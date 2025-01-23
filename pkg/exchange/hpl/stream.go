package hpl

import (
	"context"
	"encoding/json"
	"fmt"
	"lfg/pkg/order"
	"lfg/pkg/stream"
	"lfg/pkg/types"
	"lfg/pkg/utils"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const HS_TIMEOUT_S = 5   // handshake timeout in seconds
const HB_INTERVAL_S = 55 // heartbeat interval in seconds

type HplStream struct {
	exchange     *HplExchange
	wsUrl        string
	dialer       websocket.Dialer
	conn         *websocket.Conn
	lastPingpong time.Time

	// channels
	doneC          chan struct{}
	stopC          chan struct{}
	isDisconnected bool // temporary disconnection; the stream may auto-reconnect
	isClosed       bool // permanent closure; the stream will not reconnect

	// callbacks
	onConn  func(stream.Stream)
	onClose func(stream.Stream)

	// response handlers map
	actionResponseHandlers map[int64]chan wsPostActionResponse // handle ws responses related to action (open, modify, cancel)
	infoResponseHandlers   map[int64]chan wsPostInfoResponse   // handle ws responses related to info (get pending)

	mu      sync.Mutex
	writeMu sync.Mutex
	logger  *log.Entry
}

func NewStream(ctx context.Context, stream types.Stream, hplExchg *HplExchange, wsUrl string, onConn func(stream.Stream), onClose func(stream.Stream)) (*HplStream, error) {
	// validate wsUrl
	_, err := url.Parse(wsUrl)
	if err != nil {
		return nil, err
	}
	return &HplStream{
		wsUrl:    wsUrl,
		exchange: hplExchg,
		dialer: websocket.Dialer{
			HandshakeTimeout:  time.Duration(HS_TIMEOUT_S) * time.Second,
			Subprotocols:      []string{"permessage-deflate"},
			EnableCompression: true,
		},
		logger: log.WithFields(log.Fields{
			"stratId": ctx.Value("stratId"),
			"url":     wsUrl,
			"sm":      stream,
		}),
		onConn:                 onConn,
		onClose:                onClose,
		actionResponseHandlers: make(map[int64]chan wsPostActionResponse), // Initialize the map
		infoResponseHandlers:   make(map[int64]chan wsPostInfoResponse),   // Initialize the map
	}, nil
}

func (sm *HplStream) registerActionResponseHandler(nonce int64, respChan chan wsPostActionResponse) {
	sm.mu.Lock()
	sm.actionResponseHandlers[nonce] = respChan
	sm.mu.Unlock()
}

func (sm *HplStream) cleanupActionResponseHandler(nonce int64) {
	sm.mu.Lock()
	delete(sm.actionResponseHandlers, nonce)
	sm.mu.Unlock()
}

func (sm *HplStream) registerInfoResponseHandler(nonce int64, respChan chan wsPostInfoResponse) {
	sm.mu.Lock()
	sm.infoResponseHandlers[nonce] = respChan
	sm.mu.Unlock()
}

func (sm *HplStream) cleanupInfoResponseHandler(nonce int64) {
	sm.mu.Lock()
	delete(sm.infoResponseHandlers, nonce)
	sm.mu.Unlock()
}

func (sm *HplStream) ConnectAndSubscribe(params map[string]string, onEvent func(e []byte)) (doneC chan struct{}, stopC chan struct{}, err error) {
	err = sm.connect()
	if err != nil {
		return nil, nil, err
	}
	if sm.onConn != nil {
		sm.onConn(sm)
	}
	sm.lastPingpong = time.Now()

	sm.doneC = make(chan struct{})
	sm.stopC = make(chan struct{})

	go sm.subscribe(params, onEvent)
	return sm.doneC, sm.stopC, nil
}

func (sm *HplStream) connect() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	c, _, err := sm.dialer.Dial(sm.wsUrl, nil)
	if err != nil {
		sm.logger.Errorf("fail to connect stream: %v", err)
		return err
	}
	sm.conn = c
	return nil
}

