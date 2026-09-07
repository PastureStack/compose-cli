package config_test

import (
	"github.com/PastureStack/compose-cli/lookup"
	"strings"
	"testing"
)

func TestHardwareSchemaAcceptsStandardGpuForms(t *testing.T) {
	for _, gpu := range []string{"gpus: all", "gpus:\n  - driver: nvidia\n    count: 2", "deploy:\n  resources:\n    reservations:\n      devices:\n        - driver: nvidia\n          device_ids: [GPU-one]\n          capabilities: [gpu]"} {
		settings := "image: example\nruntime: nvidia\nshm_size: 2g\npids_limit: 512\n" + gpu
		for _, header := range []string{"app:\n", "version: '2'\nservices:\n  app:\n"} {
			indent := "  "
			if strings.HasPrefix(header, "version") {
				indent = "    "
			}
			compose := header + indent + strings.ReplaceAll(settings, "\n", "\n"+indent) + "\n"
			parsed, err := mergeWithResourceLookup([]byte(compose), "", lookup.NewFileResourceLookup())
			if err != nil {
				t.Fatalf("%s: %v", compose, err)
			}
			app := parsed.Services["app"]
			if app.Runtime != "nvidia" || app.ShmSize != 2147483648 || app.PidsLimit == nil || *app.PidsLimit != 512 {
				t.Fatalf("lost options: %+v", app)
			}
		}
	}
}

func TestHardwareSchemaRejectsUnimplementedDeploySettings(t *testing.T) {
	for _, extra := range []string{"deploy:\n      replicas: 2", "deploy:\n      resources:\n        reservations:\n          devices:\n            - count: all", "gpus: 1", "gpus:\n      - count: false", "gpus:\n      - count: {}", "gpus:\n      - device_ids: [1]"} {
		_, err := mergeWithResourceLookup([]byte("version: '2'\nservices:\n  app:\n    image: example\n    "+extra+"\n"), "", lookup.NewFileResourceLookup())
		if err == nil {
			t.Fatalf("silently accepted %s", extra)
		}
	}
}
