package project

import (
	"github.com/PastureStack/compose-cli/config"
	"golang.org/x/net/context"
)

type Secrets interface {
	Initialize(ctx context.Context) error
}

type SecretsFactory interface {
	Create(projectName string, secretConfigs map[string]*config.SecretConfig) (Secrets, error)
}
