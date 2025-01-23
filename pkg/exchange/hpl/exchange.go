package hpl

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"lfg/config"
	"lfg/pkg/exchange/bnf"
	"lfg/pkg/http"
	"lfg/pkg/market"
	"lfg/pkg/order"
	"lfg/pkg/stream"
	"lfg/pkg/types"
	"lfg/pkg/utils"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

type HplExchange struct {
	HplConfig *hplConfig
	IsMainnet bool

	Markets      map[string]*market.Market
	SymbolMapU2L map[string]string
	SymbolMapL2U map[string]string

	AccountPrivKey  *ecdsa.PrivateKey
	AccountAddress  common.Address
	AccountLeverage map[string]int

	IsUseBnfKLines bool
	BnfClient      *futures.Client
}

func New(exchgConfig *config.ExchangeConfig) (*HplExchange, error) {
	// (1) environment
	configFile := "hpl.test.json"
	isMainnet := false
	if config.Env.EnvName == types.EnvProd {
		configFile = "hpl.prod.json"
		isMainnet = true
	}

	// (2) load symbol
	symbolMapU2L := utils.LoadExchangeSymbolMap(string(types.ExchangeHpl))
	symbolMapL2U := utils.ReverseStrMap(symbolMapU2L)

	// (3) load config
	rawConfig, err := os.ReadFile(filepath.Join("pkg", "exchange", "hpl", "config", configFile))
	if err != nil {
		return nil, err
	}
	var hplConfig hplConfig
	if err := json.Unmarshal(rawConfig, &hplConfig); err != nil {
		return nil, err
	}

	privKey, err := crypto.HexToECDSA(utils.LoadEnv(exchgConfig.EnvPrefix + "_PRIVATE_KEY"))
	if err != nil {
		return nil, err
	}
	isUseBnfKLines := utils.LoadBoolEnvWithDefault(exchgConfig.EnvPrefix + "_USE_BNF_KLINES")

	pubKey, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("fail to parse private key to public key via ECDSA")
	}
	address := crypto.PubkeyToAddress(*pubKey)

	// (4) load markets
	markets, err := loadMarkets(hplConfig.ApiUrl)
	if err != nil {
		return nil, err
	}

	// TODO: find a way to assign default accountLeverage
	accountLeverage := make(map[string]int)

	// (5) init bnf client
	bnfClient := futures.NewClient("", "")

	hplExchange := &HplExchange{
		HplConfig:       &hplConfig,
		IsMainnet:       isMainnet,
		SymbolMapU2L:    symbolMapU2L,
		SymbolMapL2U:    symbolMapL2U,
		Markets:         markets,
		AccountPrivKey:  privKey,
		AccountAddress:  address,
		AccountLeverage: accountLeverage,
		IsUseBnfKLines:  isUseBnfKLines,
		BnfClient:       bnfClient,
	}
	return hplExchange, nil
}

func (*HplExchange) Name() types.ExchangeName {
	return types.ExchangeHpl
}

// ╔═════════════╗
//       Info
// ╚═════════════╝

func (e *HplExchange) GetMarket(symbol string) *market.Market {
	symbol = e.ToLocSymbol(symbol)
	if market, exists := e.Markets[symbol]; exists {
		return market
	}
	return nil
}

// ╔═════════════╗
//
//	Price
//
// ╚═════════════╝
func (e *HplExchange) GetBnfKLines(symbol string, interval types.Interval, window int) ([]types.KLineEvent, error) {
	// ============ Use BNF KLine ============
	bnfSymbol := strings.ReplaceAll(symbol, "_", "") + "T"
	res, err := e.BnfClient.NewKlinesService().
		Symbol(bnfSymbol).
		Interval(string(interval)).
		Limit(window).
		Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("no klines data available")
	}

	kLines, err := bnf.ParseKLines(res, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to parse klines: %w", err)
	}
	return kLines, nil
}

