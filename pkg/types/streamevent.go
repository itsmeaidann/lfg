package types

import (
	"time"
)

type TradeEvent struct {
	Event        string
	Time         time.Time
	Symbol       string
	Price        float64
	Quantity     float64
	Side         string
	ReceivedTime time.Time
}

type KLine struct {
	O float64
	H float64
	L float64
	C float64
}

type MarkPriceEvent struct {
	Event        string
	Time         time.Time
	Symbol       string
	Price        float64
	ReceivedTime time.Time
}

type KLineEvent struct {
	Event        string
	OpenTime     time.Time
	CloseTime    time.Time
	Symbol       string
	Kline        KLine
	ReceivedTime time.Time
}

type BookDepthEvent struct {
	Event        string
	Time         time.Time
	Symbol       string
	Bids         []Bid
	Asks         []Ask
	ReceivedTime time.Time
}

type OrderEvent struct {
	Event        string
	Time         time.Time
	Symbol       string
	OId          string // order ID
	ClientOId    string // client-specified order ID
	Side         OrderSide
	IsReduceOnly bool
	OrderStatus  OrderStatus
	Price        float64
	OrigQty      float64
	OrderTif     OrderTIF
	OrderType    OrderType

	// fields below valid once order is filled
	AvgPrice    float64
	FilledQty   float64
	RealizedPnL float64
	Fee         float64 // transaction fee
	FeeAsset    string  // asset used for fee e.g. USDT
}

type Bid struct {
	Price float64
	Qty   float64
}

type Ask struct {
	Price float64
	Qty   float64
}
