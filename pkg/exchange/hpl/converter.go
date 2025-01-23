package hpl

import (
	"fmt"
	"lfg/pkg/types"
)

func convertOrderTif(orderTif types.OrderTIF) (tifType, error) {
	switch orderTif {
	case types.OrderTIFIOC:
		return tifTypeIOC, nil
	case types.OrderTIFGTC:
		return tifTypeGTC, nil
	case types.OrderTIFALO:
		return tifTypeALO, nil
	default:
		return "", fmt.Errorf("fail to convert OrderTIF: %v", orderTif)
	}
}

func (e *HplExchange) convertSymbolToMarketIdx(locSymbol string) (int, error) {
	if market, exists := e.Markets[locSymbol]; exists {
		return int(market.Id), nil
	}
	return 0, fmt.Errorf("marketIdx not found from symbol: %v", locSymbol)
}

func (e *HplExchange) convertMarketIdxToSymbol(marketIdx int64) (string, error) {
	for symbol, market := range e.Markets {
		if market.Id == marketIdx {
			return symbol, nil
		}
	}
	return "", fmt.Errorf("symbol not found from marketIdx: %v", marketIdx)
}