func (e *HplExchange) GetKLines(symbol string, interval types.Interval, window int) ([]types.KLineEvent, error) {
	// convert
	if e.IsUseBnfKLines {
		return e.GetBnfKLines(symbol, interval, window)
	} else {
		symbol = e.ToLocSymbol(symbol)
		intervalDuration, err := utils.IntervalToDuration(interval)
		if err != nil {
			return nil, err
		}

		windowDuration := time.Duration(window+1) * intervalDuration
		startTime := time.Now().Add(-windowDuration).UnixMilli()
		endTime := time.Now().Add(intervalDuration).UnixMilli()

		// params
		request := map[string]interface{}{
			"coin":      symbol,
			"interval":  string(interval),
			"startTime": startTime,
			"endTime":   endTime,
		}
		req := map[string]interface{}{
			"type": "candleSnapshot",
			"req":  request,
		}
		reqBody, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		// POST request
		status, resBody, err := http.PostRequest(fmt.Sprintf("%s/info", e.HplConfig.ApiUrl), "", reqBody)
		if err != nil {
			return nil, err
		}
		if status != "200 OK" {
			return nil, fmt.Errorf("status: %v: %v", status, string(resBody))
		}
		// check response
		var kLinesRes []kLineResponse
		if err := json.Unmarshal(resBody, &kLinesRes); err != nil {
			return nil, err
		}

		klines, err := e.parseKLines(kLinesRes)
		if err != nil {
			return nil, err
		}
		return klines, nil
	}
}

// ╔═════════════╗
//      Order
// ╚═════════════╝

func (e *HplExchange) GetPendingOrders(symbol string) ([]order.Order, error) {
	// convert
	symbol = e.ToLocSymbol(symbol)

	// params
	req := metadataRequest{
		Type: "openOrders",
		User: e.AccountAddress.String(),
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/info", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return nil, err
	}
	if status != "200 OK" {
		return nil, fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var pendingOrderRes []pendingOrderResponse
	if err := json.Unmarshal(resBody, &pendingOrderRes); err != nil {
		return nil, err
	}

	orders := make([]order.Order, 0)
	for _, pendingOrder := range pendingOrderRes {
		order, err := e.parsePendingOrder(pendingOrder)
		if err != nil {
			return nil, err
		}
		// HPL, well, only returns symbol name in this endpoint e.g. "BTC" not "BTC/USD"
		if order.Symbol == strings.Split(symbol, "/")[0] {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (e *HplExchange) OpenMarketOrder(symbol string, side types.OrderSide, qty float64, lev int, reduceOnly bool) error {
	if e.AccountLeverage[e.SymbolMapU2L[symbol]] != lev {
		if err := e.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return err
		}
	}
	isBuy := side == types.OrderSideBuy

	// get mid price
	kLines, err := e.GetKLines(symbol, types.Interval1m, 1)
	if err != nil {
		return err
	}
	if len(kLines) == 0 {
		return fmt.Errorf("cannot get klines")
	}
	midPrice := kLines[len(kLines)-1].Kline.C
	limitPrice := midPrice * 0.9
	if isBuy {
		limitPrice = midPrice * 1.1
	}
	limitPrice = utils.RoundToSigFigs(limitPrice, MAX_PRICE_SIG_FIGURE)

	symbol = e.ToLocSymbol(symbol)
	orderType := orderTypeWire{
		Limit: &limit{
			Tif: tifTypeIOC,
		},
	}

	marketIdx, err := e.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return err
	}

	nonce := getNonce()
	order := orderWire{
		Asset:      marketIdx,
		IsBuy:      isBuy,
		LimitPx:    utils.FloatToStr(limitPrice),
		SizePx:     utils.FloatToStr(qty),
		ReduceOnly: reduceOnly,
		OrderType:  orderType,
	}
	action := orderAction{
		Type:     "order",
		Orders:   []orderWire{order},
		Grouping: string(groupingNa),
	}

	signature, err := e.getRequestSignature(action, "", nonce)
	if err != nil {
		return fmt.Errorf("fail to get signature when open market order: %v", err)
	}
	req := orderActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: nil,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/exchange", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return err
	}
	if status != "200 OK" {
		return fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res openOrderResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		return err
	}
	var errs []string
	for _, status := range res.Response.Data.Statuses {
		if errMsg := status.Error; errMsg != "" {
			errs = append(errs, errMsg)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("fail to open market order: %s", strings.Join(errs, "; "))
	}

	oId := strconv.FormatInt(res.Response.Data.Statuses[0].Resting.Oid, 10)
	if oId == "" {
		return fmt.Errorf("oId is missing from the response")
	}
	return nil

}

func (e *HplExchange) OpenLimitOrder(symbol string, side types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, tif types.OrderTIF, cloId string) (string, error) {
	if e.AccountLeverage[e.SymbolMapU2L[symbol]] != lev {
		if err := e.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return "", err
		}
	}

	// convert
	symbol = e.ToLocSymbol(symbol)
	// ref: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
	price = utils.RoundToSigFigs(price, MAX_PRICE_SIG_FIGURE)
	orderTif, err := convertOrderTif(tif)
	if err != nil {
		return "", err
	}
	orderType := orderTypeWire{
		Limit: &limit{
			Tif: orderTif,
		},
	}
	isBuy := side == types.OrderSideBuy
	marketIdx, err := e.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return "", err
	}

	// params
	nonce := getNonce()
	order := orderWire{
		Asset:      marketIdx,
		IsBuy:      isBuy,
		LimitPx:    utils.FloatToStr(price),
		SizePx:     utils.FloatToStr(qty),
		ReduceOnly: reduceOnly,
		OrderType:  orderType,
	}
	if cloId != "" {
		order.Cloid = &cloId
	}
	action := orderAction{
		Type:     "order",
		Orders:   []orderWire{order},
		Grouping: string(groupingNa),
	}
	signature, err := e.getRequestSignature(action, "", nonce)
	if err != nil {
		return "", fmt.Errorf("fail to get signature when open limit order: %v", err)
	}
	req := orderActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: nil,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/exchange", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return "", err
	}
	if status != "200 OK" {
		return "", fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res openOrderResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		return "", err
	}
	var errs []string
	for _, status := range res.Response.Data.Statuses {
		if errMsg := status.Error; errMsg != "" {
			errs = append(errs, errMsg)
		}
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("fail to open limit order: %s", strings.Join(errs, "; "))
	}

	oId := strconv.FormatInt(res.Response.Data.Statuses[0].Resting.Oid, 10)
	if oId == "" {
		return "", fmt.Errorf("oId is missing from the response")
	}
	return oId, nil
}

