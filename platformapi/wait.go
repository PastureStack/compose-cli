package platformapi

import (
	"time"

	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func (r *PlatformService) WaitFor(resource *client.Resource, output interface{}, transitioning func() string) error {
	for {
		if transitioning() != "yes" {
			return nil
		}

		time.Sleep(150 * time.Millisecond)

		err := r.context.Client.Reload(resource, output)
		if err != nil {
			return err
		}
	}
}

func (r *PlatformService) Wait(service *client.Service) error {
	return r.WaitFor(&service.Resource, service, func() string {
		return service.Transitioning
	})
}

func (r *PlatformService) waitInstance(instance *client.Instance) error {
	return r.WaitFor(&instance.Resource, instance, func() string {
		return instance.Transitioning
	})
}
