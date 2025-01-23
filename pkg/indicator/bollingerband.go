package indicator

import (
	"fmt"
	"lfg/pkg/types"
	"time"
)

type BollingerBand struct {
	Time       time.Time
	UpperBand  float64
	MiddleBand float64
	LowerBand  float64
}

func CalculateBollingerBand(klines []types.KLineEvent, window int, sd float64) (BollingerBand, error) {
	if len(klines) < window {
		return BollingerBand{}, fmt.Errorf("no klines data available")
	}

	closePrices := make([]float64, window)
	klineStartIndex := len(klines) - window
	for i, kline := range klines {
		if i >= klineStartIndex {
			closePrices[i-klineStartIndex] = kline.Kline.C
		}
	}

	sma := CalculateAverage(closePrices)
	stdDev := CalculateSD(closePrices, sma)
	upperBand := sma + sd*stdDev
	lowerBand := sma - sd*stdDev
	return BollingerBand{
		UpperBand:  upperBand,
		MiddleBand: sma,
		LowerBand:  lowerBand,
		Time:       klines[len(klines)-1].OpenTime,
	}, nil
}
