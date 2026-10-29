package platformapi

import (
	"encoding/json"
	"errors"

	"github.com/PastureStack/compose-cli/config"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/utils"
)

var errCatalogGraph = errors.New("CatalogLaunchGraphInvalid")

type CatalogLaunchPart struct {
	Name         string
	Config       *config.ServiceConfig
	Transform    json.RawMessage
	VolumeDriver string
}

// CatalogFinalizeService uses the same completion and digest owner as normal
// creation. Engine supplies only its existing Docker transform and verified
// resource IDs, never Java copies of Compose business rules.
func CatalogFinalizeService(primary CatalogLaunchPart, secondaries []CatalogLaunchPart, secrets map[string]string) (string, *CompositeService, error) {
	decode := func(part CatalogLaunchPart) (client.LaunchConfig, error) {
		var result client.LaunchConfig
		if len(part.Transform) == 0 || json.Unmarshal(part.Transform, &result) != nil {
			return result, errCatalogGraph
		}
		if err := appendResolvedSecrets(&result, primary.Config.Secrets, func(name string) (string, error) {
			id, ok := secrets[name]
			if !ok || id == "" {
				return "", errCatalogGraph
			}
			return id, nil
		}); err != nil {
			return result, err
		}
		finishLaunchConfig(&result, part.Config, part.VolumeDriver)
		result.HealthCheck = part.Config.HealthCheck
		return result, nil
	}
	launch, err := decode(primary)
	if err != nil {
		return "", nil, err
	}
	secondaryLaunches := []client.SecondaryLaunchConfig{}
	for _, part := range secondaries {
		value, err := decode(part)
		if err != nil {
			return "", nil, err
		}
		var secondary client.SecondaryLaunchConfig
		if utils.Convert(value, &secondary) != nil {
			return "", nil, errCatalogGraph
		}
		secondary.Name = part.Name
		if secondary.Labels == nil {
			secondary.Labels = map[string]interface{}{}
		}
		secondaryLaunches = append(secondaryLaunches, secondary)
	}
	typeName, body := CatalogServiceCreateSkeleton(primary.Name, primary.Config)
	if err := populateLbFields(&PlatformService{name: primary.Name, serviceConfig: primary.Config}, &launch, body); err != nil {
		return "", nil, err
	}
	_, body, err = serviceConfigAndHash(body, &launch, secondaryLaunches)
	return typeName, body, err
}
