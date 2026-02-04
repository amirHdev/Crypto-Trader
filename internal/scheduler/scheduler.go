package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/amirdev/internal/domain"
	"github.com/amirdev/internal/exchange"
	"github.com/amirdev/internal/notifier"
	"github.com/amirdev/internal/storage"

	"github.com/rs/zerolog/log"
)

type Scheduler struct {
	ctx      context.Context
	client   *exchange.Client
	store    *storage.Storage
	notifier *notifier.Telegram
	timers   map[string]*time.Timer
	mu       sync.Mutex
}

func New(ctx context.Context, client *exchange.Client, store *storage.Storage, notifier *notifier.Telegram) *Scheduler {
	return &Scheduler{
		ctx:      ctx,
		client:   client,
		store:    store,
		notifier: notifier,
		timers:   make(map[string]*time.Timer),
	}
}

func (s *Scheduler) Restore() error {
	orders, err := s.store.GetAllOrders()
	if err != nil {
		return err
	}

	for _, ord := range orders {
		if time.Until(ord.ExecuteAt) > 0 {
			s.schedule(ord)
		} else {
			// cleanup old
			s.store.DeleteOrder(ord.ID())
		}
	}
	return nil
}

func (s *Scheduler) AddOrder(ord domain.Order) error {
	if err := s.store.SaveOrder(ord); err != nil {
		return err
	}
	s.schedule(ord)
	return nil
}

func (s *Scheduler) schedule(ord domain.Order) {
	delay := time.Until(ord.ExecuteAt)
	if delay <= 0 {
		return
	}

	s.mu.Lock()
	timer := time.AfterFunc(delay, func() { s.executeOrder(ord) })
	s.timers[ord.ID()] = timer
	s.mu.Unlock()

	log.Info().Str("order", ord.ID()).Dur("delay", delay).Msg("scheduled")
	s.notifier.Send(fmt.Sprintf("New order scheduled: %s @ %s", ord.Coin, ord.ExecuteAt.Format(time.RFC3339)))
}

func (s *Scheduler) executeOrder(ord domain.Order) {
	s.mu.Lock()
	delete(s.timers, ord.ID())
	s.mu.Unlock()

	s.notifier.Send(fmt.Sprintf("Executing BUY order for %s", ord.Coin))

	// 1. Get available USDT
	balance, err := s.client.GetUSDTBalance(s.ctx)
	if err != nil || balance <= 0 {
		s.notifier.Send("Balance check failed or insufficient USDT")
		return
	}

	funds := balance * (ord.WalletPerc / 100)

	orderID, err := s.client.PlaceMarketBuy(s.ctx, ord.Coin, funds)
	if err != nil {
		s.notifier.Send(fmt.Sprintf("BUY failed: %v", err))
		return
	}

	s.notifier.Send(fmt.Sprintf("BUY placed: %s (order %s)", ord.Coin, orderID))

	// 2. Start price watcher (sell logic)
	go s.watchForSell(ord, funds)
}

func (s *Scheduler) watchForSell(ord domain.Order, boughtFunds float64) {
	// Placeholder – real version would use websocket
	initialPrice := 1.0 // fetch once
	target := initialPrice * (1 + ord.SellPerc/100)

	s.client.WatchPrice(s.ctx, ord.Coin, func(current float64) {
		if current >= target {
			// place sell
			s.notifier.Send(fmt.Sprintf("SELL triggered for %s @ %.4f", ord.Coin, current))
			// implement sell
		}
	})
}
