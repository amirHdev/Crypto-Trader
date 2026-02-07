package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirhdev/crypto-trader/internal/config"
	"github.com/amirhdev/crypto-trader/internal/exchange"
	"github.com/amirhdev/crypto-trader/internal/logging"
	"github.com/amirhdev/crypto-trader/internal/notifier"
	"github.com/amirhdev/crypto-trader/internal/scheduler"
	"github.com/amirhdev/crypto-trader/internal/storage"
	"github.com/amirhdev/crypto-trader/internal/ui"

	"github.com/rs/zerolog/log"
)

func main() {
	logger := logging.New()
	log.Logger = *logger

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config load failed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := storage.New(cfg.DataDir)
	if err != nil {
		log.Fatal().Err(err).Msg("storage failed")
	}
	defer store.Close()

	kc, err := exchange.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("kucoin client failed")
	}

	tg := notifier.New(cfg)
	sched := scheduler.New(ctx, kc, store, tg)
	if err := sched.Restore(); err != nil {
		log.Warn().Err(err).Msg("partial restore failure")
	}

	go ui.Run(ctx, sched, store, cfg, tg)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Info().Msg("Shutdown initiated...")
	cancel()

	time.Sleep(2 * time.Second)
	log.Info().Msg("Bye.")
}
