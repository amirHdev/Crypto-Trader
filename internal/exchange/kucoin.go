package exchange

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/api"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/account/account"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/market"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/spot/order"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/generate/websocket"
	"github.com/Kucoin/kucoin-universal-sdk/sdk/golang/pkg/types"
	"github.com/amirhdev/crypto-trader/internal/config"
)

type Client struct {
	apiClient *api.ApiClient
	spot      *api.SpotAPI
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
	return &Client{
		apiClient: client,
		spot:      client.Spot(),
	}, nil
}

func (c *Client) GetUSDTBalance(ctx context.Context) (float64, error) {
	req := &account.GetSpotAccountListReq{Currency: "USDT", Type: "trade"}
	resp, err := c.spot.Account().GetSpotAccountList(req, ctx)
	if err != nil {
		return 0, err
	}
	if len(resp.Data) == 0 {
		return 0, errors.New("no USDT account found")
	}
	bal, err := strconv.ParseFloat(resp.Data[0].Available, 64)
	if err != nil {
		return 0, err
	}
	return bal, nil
}

func (c *Client) PlaceMarketBuy(ctx context.Context, symbol string, funds float64) (string, error) {
	clientOid := fmt.Sprintf("%d", rand.Int())
	req := order.NewAddOrderReq(clientOid, "buy", symbol, "market").
		SetFunds(fmt.Sprintf("%.2f", funds))
	resp, err := c.spot.Order().AddOrder(req, ctx)
	if err != nil {
		return "", err
	}
	return resp.Data.OrderID, nil
}

func (c *Client) PlaceMarketSell(ctx context.Context, symbol string, size float64) (string, error) {
	clientOid := fmt.Sprintf("%d", rand.Int())
	req := order.NewAddOrderReq(clientOid, "sell", symbol, "market").
		SetSize(fmt.Sprintf("%.8f", size))
	resp, err := c.spot.Order().AddOrder(req, ctx)
	if err != nil {
		return "", err
	}
	return resp.Data.OrderID, nil
}

func (c *Client) GetTickerPrice(ctx context.Context, symbol string) (float64, error) {
	req := &market.GetTickerReq{Symbol: symbol}
	resp, err := c.spot.Market().GetTicker(req, ctx)
	if err != nil {
		return 0, err
	}
	price, err := strconv.ParseFloat(resp.Data.Price, 64)
	if err != nil {
		return 0, err
	}
	return price, nil
}

func (c *Client) WatchPrice(ctx context.Context, symbol string, handler func(price float64)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := c.connectAndSubscribe(ctx, symbol, handler)
			if err != nil {
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (c *Client) connectAndSubscribe(ctx context.Context, symbol string, handler func(price float64)) error {
	req := &websocket.GetBulletPrivateReq{}
	tokenResp, err := c.apiClient.WebSocket().GetBulletPrivate(req, ctx)
	if err != nil {
		return err
	}
	wsClient := websocket.NewWebSocketClient(tokenResp.Data.Token, tokenResp.Data.InstanceServers[0].Endpoint)
	messageChan, errorChan, err := wsClient.Connect()
	if err != nil {
		return err
	}
	subTopic := fmt.Sprintf("/market/ticker:%s", symbol)
	sub := websocket.NewSubscribeMessage(subTopic, true)
	if err := wsClient.Subscribe(sub); err != nil {
		return err
	}
	ticker := time.NewTicker(tokenResp.Data.InstanceServers[0].PingInterval * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			wsClient.Close()
			returctx.Err()
		case ms := <-messageChan:
			// Parse msg
			var tickerData struct {
				Type  string `json:"type"`
				Topic string `json:"topic"`
			}
			// Parse msg
			var tickerData struct {
				Type    string `json:"type"`
				Topic   string `json:"topic"`
				Subject string `json:"subject"`
				Data    struct {
					Price string `json:"price"`
				} `json:"data"`
			}
			if err := json.Unmarshal(msg, &tickerData); err == nil && tickerData.Type == "message" && tickerData.Subject == "trade.ticker" {
				price, _ := strconv.ParseFloat(tickerData.Data.Price, 64)
				handler(price)
			}
		case err := <-errorChan:
			return err
		case <-ticker.C:
			ping := map[string]interface{}{"id": fmt.Sprintf("%d", time.Now().UnixMilli()), "type": "ping"}
			pingBytes, _ := json.Marshal(ping)
			wsClient.Send(pingBytes)
		}
	}
}
