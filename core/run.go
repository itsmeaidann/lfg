package core

import (
	"context"
	"fmt"
	"lfg/pkg/ai/agent"
	"sync"

	log "github.com/sirupsen/logrus"
)

func Run(ctx context.Context) error {
	log.Info("🦿 Running...")

	var wg sync.WaitGroup
	errChan := make(chan error, len(Agents))
	for _, ag := range Agents {
		wg.Add(1)
		go func(ag *agent.UserAgent) {
			defer wg.Done()
			if err := ag.Execute(ctx); err != nil {
				errChan <- err
			}
		}(ag)
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
