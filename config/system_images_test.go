package config

import "testing"

func TestSystemImageNames(t *testing.T) {
	if LoadBalancerServiceImage != "ghcr.io/pasturestack/load-balancer-service" {
		t.Fatalf("unexpected load balancer image: %s", LoadBalancerServiceImage)
	}
	if InternalDNSServiceImage != "ghcr.io/pasturestack/internal-dns" {
		t.Fatalf("unexpected internal DNS image: %s", InternalDNSServiceImage)
	}
	if ExternalServiceImage != "ghcr.io/pasturestack/external-service" {
		t.Fatalf("unexpected external service image: %s", ExternalServiceImage)
	}
}

func TestSystemImageRecognitionIncludesPersistedLegacyNames(t *testing.T) {
	tests := []struct {
		name  string
		match func(string) bool
		image string
	}{
		{"PastureStack load balancer", IsLoadBalancerServiceImage, LoadBalancerServiceImage},
		{"persisted load balancer", IsLoadBalancerServiceImage, legacyLoadBalancerServiceImage},
		{"PastureStack internal DNS", IsInternalDNSServiceImage, InternalDNSServiceImage},
		{"persisted internal DNS", IsInternalDNSServiceImage, legacyInternalDNSServiceImage},
		{"PastureStack external service", IsExternalServiceImage, ExternalServiceImage},
		{"persisted external service", IsExternalServiceImage, legacyExternalServiceImage},
	}

	for _, test := range tests {
		if !test.match(test.image) {
			t.Errorf("%s was not recognized", test.name)
		}
		if test.match("example.invalid/custom-system-image") {
			t.Errorf("%s matcher accepted an unrelated image", test.name)
		}
	}
}

func TestCreateRawConfigEmitsPastureStackSystemImages(t *testing.T) {
	contents := []byte(`
version: "2"
external_services:
  database:
    external_ips:
      - 192.0.2.10
aliases:
  api-alias:
    services:
      - api
`)

	rawConfig, err := CreateRawConfig(contents)
	if err != nil {
		t.Fatal(err)
	}

	if image := rawConfig.Services["database"]["image"]; image != ExternalServiceImage {
		t.Errorf("external service image = %v, want %s", image, ExternalServiceImage)
	}
	if image := rawConfig.Services["api-alias"]["image"]; image != InternalDNSServiceImage {
		t.Errorf("alias image = %v, want %s", image, InternalDNSServiceImage)
	}
}
