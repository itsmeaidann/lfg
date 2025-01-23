package bnf

import (
	"context"
	"encoding/json"
	"fmt"
	"lfg/config"
	"lfg/pkg/market"
	"lfg/pkg/order"
	"lfg/pkg/stream"
	"lfg/pkg/types"
	"lfg/pkg/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

type BnfExchange struct {
	BnfConfig *bnfConfig

	sClient *binance.Client
	fClient *futures.Client

	Markets      map[string]*market.Market
	SymbolMapU2L map[string]string
	SymbolMapL2U map[string]string

	StopStreamC map[string]map[types.Stream]chan struct{}
}

func New(exchgConfig *config.ExchangeConfig) (*BnfExchange, error) {
	// (1) environment
	binance.UseTestnet = config.Env.EnvName != types.EnvProd
	futures.UseTestnet = config.Env.EnvName != types.EnvProd
	configFile := "bnf.test.json"
	if config.Env.EnvName == types.EnvProd {
		configFile = "bnf.prod.json"
	}

	// (2) load symbol
	symbolMapU2L := utils.LoadExchangeSymbolMap(string(types.ExchangeBnf))
	symbolMapL2U := utils.ReverseStrMap(symbolMapU2L)

	// (3) validate config
	key := utils.LoadEnv(exchgConfig.EnvPrefix + "_API_KEY")
	secret := utils.LoadEnv(exchgConfig.EnvPrefix + "_API_SECRET")
	if key == "" || secret == "" {
		return nil, fmt.Errorf("API key or secret is not set: prefix %v", exchgConfig.EnvPrefix)
	}
	sClient := binance.NewClient(key, secret)
	fClient := futures.NewClient(key, secret)

	rawConfig, err := os.ReadFile(filepath.Join("pkg", "exchange", "bnf", "config", configFile))
	if err != nil {
		return nil, err
	}
	var bnfConfig bnfConfig
	if err := json.Unmarshal(rawConfig, &bnfConfig); err != nil {
		return nil, err
	}

	// (4) load markets
	markets, err := loadMarkets(fClient)
	if err != nil {
		return nil, err
	}

	return &BnfExchange{
		BnfConfig:    &bnfConfig,
		sClient:      sClient,
		fClient:      fClient,
		SymbolMapU2L: symbolMapU2L,
		SymbolMapL2U: symbolMapL2U,
		Markets:      markets,
		StopStreamC:  make(map[string]map[types.Stream]chan struct{}),
	}, nil
}

func (e *BnfExchange) Name() types.ExchangeName {
	return types.ExchangeBnf
}

// ╔═════════════╗
//       Info
// ╚═════════════╝

func (e *BnfExchange) GetMarket(symbol string) *market.Market {
	symbol = e.ToLocSymbol(symbol)
	if market, exists := e.Markets[symbol]; exists {
		return market
	}
	return nil
}

func (e *BnfExchange) getListenKey() (string, error) {
	listenKey, err := e.fClient.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		return "", err
	}
	return listenKey, nil
}

// ╔═════════════╗
//      Price
// ╚═════════════╝

func (e *BnfExchange) GetMarkPrice(symbol string) (float64, error) {
	res, err := e.fClient.NewPremiumIndexService().Do(context.Background())
	if err != nil {
		return 0, err
	}
	for _, price := range res {
		if price.Symbol == symbol {
			return utils.StrToFloat(price.MarkPrice)
		}
	}
	return 0, fmt.Errorf("bad symbol: %s", symbol)
}

func (e *BnfExchange) GetKLines(symbol string, interval types.Interval, window int) ([]types.KLineEvent, error) {
	symbol = e.ToLocSymbol(symbol)
	res, err := e.fClient.NewKlinesService().
		Symbol(symbol).
		Interval(string(interval)).
		Limit(window).
		Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no klines data available")
	}

	kLines, err := ParseKLines(res, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to parse klines: %w", err)
	}
	return kLines, nil
}

// ╔═════════════╗
//      Order
// ╚═════════════╝

func (e *BnfExchange) CancelOrder(symbol string, orderId string, cloId string) error {
	return fmt.Errorf("not implemented")
}

func (e *BnfExchange) CancelBatchOrders(symbol string, orderIds []string) error {
	return fmt.Errorf("not implemented")
}

func (e *BnfExchange) CancelAllOrders(symbol string) error {
	return fmt.Errorf("not implemented")
}

