package types

type Interval string

const (
	Interval1s  = Interval("1s")
	Interval15s = Interval("15s")
	Interval1m  = Interval("1m")
	Interval5m  = Interval("5m")
	Interval15m = Interval("15m")
	Interval1h  = Interval("1h")
	Interval4h  = Interval("4h")
	Interval1d  = Interval("1d")
)