func (e *HplExchange) OpenBatchLimitOrders(symbol string, inputs []types.LimitOrderInput, lev int) ([]string, error) {
	if e.AccountLeverage[e.SymbolMapU2L[symbol]] != lev {
		if err := e.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return nil, err
		}
	}

	if len(inputs) == 0 {
		return nil, fmt.Errorf("inputs length is 0")
	}

	// convert
	symbol = e.ToLocSymbol(symbol)
	orderTif, err := convertOrderTif(inputs[0].Tif)
	if err != nil {
		return nil, err
	}
	orderType := orderTypeWire{
		Limit: &limit{
			Tif: orderTif,
		},
	}
	marketIdx, err := e.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return nil, err
	}
	// ref: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
	orders := make([]orderWire, 0, len(inputs))
	for _, order := range inputs {
		price := utils.RoundToSigFigs(order.Price, MAX_PRICE_SIG_FIGURE)
		isBuy := order.Side == types.OrderSideBuy

		// params
		orders = append(orders, orderWire{
			Asset:      marketIdx,
			IsBuy:      isBuy,
			LimitPx:    utils.FloatToStr(price),
			SizePx:     utils.FloatToStr(order.Qty),
			ReduceOnly: false,
			OrderType:  orderType,
		})
	}
	nonce := getNonce()
	action := orderAction{
		Type:     "order",
		Orders:   orders,
		Grouping: string(groupingNa),
	}
	signature, err := e.getRequestSignature(action, "", nonce)
	if err != nil {
		return nil, fmt.Errorf("fail to get signature when open limit order: %v", err)
	}
	req := orderActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: nil,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/exchange", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return nil, err
	}
	if status != "200 OK" {
		return nil, fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res openOrderResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		log.Warnf("fail unmarshal response: %v", string(resBody))
		return nil, err
	}
	for _, status := range res.Response.Data.Statuses {
		if errMsg := status.Error; errMsg != "" {
			log.Warnf("fail to open some limit order in batch: %s", errMsg)
		}
	}
	oIds := make([]string, 0, len(res.Response.Data.Statuses))
	for _, status := range res.Response.Data.Statuses {
		oId := strconv.FormatInt(status.Resting.Oid, 10)
		if oId == "" {
			continue
		}
		oIds = append(oIds, oId)
	}
	return oIds, nil
}

