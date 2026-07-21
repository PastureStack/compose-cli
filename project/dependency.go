package project

import (
	"github.com/PastureStack/compose-cli/config"
	"golang.org/x/net/context"
)

type Dependencies interface {
	Initialize(ctx context.Context) error
}

type DependenciesFactory interface {
	Create(projectName string, dependencyConfigs map[string]*config.DependencyConfig) (Dependencies, error)
}
