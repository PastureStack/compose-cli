package project

import (
	"context"
	"github.com/PastureStack/compose-cli/config"
)

type Hosts interface {
	Initialize(ctx context.Context) error
}

type HostsFactory interface {
	Create(projectName string, hostConfigs map[string]*config.HostConfig) (Hosts, error)
}
