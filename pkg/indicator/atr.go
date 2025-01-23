package indicator

import (
	"fmt"
	"lfg/pkg/types"
	"math"
)

func CalculateTR(high, low, prevClose float64) float64 {
	return math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
}

func CalculateATRLines(klines []types.KLineEvent, window int) []float64 {

	if len(klines) < 2*window+1 {
		return []float64{}
	}
	trValues := make([]float64, 2*window)
	atrValues := make([]float64, window)

	klineStartIndex := len(klines) - 2*window

	for i := klineStartIndex; i < len(klines); i++ {
		high := klines[i].Kline.H
		low := klines[i].Kline.L
		prevClose := klines[i-1].Kline.C
		trValues[i-klineStartIndex] = CalculateTR(high, low, prevClose)
	}

	for i := 0; i < window; i++ {
		atrValues[i] = CalculateAverage(trValues[i+1 : i+window+1])
	}

	return atrValues
}

func CalculateATRIndex(klines []types.KLineEvent, window int) (float64, error) {
	atr, err := CalculateATR(klines, window)
	if err != nil {
		return 0, err
	}

	closePrices := make([]float64, window)
	for i := len(klines) - window; i < len(klines); i++ {
		closePrices[i-(len(klines)-window)] = klines[i].Kline.C
	}

	atrIndex := atr / CalculateAverage(closePrices) * 100
	return atrIndex, nil
}

func CalculateATR(klines []types.KLineEvent, window int) (float64, error) {
	atrValues := CalculateATRLines(klines, window)
	if len(atrValues) == 0 {
		return 0, fmt.Errorf("no atr values available")
	}
	return atrValues[len(atrValues)-1], nil
}