func (sm *HplStream) sendSubMsg(params map[string]string) error {
	sm.writeMu.Lock()
	defer sm.writeMu.Unlock()

	if params != nil {
		subMsg := map[string]interface{}{
			"method":       "subscribe",
			"subscription": params,
		}
		return sm.conn.WriteJSON(subMsg)
	}
	return nil
}

func (sm *HplStream) writeMessage(messageType int, data []byte) error {
	sm.writeMu.Lock()
	defer sm.writeMu.Unlock()
	return sm.conn.WriteMessage(messageType, data)
}

func (sm *HplStream) handleReconnect(params map[string]string) {
	if !sm.IsDisconnected() {
		sm.forceDisconnect()
	}

	for {
		if sm.IsClosed() {
			return
		}
		select {
		case <-sm.stopC:
			sm.Close()
			return
		default:
			time.Sleep(1 * time.Second)
			if err := sm.connect(); err != nil {
				sm.logger.Errorf("fail to reconnect stream (retrying...): %v", err)
				continue
			} else {
				sm.logger.Info("reconnect stream success")
			}

			if err := sm.sendSubMsg(params); err != nil {
				sm.logger.Errorf("fail to resubscribe stream: %v", err)
				sm.forceDisconnect()
				continue
			}
			sm.logger.Info("reconnect and resubscribe stream success")
			sm.mu.Lock()
			sm.isDisconnected = false
			sm.mu.Unlock()
			return
		}
	}
}

func (sm *HplStream) subscribe(params map[string]string, onEvent func(e []byte)) {
	err := sm.sendSubMsg(params)
	if err != nil {
		sm.logger.Errorf("fail to subscribe stream: %v", err)
		sm.Close()
	}
	sm.isDisconnected = false

	// keep stream connection alive
	sm.keepAlive(time.Duration(HB_INTERVAL_S) * time.Second)

	for {
		select {
		case <-sm.stopC:
			sm.Close()
			return
		default:
			if sm.IsClosed() {
				return
			}
			_, msg, err := sm.conn.ReadMessage()
			if err != nil {
				sm.logger.Errorf("fail to read stream message (trying to reconnect): %v", err)
				sm.handleReconnect(params)
				continue
			}

			// @dev
			// HPL sends `{"channel": "pong"}` as a regular ws message
			// we need to handle `lastPingpong` in the main stream loop
			var wsGenericRes wsGenericResponse
			if err := json.Unmarshal(msg, &wsGenericRes); err != nil {
				sm.logger.Warnf("found unknown message format: %v: %v", err, string(msg))
				continue
			}
			sm.lastPingpong = time.Now()
			if wsGenericRes.Channel == "pong" {
				sm.logger.Debug("received pong")
			} else if wsGenericRes.Channel == "error" {
				sm.logger.Errorf("found err message during stream: %v", string(msg))
			} else if wsGenericRes.Channel == "post" {
				var wsPostRes wsPostResponse
				if err = json.Unmarshal(msg, &wsPostRes); err != nil {
					sm.logger.Warnf("found unknown post message format: %v: %v", err, string(msg))
					continue
				}
				if wsPostRes.Data.Response.Type == "action" {
					var wsPostActionRes wsPostActionResponse
					if err := json.Unmarshal(msg, &wsPostActionRes); err == nil {
						if wsPostActionRes.Data.Id != 0 { // Assuming 0 is not a valid request ID
							sm.mu.Lock()
							if ch, exists := sm.actionResponseHandlers[wsPostActionRes.Data.Id]; exists {
								ch <- wsPostActionRes
							}
							sm.mu.Unlock()
							continue
						}
					} else {
						sm.logger.Errorf("fail to unmarshall wsPostActionResponse: %v: %v", string(msg), err)
					}
				} else if wsPostRes.Data.Response.Type == "info" {
					var wsPostInfoRes wsPostInfoResponse
					if err := json.Unmarshal(msg, &wsPostInfoRes); err == nil {
						if wsPostInfoRes.Data.Id != 0 { // Assuming 0 is not a valid request ID
							sm.mu.Lock()
							if ch, exists := sm.infoResponseHandlers[wsPostInfoRes.Data.Id]; exists {
								ch <- wsPostInfoRes
							}
							sm.mu.Unlock()
							continue
						}
					}
				}
			}
			onEvent(msg)
		}
	}
}

