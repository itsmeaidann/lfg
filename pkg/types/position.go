package types

type Position struct {
	EntryPrice float64   `json:"entry_price"`
	Qty        float64   `json:"qty"`
	Side       OrderSide `json:"side"`
}
