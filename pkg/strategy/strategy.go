package strategy

import (
	"context"
	"lfg/pkg/types"
)

type Strategy interface {
	Id() string
	Name() types.StrategyName
	Validate() error
	Run(ctx context.Context) error
	Shutdown() error
}
