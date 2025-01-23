package hpl

import (
	"encoding/json"
	"fmt"
	"lfg/pkg/http"
	"lfg/pkg/market"
	"lfg/pkg/types"
	"math"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ref: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
const MAX_PRICE_SIG_FIGURE = 5
const MAX_PRICE_DECIMALS = 6

func loadMarkets(baseUrl string) (map[string]*market.Market, error) {
	// retrieve market filters from api
	var marketInfos marketInfoResponse
	reqBody, err := json.Marshal(map[string]string{
		"type": "meta",
	})
	if err != nil {
		return nil, err
	}
	status, resBody, err := http.PostRequest(fmt.Sprintf("%s/info", baseUrl), "", reqBody)
	if err != nil {
		return nil, err
	}
	if status != "200 OK" {
		return nil, fmt.Errorf("status: %v: %v", status, string(resBody))
	}
	if err := json.Unmarshal(resBody, &marketInfos); err != nil {
		return nil, err
	}

	// map into market.Market
	var markets = make(map[string]*market.Market)
	for id, marketFilter := range marketInfos.Universe {
		// TODO: complete all field mapping
		market := market.New(types.ExchangeHpl, int64(id), marketFilter.Name)
		market.LotMinQty = math.Pow(10, float64(-marketFilter.SzDecimals))
		market.LotStepSize = math.Pow(10, float64(-marketFilter.SzDecimals))
		market.MarketLotMinQty = math.Pow(10, float64(-marketFilter.SzDecimals))
		market.MarketLotStepSize = math.Pow(10, float64(-marketFilter.SzDecimals))
		priceDecimals := MAX_PRICE_DECIMALS - marketFilter.SzDecimals
		market.TickSize = math.Pow(10, float64(-priceDecimals))
		market.MaxLeverage = float64(marketFilter.MaxLeverage)

		// ref: https://hyperliquid.gitbook.io/hyperliquid-docs/trading/fees
		market.MinNotional = 10
		market.MakerFeePct = 0.0001  // 1 bps
		market.TakerFeePct = 0.00035 // 3.5 bps

		markets[market.Symbol] = market
	}
	return markets, nil
}

func getRsvSignature(r [32]byte, s [32]byte, v byte) RsvSignature {
	return RsvSignature{
		R: hexutil.Encode(r[:]),
		S: hexutil.Encode(s[:]),
		V: v,
	}
}