func (sm *HplStream) keepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// @dev: must check the state inside the ticker loop to handle reconnections
				if sm.IsClosed() {
					return
				}
				if sm.IsDisconnected() {
					continue
				}
				if time.Since(sm.lastPingpong) > time.Duration((HS_TIMEOUT_S+HB_INTERVAL_S)*time.Second) {
					sm.logger.Warn("KeepAlive timeout: force disconnecting")
					sm.forceDisconnect()
					continue
				}

				ping, _ := json.Marshal(map[string]string{"method": "ping"})
				if err := sm.writeMessage(websocket.TextMessage, ping); err != nil {
					sm.logger.Errorf("fail to set write writeMessage during keepAlive: %v", err)
					return
				}
			case <-sm.stopC:
				sm.Close()
				return
			}
		}
	}()
}

// ╔══════════════════════════╗
//   Websocket write function
// ╚══════════════════════════╝

func (sm *HplStream) OpenLimitOrder(symbol string, side types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, tif types.OrderTIF, cloId string) (string, error) {
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return "", fmt.Errorf("fail to open limit order %v %v %v at price %v: websocket already closed", side, qty, symbol, price)
	}
	if sm.exchange.AccountLeverage[sm.exchange.SymbolMapU2L[symbol]] != lev {
		if err := sm.exchange.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return "", err
		}
	}
	// convert
	symbol = sm.exchange.ToLocSymbol(symbol)
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
	marketIdx, err := sm.exchange.convertSymbolToMarketIdx(symbol)
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
	signature, err := sm.exchange.getRequestSignature(action, "", nonce)
	if err != nil {
		return "", fmt.Errorf("fail to get signature when open limit order: %v", err)
	}
	req := map[string]interface{}{
		"method": "post",
		"id":     nonce,
		"request": map[string]interface{}{
			"type": "action",
			"payload": orderActionRequest{
				Action:       action,
				Nonce:        nonce,
				Signature:    signature,
				VaultAddress: nil,
			}},
	}

	// marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal order: %v", err)
		return "", err
	}

	// prepare responseHandler channel and cleanup
	respChan := make(chan wsPostActionResponse)
	sm.registerActionResponseHandler(nonce, respChan)
	defer sm.cleanupActionResponseHandler(nonce)

	// write ws
	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send order: %v", err)
		return "", err
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Data.Response.Type == "error" {
			return "", fmt.Errorf("server returned error: %v", resp.Data.Response.Payload)
		}

		var openOrderRes openOrderResponse
		if err := json.Unmarshal(resp.Data.Response.Payload, &openOrderRes); err != nil {
			return "", fmt.Errorf("failed to parse response: %v", err)
		}
		if len(openOrderRes.Response.Data.Statuses) == 0 {
			return "", fmt.Errorf("server returned 0 order")
		}
		if openOrderRes.Response.Data.Statuses[0].Resting.Oid == 0 {
			return "", fmt.Errorf(openOrderRes.Response.Data.Statuses[0].Error)
		}
		return strconv.FormatInt(openOrderRes.Response.Data.Statuses[0].Resting.Oid, 10), nil
	case <-time.After(time.Duration(HS_TIMEOUT_S) * time.Second):
		return "", fmt.Errorf("timeout waiting for response")
	}
}

