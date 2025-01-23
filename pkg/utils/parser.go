package utils

import (
	"encoding/hex"
	"fmt"
	"lfg/pkg/types"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func RoundFloat(val float64, decimals int64) float64 {
	ratio := math.Pow(10, float64(decimals))
	return math.Round(val*ratio) / ratio
}

func StrToFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}

func BigIntStrToFloat(s string, scaleFactor int64) (float64, error) {
	bigFloat, ok := new(big.Float).SetString(s)
	if !ok {
		return 0, fmt.Errorf("fail to parse string to bigint: %s", s)
	}
	sf := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(scaleFactor), nil))
	bigFloat.Quo(bigFloat, sf)
	floatValue, _ := bigFloat.Float64()
	return floatValue, nil
}

func FloatToStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func FloatToBigInt(val float64, scaleFactor int64) (*big.Int, error) {
	scaledValue := val * math.Pow10(int(scaleFactor))
	bigFloat := new(big.Float).SetFloat64(scaledValue)
	bigInt := new(big.Int)
	bigFloat.Int(bigInt)
	return bigInt, nil
}

func FloatToBigIntPrecision(val float64, scaleFactor int) (*big.Int, error) {
	// split the string on the decimal point
	parts := strings.Split(strconv.FormatFloat(val, 'f', -1, 64), ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid float value: %v", val)
	}

	// Combine the parts, adding 18 zeros after the integer part
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) == 2 {
		decimalPart = parts[1]
	}
	combined := integerPart + decimalPart + strings.Repeat("0", scaleFactor-len(decimalPart))

	// Convert the string to a big.Int
	result := new(big.Int)
	result, ok := result.SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("fail to convert string to big.Int: %s", combined)
	}

	return result, nil
}

func HexToFloat(val string, scaleFactor int64) (float64, error) {
	if len(val) > 2 && val[:2] == "0x" {
		val = val[2:]
	}
	unscaledBigInt, ok := new(big.Int).SetString(val, 16)
	if !ok {
		return 0, fmt.Errorf("fail to parse hex to bigint: %v", val)
	}
	// create the sf (scale factor) as a big.Float
	sf := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(scaleFactor), nil))
	bigFloat := new(big.Float).SetInt(unscaledBigInt)
	bigFloat.Quo(bigFloat, sf)
	floatValue, _ := bigFloat.Float64()
	return floatValue, nil
}

func HexToBytes(val string) ([]byte, error) {
	if len(val) > 2 && val[:2] == "0x" {
		val = val[2:]
	}
	bytes, err := hex.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("fail to parse address to bytes: %v", val)
	}
	return bytes, nil
}

func SignatureToVRS(sig []byte) (byte, [32]byte, [32]byte) {
	var v byte
	var r [32]byte
	var s [32]byte

	v = sig[64] + 27
	copy(r[:], sig[:32])
	copy(s[:], sig[32:64])

	return v, r, s
}

func RoundToSigFigs(val float64, sigFigs int) float64 {
	if val == 0 {
		return 0
	}
	d := math.Ceil(math.Log10(math.Abs(val)))
	power := float64(sigFigs) - d
	magnitude := math.Pow(10, power)
	return math.Round(val*magnitude) / magnitude
}

func IntervalToDuration(interval types.Interval) (time.Duration, error) {
	switch interval {
	case types.Interval1s:
		return time.Second, nil
	case types.Interval15s:
		return 15 * time.Second, nil
	case types.Interval1m:
		return time.Minute, nil
	case types.Interval5m:
		return 5 * time.Minute, nil
	case types.Interval15m:
		return 15 * time.Minute, nil
	case types.Interval1h:
		return time.Hour, nil
	case types.Interval4h:
		return 4 * time.Hour, nil
	case types.Interval1d:
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid interval: %v", interval)
	}
}
