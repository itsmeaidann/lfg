package bnf

import (
	"context"
	"fmt"
	"lfg/pkg/market"
	"lfg/pkg/types"
	"lfg/pkg/utils"

	"github.com/adshao/go-binance/v2/futures"
)

func loadMarkets(fClient *futures.Client) (map[string]*market.Market, error) {
	marketFilters, err := getMarketFilters(fClient)
	if err != nil {
		return nil, err
	}
	var markets = make(map[string]*market.Market)
	for _, marketFilter := range marketFilters {
		// market ID does not apply on bnf, default to 0
		market := market.New(types.ExchangeBnf, 0, marketFilter.symbol)
		market.MinNotional = marketFilter.minNotional
		market.LotMinQty = marketFilter.lotMinQty
		market.LotMaxQty = marketFilter.lotMaxQty
		market.LotStepSize = marketFilter.lotStepSize
		market.MarketLotMinQty = marketFilter.marketLotMinQty
		market.MarketLotMaxQty = marketFilter.marketLotMaxQty
		market.MarketLotStepSize = marketFilter.marketLotStepSize

		// ref: https://www.binance.com/en/fee/futureFee
		market.MakerFeePct = 0.0002 // 2 bps
		market.TakerFeePct = 0.0005 // 5 bps

		markets[marketFilter.symbol] = market
	}
	return markets, nil
}

func getMarketFilters(fClient *futures.Client) (map[string]bnfMarketFilter, error) {
	exchangeInfo, err := fClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, err
	}

	marketFilters := make(map[string]bnfMarketFilter)
	for _, symbol := range exchangeInfo.Symbols {
		minNotional, lotMinQty, lotMaxQty, lotStepSize, marketLotMinQty, marketLotMaxQty, marketLotStepSize := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0
		for _, filter := range symbol.Filters {
			if filter["filterType"] == "MIN_NOTIONAL" {
				minNotional, err = extractFilter(filter, "notional")
				if err != nil {
					return nil, err
				}
			}
			if filter["filterType"] == "LOT_SIZE" {
				lotStepSize, err = extractFilter(filter, "stepSize")
				if err != nil {
					return nil, err
				}
				lotMinQty, err = extractFilter(filter, "minQty")
				if err != nil {
					return nil, err
				}
				lotMaxQty, err = extractFilter(filter, "maxQty")
				if err != nil {
					return nil, err
				}
			}
			if filter["filterType"] == "MARKET_LOT_SIZE" {
				marketLotStepSize, err = extractFilter(filter, "stepSize")
				if err != nil {
					return nil, err
				}
				marketLotMinQty, err = extractFilter(filter, "minQty")
				if err != nil {
					return nil, err
				}
				marketLotMaxQty, err = extractFilter(filter, "maxQty")
				if err != nil {
					return nil, err
				}
			}
		}
		marketFilters[symbol.Symbol] = bnfMarketFilter{
			symbol:            symbol.Symbol,
			minNotional:       minNotional,
			lotMinQty:         lotMinQty,
			lotMaxQty:         lotMaxQty,
			lotStepSize:       lotStepSize,
			marketLotMinQty:   marketLotMinQty,
			marketLotMaxQty:   marketLotMaxQty,
			marketLotStepSize: marketLotStepSize,
		}
	}
	return marketFilters, nil
}

func extractFilter(filter map[string]interface{}, key string) (float64, error) {
	notional, ok := filter[key].(string)
	if !ok {
		return 0, fmt.Errorf("bad string assertion: %s", key)
	}
	parsedFloat, err := utils.StrToFloat(notional)
	return parsedFloat, err
}
