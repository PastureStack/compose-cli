package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/lookup"
	"github.com/PastureStack/compose-cli/template"
)

func mergeWithResourceLookup(contents []byte, sourceFile string, resourceLookup config.ResourceLookup) (*config.Config, error) {
	return config.Merge(
		config.NewServiceConfigs(),
		&lookup.MapEnvLookup{Env: map[string]interface{}{}},
		resourceLookup,
		template.StackInfo{},
		template.EnvironmentInfo{},
		sourceFile,
		contents,
	)
}

func TestInMemoryComposeCannotReadExecutorHostEnvFile(t *testing.T) {
	hostFile := filepath.Join(t.TempDir(), "executor-secret.env")
	if err := os.WriteFile(hostFile, []byte("TOKEN=must-not-leak\n"), 0600); err != nil {
		t.Fatal(err)
	}
	compose := []byte(fmt.Sprintf("version: '2'\nservices:\n  app:\n    image: busybox\n    env_file:\n      - %s\n", filepath.ToSlash(hostFile)))

	parsed, err := mergeWithResourceLookup(compose, "", lookup.NewFileResourceLookup())
	if err == nil {
		t.Fatalf("in-memory Compose loaded executor host data: %#v", parsed.Services["app"].Environment)
	}
}

func TestLocalComposeCanReadEnvFileWithinConfiguredDirectory(t *testing.T) {
	projectDir := t.TempDir()
	composeFile := filepath.Join(projectDir, "compose.yml")
	envFile := filepath.Join(projectDir, "config", "app.env")
	if err := os.MkdirAll(filepath.Dir(envFile), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, []byte("TOKEN=allowed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	compose := []byte("version: '2'\nservices:\n  app:\n    image: busybox\n    env_file:\n      - config/app.env\n")
	if err := os.WriteFile(composeFile, compose, 0600); err != nil {
		t.Fatal(err)
	}

	parsed, err := mergeWithResourceLookup(compose, composeFile, lookup.NewFileResourceLookup(composeFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := []string(parsed.Services["app"].Environment); len(got) != 1 || got[0] != "TOKEN=allowed" {
		t.Fatalf("unexpected environment %#v", got)
	}
}
