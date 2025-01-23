package bnf

import (
	"fmt"
	"lfg/pkg/types"

	"github.com/adshao/go-binance/v2/futures"
)

func convertOrderSide(side types.OrderSide) (futures.SideType, error) {
	switch side {
	case types.OrderSideBuy:
		return futures.SideTypeBuy, nil
	case types.OrderSideSell:
		return futures.SideTypeSell, nil
	default:
		return "", fmt.Errorf("unknown order side: %s", side)
	}
}

func convertOrderTIF(tif types.OrderTIF) (futures.TimeInForceType, error) {
	switch tif {
	case types.OrderTIFGTC:
		return futures.TimeInForceTypeGTC, nil
	case types.OrderTIFGTX:
		return futures.TimeInForceTypeGTX, nil
	case types.OrderTIFIOC:
		return futures.TimeInForceTypeIOC, nil
	case types.OrderTIFFOK:
		return futures.TimeInForceTypeFOK, nil
	default:
		return "", fmt.Errorf("unknown tif: %s", tif)
	}
}
