package types

type OrderSide string

const (
	OrderSideBuy  = OrderSide("buy")
	OrderSideSell = OrderSide("sell")
)

type OrderTIF string // TimeInForce

const (
	OrderTIFGTC = OrderTIF("GTC") // Good 'Til Canceled
	OrderTIFGTX = OrderTIF("GTX") // Good 'Till Crossing
	OrderTIFIOC = OrderTIF("IOC") // Immediate or Cancel
	OrderTIFFOK = OrderTIF("FOK") // Fill or Kill
	OrderTIFALO = OrderTIF("ALO") // Add Liquidity Only (Post-only)
)

type OrderType string

const (
	OrderLimit  = OrderType("limit")
	OrderMarket = OrderType("market")
)

type OrderStatus string

const (
	OrderStatusNew           = OrderStatus("new")
	OrderStatusPartialFilled = OrderStatus("partial_filled")
	OrderStatusFilled        = OrderStatus("filled")
	OrderStatusCanceled      = OrderStatus("canceled")
	OrderStatusRejected      = OrderStatus("rejected")
	OrderStatusExpired       = OrderStatus("expired")
)

type LimitOrderInput struct {
	Side  OrderSide `json:"side"`
	Price float64   `json:"price"`
	Qty   float64   `json:"qty"`
	Tif   OrderTIF  `json:"tif"`
}
