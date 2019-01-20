package exchangeconsumer

import (
	"time"
)

type GdaxMessage struct {
	Sequence  uint64    `json:"sequence"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"time"`
	ProductId string    `json:"product_id"`
	ClientId  string    `json:"client_oid,omitempty"`
	OrderId   string    `json:"order_id,omitempty"`
	OrderType string    `json:"order_type,omitempty"`
	Bid       string    `json:"bid,omitempty"`
	Ask       string    `json:"ask,omitempty"`
	Size      string    `json:"size,omitempty"`
	Price     string    `json:"price"`
	Funds     string    `json:"funds,omitempty"`
	Side      string    `json:"side,omitempty"`
	Volume    string    `json:"volume,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

type OandaMessage struct {
	Type        string                   `json:"type"`
	Timestamp   time.Time                `json:"time"`
	Bids        []map[string]interface{} `json:"bids,omitempty"`
	Asks        []map[string]interface{} `json:"asks,omitempty"`
	CloseoutBid string                   `json:"closeoutBid,omitempty"`
	CloseoutAsk string                   `json:"closeoutAsk,omitempty"`
	Status      string                   `json:"status,omitempty"`
	Tradeable   bool                     `json:"tradeable,omitempty"`
	Instrument  string                   `json:"instrument,omitempty"`
}
