package exchange

import (
	"context"
	"errors"
	"lfg/config"
	"lfg/pkg/exchange/bnf"
	"lfg/pkg/exchange/hpl"
	"lfg/pkg/market"
	"lfg/pkg/order"
	"lfg/pkg/stream"
	"lfg/pkg/types"
)

type Exchange interface {
	Name() types.ExchangeName
	GetMarket(symbol string) *market.Market

	GetPendingOrders(symbol string) ([]order.Order, error)
	OpenMarketOrder(symbol string, side types.OrderSide, qty float64, lev int, reduceOnly bool) error
	OpenLimitOrder(symbol string, side types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, tif types.OrderTIF, cloId string) (string, error)
	OpenBatchLimitOrders(symbol string, inputs []types.LimitOrderInput, lev int) ([]string, error)
	CancelOrder(symbol string, orderId string, cloId string) error
	CancelAllOrders(symbol string) error
	CancelBatchOrders(symbol string, orderIds []string) error
	GetKLines(symbol string, interval types.Interval, window int) ([]types.KLineEvent, error)
	GetAccountBalance() (float64, error) // in USD
	GetActivePositionByMarket(symbol string) ([]types.Position, error)
	CloseActivePositionByMarket(symbol string, lev int) error

	// ╔═════ WS callback functions ═════╗
	// - onConn(): invoked when ws connection is established for the 1st time, NOT when reconnecting
	// - onEvent(): invoked when a relevant event is received through the ws
	// - onClose(): invoked when ws connection is closed intentionally, NOT when connection is lost unexpectedly
	// ╚═════════════════════════════════╝
	ConnectOrderMgmtStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.OrderEvent), onClose func(stream.Stream)) (stream.Stream, error)
	SubscribeTradeStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.TradeEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error)
	SubscribeKLineStream(ctx context.Context, symbol string, interval types.Interval, onConn func(stream.Stream), onEvent func(stream.Stream, types.KLineEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error)
	SubscribeMarkPriceStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.MarkPriceEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error)
	SubscribeBookDepthStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.BookDepthEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error)
	SubscribeOrderStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.OrderEvent), onClose func(stream.Stream)) (stream.Stream, error) // maxDelayMs is not valid with order update (as every event is crucial)

	ToUniSymbol(locSymbol string) string
	ToLocSymbol(uniSymbol string) string
}

// creates a new exchange instance based on the provided name and credentials
func NewExchange(exchgId string, exchgConfig *config.ExchangeConfig) (Exchange, error) {
	switch exchgConfig.ExchangeName {
	case types.ExchangeBnf:
		return bnf.New(exchgConfig)
	case types.ExchangeHpl:
		return hpl.New(exchgConfig)
	default:
		return nil, errors.New("unsupported exchange")
	}
}