func (sm *HplStream) OpenBatchLimitOrders(symbol string, inputs []types.LimitOrderInput, lev int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return fmt.Errorf("fail to open %v batch limit orders: websocket already closed", len(inputs))
	}
	if sm.exchange.AccountLeverage[sm.exchange.SymbolMapU2L[symbol]] != lev {
		if err := sm.exchange.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return err
		}
	}

	if len(inputs) == 0 {
		return fmt.Errorf("inputs length is 0")
	}

	// convert
	symbol = sm.exchange.ToLocSymbol(symbol)
	orderTif, err := convertOrderTif(inputs[0].Tif)
	if err != nil {
		return err
	}
	orderType := orderTypeWire{
		Limit: &limit{
			Tif: orderTif,
		},
	}
	marketIdx, err := sm.exchange.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return err
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
	signature, err := sm.exchange.getRequestSignature(action, "", nonce)
	if err != nil {
		return fmt.Errorf("fail to get signature when open limit order: %v", err)
	}
	req := map[string]interface{}{
		"method": "post",
		"id":     0,
		"request": map[string]interface{}{
			"type": "action",
			"payload": orderActionRequest{
				Action:       action,
				Nonce:        nonce,
				Signature:    signature,
				VaultAddress: nil,
			}},
	}

	// marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal order: %v", err)
		return err
	}
	// write ws
	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send order: %v", err)
		return err
	}

	return nil
}

func (sm *HplStream) CancelOrder(symbol string, orderId string, cloId string) error {
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return fmt.Errorf("fail to cancel order %v: oId %v: websocket already closed", symbol, orderId)
	}

	// convert
	symbol = sm.exchange.ToLocSymbol(symbol)

	marketIdx, err := sm.exchange.convertSymbolToMarketIdx(symbol)
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
	signature, err := sm.exchange.getRequestSignature(action, "", nonce)
	if err != nil {
		log.Errorf("fail to get signature when cancel order: %v", err)
	}

	req := map[string]interface{}{
		"method": "post",
		"id":     nonce,
		"request": map[string]interface{}{
			"type": "action",
			"payload": orderActionRequest{
				Action:       action,
				Nonce:        nonce,
				Signature:    signature,
				VaultAddress: nil,
			}},
	}

	// prepare responseHandler channel and cleanup
	respChan := make(chan wsPostActionResponse)
	sm.registerActionResponseHandler(nonce, respChan)
	defer sm.cleanupActionResponseHandler(nonce)

	// marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal cancel order: %v", err)
		return err
	}

	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send cancel order: %v", err)
		return err
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Data.Response.Type == "error" {
			return fmt.Errorf("server returned error: %v", resp.Data.Response.Payload)
		}

		var cancelOrderRes cancelOrderResponse
		if err := json.Unmarshal(resp.Data.Response.Payload, &cancelOrderRes); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
		if string(cancelOrderRes.Response.Data.Statuses) == `["success"]` {
			return nil
		} else {
			return fmt.Errorf("%v", string(cancelOrderRes.Response.Data.Statuses))
		}
	case <-time.After(time.Duration(HS_TIMEOUT_S) * time.Second):
		return fmt.Errorf("timeout waiting for response")
	}
}

