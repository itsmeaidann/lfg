package types

type Stream string

const (
	StreamTrade     = Stream("Trade")
	StreamKLine     = Stream("KLine")
	StreamMarkPrice = Stream("MarkPrice")
	StreamBookDepth = Stream("BookDepth")
	StreamOrder     = Stream("Order")
	StreamOrderMgmt = Stream("OrderMgmt")
)
