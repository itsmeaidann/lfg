package market

import "lfg/pkg/types"

type Market struct {
	Id           int64 // market id (or index), usually for API usage
	ExchangeName types.ExchangeName
	Symbol       string

	TickSize          float64 // min price movement
	MinNotional       float64 // min order value in USD
	LotMinQty         float64 // min limit order quantity (e.g. 0.01 means 0.009 BTC is invalid)
	LotMaxQty         float64 // max limit order quantity (e.g. 5000 means 5001 BTC is invalid)
	LotStepSize       float64 // limit order quantity granularity (e.g. 0.01 means 25.001 BTC is invalid)
	MarketLotMinQty   float64 // minimum market order quantity (e.g. 0.01 means 0.009 BTC is invalid)
	MarketLotMaxQty   float64 // max market order quantity (e.g. 5000 means 5001 BTC is invalid)
	MarketLotStepSize float64 // market order quantity granularity (e.g. 0.01 means 25.001 BTC is invalid)
	MaxLeverage       float64 // max leverage gearing

	MakerFeePct float64 // maker fee (e.g. 0.0001 means 0.01% aka 1bps)
	TakerFeePct float64 // maker fee (e.g. 0.0005 means 0.05% aka 5bps)
}

func New(exchangeName types.ExchangeName, id int64, symbol string) *Market {
	return &Market{
		ExchangeName: exchangeName,
		Id:           id,
		Symbol:       symbol,
	}
}