func (sm *HplStream) ModifyOrder(symbol string, oId string, cloId string, side types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, tif types.OrderTIF) error {
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return fmt.Errorf("fail to open limit order %v %v %v at price %v: websocket already closed", side, qty, symbol, price)
	}
	if sm.exchange.AccountLeverage[sm.exchange.SymbolMapU2L[symbol]] != lev {
		if err := sm.exchange.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return err
		}
	}
	// convert
	symbol = sm.exchange.ToLocSymbol(symbol)
	// ref: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
	price = utils.RoundToSigFigs(price, MAX_PRICE_SIG_FIGURE)
	orderTif, err := convertOrderTif(tif)
	if err != nil {
		return err
	}
	orderType := orderTypeWire{
		Limit: &limit{
			Tif: orderTif,
		},
	}
	isBuy := side == types.OrderSideBuy
	marketIdx, err := sm.exchange.convertSymbolToMarketIdx(symbol)
	if err != nil {
		return err
	}

	// params
	nonce := getNonce()
	var action any
	if cloId != "" {
		action = orderActionCloId{
			Type:   "modify",
			OId:    cloId,
			Orders: nil,
			Order: &orderWire{
				Asset:      marketIdx,
				IsBuy:      isBuy,
				LimitPx:    utils.FloatToStr(price),
				SizePx:     utils.FloatToStr(qty),
				ReduceOnly: reduceOnly,
				OrderType:  orderType,
				Cloid:      &cloId,
			},
		}
	} else {
		orderId, err := strconv.Atoi(oId)
		if err != nil {
			return err
		}
		action = orderAction{
			Type:   "modify",
			OId:    orderId,
			Orders: nil,
			Order: &orderWire{
				Asset:      marketIdx,
				IsBuy:      isBuy,
				LimitPx:    utils.FloatToStr(price),
				SizePx:     utils.FloatToStr(qty),
				ReduceOnly: reduceOnly,
				OrderType:  orderType,
			},
		}
	}
	signature, err := sm.exchange.getRequestSignature(action, "", nonce)
	if err != nil {
		return fmt.Errorf("fail to get signature when open limit order: %v", err)
	}
	req := map[string]interface{}{
		"method": "post",
		"id":     nonce,
		"request": map[string]interface{}{
			"type": "action",
			"payload": orderActionRequest{
				Action:       action,
				Nonce:        nonce,
				Signature:    signature,
				VaultAddress: nil,
			}},
	}

	// prepare responseHandler channel and cleanup
	respChan := make(chan wsPostActionResponse)
	sm.registerActionResponseHandler(nonce, respChan)
	defer sm.cleanupActionResponseHandler(nonce)

	// marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal order: %v", err)
		return err
	}
	// write ws
	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send order: %v", err)
		return err
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Data.Response.Type == "error" {
			return fmt.Errorf("server returned error: %v", resp.Data.Response.Payload)
		}

		var modifyOrderRes modifyOrderResponse
		if err := json.Unmarshal(resp.Data.Response.Payload, &modifyOrderRes); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
		if modifyOrderRes.Status == "ok" {
			return nil
		} else {
			return fmt.Errorf("%v", string(modifyOrderRes.Response))
		}
	case <-time.After(time.Duration(HS_TIMEOUT_S) * time.Second):
		return fmt.Errorf("timeout waiting for response")
	}
}

func (sm *HplStream) GetPendingOrders(symbol string) ([]order.Order, error) {
	symbol = sm.exchange.ToLocSymbol(symbol)
	nonce := getNonce()

	if sm.IsClosed() {
		return nil, fmt.Errorf("fail to get pending orders: websocket already closed")
	}

	// Set write deadline
	deadline := time.Now().Add(time.Duration(HS_TIMEOUT_S) * time.Second)
	if err := sm.conn.SetWriteDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %v", err)
	}
	defer sm.conn.SetWriteDeadline(time.Time{}) // Reset deadline

	// Create request
	req := map[string]interface{}{
		"method": "post",
		"id":     nonce,
		"request": map[string]interface{}{
			"type": "info",
			"payload": metadataRequest{
				Type: "openOrders",
				User: sm.exchange.AccountAddress.String(),
			}},
	}

	// Marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal getPendingOrders request: %v", err)
		return nil, err
	}

	// prepare responseHandler channel and cleanup
	respChan := make(chan wsPostInfoResponse)
	sm.registerInfoResponseHandler(nonce, respChan)
	defer sm.cleanupInfoResponseHandler(nonce)

	// write ws
	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send getPendingOrders request: %v", err)
		return nil, err
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Data.Response.Type == "error" {
			return nil, fmt.Errorf("server returned error: %v", resp.Data.Response.Payload)
		}

		var pendingOrderRes []pendingOrderResponse
		if err := json.Unmarshal(resp.Data.Response.Payload.Data, &pendingOrderRes); err != nil {
			return nil, fmt.Errorf("failed to parse response: %v", err)
		}

		orders := make([]order.Order, 0)
		for _, pendingOrder := range pendingOrderRes {
			order, err := sm.exchange.parsePendingOrder(pendingOrder)
			if err != nil {
				return nil, err
			}
			// HPL, well, only returns symbol name in this endpoint e.g. "BTC" not "BTC/USD"
			if order.Symbol == strings.Split(symbol, "/")[0] {
				orders = append(orders, order)
			}
		}

		return orders, nil

	case <-time.After(time.Duration(HS_TIMEOUT_S) * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (sm *HplStream) OpenMarketOrder(symbol string, side types.OrderSide, qty float64, lev int, reduceOnly bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return fmt.Errorf("fail to open market order %v %v %v: websocket already closed", side, qty, symbol)
	}

	if sm.exchange.AccountLeverage[sm.exchange.SymbolMapU2L[symbol]] != lev {
		if err := sm.exchange.UpdateAccountLeverage(symbol, lev, false); err != nil {
			return err
		}
	}
	isBuy := side == types.OrderSideBuy

	// get mid price
	kLines, err := sm.exchange.GetKLines(symbol, types.Interval1m, 1)
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

	symbol = sm.exchange.ToLocSymbol(symbol)
	orderType := orderTypeWire{
		Limit: &limit{
			Tif: tifTypeIOC,
		},
	}

	marketIdx, err := sm.exchange.convertSymbolToMarketIdx(symbol)
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

	signature, err := sm.exchange.getRequestSignature(action, "", nonce)
	if err != nil {
		return fmt.Errorf("fail to get signature when open market order: %v", err)
	}
	req := map[string]interface{}{
		"method": "post",
		"id":     0,
		"request": map[string]interface{}{
			"type": "action",
			"payload": orderActionRequest{
				Action:       action,
				Nonce:        nonce,
				Signature:    signature,
				VaultAddress: nil,
			}},
	}

	// marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal order: %v", err)
		return err
	}

	// write ws
	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send order: %v", err)
		return err
	}
	return nil
}

