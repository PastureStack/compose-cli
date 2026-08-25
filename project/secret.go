package project

import (
	"context"
	"github.com/PastureStack/compose-cli/config"
)

type Secrets interface {
	Initialize(ctx context.Context) error
}

type SecretsFactory interface {
	Create(projectName string, secretConfigs map[string]*config.SecretConfig) (Secrets, error)
}
