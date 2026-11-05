package platformapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/digest"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/project"
	"github.com/PastureStack/compose-cli/yaml"
)

func TestCatalogHashEmptyMapsAndInvalidInputs(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicitEmpty=%v", empty), func(t *testing.T) {
			body := &CompositeService{Service: client.Service{Name: "app", StackId: "1st1600"}}
			launch := &client.LaunchConfig{ImageUuid: "docker:busybox:1.37.0"}
			secondaries := []client.SecondaryLaunchConfig{{Name: "side", ImageUuid: "docker:busybox:1.37.0"}}
			if empty {
				body.Metadata = map[string]interface{}{}
				launch.Labels = map[string]interface{}{}
				secondaries[0].Labels = map[string]interface{}{}
			}
			hash, result, err := catalogServiceConfigAndHash(body, launch, secondaries)
			if err != nil || hash.Service == "" || hash.LaunchConfig == "" || result.StackId != "1st1600" ||
				result.Metadata[digest.ServiceHashKey] != hash.Service || result.LaunchConfig.Labels[digest.ServiceHashKey] != hash.LaunchConfig ||
				result.SecondaryLaunchConfigs[0].Labels[digest.ServiceHashKey] != hash.SecondaryLaunchConfigs["side"] {
				t.Fatal("empty-map roundtrip did not produce writable owned hash slots")
			}
			if len(body.Metadata) != 0 || len(launch.Labels) != 0 || len(secondaries[0].Labels) != 0 {
				t.Fatal("hash completion mutated the caller's empty business maps")
			}
		})
	}
	for _, input := range []struct {
		body   *CompositeService
		launch *client.LaunchConfig
	}{
		{nil, &client.LaunchConfig{}}, {&CompositeService{}, nil},
	} {
		if _, _, err := catalogServiceConfigAndHash(input.body, input.launch, nil); err == nil {
			t.Fatal("missing typed service/launch input was accepted")
		}
	}
}

func TestCatalogHashPreservesTypedBusinessMapsNumbersAndSource(t *testing.T) {
	business := map[string]interface{}{"id": "business-id", "type": "business-type", "links": nil,
		"actions": []interface{}{"business-action"}, "large": json.Number("9007199254740993"), "fraction": json.Number("1.25")}
	header := client.Resource{Id: "response-id", Type: "container", Links: map[string]string{"self": "response-link"}, Actions: map[string]string{"act": "response-action"}}
	body := &CompositeService{Service: client.Service{Name: "app", StackId: "1st1600", Metadata: business}}
	launch := &client.LaunchConfig{Resource: header, ImageUuid: "docker:busybox:1.37.0", Labels: business, Environment: business,
		Ports: []string{"8080:80/tcp"}, DataVolumes: []string{"/data"}, CapAdd: []string{"NET_BIND_SERVICE"}, SecurityOpt: []string{"no-new-privileges"},
		LogConfig: &client.LogConfig{Resource: header, Driver: "json-file", Config: business}}
	secondaries := []client.SecondaryLaunchConfig{{Resource: header, Name: "side", ImageUuid: "docker:busybox:1.37.0", Labels: business}}
	input := *body
	input.LaunchConfig, input.SecondaryLaunchConfigs = launch, secondaries
	before, _ := json.Marshal(&input)
	want, err := client.CatalogMutationRequest(&input)
	if err != nil {
		t.Fatal("typed business fixture failed")
	}
	_, result, err := catalogServiceConfigAndHash(body, launch, secondaries)
	if err != nil {
		t.Fatal("Catalog hash rejected valid typed business fields")
	}
	got, err := client.CatalogMutationRequest(result)
	if err != nil {
		t.Fatal("hashed body is not a typed mutation")
	}
	delete(got["metadata"].(map[string]interface{}), digest.ServiceHashKey)
	delete(got["launchConfig"].(map[string]interface{})["labels"].(map[string]interface{}), digest.ServiceHashKey)
	for _, item := range got["secondaryLaunchConfigs"].([]interface{}) {
		delete(item.(map[string]interface{})["labels"].(map[string]interface{}), digest.ServiceHashKey)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatal("business keys, numbers or typed configuration changed beyond the owned hash slots")
	}
	after, _ := json.Marshal(&input)
	if !bytes.Equal(before, after) {
		t.Fatal("Catalog hash projection mutated the source DTO or business maps")
	}
}

// This golden is captured from the published 0.14.42 source with the same
// synthetic native transform. It covers both legacy/no-proof SDK callers.
func TestNonDelegatedFactoryKeeps042HashAndRequestBytes(t *testing.T) {
	const expected042SHA256 = "50f022028ee449288ae7ffe5bf8600b01e5a3417141c8005e225c960ed853135"
	var actual []byte
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(client.CatalogLaunchDelegationHeader) != "" {
			t.Error("nondelegated caller acquired delegation authority")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/schemas":
			w.Header().Set("X-API-Schemas", server.URL+"/schemas")
			fmt.Fprintf(w, `{"links":{"self":%q},"data":[{"id":"service","collectionMethods":["POST"],"links":{"collection":%q}}]}`, server.URL+"/schemas", server.URL+"/services")
		case "/scripts/transform":
			fmt.Fprint(w, `{"type":"container","links":{"self":"response-link"},"actions":{"act":"response-action"},"imageUuid":"docker:busybox:1.37.0","labels":{"id":"business-id"},"environment":{"type":"business-type"}}`)
		case "/services":
			actual, _ = io.ReadAll(io.LimitReader(r.Body, 1<<20))
			fmt.Fprint(w, `{}`)
		default:
			t.Error("nondelegated fixture received an unexpected route")
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
	if err != nil {
		t.Fatal("synthetic ordinary bootstrap failed")
	}
	cfg := &config.ServiceConfig{Image: "busybox:1.37.0", Labels: yaml.SliceorMap{}}
	cfg.Metadata = map[string]interface{}{"type": "business-type"}
	ctx := &Context{Client: c, Stack: &client.Stack{Resource: client.Resource{Id: "1st1600"}}}
	p := project.NewProject(&ctx.Context)
	ctx.Project = p
	p.ServiceConfigs.Add("app", cfg)
	ctx.SidekickInfo, err = NewSidekickInfo(p)
	if err != nil {
		t.Fatal("ordinary sidekick fixture failed")
	}
	worker := NewService("app", cfg, ctx)
	factory := &NormalFactory{}
	hash, err := factory.Hash(worker)
	if err != nil || factory.Create(worker) != nil {
		t.Fatal("ordinary factory path failed")
	}
	var posted client.Service
	if json.Unmarshal(actual, &posted) != nil {
		t.Fatal("ordinary SDK emitted invalid JSON")
	}
	stored, complete := digest.LookupHash(&posted)
	if !complete || !hash.Equals(stored) || posted.LaunchConfig.Type != "container" || posted.LaunchConfig.Links["self"] != "response-link" {
		t.Fatal("ordinary hash/update or original resource-envelope behavior changed")
	}
	sum := sha256.Sum256(actual)
	gotSHA := hex.EncodeToString(sum[:])
	t.Logf("synthetic ordinary request SHA256 only: %s", gotSHA)
	if gotSHA != expected042SHA256 {
		t.Fatal("nondelegated request bytes differ from the published042 worker baseline")
	}
}
