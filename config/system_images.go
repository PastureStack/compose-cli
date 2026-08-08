package config

const (
	LoadBalancerServiceImage = "ghcr.io/pasturestack/load-balancer-service"
	InternalDNSServiceImage  = "ghcr.io/pasturestack/internal-dns"
	ExternalServiceImage     = "ghcr.io/pasturestack/external-service"

	legacyLoadBalancerServiceImage = "rancher/load-balancer-service"
	legacyInternalDNSServiceImage  = "rancher/dns-service"
	legacyExternalServiceImage     = "rancher/external-service"
)

func IsLoadBalancerServiceImage(image string) bool {
	return image == LoadBalancerServiceImage || image == legacyLoadBalancerServiceImage
}

func IsInternalDNSServiceImage(image string) bool {
	return image == InternalDNSServiceImage || image == legacyInternalDNSServiceImage
}

func IsExternalServiceImage(image string) bool {
	return image == ExternalServiceImage || image == legacyExternalServiceImage
}
