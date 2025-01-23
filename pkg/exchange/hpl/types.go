package hpl

import "encoding/json"

type hplConfig struct {
	ApiUrl  string `json:"apiUrl"`
	WsUrl   string `json:"wsUrl"`
	ChainId int64  `json:"chainId"`
}

// ╔══════════════╗
//     Ws Event
// ╚══════════════╝

type wsGenericResponse struct {
	Channel string `json:"channel"`
}
type wsPostResponse struct {
	Channel string `json:"channel"`
	Data    struct {
		Id       int64 `json:"id"`
		Response struct {
			Type string `json:"type"`
		}
	}
}

type wsPostInfoResponse struct {
	Channel string `json:"channel"`
	Data    struct {
		Id       int64 `json:"id"`
		Response struct {
			Type    string `json:"type"`
			Payload struct {
				Type     string          `json:"type"`
				Status   string          `json:"status,omitempty"`
				Data     json.RawMessage `json:"data,omitempty"`
				Response string          `json:"response,omitempty"`
			} `json:"payload"`
		} `json:"response"`
	} `json:"data"`
}

type wsPostActionResponse struct {
	Channel string `json:"channel"`
	Data    struct {
		Id       int64 `json:"id"`
		Response struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		} `json:"response"`
	} `json:"data"`
}

type wsActiveAssetCtxResponse struct {
	Channel string           `json:"channel"`
	Data    wsActiveAssetCtx `json:"data"`
}

type wsActiveAssetCtx struct {
	Coin string          `json:"coin"`
	Ctx  wsPerpsAssetCtx `json:"ctx"`
}

type wsPerpsAssetCtx struct {
	DayNtlVlm    string  `json:"dayNtlVlm"`
	PrevDayPx    string  `json:"prevDayPx"`
	MarkPx       string  `json:"markPx"`
	MidPx        *string `json:"midPx,omitempty"` // pointer allows for optional fields
	Funding      string  `json:"funding"`
	OpenInterest string  `json:"openInterest"`
	OraclePx     string  `json:"oraclePx"`
}

type wsTradeResponse struct {
	Channel string         `json:"channel"`
	Data    []wsTradeEvent `json:"data"`
}

type wsTradeEvent struct {
	Coin string `json:"coin"`
	Side string `json:"side"`
	Px   string `json:"px"`
	Sz   string `json:"sz"`
	Time int64  `json:"time"`
	Hash string `json:"hash"`
	Tid  int64  `json:"tid"`
}

type wsOrderResponse struct {
	Channel string         `json:"channel"`
	Data    []wsOrderEvent `json:"data"`
}

type wsOrderEvent struct {
	Order           wsBasicOrder `json:"order"`
	Status          string       `json:"status"`
	StatusTimestamp int64        `json:"statusTimestamp"`
}

type wsBasicOrder struct {
	Symbol    string  `json:"coin"`
	Side      string  `json:"side"`
	LimitPx   string  `json:"limitPx"`
	Sz        string  `json:"sz"`
	Oid       int64   `json:"oid"`
	Timestamp int64   `json:"timestamp"`
	OrigSz    string  `json:"origSz"`
	Cloid     *string `json:"cloid,omitempty"`
}

type wsKLineRes struct {
	Channel string       `json:"channel"`
	Data    wsKLineEvent `json:"data"`
}

type wsKLineEvent struct {
	OpenT    int64  `json:"t"` // open millis
	CloseT   int64  `json:"T"` // close millis
	Symbol   string `json:"s"` // coin
	Interval string `json:"i"` // interval
	Open     string `json:"o"` // open price
	Close    string `json:"c"` // close price
	High     string `json:"h"` // high price
	Low      string `json:"l"` // low price
	Volume   string `json:"v"` // volume (base unit)
	Trades   int    `json:"n"` // number of trades
}

type wsBookDepthRes struct {
	Channel string           `json:"channel"`
	Data    wsBookDepthEvent `json:"data"`
}

type wsBookDepthEvent struct {
	Coin   string           `json:"coin"`
	Levels [2][]wsBookLevel `json:"levels"`
	Time   int64            `json:"time"`
}

type wsBookLevel struct {
	Price  string `json:"px"`
	Sz     string `json:"sz"`
	Orders int64  `json:"n"`
}

// ╔════════════════════════╗
//    API request/response
// ╚════════════════════════╝

type universe struct {
	SzDecimals   int    `json:"szDecimals"`
	Name         string `json:"name"`
	MaxLeverage  int    `json:"maxLeverage"`
	OnlyIsolated bool   `json:"onlyIsolated"`
}

type tifType string

const (
	tifTypeALO = tifType("Alo")
	tifTypeIOC = tifType("Ioc")
	tifTypeGTC = tifType("Gtc")
)

type grouping string

const (
	groupingNa   grouping = "na"
	groupingTpSl grouping = "positionTpsl"
)

type orderTypeWire struct {
	Limit   *limit   `msgpack:"limit,omitempty" json:"limit,omitempty"`
	Trigger *trigger `msgpack:"trigger,omitempty" json:"trigger,omitempty"`
}

type limit struct {
	Tif tifType `msgpack:"tif" json:"tif"`
}

type trigger struct {
	IsMarket  bool   `json:"isMarket" msgpack:"isMarket"`
	TriggerPx string `json:"triggerPx" msgpack:"triggerPx"`
	TpSl      tpSl   `json:"tpsl" msgpack:"tpsl"`
}

