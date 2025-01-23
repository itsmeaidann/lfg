package core

import (
	"context"
	"fmt"
	"lfg/pkg/ai"
	"sync"

	log "github.com/sirupsen/logrus"
)

func Run(ctx context.Context) error {
	log.Info("🦿 Running...")

	var wg sync.WaitGroup
	errChan := make(chan error, len(Agents))
	for _, agent := range Agents {
		wg.Add(1)
		go func(agent *ai.Agent) {
			defer wg.Done()
			if err := agent.Execute(ctx); err != nil {
				errChan <- err
			}
		}(agent)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors during execution: %v", errs)
	}
	return nil
}
