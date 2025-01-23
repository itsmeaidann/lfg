package indicator

import (
	"fmt"
	"lfg/pkg/types"
	"math"
)

func CalculateVolatility(klines []types.KLineEvent, window int) (float64, error) {
	if len(klines) < window+1 {
		return 0, fmt.Errorf("insufficient klines candle: have %v/%v", len(klines), window)
	}

	var sumSquaredDiff float64
	klineStartIndex := len(klines) - window
	for i := klineStartIndex; i < len(klines); i++ {
		diff := klines[i].Kline.C - klines[i-1].Kline.C
		sumSquaredDiff += diff * diff
	}
	avg := sumSquaredDiff / float64(window)
	volatility := math.Sqrt(avg)

	return volatility, nil
}
