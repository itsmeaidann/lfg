package types

type ExchangeName string

const (
	ExchangeDummy = ExchangeName("dummy") // dummy exchange
	ExchangeBnf   = ExchangeName("bnf")
	ExchangeHpl   = ExchangeName("hpl")
)
