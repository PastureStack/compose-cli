package platformapi

import (
	"github.com/PastureStack/compose-cli/project"
	"github.com/sirupsen/logrus"
)

func NewProject(context *Context) (*project.Project, error) {
	context.ServiceFactory = &PlatformServiceFactory{
		Context: context,
	}

	context.ContainerFactory = &PlatformContainerFactory{
		Context: context,
	}

	context.DependenciesFactory = &PlatformDependenciesFactory{
		Context: context,
	}

	context.VolumesFactory = &PlatformVolumesFactory{
		Context: context,
	}

	context.HostsFactory = &PlatformHostsFactory{
		Context: context,
	}

	context.SecretsFactory = &PlatformSecretsFactory{
		Context: context,
	}

	p := project.NewProject(&context.Context)
	err := p.Open()
	if err != nil {
		return nil, err
	}

	if err := context.open(); err != nil {
		logrus.Errorf("Failed to open project %s: %v", p.Name, err)
		return nil, err
	}

	if err := p.Parse(); err != nil {
		return nil, err
	}

	if context.Prune {
		pruneServices(context, p)
	}
	sidekickInfo, err := NewSidekickInfo(p)
	if err != nil {
		return nil, err
	}

	context.SidekickInfo = sidekickInfo

	return p, nil
}
