package core

import (
	"context"
	"fmt"
	"lfg/config"

	log "github.com/sirupsen/logrus"
)

func Bootstrap(ctx context.Context, config config.Config) error {
	log.Info("🦾 Bootstrapping...")

	// register exchanges
	for exchgId, exchgConfig := range config.ExchangeConfigs {
		RegisterExchange(exchgId, exchgConfig)
		log.Infof("exchange '%v' registered", exchgId)
	}

	// register agents and plan tasks
	for agentId, agentConfig := range config.AgentConfigs {
		exchanges := make([]string, len(agentConfig.Exchange))
		for i, e := range agentConfig.Exchange {
			exchanges[i] = *e
		}
		if err := RegisterAgent(agentId, agentConfig.Prompt, exchanges); err != nil {
			return fmt.Errorf("failed to register agent %v: %w", agentId, err)
		}
		log.Infof("agent '%v' registered", agentId)

		agent, exists := Agents[agentId]
		if !exists {
			return fmt.Errorf("agent %v not found", agentId)
		}

		if err := agent.Plan(ctx); err != nil {
			return fmt.Errorf("failed to plan tasks for agent %v: %w", agentId, err)
		}
		log.Infof("agent %v tasks planned successfully: %v steps", agentId, len(agent.Tasks))
	}
	return nil
}