func (sm *HplStream) CancelBatchOrders(symbol string, orderIds []string) error {
	if len(orderIds) == 0 {
		return nil
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return fmt.Errorf("fail to cancel %v batch orders: websocket already closed", len(orderIds))
	}

	// convert
	symbol = sm.exchange.ToLocSymbol(symbol)
	marketIdx, err := sm.exchange.convertSymbolToMarketIdx(symbol)
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

	signature, err := sm.exchange.getRequestSignature(action, "", nonce)
	if err != nil {
		log.Errorf("fail to get signature when cancel order: %v", err)
	}
	req := map[string]interface{}{
		"method": "post",
		"id":     0,
		"request": map[string]interface{}{
			"type": "action",
			"payload": orderActionRequest{
				Action:       action,
				Nonce:        nonce,
				Signature:    signature,
				VaultAddress: nil,
			}},
	}

	// marshall to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		sm.logger.Errorf("fail to marshal order: %v", err)
		return err
	}
	// write ws
	err = sm.writeMessage(websocket.TextMessage, reqBody)
	if err != nil {
		sm.logger.Errorf("fail to send order: %v", err)
		return err
	}
	return nil
}

// Close() is the final function to be called; the stream cannot be reopened afterward
func (sm *HplStream) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// @dev: must directly read sm.isClosed here to prevent mutex deadlock
	if sm.isClosed {
		return
	}
	if sm.onClose != nil {
		sm.onClose(sm)
	}
	// close the websocket connection
	err := sm.conn.Close()
	if err != nil {
		sm.logger.Fatalf("fail to close stream: %v", err)
	}
	sm.isDisconnected = true
	sm.isClosed = true

	select {
	case <-sm.doneC:
	default:
		// safely close the doneC channel
		close(sm.doneC)
	}
	sm.logger.Info("🔌 stream closed")
}

func (sm *HplStream) forceDisconnect() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// @dev: must directly read sm.isDisconnected here to prevent mutex deadlock
	if sm.isDisconnected {
		return
	}

	sm.conn.Close()
	sm.isDisconnected = true
}

func (sm *HplStream) IsDisconnected() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.isDisconnected
}

func (sm *HplStream) IsClosed() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.isClosed
}
