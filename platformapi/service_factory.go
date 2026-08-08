package platformapi

import (
	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/project"
)

type PlatformServiceFactory struct {
	Context *Context
}

func (r *PlatformServiceFactory) Create(project *project.Project, name string, serviceConfig *config.ServiceConfig) (project.Service, error) {
	if len(r.Context.SidekickInfo.sidekickToPrimaries[name]) > 0 {
		return NewSidekick(name, serviceConfig, r.Context), nil
	} else {
		return NewService(name, serviceConfig, r.Context), nil
	}
}
