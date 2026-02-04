package domain

import "time"

type Order struct {
	ID         string    `json:"id"`
	Coin       string    `json:"coin_name"`
	WalletPerc float64   `json:"wallet_per"`
	SellPerc   float64   `json:"sell_per"`
	ExecuteAt  time.Time `json:"execute_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func (o Order) ID() string {
	return o.Coin + "-" + o.ExecuteAt.Format("20060102-150405")
}
