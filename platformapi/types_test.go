package platformapi

import (
	"testing"

	"github.com/PastureStack/compose-cli/config"
)

func TestFindServiceTypeAcceptsPastureStackAndPersistedSystemImages(t *testing.T) {
	tests := []struct {
		image string
		want  ServiceType
	}{
		{config.ExternalServiceImage, ExternalServiceType},
		{"rancher/external-service", ExternalServiceType},
		{config.LoadBalancerServiceImage, LegacyLbServiceType},
		{"rancher/load-balancer-service", LegacyLbServiceType},
		{config.InternalDNSServiceImage, DnsServiceType},
		{"rancher/dns-service", DnsServiceType},
	}

	for _, test := range tests {
		service := &PlatformService{
			serviceConfig: &config.ServiceConfig{Image: test.image},
		}
		if got := FindServiceType(service); got != test.want {
			t.Errorf("FindServiceType(%q) = %v, want %v", test.image, got, test.want)
		}
	}
}
