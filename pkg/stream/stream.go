package stream

import (
	"lfg/pkg/order"
	"lfg/pkg/types"
)

type Stream interface {
	ConnectAndSubscribe(params map[string]string, cb func(e []byte)) (doneC chan struct{}, stopC chan struct{}, err error)
	Close()
	IsClosed() bool

	// @dev:
	// for order mgmt stream; normal read-only stream should not use this to avoid concurrent writes
	OpenLimitOrder(symbol string, orderSide types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, orderTif types.OrderTIF, cloId string) (string, error)
	OpenMarketOrder(symbol string, side types.OrderSide, qty float64, lev int, reduceOnly bool) error
	OpenBatchLimitOrders(symbol string, inputs []types.LimitOrderInput, lev int) error // TODO: return orderIds []string
	ModifyOrder(symbol string, oId string, cloId string, orderSide types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, orderTif types.OrderTIF) error
	CancelOrder(symbol string, orderId string, cloId string) error
	CancelBatchOrders(symbol string, orderIds []string) error
	GetPendingOrders(symbol string) ([]order.Order, error)
}
