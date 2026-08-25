package project

import (
	"context"
	"github.com/PastureStack/compose-cli/config"
)

type Dependencies interface {
	Initialize(ctx context.Context) error
}

type DependenciesFactory interface {
	Create(projectName string, dependencyConfigs map[string]*config.DependencyConfig) (Dependencies, error)
}
