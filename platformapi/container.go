package platformapi

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/project"
	"github.com/PastureStack/compose-cli/project/options"
	"github.com/rancher/go-rancher/v2"
)

type PlatformContainer struct {
	name          string
	serviceConfig *config.ServiceConfig
	context       *Context
}

func (r *PlatformContainer) ID() string {
	return ""
}

func (r *PlatformContainer) Name() string {
	return r.name
}

func (r *PlatformContainer) Config() *config.ServiceConfig {
	return r.serviceConfig
}

func (r *PlatformContainer) Context() *Context {
	return r.context
}

func NewContainer(name string, config *config.ServiceConfig, context *Context) *PlatformContainer {
	return &PlatformContainer{
		name:          name,
		serviceConfig: config,
		context:       context,
	}
}

func (r *PlatformContainer) Create(ctx context.Context, options options.Create) error {
	fmt.Println(r.Name(), "Create")
	return nil
}

func (r *PlatformContainer) Up(ctx context.Context, options options.Up) error {
	fmt.Println(r.Name(), "Up")
	return nil
}

func (r *PlatformContainer) Build(ctx context.Context, buildOptions options.Build) error {
	fmt.Println(r.Name(), "Build")
	return nil
}

func (r *PlatformContainer) Log(ctx context.Context, follow bool) error {
	fmt.Println(r.Name(), "Log")
	return nil
}

func (r *PlatformContainer) DependentServices() []project.ServiceRelationship {
	return []project.ServiceRelationship{}
}

func (r *PlatformContainer) Client() *client.RancherClient {
	return r.context.Client
}

func (r *PlatformContainer) Pull(ctx context.Context) error {
	fmt.Println(r.Name(), "Pull")
	return nil
}
