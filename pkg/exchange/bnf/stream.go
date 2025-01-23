package bnf

import (
	"context"
	"fmt"
	"lfg/pkg/order"
	"lfg/pkg/stream"
	"lfg/pkg/types"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const HS_TIMEOUT_S = 10 // handshake timeout in seconds

// TODO: Reevaluate this interval if no frequent connection losses
// https://binance-docs.github.io/apidocs/futures/en/#websocket-api-general-info
const CONN_AUTORESET_S = 3600 // auto reset ws connection every 1hr

type BnfStream struct {
	exchange *BnfExchange
	wsUrl    string
	dialer   websocket.Dialer
	conn     *websocket.Conn

	// channels
	resetC         chan struct{}
	doneC          chan struct{}
	stopC          chan struct{}
	isDisconnected bool // temporary disconnection; the stream may auto-reconnect
	isClosed       bool // permanent closure; the stream will not reconnect

	// callbacks
	onConn  func(stream.Stream)
	onClose func(stream.Stream)

	mu     sync.Mutex
	logger *log.Entry
}

func NewStream(ctx context.Context, streamName types.Stream, bnfExchg *BnfExchange, wsUrl string, onConn func(stream.Stream), onClose func(stream.Stream)) (*BnfStream, error) {
	// validate wsUrl
	_, err := url.Parse(wsUrl)
	if err != nil {
		return nil, err
	}
	return &BnfStream{
		wsUrl:    wsUrl,
		exchange: bnfExchg,
		dialer: websocket.Dialer{
			HandshakeTimeout:  time.Duration(HS_TIMEOUT_S) * time.Second,
			Subprotocols:      []string{"permessage-deflate"},
			EnableCompression: false,
		},
		resetC: make(chan struct{}, 1),
		logger: log.WithFields(log.Fields{
			"stratId": ctx.Value("stratId"),
			"stream":  wsUrl,
			"name":    streamName,
		}),
		onConn:  onConn,
		onClose: onClose,
	}, nil
}

func (sm *BnfStream) ConnectAndSubscribe(_ map[string]string, onEvent func(e []byte)) (doneC chan struct{}, stopC chan struct{}, err error) {
	// connect
	err = sm.connect()
	if err != nil {
		return nil, nil, err
	}
	if sm.onConn != nil {
		sm.onConn(sm)
	}

	// subscribe
	sm.doneC = make(chan struct{})
	sm.stopC = make(chan struct{})
	go sm.subscribe(onEvent)
	go sm.autoReset()

	return sm.doneC, sm.stopC, nil
}

func (sm *BnfStream) connect() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	c, _, err := sm.dialer.Dial(sm.wsUrl, nil)
	if err != nil {
		sm.logger.Errorf("fail to connect stream: %v", err)
		return err
	}
	sm.conn = c

	// keep stream connection alive: Binance pings every 3m, respond with matching pong payload
	// ref: https://binance-docs.github.io/apidocs/futures/en/#websocket-market-streams
	sm.conn.SetPingHandler(func(msg string) error {
		sm.logger.Debugf("received ping, sending pong: %s", msg)
		err := sm.conn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(HS_TIMEOUT_S*time.Second))
		if err != nil {
			sm.logger.Warnf("fail to send pong: %v", err)
			return nil // intentionally return nil even err to prevent connection teardown
		}
		return nil
	})
	return nil
}

func (sm *BnfStream) handleReconnect() {
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
			}
			sm.logger.Info("reconnect and resubscribe stream success")
			sm.mu.Lock()
			sm.isDisconnected = false
			sm.mu.Unlock()
			return
		}
	}
}

func (sm *BnfStream) subscribe(onEvent func(e []byte)) {
	sm.isDisconnected = false

	for {
		select {
		case <-sm.stopC:
			sm.Close()
			return
		case <-sm.resetC:
			sm.handleReconnect()
		default:
			if sm.IsClosed() {
				return
			}
			_, msg, err := sm.conn.ReadMessage()
			if err != nil {
				sm.logger.Errorf("fail to read stream message (trying to reconnect): %v", err)
				sm.handleReconnect()
				continue
			}
			onEvent(msg)
		}
	}
}

// Binance auto-resets connections after 24h; we self-reset every `CONN_AUTORESET_S` to reconnect smoothly before that
func (sm *BnfStream) autoReset() {
	timer := time.NewTicker(time.Duration(CONN_AUTORESET_S) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// @dev: must check the state inside the ticker loop to handle reconnections
			if sm.IsClosed() {
				return
			}
			if sm.IsDisconnected() {
				continue
			}
			sm.logger.Infof("auto-reset triggered after %d seconds", CONN_AUTORESET_S)
			sm.resetC <- struct{}{} // trigger reset
			timer.Reset(time.Duration(CONN_AUTORESET_S) * time.Second)
		case <-sm.stopC:
			return
		}
	}
}

// ╔══════════════════════════╗
//
//	Websocket write function
//
// ╚══════════════════════════╝
func (sm *BnfStream) OpenLimitOrder(symbol string, orderSide types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, orderTif types.OrderTIF, cloId string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (sm *BnfStream) OpenBatchLimitOrders(symbol string, inputs []types.LimitOrderInput, lev int) error {
	return fmt.Errorf("not implemented")
}

func (sm *BnfStream) ModifyOrder(symbol string, oId string, cloId string, orderSide types.OrderSide, price float64, qty float64, lev int, reduceOnly bool, orderTif types.OrderTIF) error {
	return fmt.Errorf("not implemented")
}

func (sm *BnfStream) GetPendingOrders(symbol string) ([]order.Order, error) {
	return nil, fmt.Errorf("not implemented")
}

func (sm *BnfStream) OpenMarketOrder(symbol string, side types.OrderSide, qty float64, lev int, reduceOnly bool) error {
	return fmt.Errorf("not implemented")
}

func (sm *BnfStream) CancelOrder(symbol string, orderId string, cloId string) error {
	return fmt.Errorf("not implemented")
}

func (sm *BnfStream) CancelBatchOrders(symbol string, orderIds []string) error {
	return fmt.Errorf("not implemented")
}

// Close() is the final function to be called; the stream cannot be reopened afterward
func (sm *BnfStream) Close() {
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

func (sm *BnfStream) forceDisconnect() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isDisconnected {
		return
	}

	sm.conn.Close()
	sm.isDisconnected = true
}

func (sm *BnfStream) IsDisconnected() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.isDisconnected
}

func (sm *BnfStream) IsClosed() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.isClosed
}
