package cataloglaunch

import client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"

// These are mutation inputs, not generic API resources. Empty links remain an
// explicit business value, and link aliases/targets retain the native parser's values.
type serviceLinkWire struct {
	Name      string `json:"name,omitempty"`
	ServiceID string `json:"serviceId"`
}

type serviceLinksWire struct {
	ServiceLinks []serviceLinkWire `json:"serviceLinks"`
}

func catalogLinksWire(input *client.SetServiceLinksInput) *serviceLinksWire {
	if input == nil {
		return nil
	}
	result := &serviceLinksWire{ServiceLinks: make([]serviceLinkWire, 0, len(input.ServiceLinks))}
	for _, link := range input.ServiceLinks {
		result.ServiceLinks = append(result.ServiceLinks, serviceLinkWire{Name: link.Name, ServiceID: link.ServiceId})
	}
	return result
}

// Keep the private command's existing error contract while sharing its DTO owner.
func catalogRequestWire(value interface{}) (map[string]interface{}, error) {
	wire, err := client.CatalogMutationRequest(value)
	if err != nil {
		return nil, ErrUnsupported
	}
	return wire, nil
}