func (e *BnfExchange) GetPendingOrders(symbol string) ([]order.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *BnfExchange) OpenMarketOrder(symbol string, orderSide types.OrderSide, qty float64, lev int, reduceOnly bool) error {
	// TODO: use e.markets to filter invalid params
	symbol = e.ToLocSymbol(symbol)
	side, err := convertOrderSide(orderSide)
	if err != nil {
		return err
	}
	_, err = e.fClient.NewCreateOrderService().
		Symbol(symbol).
		Type(futures.OrderTypeMarket).
		Side(side).
		Quantity(fmt.Sprintf("%f", qty)).
		ReduceOnly(reduceOnly).
		Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (e *BnfExchange) OpenLimitOrder(symbol string, orderSide types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, orderTif types.OrderTIF, cloId string) (string, error) {
	// TODO:
	// - use e.markets to filter invalid params
	// - enforce cloId usage
	side, err := convertOrderSide(orderSide)
	if err != nil {
		return "", err
	}
	tif, err := convertOrderTIF(orderTif)
	if err != nil {
		return "", err
	}
	res, err := e.fClient.NewCreateOrderService().
		Symbol(symbol).
		Type(futures.OrderTypeLimit).
		Side(side).
		Price(utils.FloatToStr(price)).
		Quantity(utils.FloatToStr(qty)).
		ReduceOnly(reduceOnly).
		TimeInForce(tif).
		Do(context.Background())
	if err != nil {
		return "", err
	}

	oId := strconv.FormatInt(res.OrderID, 10)
	if oId == "" {
		return "", fmt.Errorf("oId is missing from the response")
	}
	return oId, nil
}

func (e *BnfExchange) OpenBatchLimitOrders(symbol string, inputs []types.LimitOrderInput, lev int) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// ╔═══════════════════╗
//    OrderMgmtStream
// ╚═══════════════════╝

func (e *BnfExchange) ConnectOrderMgmtStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.OrderEvent), onClose func(stream.Stream)) (stream.Stream, error) {
	return nil, fmt.Errorf("not implemented")
}

// ╔══════════════╗
//    TradeSteam
// ╚══════════════╝

