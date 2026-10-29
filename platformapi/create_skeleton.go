package platformapi

import (
	"github.com/PastureStack/compose-cli/config"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/utils"
)

// serviceCreateFields is shared by ordinary creation and the typed preflight.
// Launch configs are supplied by the existing Docker transform owner later.
func serviceCreateFields(name string, c *config.ServiceConfig, stackID string) *CompositeService {
	scale := int64(c.Scale)
	if scale <= 0 {
		scale = 1
	}
	service := &CompositeService{Service: client.Service{Name: name, Metadata: utils.NestedMapsToMapInterface(c.Metadata), Scale: scale, ScalePolicy: c.ScalePolicy,
		RetainIp: c.RetainIp, StackId: stackID, SelectorContainer: c.Labels["io.rancher.service.selector.container"], SelectorLink: c.Labels["io.rancher.service.selector.link"]},
		ExternalIpAddresses: c.ExternalIps, Hostname: c.Hostname, HealthCheck: c.HealthCheck, StorageDriver: c.StorageDriver, NetworkDriver: c.NetworkDriver}
	if service.NetworkDriver != nil {
		service.NetworkDriver.CniConfig = convertNestedMapKeysToStrings(service.NetworkDriver.CniConfig)
	}
	return service
}

func serviceCreateType(r *PlatformService) string {
	switch FindServiceType(r) {
	case ExternalServiceType:
		return client.EXTERNAL_SERVICE_TYPE
	case DnsServiceType:
		return client.DNS_SERVICE_TYPE
	case LegacyLbServiceType, LbServiceType:
		return client.LOAD_BALANCER_SERVICE_TYPE
	case StorageDriverType:
		return client.STORAGE_DRIVER_SERVICE_TYPE
	case NetworkDriverType:
		return client.NETWORK_DRIVER_SERVICE_TYPE
	default:
		return client.SERVICE_TYPE
	}
}

// CatalogServiceCreateSkeleton has a closed symbolic Stack slot. Engine must
// bind it to the Stack created by this exact claimed plan, never a caller ID.
func CatalogServiceCreateSkeleton(name string, c *config.ServiceConfig) (string, *CompositeService) {
	return serviceCreateType(&PlatformService{serviceConfig: c}), serviceCreateFields(name, c, "$createdStack")
}

func volumeCreateFields(name string, c *config.VolumeConfig, stackID string) *client.VolumeTemplate {
	options := map[string]interface{}{}
	for key, value := range c.DriverOpts {
		options[key] = value
	}
	return &client.VolumeTemplate{Name: name, Driver: c.Driver, DriverOpts: options, External: c.External.External, PerContainer: c.PerContainer, StackId: stackID}
}

func CatalogVolumeCreateSkeleton(name string, c *config.VolumeConfig) *client.VolumeTemplate {
	return volumeCreateFields(name, c, "$createdStack")
}

func serviceCreatePayload(kind string, body *CompositeService) interface{} {
	if kind == client.SERVICE_TYPE {
		return &body.Service
	}
	return body
}

func CatalogServiceRequestPayload(kind string, body *CompositeService) interface{} {
	return serviceCreatePayload(kind, body)
}
