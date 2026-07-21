package platformapi

import (
	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/project"
)

type PlatformContainerFactory struct {
	Context *Context
}

func (r *PlatformContainerFactory) Create(project *project.Project, name string, serviceConfig *config.ServiceConfig) (project.Service, error) {
	return NewContainer(name, serviceConfig, r.Context), nil
}
