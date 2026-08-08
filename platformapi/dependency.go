package platformapi

import (
	"golang.org/x/net/context"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/project"
)

type PlatformDependenciesFactory struct {
	Context *Context
}

func (f *PlatformDependenciesFactory) Create(projectName string, dependencyConfigs map[string]*config.DependencyConfig) (project.Dependencies, error) {
	dependencies := make([]*Dependency, 0, len(dependencyConfigs))
	for name, config := range dependencyConfigs {
		dependencies = append(dependencies, &Dependency{
			context:     f.Context,
			name:        name,
			projectName: projectName,
			template:    config.Template,
			version:     config.Version,
		})
	}
	return &Dependencies{
		dependencies: dependencies,
	}, nil
}

type Dependencies struct {
	dependencies []*Dependency
	Context      *Context
}

func (h *Dependencies) Initialize(ctx context.Context) error {
	for _, dependency := range h.dependencies {
		if err := dependency.EnsureItExists(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Dependency struct {
	context     *Context
	name        string
	projectName string
	template    string
	version     string
}

func (d *Dependency) EnsureItExists(ctx context.Context) error {
	return nil
}
