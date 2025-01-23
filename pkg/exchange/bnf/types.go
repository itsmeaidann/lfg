package bnf

type bnfConfig struct {
	WsUrl string `json:"wsUrl"`
}

type bnfMarketFilter struct {
	symbol            string
	minNotional       float64
	lotMinQty         float64
	lotMaxQty         float64
	lotStepSize       float64
	marketLotMinQty   float64
	marketLotMaxQty   float64
	marketLotStepSize float64
}

// ╔══════════════╗
//     Ws Event
// ╚══════════════╝

type wsDepthEvent struct {
	Event            string     `json:"e"`
	Time             int64      `json:"E"`
	TransactionTime  int64      `json:"T"`
	Symbol           string     `json:"s"`
	FirstUpdateID    int64      `json:"U"`
	LastUpdateID     int64      `json:"u"`
	PrevLastUpdateID int64      `json:"pu"`
	Bids             [][]string `json:"b"`
	Asks             [][]string `json:"a"`
}
