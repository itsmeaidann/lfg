package test

import (
	"context"
	"fmt"
	"lfg/pkg/ai/agent"
	"lfg/pkg/exchange"
	"lfg/pkg/exchange/hpl"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func RunTest() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize exchange
	hplExchange, err := hpl.NewExchange()
	if err != nil {
		log.Fatal(err)
	}

	// Create agent
	prompt := "I want to buy BTC when price is below 60000 and sell when price is above 70000"
	exchanges := map[string]*exchange.Exchange{
		"hpl": hplExchange,
	}
	agent, err := agent.NewUserAgent("test", prompt, exchanges)
	if err != nil {
		log.Fatal(err)
	}

	// Plan strategy
	err = agent.Plan(context.Background())
	if err != nil {
		fmt.Printf("Error planning strategy: %v\n", err)
		os.Exit(1)
	}

	// Execute strategy
	err = agent.Execute(context.Background())
	if err != nil {
		fmt.Printf("Error executing strategy: %v\n", err)
		os.Exit(1)
	}
}
