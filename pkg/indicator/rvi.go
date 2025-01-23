package indicator

import (
	"fmt"
	"lfg/pkg/types"
)

func CalculateRVI(klines []types.KLineEvent) (float64, error) {

	period := len(klines)

	if period == 0 {
		return 0, fmt.Errorf("no klines data available")
	}

	upMoves := make([]float64, period)
	downMoves := make([]float64, period)

	for i := 1; i <= period; i++ {
		change := klines[len(klines)-i].Kline.C - klines[len(klines)-i-1].Kline.C
		if change > 0 {
			upMoves[i-1] = change
			downMoves[i-1] = 0
		} else {
			upMoves[i-1] = 0
			downMoves[i-1] = -change
		}
	}

	upStdDev := CalculateSD(upMoves, CalculateAverage(upMoves))
	downStdDev := CalculateSD(downMoves, CalculateAverage(downMoves))

	rvi := 100 * upStdDev / (upStdDev + downStdDev)
	return rvi, nil
}
