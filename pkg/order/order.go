package order

import "lfg/pkg/types"

type Order struct {
	Id           string
	Symbol       string
	OrderType    types.OrderType
	OrderSide    types.OrderSide
	Price        float64
	OriginalQty  float64
	RemainingQty float64
}

func New(id string, symbol string) (*Order, error) {
	return &Order{
		Id:     id,
		Symbol: symbol,
	}, nil
}
