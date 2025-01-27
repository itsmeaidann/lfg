package indicator

import "lfg/pkg/types"

func CalculateMovingAverage(klines []types.KLineEvent, window int) []float64 {
	if len(klines) < window {
		return []float64{}
	}

	maValues := make([]float64, len(klines)-window+1)
	for i := window - 1; i < len(klines); i++ {
		sum := 0.0
		for j := i - window + 1; j <= i; j++ {
			sum += klines[j].Kline.C
		}
		maValues[i-window+1] = sum / float64(window)
	}

	// flip the array
	for i := 0; i < len(maValues)/2; i++ {
		j := len(maValues) - i - 1
		maValues[i], maValues[j] = maValues[j], maValues[i]
	}
	return maValues
}
