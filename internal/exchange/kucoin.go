package exchange

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/api"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/account/account"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/spot/market"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/spot/order"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/types"
	"github.com/gorilla/websocket"

	"github.com/amirhdev/crypto-trader/internal/config"
	"github.com/rs/zerolog/log"
)

const (
	TYPE     = "trade"
	CURRENCY = "USDT"
)

type Client struct {
	apiClient api.KucoinRestService
}

func New(cfg *config.Config) (*Client, error) {
	// Build client options (as shown in official docs)
	optBuilder := types.ClientOptionBuilder{}
	optBuilder.WithKey(cfg.KuCoin.Key)
	optBuilder.WithSecret(cfg.KuCoin.Secret)
	optBuilder.WithPassphrase(cfg.KuCoin.Passphrase)
	optBuilder.WithBrokerEndpoint(types.GlobalApiEndpoint)

	// Transport options
	transportBuilder := types.TransportOptionBuilder{}
	transportBuilder.SetKeepAlive(true)
	transportBuilder.SetMaxIdleConnsPerHost(10)

	// Create client
	client := api.NewClient(optBuilder.Build())

	return &Client{
		apiClient: client.RestService(),
	}, nil
}

func (c *Client) GetUSDTBalance(ctx context.Context) (float64, error) {
	req := account.NewGetSpotAccountListReqBuilder().
		SetCurrency("USDT").
		Build()

	resp, err := c.apiClient.GetAccountService().GetAccountAPI().GetSpotAccountList(req, ctx)
	if err != nil {
		return 0, fmt.Errorf("get spot account list failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return 0, errors.New("no USDT trade account found")
	}

	bal, err := strconv.ParseFloat(resp.Data[0].Available, 64)
	if err != nil {
		return 0, fmt.Errorf("parse available balance failed: %w", err)
	}

	if bal <= 0 {
		log.Warn().Float64("balance", bal).Msg("USDT balance is zero or negative")
	}

	return bal, nil
}

func (c *Client) PlaceMarketBuy(ctx context.Context, symbol string, funds float64) (string, error) {
	clientOid, _ := rand.Int(rand.Reader, big.NewInt(1_000_000_000_000_000_000))
	oidStr := fmt.Sprintf("%d", clientOid)

	req := order.NewAddOrderReq(oidStr, symbol, "buy")

	resp, err := c.apiClient.GetSpotService().GetOrderAPI().AddOrder(req, ctx)
	if err != nil {
		return "", fmt.Errorf("place market buy failed: %w", err)
	}

	if resp.CommonResponse.Data == nil || resp.OrderId == "" {
		return "", errors.New("empty order ID in response")
	}

	return resp.OrderId, nil
}

func (c *Client) PlaceMarketSell(ctx context.Context, symbol string, size float64) (string, error) {
	clientOid, _ := rand.Int(rand.Reader, big.NewInt(1_000_000_000_000_000_000))
	oidStr := fmt.Sprintf("%d", clientOid)

	req := order.NewAddOrderReq(oidStr, symbol, "sell")

	resp, err := c.apiClient.GetSpotService().GetOrderAPI().AddOrder(req, ctx)
	if err != nil {
		return "", fmt.Errorf("place market sell failed: %w", err)
	}

	if resp.CommonResponse.Data == nil || resp.OrderId == "" {
		return "", errors.New("empty order ID in response")
	}

	return resp.OrderId, nil
}

func (c *Client) GetTickerPrice(ctx context.Context, symbol string) (float64, error) {
	req := &market.GetTickerReq{Symbol: &symbol}

	resp, err := c.apiClient.GetSpotService().GetMarketAPI().GetTicker(req, ctx)
	if err != nil {
		return 0, fmt.Errorf("get ticker failed: %w", err)
	}

	if resp.CommonResponse.Data == nil || resp.Price == "" {
		return 0, errors.New("no price in ticker response")
	}

	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("parse price failed: %w", err)
	}

	return price, nil
}

// ────────────────────────────────────────────────────────────────────────────────
//  Public Ticker Websocket with reconnection + backoff
// ────────────────────────────────────────────────────────────────────────────────

func (c *Client) WatchPrice(ctx context.Context, symbol string, handler func(price float64)) error {
	backoff := 5 * time.Second
	maxBackoff := 60 * time.Second

	for {
		err := c.connectAndWatchPublicTicker(ctx, symbol, handler)
		if err == nil || ctx.Err() != nil {
			return ctx.Err()
		}

		log.Error().
			Err(err).
			Str("symbol", symbol).
			Dur("backoff", backoff).
			Msg("websocket disconnected → reconnecting")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

func (c *Client) connectAndWatchPublicTicker(ctx context.Context, symbol string, handler func(price float64)) error {
	// 1. Get public websocket connection info
	resp, err := c.apiClient.GetSpotService().GetMarketAPI().GetPublicToken(ctx)
	if err != nil {
		return fmt.Errorf("get public bullet failed: %w", err)
	}

	if len(resp.InstanceServers) == 0 {
		return errors.New("no instance servers in public bullet response")
	}

	server := resp.InstanceServers[0]
	url := fmt.Sprintf("%s?token=%s&connectId=%d",
		server.Endpoint, resp.Token, time.Now().UnixMilli())

	// 2. Dial websocket (use gorilla/websocket or SDK wrapper)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	// 3. Subscribe to ticker
	sub := map[string]interface{}{
		"id":       time.Now().UnixMilli(),
		"type":     "subscribe",
		"topic":    "/market/ticker:" + symbol,
		"private":  false,
		"response": true,
	}
	if err := conn.WriteJSON(sub); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	// 4. Ping ticker
	pingTicker := time.NewTicker(time.Duration(server.PingInterval) * time.Millisecond)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-pingTicker.C:
			ping := map[string]interface{}{
				"id":   time.Now().UnixMilli(),
				"type": "ping",
			}
			if err := conn.WriteJSON(ping); err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}

		default:
			var raw json.RawMessage
			if err := conn.ReadJSON(&raw); err != nil {
				return fmt.Errorf("read failed: %w", err)
			}

			var msg struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				log.Debug().Err(err).Msg("invalid message format → skipping")
				continue
			}

			if msg.Type == "message" {
				var ticker struct {
					Price string `json:"price"`
				}
				if err := json.Unmarshal(msg.Data, &ticker); err == nil && ticker.Price != "" {
					p, err := strconv.ParseFloat(ticker.Price, 64)
					if err == nil {
						handler(p)
					} else {
						log.Warn().Err(err).Str("priceStr", ticker.Price).Msg("price parse failed")
					}
				}
			}
		}
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