func (e *HplExchange) CancelOrder(symbol string, orderId string, cloId string) error {
	// convert
	symbol = e.ToLocSymbol(symbol)
	marketIdx, err := e.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return err
	}

	// params
	nonce := getNonce()
	var action orderAction
	if cloId != "" {
		action = orderAction{
			Type: "cancelByCloid",
			Cancels: []cancelWire{{
				AssetCloId: marketIdx,
				CloId:      cloId,
			}},
		}
	} else {
		oId, err := strconv.Atoi(orderId)
		if err != nil {
			return err
		}
		action = orderAction{
			Type: "cancel",
			Cancels: []cancelWire{{
				Asset:   marketIdx,
				OrderId: oId,
			}},
		}
	}
	signature, err := e.getRequestSignature(action, "", nonce)
	if err != nil {
		log.Errorf("fail to get signature when cancel order: %v", err)
	}
	req := orderActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: nil,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/exchange", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return err
	}
	if status != "200 OK" {
		return fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res cancelOrderResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		return err
	}
	if string(res.Response.Data.Statuses) == `["success"]` {
		return nil
	} else {
		return fmt.Errorf("%v", string(res.Response.Data.Statuses))
	}
}

func (e *HplExchange) CancelBatchOrders(symbol string, orderIds []string) error {
	if len(orderIds) == 0 {
		return nil
	}
	// convert
	symbol = e.ToLocSymbol(symbol)
	marketIdx, err := e.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return err
	}

	cancels := make([]cancelWire, 0, len(orderIds))

	for _, orderId := range orderIds {
		oId, err := strconv.Atoi(orderId)
		if err != nil {
			return err
		}
		cancels = append(cancels, cancelWire{
			Asset:   marketIdx,
			OrderId: oId,
		})
	}

	// params
	nonce := getNonce()
	action := orderAction{
		Type:    "cancel",
		Cancels: cancels,
	}

	signature, err := e.getRequestSignature(action, "", nonce)
	if err != nil {
		log.Errorf("fail to get signature when cancel order: %v", err)
	}
	req := orderActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: nil,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/exchange", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return err
	}
	if status != "200 OK" {
		return fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res cancelOrderResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		return err
	}
	if res.Status == "err" {
		// TODO: there're inconsistent data type while unmarshall if error case happen
		// "response" key can be both string and struct
		log.Warn(string(resBody))
		return fmt.Errorf("fail to cancel order: %s", res.Status)
	}
	return nil
}

func (e *HplExchange) CancelAllOrders(symbol string) error {
	return fmt.Errorf("not implemented")
}

func (e *HplExchange) UpdateAccountLeverage(symbol string, lev int, isCross bool) error {
	// convert
	symbol = e.ToLocSymbol(symbol)
	marketIdx, err := e.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return err
	}

	// params
	nonce := getNonce()
	action := leverageAction{
		Type:     "updateLeverage",
		Asset:    marketIdx,
		IsCross:  isCross,
		Leverage: lev,
	}
	signature, err := e.getRequestSignature(action, "", nonce)
	if err != nil {
		log.Errorf("fail to get signature when update account leverage: %v", err)
	}
	req := updateLeverageRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: nil,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/exchange", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return err
	}
	if status != "200 OK" {
		return fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res updateLeverageResponse
	if err := json.Unmarshal([]byte(resBody), &res); err != nil {
		return err
	}
	if res.Status == "err" {
		return fmt.Errorf("fail to update account leverage for %s to %v: %s", symbol, lev, res.Response)
	}

	e.AccountLeverage[symbol] = lev
	return nil
}

// ╔═══════════════════╗
//    OrderMgmtStream
// ╚═══════════════════╝

