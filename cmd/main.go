package main

import (
	"context"
	"lfg/config"
	"lfg/core"
	"lfg/pkg/types"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	configureLog(config.Env.EnvName)

	// init context for graceful shutdown
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load config & prompt
	config, err := config.LoadConfig(config.Env.EnvName)
	if err != nil {
		log.Fatalf("fail to load config: %v", err)
	}

	// trap signal for graceful shutdown
	setupSignalHandler(cancel)

	// 📊 core: lfg module
	err = core.Bootstrap(rootCtx, *config)
	if err != nil {
		log.Panicf("fail to bootstrap app: %v", err)
	}
	go func() {
		if err := core.Run(rootCtx); err != nil {
			log.Errorf("Runtime error: %v", err)
			cancel()
		}
	}()

	// 🌩️ fiber: rest API module
	fApp := core.SetupFiberApp()
	if err := fApp.Listen(":3000"); err != nil {
		log.Panic(err)
	}
}

func configureLog(envName types.EnvName) {
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	if envName == types.EnvLocal || envName == types.EnvDev {
		log.SetLevel(log.DebugLevel)
	}
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigC
		log.Info("🚩 received shutdown signal")
		cancel()
	}()
}
