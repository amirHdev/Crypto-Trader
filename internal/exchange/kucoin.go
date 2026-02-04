package exchange

import (
	"context"
	"fmt"
	"time"

	"github.com/amirhdev/internal/config"

	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/api"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/types"
)

type Client struct {
	Spot *api.SpotAPI
}

func New(cfg *config.Config) (*Client, error) {
	key := types.NewApiKeyTransport(
		cfg.KuCoin.Key,
		cfg.KuCoin.Secret,
		cfg.KuCoin.Passphrase,
		types.GlobalApiEndpoint,
	)

	transport := types.NewTransport().
		SetKeepAlive(true).
		SetMaxIdleConnsPerHost(10)

	client := api.NewApiClient(key, transport)

	return &Client{Spot: client.Spot()}, nil
}

func (c *Client) GetUSDTBalance(ctx context.Context) (float64, error) {
	// Implement via c.Spot.Account().GetAccounts(...)
	// Placeholder
	return 0, fmt.Errorf("not implemented yet")
}

func (c *Client) PlaceMarketBuy(ctx context.Context, symbol string, funds float64) (string, error) {
	// Implement real call
	return "order-id-placeholder", nil
}

// WatchPriceWithReconnect – basic version, improve with backoff
func (c *Client) WatchPrice(ctx context.Context, symbol string, handler func(price float64)) error {
	// Use SDK websocket feed client
	// This is placeholder – real impl needs ws client from SDK + ping/pong + reconnect
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Simulate price
			handler(1.2345)
			time.Sleep(3 * time.Second)
		}
	}
}
