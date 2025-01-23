package hpl

import (
	"encoding/json"
	"fmt"
	"lfg/pkg/order"
	"lfg/pkg/types"
	"lfg/pkg/utils"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func parseOrderSide(orderSide string) (types.OrderSide, error) {
	switch strings.ToUpper(orderSide) {
	case "B":
		return types.OrderSideBuy, nil
	case "A":
		return types.OrderSideSell, nil
	default:
		return "", fmt.Errorf("unknown orderType: %v", orderSide)
	}
}

func parseMarkPriceEvent(e []byte) (types.MarkPriceEvent, error) {
	receivedTime := time.Now()
	var wsRes wsGenericResponse
	if err := json.Unmarshal(e, &wsRes); err != nil {
		return types.MarkPriceEvent{}, err
	}
	if wsRes.Channel != "activeAssetCtx" {
		// HPL also send other event i.e. `channel: "subscriptionResponse"` during stream, ignore them
		return types.MarkPriceEvent{}, nil
	}

	// parse mark price event
	var res wsActiveAssetCtxResponse
	if err := json.Unmarshal(e, &res); err != nil {
		return types.MarkPriceEvent{}, err
	}
	price, err := utils.StrToFloat(res.Data.Ctx.MarkPx)
	if err != nil {
		return types.MarkPriceEvent{}, err
	}

	return types.MarkPriceEvent{
		Event:        res.Channel,
		Time:         time.Now(), // no time field in response, fallback to local time
		Symbol:       res.Data.Coin,
		Price:        price,
		ReceivedTime: receivedTime,
	}, nil
}

func parseTradeEvents(e []byte) ([]types.TradeEvent, error) {
	receivedTime := time.Now()
	var wsRes wsGenericResponse
	if err := json.Unmarshal(e, &wsRes); err != nil {
		return []types.TradeEvent{}, err
	}
	if wsRes.Channel != "trades" {
		// HPL also send other event i.e. `channel: "subscriptionResponse"` during stream, ignore them
		return []types.TradeEvent{}, nil
	}

	// parse trade event
	var res wsTradeResponse
	if err := json.Unmarshal(e, &res); err != nil {
		return []types.TradeEvent{}, err
	}
	var tradeEvents []types.TradeEvent
	for _, evt := range res.Data {
		price, err := utils.StrToFloat(evt.Px)
		if err != nil {
			return []types.TradeEvent{}, err
		}
		qty, err := utils.StrToFloat(evt.Sz)
		if err != nil {
			return []types.TradeEvent{}, err
		}

		tradeEvents = append(tradeEvents, types.TradeEvent{
			Event:        res.Channel,
			Time:         time.UnixMilli(evt.Time),
			Symbol:       evt.Coin,
			Price:        price,
			Quantity:     qty,
			Side:         evt.Side,
			ReceivedTime: receivedTime,
		})
	}
	return tradeEvents, nil
}

func parseKLineEvent(e []byte) (types.KLineEvent, error) {
	receivedTime := time.Now()

	var wsRes wsGenericResponse
	if err := json.Unmarshal(e, &wsRes); err != nil {
		return types.KLineEvent{}, err
	}
	if wsRes.Channel != "candle" {
		// HPL also send other event i.e. `channel: "subscriptionResponse"` during stream, ignore them
		return types.KLineEvent{}, nil
	}

	// parse kline event
	var res wsKLineRes
	if err := json.Unmarshal(e, &res); err != nil {
		return types.KLineEvent{}, err
	}
	o, err := utils.StrToFloat(res.Data.Open)
	if err != nil {
		return types.KLineEvent{}, err
	}
	c, err := utils.StrToFloat(res.Data.Close)
	if err != nil {
		return types.KLineEvent{}, err
	}
	h, err := utils.StrToFloat(res.Data.High)
	if err != nil {
		return types.KLineEvent{}, err
	}
	l, err := utils.StrToFloat(res.Data.Low)
	if err != nil {
		return types.KLineEvent{}, err
	}

	return types.KLineEvent{
		Event:     res.Channel,
		OpenTime:  time.UnixMilli(res.Data.OpenT),
		CloseTime: time.UnixMilli(res.Data.CloseT),
		Symbol:    res.Data.Symbol,
		Kline: types.KLine{
			O: o,
			C: c,
			H: h,
			L: l,
		},
		ReceivedTime: receivedTime,
	}, nil
}

func parseBookDepthEvent(e []byte) (types.BookDepthEvent, error) {
	receivedTime := time.Now()
	var wsRes wsGenericResponse
	if err := json.Unmarshal(e, &wsRes); err != nil {
		return types.BookDepthEvent{}, err
	}

	if wsRes.Channel != "l2Book" {
		// HPL also send other event i.e. `channel: "subscriptionResponse"` during stream, ignore them
		return types.BookDepthEvent{}, nil
	}

	// parse bookDepth event
	var res wsBookDepthRes
	if err := json.Unmarshal(e, &res); err != nil {
		return types.BookDepthEvent{}, err
	}

	// res.Data.Levels[0] = bids, res.Data.Levels[1] = asks
	bids := make([]types.Bid, len(res.Data.Levels[0]))
	for i, bid := range res.Data.Levels[0] {
		price, err := utils.StrToFloat(bid.Price)
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		qty, err := utils.StrToFloat(bid.Sz)
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		bids[i] = types.Bid{Price: price, Qty: qty}
	}
	asks := make([]types.Ask, len(res.Data.Levels[1]))
	for i, ask := range res.Data.Levels[1] {
		price, err := utils.StrToFloat(ask.Price)
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		qty, err := utils.StrToFloat(ask.Sz)
		if err != nil {
			return types.BookDepthEvent{}, err
		}
		asks[i] = types.Ask{Price: price, Qty: qty}
	}

	return types.BookDepthEvent{
		Event:        res.Channel,
		Time:         time.UnixMilli(res.Data.Time),
		Symbol:       res.Data.Coin,
		Bids:         bids,
		Asks:         asks,
		ReceivedTime: receivedTime,
	}, nil
}

func parseOrderEvent(e []byte) ([]types.OrderEvent, error) {
	var wsRes wsGenericResponse
	if err := json.Unmarshal(e, &wsRes); err != nil {
		return []types.OrderEvent{}, err
	}
	if wsRes.Channel != "orderUpdates" {
		if wsRes.Channel == "post" {
			var res wsPostActionResponse
			if err := json.Unmarshal(e, &res); err != nil {
				log.Errorf("parseOrderEventError: %v)", err)
			}
			if res.Data.Response.Type == "error" {
				log.Errorf("parseOrderEventError (response: %v)", string(e))
			}
		}
		// HPL also send other event i.e. `channel: "subscriptionResponse"` during stream, ignore them
		return []types.OrderEvent{}, nil
	}

	// parse orderUpdates event
	var res wsOrderResponse
	if err := json.Unmarshal(e, &res); err != nil {
		return []types.OrderEvent{}, err
	}

	var orderEvents []types.OrderEvent
	for _, orderEvt := range res.Data {
		price, err := utils.StrToFloat(orderEvt.Order.LimitPx)
		if err != nil {
			return []types.OrderEvent{}, err
		}
		origQty, err := utils.StrToFloat(orderEvt.Order.OrigSz)
		if err != nil {
			return []types.OrderEvent{}, err
		}
		remQty, err := utils.StrToFloat(orderEvt.Order.Sz)
		if err != nil {
			return []types.OrderEvent{}, err
		}
		orderStatus, err := parseOrderStatus(orderEvt.Status)
		if err != nil {
			return []types.OrderEvent{}, err
		}
		clientOId := ""
		if orderEvt.Order.Cloid != nil {
			clientOId = *orderEvt.Order.Cloid
		}
		orderSide, err := parseOrderSide(orderEvt.Order.Side)
		if err != nil {
			return []types.OrderEvent{}, err
		}
		orderEvents = append(orderEvents, types.OrderEvent{
			Event:       wsRes.Channel,
			Time:        time.UnixMilli(orderEvt.StatusTimestamp),
			Symbol:      orderEvt.Order.Symbol,
			OId:         fmt.Sprintf("%v", orderEvt.Order.Oid),
			ClientOId:   clientOId,
			Side:        orderSide,
			OrderStatus: orderStatus,
			Price:       price,
			OrigQty:     origQty,
			FilledQty:   origQty - remQty,
		})
	}
	return orderEvents, nil
}

func (e *HplExchange) parseKLines(kLinesRes []kLineResponse) ([]types.KLineEvent, error) {
	var kLines []types.KLineEvent
	for _, kLine := range kLinesRes {
		o, err := utils.StrToFloat(kLine.Open)
		if err != nil {
			return nil, err
		}
		c, err := utils.StrToFloat(kLine.Close)
		if err != nil {
			return nil, err
		}
		h, err := utils.StrToFloat(kLine.High)
		if err != nil {
			return nil, err
		}
		l, err := utils.StrToFloat(kLine.Low)
		if err != nil {
			return nil, err
		}

		kLines = append(kLines, types.KLineEvent{
			OpenTime:  time.UnixMilli(kLine.OpenT),
			CloseTime: time.UnixMilli(kLine.CloseT),
			Symbol:    kLine.Symbol,
			Kline: types.KLine{
				O: o,
				C: c,
				H: h,
				L: l,
			},
		})
	}
	return kLines, nil
}

func (e *HplExchange) parsePendingOrder(pendingOrder pendingOrderResponse) (order.Order, error) {
	price, err := utils.BigIntStrToFloat(pendingOrder.LimitPx, 18)
	if err != nil {
		return order.Order{}, fmt.Errorf("fail to parse price: %v", err)
	}

	originalQty, err := utils.BigIntStrToFloat(pendingOrder.OrigSz, 18)
	if err != nil {
		return order.Order{}, fmt.Errorf("fail to parse original quantity: %v", err)
	}

	remainingQty, err := utils.BigIntStrToFloat(pendingOrder.Sz, 18)
	if err != nil {
		return order.Order{}, fmt.Errorf("fail to parse remaining quantity: %v", err)
	}

	orderType := types.OrderLimit
	orderSide, err := parseOrderSide(pendingOrder.Side)
	if err != nil {
		return order.Order{}, err
	}

	return order.Order{
		Id:           fmt.Sprintf("%d", pendingOrder.OId),
		Symbol:       pendingOrder.Coin,
		OrderType:    orderType,
		OrderSide:    orderSide,
		Price:        price,
		OriginalQty:  originalQty,
		RemainingQty: remainingQty,
	}, nil
}

func parseOrderStatus(orderStatus string) (types.OrderStatus, error) {
	switch orderStatus {
	case "open":
		return types.OrderStatusNew, nil
	case "filled":
		return types.OrderStatusFilled, nil
	case "canceled":
		return types.OrderStatusCanceled, nil
	case "rejected":
		return types.OrderStatusRejected, nil
	// TODO: map to corresponding status
	case "triggered", "marginCanceled":
		return "", fmt.Errorf("orderStatusType is known but not yet supported: %v", string(orderStatus))
	default:
		return "", fmt.Errorf("fail to parse unknown orderStatusType: %v", string(orderStatus))
	}
}