func (e *HplExchange) ConnectOrderMgmtStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.OrderEvent), onClose func(stream.Stream)) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)

	// connect stream
	stream, err := NewStream(ctx, types.StreamOrderMgmt, e, e.HplConfig.WsUrl, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := stream.ConnectAndSubscribe(nil, func(e []byte) {
		evts, err := parseOrderEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		for _, evt := range evts {
			if (evt.Symbol != symbol || evt == types.OrderEvent{}) {
				return
			}
			onEvent(stream, evt)
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return stream, nil
}

// ╔══════════════╗
//    TradeSteam
// ╚══════════════╝

func (e *HplExchange) SubscribeTradeStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.TradeEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	params := map[string]string{
		"type": "trades",
		"coin": symbol,
	}

	// connect stream
	stream, err := NewStream(ctx, types.StreamTrade, e, e.HplConfig.WsUrl, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := stream.ConnectAndSubscribe(params, func(e []byte) {
		evts, err := parseTradeEvents(e)
		if err != nil {
			log.Error(err)
			return
		}
		for _, evt := range evts {
			// check if the event is within the allowed delay
			delayMs := time.Now().UnixMilli() - evt.Time.UnixMilli()
			if delayMs > maxDelayMs {
				return
			}
			onEvent(stream, evt)
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return stream, nil
}

// ╔══════════════╗
//    KLineSteam
// ╚══════════════╝

func (e *HplExchange) SubscribeKLineStream(ctx context.Context, symbol string, interval types.Interval, onConn func(stream.Stream), onEvent func(stream.Stream, types.KLineEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	if e.IsUseBnfKLines {
		return e.SubscribeBnfKLineStream(ctx, symbol, interval, onConn, onEvent, onClose, maxDelayMs)
	} else {
		symbol = e.ToLocSymbol(symbol)
		params := map[string]string{
			"type":     "candle",
			"coin":     symbol,
			"interval": string(interval),
		}

		// connect stream
		stream, err := NewStream(ctx, types.StreamKLine, e, e.HplConfig.WsUrl, onConn, onClose)
		if err != nil {
			return nil, err
		}
		doneC, stopC, err := stream.ConnectAndSubscribe(params, func(e []byte) {
			evt, err := parseKLineEvent(e)
			if err != nil {
				log.Error(err)
				return
			}
			// check if the event is within the allowed delay and non-empty struct
			delayMs := time.Now().UnixMilli() - evt.CloseTime.UnixMilli()
			if (delayMs > maxDelayMs && evt == types.KLineEvent{}) {
				return
			}
			onEvent(stream, evt)
		})
		if err != nil {
			return nil, err
		}

		go func() {
			select {
			case <-ctx.Done():
				close(stopC)
			case <-doneC:
			}
		}()

		return stream, nil
	}
}

func (e *HplExchange) SubscribeBnfKLineStream(ctx context.Context, symbol string, interval types.Interval, onConn func(stream.Stream), onEvent func(stream.Stream, types.KLineEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	bnfSymbol := strings.ReplaceAll(symbol, "_", "") + "T"
	bnfWsEndpoint := fmt.Sprintf("wss://fstream.binance.com/ws/%s@kline_%s", strings.ToLower(bnfSymbol), interval)

	// connect bnfStream
	bnfStream, err := bnf.NewStream(ctx, types.StreamKLine, nil, bnfWsEndpoint, onConn, onClose)
	if err != nil {
		return bnfStream, err
	}
	doneC, stopC, err := bnfStream.ConnectAndSubscribe(nil, func(e []byte) {
		evt, err := bnf.ParseKLineEvent(symbol, e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay
		delayMs := time.Now().UnixMilli() - evt.CloseTime.UnixMicro()
		if delayMs > maxDelayMs {
			return
		}
		onEvent(bnfStream, evt)
	})
	if err != nil {
		log.Errorf("fail to connect and subscribe: %v", err)
		return bnfStream, nil
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
//    MarkPriceStream
// ╚═══════════════════╝

func (e *HplExchange) SubscribeMarkPriceStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.MarkPriceEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	params := map[string]string{
		"type": "activeAssetCtx",
		"coin": symbol,
	}

	// connect stream
	stream, err := NewStream(ctx, types.StreamMarkPrice, e, e.HplConfig.WsUrl, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := stream.ConnectAndSubscribe(params, func(e []byte) {
		evt, err := parseMarkPriceEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay and non-empty struct
		delayMs := time.Now().UnixMilli() - evt.Time.UnixMilli()
		if (delayMs > maxDelayMs && evt == types.MarkPriceEvent{}) {
			return
		}
		onEvent(stream, evt)
	})
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return stream, nil
}

// ╔═══════════════════╗
//    BookDepthStream
// ╚═══════════════════╝

func (e *HplExchange) SubscribeBookDepthStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.BookDepthEvent), onClose func(stream.Stream), maxDelayMs int64) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	params := map[string]string{
		"type": "l2Book",
		"coin": symbol,
	}

	// connect stream
	stream, err := NewStream(ctx, types.StreamBookDepth, e, e.HplConfig.WsUrl, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := stream.ConnectAndSubscribe(params, func(e []byte) {
		evt, err := parseBookDepthEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		// check if the event is within the allowed delay and non-empty struct
		delayMs := time.Now().UnixMilli() - evt.Time.UnixMilli()
		if delayMs > maxDelayMs && evt.Event == "" {
			return
		}
		onEvent(stream, evt)
	})
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return stream, nil
}

// ╔═══════════════╗
//    OrderStream
// ╚═══════════════╝

func (e *HplExchange) SubscribeOrderStream(ctx context.Context, symbol string, onConn func(stream.Stream), onEvent func(stream.Stream, types.OrderEvent), onClose func(stream.Stream)) (stream.Stream, error) {
	symbol = e.ToLocSymbol(symbol)
	params := map[string]string{
		"type": "orderUpdates",
		"user": e.AccountAddress.String(),
	}

	// connect stream
	stream, err := NewStream(ctx, types.StreamOrder, e, e.HplConfig.WsUrl, onConn, onClose)
	if err != nil {
		return nil, err
	}
	doneC, stopC, err := stream.ConnectAndSubscribe(params, func(e []byte) {
		evts, err := parseOrderEvent(e)
		if err != nil {
			log.Error(err)
			return
		}
		for _, evt := range evts {
			if (evt.Symbol != symbol || evt == types.OrderEvent{}) {
				return
			}
			onEvent(stream, evt)
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-ctx.Done():
			close(stopC)
		case <-doneC:
		}
	}()

	return stream, nil
}

func (e *HplExchange) ToUniSymbol(locSymbol string) string {
	if locSymbol, ok := e.SymbolMapL2U[locSymbol]; ok {
		return locSymbol
	}
	log.Fatalf("fail to convert local symbol to universal symbol: %v", locSymbol)
	return ""
}

func (e *HplExchange) ToLocSymbol(uniSymbol string) string {
	if locSymbol, ok := e.SymbolMapU2L[uniSymbol]; ok {
		return locSymbol
	}
	log.Fatalf("fail to convert universal symbol to local symbol: %v", uniSymbol)
	return ""
}

func (e *HplExchange) GetAccountBalance() (float64, error) {
	// params
	req := map[string]interface{}{
		"type": "clearinghouseState",
		"user": e.AccountAddress.String(),
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/info", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return 0, err
	}
	if status != "200 OK" {
		return 0, fmt.Errorf("status: %v: %v", status, string(resBody))
	}
	// check response
	var res accountBalanceResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		return 0, err
	}

	return utils.StrToFloat(res.MarginSummary.AccountValue)
}

func (e *HplExchange) GetActivePositionByMarket(symbol string) ([]types.Position, error) {
	// params
	symbol = e.ToLocSymbol(symbol)
	req := map[string]interface{}{
		"type": "clearinghouseState",
		"user": e.AccountAddress.String(),
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// POST request
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/info", e.HplConfig.ApiUrl), "", reqBody)
	if err != nil {
		return nil, err
	}
	if status != "200 OK" {
		return nil, fmt.Errorf("status: %v: %v", status, string(resBody))
	}

	// check response
	var res accountBalanceResponse
	if err := json.Unmarshal(resBody, &res); err != nil {
		return nil, err
	}

	positions := []types.Position{}
	for _, pos := range res.AssetPositions {
		if pos.Position.Coin == symbol {
			qty, err := utils.StrToFloat(pos.Position.Szi)
			if err != nil {
				return nil, err
			}
			entryPx, err := utils.StrToFloat(pos.Position.EntryPx)
			if err != nil {
				return nil, err
			}
			side := types.OrderSideBuy
			if qty < 0 {
				side = types.OrderSideSell
			}
			positions = append(positions, types.Position{
				EntryPrice: entryPx,
				Qty:        math.Abs(qty),
				Side:       side,
			})
		}
	}

	return positions, nil
}

func (e *HplExchange) CloseActivePositionByMarket(symbol string, lev int) error {
	positions, err := e.GetActivePositionByMarket(symbol)
	if err != nil {
		return err
	}
	for _, position := range positions {
		if position.Qty > 0 {
			if position.Side == types.OrderSideBuy {
				err = e.OpenMarketOrder(symbol, types.OrderSideSell, position.Qty, lev, true)
				if err != nil {
					return err
				}
			} else {
				err = e.OpenMarketOrder(symbol, types.OrderSideBuy, position.Qty, lev, true)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
