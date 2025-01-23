package skeleton

import (
	"context"
	"fmt"
	"lfg/pkg/exchange"
	"lfg/pkg/market"
	"lfg/pkg/stream"
	"lfg/pkg/types"
	"time"

	log "github.com/sirupsen/logrus"
)

type Strategy struct {
	Exchange       *exchange.Exchange
	Symbol         string
	Market         *market.Market
	LastKLineEvent string
	LastTradeEvent string

	logger *log.Entry
}

func (s *Strategy) Init() {

}

func (s *Strategy) Id() string {
	return fmt.Sprintf("%s:%s:%s", (*s.Exchange).Name(), types.StrategySkeleton, s.Symbol)
}

func (s *Strategy) Name() types.StrategyName {
	return types.StrategySkeleton
}

func (s *Strategy) Validate() error {
	return nil
}

func (s *Strategy) onTradeEvent(_ stream.Stream, event types.TradeEvent) {
	log.Infof("Trade event: %v\n", event)
	s.LastTradeEvent = event.Event
	// @dev: DEMO
	// err := s.Exchange.OpenLimitOrder(s.Symbol, types.OrderSell, 168, 1, 0, true, types.OrderTIFGTC)
	// if err != nil {
	// 	log.Error(err)
	// }
}

func (s *Strategy) onKLineEvent(_ stream.Stream, event types.KLineEvent) {
	log.Infof("KLine event: %v\n", event)
	s.LastKLineEvent = event.Event
}

func (s *Strategy) onMarkPriceEvent(_ stream.Stream, event types.MarkPriceEvent) {
	log.Infof("MarkPrice event: %v\n", event)
}

func (s *Strategy) onOrderEvent(_ stream.Stream, event types.OrderEvent) {
	log.Infof("Order event: %v\n", event)
}

func (s *Strategy) Run(ctx context.Context) error {
	// setup logger
	if ctx == nil {
		return fmt.Errorf("invalid context provided: %v: %v", s.Id(), ctx)
	}
	s.logger = log.WithFields(log.Fields{
		"stratId": ctx.Value("stratId"),
	})

	// retrieve market filter
	s.Market = (*s.Exchange).GetMarket(s.Symbol)
	if s.Market == nil {
		return fmt.Errorf("cannot find market %v (exchange %v)", s.Symbol, (*s.Exchange).Name())
	}

	// run strategy
	_, err := (*s.Exchange).SubscribeTradeStream(ctx, s.Symbol, nil, s.onTradeEvent, nil, 1000)
	if err != nil {
		s.logger.Errorf("fail to subscribe trade stream: %v", err)
	}
	_, err = (*s.Exchange).SubscribeKLineStream(ctx, s.Symbol, types.Interval1m, nil, s.onKLineEvent, nil, 1000)
	if err != nil {
		s.logger.Errorf("fail to subscribe kline stream: %v", err)
	}
	_, err = (*s.Exchange).SubscribeOrderStream(ctx, s.Symbol, nil, s.onOrderEvent, nil)
	if err != nil {
		s.logger.Errorf("fail to subscribe order stream: %v", err)
	}
	_, err = (*s.Exchange).SubscribeMarkPriceStream(ctx, s.Symbol, nil, s.onMarkPriceEvent, nil, 1000)
	if err != nil {
		s.logger.Errorf("fail to subscribe markprice stream: %v", err)
	}

	// wait
	<-ctx.Done()
	return s.Shutdown()
}

func (s *Strategy) Shutdown() error {
	s.logger.Info("💤 shutting down...")

	// wait for all go routine to stop + 5s buffer for straggler routine
	time.Sleep(5 * time.Second)

	s.logger.Info("😴 shutdown gracefully")
	return nil
}
