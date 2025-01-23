package bnf

import (
	"encoding/json"
	"fmt"
	"lfg/pkg/types"
	"lfg/pkg/utils"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

func parseTradeEvent(e []byte) (types.TradeEvent, error) {
	var evt futures.WsAggTradeEvent
	err := json.Unmarshal(e, &evt)
	if err != nil {
		return types.TradeEvent{}, err
	}
	price, err := utils.StrToFloat(evt.Price)
	if err != nil {
		return types.TradeEvent{}, err
	}
	qty, err := utils.StrToFloat(evt.Quantity)
	if err != nil {
		return types.TradeEvent{}, err
	}
	return types.TradeEvent{
		Event:    evt.Event,
		Time:     time.UnixMilli(evt.Time),
		Symbol:   evt.Symbol,
		Price:    price,
		Quantity: qty,
	}, nil
}

func parseMarkPriceEvent(e []byte) (types.MarkPriceEvent, error) {
	var evt futures.WsMarkPriceEvent
	err := json.Unmarshal(e, &evt)
	if err != nil {
		return types.MarkPriceEvent{}, err
	}
	price, err := utils.StrToFloat(evt.MarkPrice)
	if err != nil {
		return types.MarkPriceEvent{}, err
	}
	return types.MarkPriceEvent{
		Event:  evt.Event,
		Time:   time.UnixMilli(evt.Time),
		Symbol: evt.Symbol,
		Price:  price,
	}, nil
}

func ParseKLineEvent(symbol string, e []byte) (types.KLineEvent, error) {
	var evt futures.WsKlineEvent
	err := json.Unmarshal(e, &evt)
	if err != nil {
		return types.KLineEvent{}, err
	}
	o, err := utils.StrToFloat(evt.Kline.Open)
	if err != nil {
		return types.KLineEvent{}, err
	}
	c, err := utils.StrToFloat(evt.Kline.Close)
	if err != nil {
		return types.KLineEvent{}, err
	}
	h, err := utils.StrToFloat(evt.Kline.High)
	if err != nil {
		return types.KLineEvent{}, err
	}
	l, err := utils.StrToFloat(evt.Kline.Low)
	if err != nil {
		return types.KLineEvent{}, err
	}
	return types.KLineEvent{
		Event:     evt.Event,
		OpenTime:  time.UnixMilli(evt.Kline.StartTime),
		CloseTime: time.UnixMilli(evt.Kline.EndTime),
		Symbol:    evt.Symbol,
		Kline: types.KLine{
			O: o,
			C: c,
			H: h,
			L: l,
		},
	}, nil
}

func parseBookDepthEvent(e []byte) (types.BookDepthEvent, error) {
	var evt wsDepthEvent
	err := json.Unmarshal(e, &evt)
	if err != nil {
		return types.BookDepthEvent{}, err
	}

	bids := make([]types.Bid, len(evt.Bids))
	for i, bid := range evt.Bids {
		price, err := utils.StrToFloat(bid[0])
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		qty, err := utils.StrToFloat(bid[1])
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		bids[i] = types.Bid{Price: price, Qty: qty}
	}

	asks := make([]types.Ask, len(evt.Asks))
	for i, ask := range evt.Asks {
		price, err := utils.StrToFloat(ask[0])
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		qty, err := utils.StrToFloat(ask[1])
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		asks[i] = types.Ask{Price: price, Qty: qty}
	}

	return types.BookDepthEvent{
		Event:  evt.Event,
		Time:   time.UnixMilli(evt.Time),
		Symbol: evt.Symbol,
		Bids:   bids,
		Asks:   asks,
	}, nil
}

func parseOrderEvent(e []byte) (types.OrderEvent, error) {
	var evt futures.WsUserDataEvent
	err := json.Unmarshal(e, &evt)
	if err != nil {
		return types.OrderEvent{}, err
	}

	if evt.Event != futures.UserDataEventTypeOrderTradeUpdate {
		return types.OrderEvent{}, fmt.Errorf("ignore as order type: %v", evt.Event)
	}
	o := evt.OrderTradeUpdate
	origPrice, err := utils.StrToFloat(o.OriginalPrice)
	if err != nil {
		return types.OrderEvent{}, err
	}
	origQty, err := utils.StrToFloat(o.OriginalQty)
	if err != nil {
		return types.OrderEvent{}, err
	}
	orderStatus, err := parseOrderStatus(o.Status)
	if err != nil {
		return types.OrderEvent{}, err
	}
	avgPrice, err := utils.StrToFloat(o.AveragePrice)
	if err != nil {
		return types.OrderEvent{}, err
	}
	filledQty, err := utils.StrToFloat(o.AccumulatedFilledQty)
	if err != nil {
		return types.OrderEvent{}, err
	}
	realizedPnL, err := utils.StrToFloat(o.RealizedPnL)
	if err != nil {
		return types.OrderEvent{}, err
	}
	fee, err := utils.StrToFloat(o.Commission)
	if err != nil {
		return types.OrderEvent{}, err
	}

	return types.OrderEvent{
		Event:        string(evt.Event),
		Time:         time.UnixMilli(evt.TransactionTime),
		Symbol:       o.Symbol,
		OId:          fmt.Sprintf("%v", o.ID),
		ClientOId:    o.ClientOrderID,
		Side:         types.OrderSide(o.Side),
		IsReduceOnly: o.IsReduceOnly,
		OrderStatus:  orderStatus,
		Price:        origPrice,
		OrigQty:      origQty,
		OrderTif:     types.OrderTIF(o.TimeInForce), // TIF format is expected to be universal standard, so parse directly
		AvgPrice:     avgPrice,
		FilledQty:    filledQty,
		RealizedPnL:  realizedPnL,
		Fee:          fee,
		FeeAsset:     o.CommissionAsset,
	}, nil
}

func parseOrderStatus(orderStatusType futures.OrderStatusType) (types.OrderStatus, error) {
	switch orderStatusType {
	case futures.OrderStatusTypeNew:
		return types.OrderStatusNew, nil
	case futures.OrderStatusTypePartiallyFilled:
		return types.OrderStatusPartialFilled, nil
	case futures.OrderStatusTypeFilled:
		return types.OrderStatusFilled, nil
	case futures.OrderStatusTypeCanceled:
		return types.OrderStatusCanceled, nil
	case futures.OrderStatusTypeRejected:
		return types.OrderStatusRejected, nil
	case futures.OrderStatusTypeExpired:
		return types.OrderStatusExpired, nil
	default:
		return "", fmt.Errorf("fail to parse unknown orderStatusType: %v", string(orderStatusType))
	}
}

func ParseKLines(bnfKLines []*futures.Kline, symbol string) ([]types.KLineEvent, error) {
	kLines := make([]types.KLineEvent, len(bnfKLines))
	for i, kLine := range bnfKLines {
		open, err := utils.StrToFloat(kLine.Open)
		if err != nil {
			return nil, err
		}
		high, err := utils.StrToFloat(kLine.High)
		if err != nil {
			return nil, err
		}
		low, err := utils.StrToFloat(kLine.Low)
		if err != nil {
			return nil, err
		}
		close, err := utils.StrToFloat(kLine.Close)
		if err != nil {
			return nil, err
		}
		kLines[i] = types.KLineEvent{
			OpenTime:  time.UnixMilli(kLine.OpenTime),
			CloseTime: time.UnixMilli(kLine.CloseTime),
			Symbol:    symbol,
			Kline: types.KLine{
				O: open,
				H: high,
				L: low,
				C: close,
			},
		}
	}
	return kLines, nil
}
