package config_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/PastureStack/compose-cli/lookup"
)

func TestYAMLV3MappingsRemainStructuredDuringPreprocessing(t *testing.T) {
	compose := []byte(`version: '2'
services:
  app:
    image: example
    environment:
      ENABLED: true
      RETRIES: 3
    labels:
      io.example.enabled: true
      io.example.retries: 3
`)

	parsed, err := mergeWithResourceLookup(compose, "", lookup.NewFileResourceLookup())
	if err != nil {
		t.Fatal(err)
	}

	app := parsed.Services["app"]
	wantEnvironment := []string{"ENABLED=true", "RETRIES=3"}
	gotEnvironment := []string(app.Environment)
	sort.Strings(gotEnvironment)
	if !reflect.DeepEqual(gotEnvironment, wantEnvironment) {
		t.Fatalf("unexpected environment: got %#v want %#v", app.Environment, wantEnvironment)
	}

	wantLabels := map[string]string{
		"io.example.enabled": "true",
		"io.example.retries": "3",
	}
	if !reflect.DeepEqual(map[string]string(app.Labels), wantLabels) {
		t.Fatalf("unexpected labels: got %#v want %#v", app.Labels, wantLabels)
	}
}
