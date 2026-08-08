package platformapi

import (
	"encoding/base64"
	"fmt"

	"golang.org/x/net/context"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/project"
	"github.com/rancher/go-rancher/v2"
	log "github.com/sirupsen/logrus"
)

type PlatformSecretsFactory struct {
	Context *Context
}

func (f *PlatformSecretsFactory) Create(projectName string, secretConfigs map[string]*config.SecretConfig) (project.Secrets, error) {
	secrets := make([]*Secret, 0, len(secretConfigs))
	for name, config := range secretConfigs {
		secrets = append(secrets, &Secret{
			context:     f.Context,
			name:        name,
			projectName: projectName,
			file:        config.File,
			external:    config.External,
		})
	}
	return &Secrets{
		secrets: secrets,
		Context: f.Context,
	}, nil
}

type Secrets struct {
	secrets []*Secret
	Context *Context
}

func (s *Secrets) Initialize(ctx context.Context) error {
	for _, secret := range s.secrets {
		if err := secret.EnsureItExists(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Secret struct {
	context     *Context
	name        string
	projectName string
	file        string
	external    string
}

func secretResourceBase(composeFiles []string) string {
	if len(composeFiles) > 0 {
		return composeFiles[0]
	}
	return "./"
}

func (s *Secret) EnsureItExists(ctx context.Context) error {
	existingSecrets, err := s.context.Client.Secret.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name": s.name,
		},
	})
	if err != nil {
		return err
	}
	if len(existingSecrets.Data) > 0 {
		log.Infof("Secret %s already exists", s.name)
		return nil
	}
	if s.external != "" {
		return fmt.Errorf("Existing secret %s not found", s.name)
	}
	relativeTo := secretResourceBase(s.context.ComposeFiles)
	contents, filename, err := s.context.ResourceLookup.Lookup(s.file, relativeTo)
	if err != nil {
		return err
	}
	log.Infof("Creating secret %s with contents from file %s", s.name, filename)
	_, err = s.context.Client.Secret.Create(&client.Secret{
		Name:  s.name,
		Value: base64.StdEncoding.EncodeToString(contents),
	})
	return err
}
