package platformapi

import (
	"bytes"
	"encoding/json"

	"github.com/PastureStack/compose-cli/digest"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

// Catalog hashes describe the approved typed request, before the owned Stack
// slot is bound. Native transform response envelopes and deployment-assigned
// Stack IDs must not change those hashes. Ordinary/legacy hashing is unchanged.
func catalogServiceConfigAndHash(service *CompositeService, launch *client.LaunchConfig, secondaries []client.SecondaryLaunchConfig) (digest.ServiceHash, *CompositeService, error) {
	if service == nil || launch == nil {
		return digest.ServiceHash{}, nil, errCatalogGraph
	}
	input := *service
	input.LaunchConfig = launch
	input.SecondaryLaunchConfigs = secondaries
	wire, err := client.CatalogMutationRequest(&input)
	if err != nil {
		return digest.ServiceHash{}, nil, err
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		return digest.ServiceHash{}, nil, err
	}
	var canonical CompositeService
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&canonical); err != nil {
		return digest.ServiceHash{}, nil, err
	}
	launch = canonical.LaunchConfig
	secondaries = canonical.SecondaryLaunchConfigs
	if launch == nil {
		return digest.ServiceHash{}, nil, errCatalogGraph
	}
	// JSON omitempty can drop empty maps. Restore only the maps this existing
	// hash owner writes; business entries and numeric values remain unchanged.
	if canonical.Metadata == nil {
		canonical.Metadata = map[string]interface{}{}
	}
	if launch.Labels == nil {
		launch.Labels = map[string]interface{}{}
	}
	for index := range secondaries {
		if secondaries[index].Labels == nil {
			secondaries[index].Labels = map[string]interface{}{}
		}
	}
	canonical.LaunchConfig = nil
	canonical.SecondaryLaunchConfigs = nil
	boundStackID := canonical.StackId
	canonical.StackId = "$createdStack"
	hash, body, err := serviceConfigAndHash(&canonical, launch, secondaries)
	if err != nil {
		return digest.ServiceHash{}, nil, err
	}
	body.StackId = boundStackID
	return hash, body, nil
}