type tpSl string

const (
	triggerTp tpSl = "tp"
	triggerSl tpSl = "sl"
)

type orderWire struct {
	Asset      int           `msgpack:"a" json:"a"`
	IsBuy      bool          `msgpack:"b" json:"b"`
	LimitPx    string        `msgpack:"p" json:"p"`
	SizePx     string        `msgpack:"s" json:"s"`
	ReduceOnly bool          `msgpack:"r" json:"r"`
	OrderType  orderTypeWire `msgpack:"t" json:"t"`
	Cloid      *string       `msgpack:"c,omitempty" json:"c,omitempty"`
}

type cancelWire struct {
	Asset      int    `msgpack:"a,omitempty" json:"a,omitempty"` // set this field if using oId
	OrderId    int    `msgpack:"o,omitempty" json:"o,omitempty"`
	AssetCloId int    `msgpack:"asset,omitempty" json:"asset,omitempty"` // set this field if using cloId
	CloId      string `msgpack:"cloid" json:"cloid"`
}

type orderAction struct {
	Type     string       `msgpack:"type" json:"type"`
	OId      int          `msgpack:"oid,omitempty" json:"oid,omitempty"`
	Order    *orderWire   `msgpack:"order,omitempty" json:"order,omitempty"`
	Orders   []orderWire  `msgpack:"orders,omitempty" json:"orders,omitempty"`
	Cancels  []cancelWire `msgpack:"cancels,omitempty" json:"cancels,omitempty"`
	Grouping string       `msgpack:"grouping,omitempty" json:"grouping,omitempty"`
}

type orderActionCloId struct {
	Type     string       `msgpack:"type" json:"type"`
	OId      string       `msgpack:"oid,omitempty" json:"oid,omitempty"`
	Order    *orderWire   `msgpack:"order,omitempty" json:"order,omitempty"`
	Orders   []orderWire  `msgpack:"orders,omitempty" json:"orders,omitempty"`
	Cancels  []cancelWire `msgpack:"cancels,omitempty" json:"cancels,omitempty"`
	Grouping string       `msgpack:"grouping,omitempty" json:"grouping,omitempty"`
}

type leverageAction struct {
	Type     string `msgpack:"type" json:"type"`
	Asset    int    `msgpack:"asset" json:"asset"`
	IsCross  bool   `msgpack:"isCross" json:"isCross"`
	Leverage int    `msgpack:"leverage" json:"leverage"`
}

type orderActionRequest struct {
	Action       any          `json:"action"`
	Nonce        int64        `json:"nonce"`
	Signature    RsvSignature `json:"signature"`
	VaultAddress *string      `json:"vaultAddress"`
}

type metadataRequest struct {
	Type string `json:"type"`
	User string `json:"user"`
}

type updateLeverageRequest struct {
	Action       leverageAction `json:"action"`
	Nonce        int64          `json:"nonce"`
	Signature    RsvSignature   `json:"signature"`
	VaultAddress *string        `json:"vaultAddress"`
}

type marketInfoResponse struct {
	Universe []universe `json:"universe"`
}

type openOrderResponse struct {
	Status   string `json:"status"`
	Response struct {
		Type string `json:"type"`
		Data struct {
			Statuses []struct {
				Error   string `json:"error,omitempty"`
				Resting struct {
					Oid int64 `json:"oid,omitempty"`
				} `json:"resting,omitempty"`
			} `json:"statuses"`
		} `json:"data"`
	} `json:"response"`
}

type modifyOrderResponse struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

type cancelOrderResponse struct {
	Status   string `json:"status"`
	Response struct {
		Type string `json:"type"`
		Data struct {
			Statuses json.RawMessage `json:"statuses"`
		} `json:"data"`
	} `json:"response"`
}

type pendingOrderResponse struct {
	OId       int64  `json:"oid"`
	Coin      string `json:"coin"`
	Side      string `json:"side"`
	LimitPx   string `json:"limitPx"`
	Sz        string `json:"sz"`
	OrigSz    string `json:"origSz"`
	Timestamp int64  `json:"timestamp"`
}

type kLineResponse struct {
	OpenT  int64  `json:"t"`
	CloseT int64  `json:"T"`
	Symbol string `json:"s"`
	Open   string `json:"o"`
	Close  string `json:"c"`
	High   string `json:"h"`
	Low    string `json:"l"`
	Volume string `json:"v"`
}

type accountBalanceResponse struct {
	MarginSummary struct {
		AccountValue string `json:"accountValue"`
	} `json:"marginSummary"`
	AssetPositions []assetPosition `json:"assetPositions"`
}

type assetPosition struct {
	Position struct {
		Coin    string `json:"coin"`
		Szi     string `json:"szi"`
		EntryPx string `json:"entryPx"`
	} `json:"position"`
}

type updateLeverageResponse struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response"`
}

// ╔════════════════════════╗
//         Signature
// ╚════════════════════════╝

type EIP712Domain struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainId           int    `json:"chainId"`
	VerifyingContract string `json:"verifyingContract"`
}

type Agent struct {
	Source       string `json:"source"`
	ConnectionId string `json:"connectionId"`
}
type RsvSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V uint8  `json:"v"`
}