func (e *BnfExchange) SubscribeTradeStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.TradeEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	bnfWsEndpoint := fmt.Sprintf("%s/%s@aggTrade", e.BnfConfig.WsUrl, symbol)

	// connect bnfStream
	bnfStream, err := NewStream(ctx, types.StreamTrade, e, bnfWsEndpoint, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := bnfStream.ConnectAndSubscribe(map[string]string{}, func(e []byte) {
		evt, err := parseTradeEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay
		delayMs := time.Now().UnixMilli() - evt.Time.UnixMilli()
		if (delayMs > maxDelayMs || evt == types.TradeEvent{}) {
			return
		}
		onEvent(bnfStream, evt)
	})
	if err != nil {
		log.Errorf("fail to connect and subscribe: %v", err)
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()
	return bnfStream, nil
}

// ╔══════════════╗
//    KLineSteam
// ╚══════════════╝

func (e *BnfExchange) SubscribeKLineStream(ctx context.Context, symbol string, interval types.Interval, onConn func(stream.Stream), onEvent func(stream.Stream, types.KLineEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	bnfWsEndpoint := fmt.Sprintf("%s/%s@kline_%s", e.BnfConfig.WsUrl, strings.ToLower(symbol), interval)

	// connect bnfStream
	bnfStream, err := NewStream(ctx, types.StreamKLine, e, bnfWsEndpoint, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := bnfStream.ConnectAndSubscribe(map[string]string{}, func(e []byte) {
		evt, err := ParseKLineEvent(symbol, e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay
		delayMs := time.Now().UnixMilli() - evt.CloseTime.UnixMilli()
		if (delayMs > maxDelayMs || evt == types.KLineEvent{}) {
			return
		}
		onEvent(bnfStream, evt)
	})
	if err != nil {
		log.Errorf("fail to connect and subscribe: %v", err)
		return nil, err
	}

	// wait
	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()
	return bnfStream, nil
}

// ╔═══════════════════╗
//    MarkPriceStream
// ╚═══════════════════╝

func (e *BnfExchange) SubscribeMarkPriceStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.MarkPriceEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	bnfWsEndpoint := fmt.Sprintf("%s/%s@markPrice@1s", e.BnfConfig.WsUrl, strings.ToLower(symbol))

	// connect bnfStream
	bnfStream, err := NewStream(ctx, types.StreamMarkPrice, e, bnfWsEndpoint, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := bnfStream.ConnectAndSubscribe(map[string]string{}, func(e []byte) {
		evt, err := parseMarkPriceEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay
		delayMs := time.Now().UnixMilli() - evt.Time.UnixMilli()
		if (delayMs > maxDelayMs || evt == types.MarkPriceEvent{}) {
			return
		}
		onEvent(bnfStream, evt)
	})
	if err != nil {
		log.Errorf("fail to connect and subscribe: %v", err)
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return bnfStream, nil
}

// ╔═══════════════════╗
//    BookDepthStream
// ╚═══════════════════╝

func (e *BnfExchange) SubscribeBookDepthStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.BookDepthEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	// @dev: fixed to fastest updates (every 100ms) & largest depth (20 levels)
	bnfWsEndpoint := fmt.Sprintf("%s/%s@depth20@100ms", e.BnfConfig.WsUrl, strings.ToLower(symbol))

	// connect bnfStream
	bnfStream, err := NewStream(ctx, types.StreamBookDepth, e, bnfWsEndpoint, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := bnfStream.ConnectAndSubscribe(map[string]string{}, func(e []byte) {
		evt, err := parseBookDepthEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay
		delayMs := time.Now().UnixMilli() - evt.Time.UnixMilli()
		if delayMs > maxDelayMs || evt.Event == "" {
			return
		}
		onEvent(bnfStream, evt)
	})
	if err != nil {
		log.Errorf("fail to connect and subscribe: %v", err)
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return bnfStream, nil
}

// ╔═══════════════╗
//    OrderStream
// ╚═══════════════╝

func (e *BnfExchange) SubscribeOrderStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.OrderEvent), onClose func(stream.Stream)) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	listenKey, err := e.getListenKey()
	if err != nil {
		return nil, err
	}
	bnfWsEndpoint := fmt.Sprintf("%s/%s", e.BnfConfig.WsUrl, listenKey)

	// connect bnfStream
	bnfStream, err := NewStream(ctx, types.StreamOrder, e, bnfWsEndpoint, onConn, onClose)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	doneC, stopC, err := bnfStream.ConnectAndSubscribe(map[string]string{}, func(e []byte) {
		// process only "ORDER_TRADE_UPDATE" event; ignore others
		if err := json.Unmarshal(e, &data); err != nil {
			log.Errorf("fail to unmarshal order stream event from []byte: %v", err)
			return
		}
		if evtName, ok := data["e"].(string); !ok || evtName != "ORDER_TRADE_UPDATE" {
			return
		}
		evt, err := parseOrderEvent(e)
		if err != nil {
			log.Errorf("fail to parse order event: %v: %v", string(e), err)
			return
		}
		if (evt.Symbol != symbol || evt == types.OrderEvent{}) {
			return
		}
		onEvent(bnfStream, evt)
	})
	if err != nil {
		log.Errorf("fail to connect and subscribe: %v", err)
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return bnfStream, nil
}

func (e *BnfExchange) ToUniSymbol(locSymbol string) string {
	if locSymbol, ok := e.SymbolMapL2U[locSymbol]; ok {
		return locSymbol
	}
	log.Fatalf("fail to convert local symbol to universal symbol: %v", locSymbol)
	return ""
}

func (e *BnfExchange) ToLocSymbol(uniSymbol string) string {
	if locSymbol, ok := e.SymbolMapU2L[uniSymbol]; ok {
		return locSymbol
	}
	log.Fatalf("fail to convert universal symbol to local symbol: %v", uniSymbol)
	return ""
}

func (e *BnfExchange) GetAccountBalance() (float64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *BnfExchange) GetActivePositionByMarket(symbol string) ([]types.Position, error) {
	symbol = e.ToLocSymbol(symbol)
	account, err := e.fClient.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fail to get active positions: %w", err)
	}

	var positions []types.Position
	for _, pos := range account.Positions {
		if pos.Symbol != symbol || pos.PositionAmt == "0" {
			continue
		}
		qty, err := utils.StrToFloat(pos.PositionAmt)
		if err != nil {
			return nil, fmt.Errorf("fail to convert position qty: %v", err)
		}
		entryPrice, err := utils.StrToFloat(pos.EntryPrice)
		if err != nil {
			return nil, fmt.Errorf("fail to convert entry price: %v", err)
		}
		posSide := types.OrderSideBuy
		if qty < 0 {
			posSide = types.OrderSideSell
		}
		positions = append(positions, types.Position{
			Qty:        qty,
			EntryPrice: entryPrice,
			Side:       posSide,
		})
	}
	return positions, nil
}

func (e *BnfExchange) CloseActivePositionByMarket(symbol string, lev int) error {
	return fmt.Errorf("not implemented")
}
