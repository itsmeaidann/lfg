package core

import (
	"lfg/config"
	"lfg/pkg/ai/agent"
	"lfg/pkg/exchange"
)

var Exchanges map[string]*exchange.Exchange
var Agents map[string]*agent.UserAgent

func init() {
	Exchanges = make(map[string]*exchange.Exchange)
	Agents = make(map[string]*agent.UserAgent)
}

func RegisterAgent(agentId string, prompt string, exchangeIds []string) error {
	agentExchanges := make(map[string]*exchange.Exchange)
	for _, exchangeId := range exchangeIds {
		if exchange, exists := Exchanges[exchangeId]; exists {
			agentExchanges[exchangeId] = exchange
		}
	}
	agent, err := agent.NewUserAgent(agentId, prompt, agentExchanges)
	if err != nil {
		return err
	}
	Agents[agentId] = agent
	return nil
}

func RegisterExchange(exchgId string, exchgConfig *config.ExchangeConfig) error {
	exchange, err := exchange.NewExchange(exchgId, exchgConfig)
	if err != nil {
		return err
	}
	Exchanges[exchgId] = &exchange
	return nil
}
