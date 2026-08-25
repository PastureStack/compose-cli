package platformapi

import (
	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

const (
	PlatformType        = ServiceType(iota)
	LegacyLbServiceType = ServiceType(iota)
	LbServiceType       = ServiceType(iota)
	DnsServiceType      = ServiceType(iota)
	ExternalServiceType = ServiceType(iota)
	StorageDriverType   = ServiceType(iota)
	NetworkDriverType   = ServiceType(iota)
)

type ServiceType int

func FindServiceType(r *PlatformService) ServiceType {
	if config.IsExternalServiceImage(r.serviceConfig.Image) {
		return ExternalServiceType
	} else if config.IsLoadBalancerServiceImage(r.serviceConfig.Image) {
		return LegacyLbServiceType
	} else if isLbServiceType(r.serviceConfig.LbConfig) {
		return LbServiceType
	} else if config.IsInternalDNSServiceImage(r.serviceConfig.Image) {
		return DnsServiceType
	} else if r.serviceConfig.NetworkDriver != nil {
		return NetworkDriverType
	} else if r.serviceConfig.StorageDriver != nil {
		return StorageDriverType
	}

	return PlatformType
}

func isLbServiceType(lbConfig *config.LBConfig) bool {
	if lbConfig == nil {
		return false
	}

	for _, portRule := range lbConfig.PortRules {
		if portRule.SourcePort != 0 {
			return true
		}
	}

	return false
}

type CompositeService struct {
	client.Service

	StorageDriver *client.StorageDriver `json:"storageDriver,omitempty" yaml:"storageDriver,omitempty"`
	NetworkDriver *client.NetworkDriver `json:"networkDriver,omitempty" yaml:"networkDriver,omitempty"`
	RealLbConfig  *client.LbConfig      `json:"lbConfig,omitempty" yaml:"lb_config,omitempty"`

	// External Service Fields
	ExternalIpAddresses []string                    `json:"externalIpAddresses,omitempty" yaml:"external_ip_addresses,omitempty"`
	Hostname            string                      `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	HealthCheck         *client.InstanceHealthCheck `json:"healthCheck,omitempty" yaml:"health_check,omitempty"`
}
