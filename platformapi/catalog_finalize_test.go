package platformapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/project"
	"github.com/PastureStack/compose-cli/yaml"
)

func TestCatalogFinalizeBodyEqualsActualOrdinaryServicePost(t *testing.T) {
	assertCatalogBody(t, false)
}

func TestCatalogFinalizeSidekickSecretAndHashBodyEqualsActualOrdinaryPost(t *testing.T) {
	assertCatalogBody(t, true)
}

func assertCatalogBody(t *testing.T, sidekick bool) {
	t.Helper()
	posted := map[string]interface{}{}
	transformed := map[string]json.RawMessage{}
	for _, name := range []string{"app", "helper"} {
		raw, _ := json.Marshal(map[string]interface{}{"imageUuid": "docker:busybox:1.37.0", "labels": map[string]string{"source": "preserved"}, "environment": map[string]string{"private": "private-owner-input"}, "logConfig": map[string]interface{}{"config": map[string]string{}}, "ports": []string{"discarded-by-owner-primary"}, "unexpectedCoreField": "must-not-copy"})
		transformed[name] = raw
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/schemas":
			w.Header().Set("X-API-Schemas", server.URL+"/schemas")
			fmt.Fprintf(w, `{"links":{"self":%q},"data":[{"id":"service","collectionMethods":["POST"],"links":{"collection":%q}},{"id":"secret","collectionMethods":["GET"],"links":{"collection":%q}}]}`, server.URL+"/schemas", server.URL+"/services", server.URL+"/secrets")
		case "/scripts/transform":
			var input struct{ Name string }
			_ = json.NewDecoder(r.Body).Decode(&input)
			_, _ = w.Write(transformed[strings.TrimPrefix(input.Name, "/")])
		case "/secrets":
			fmt.Fprint(w, `{"data":[{"id":"1se7","name":"chosen"}]}`)
		case "/services":
			_ = json.NewDecoder(r.Body).Decode(&posted)
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected owner route %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
	if err != nil {
		t.Fatal("fake owner API unavailable")
	}
	primary := &config.ServiceConfig{Image: "busybox:1.37.0", Ports: []string{"18080:80"}, Expose: []string{"81"}, Environment: yaml.MaporEqualSlice{"Z=z", "A=a"}, Labels: yaml.SliceorMap{}, MilliCpuReservation: 100, NetworkMode: "managed", VolumeDriver: "driver"}
	primary.Metadata = map[string]interface{}{"owned": "value"}
	primary.Scale = 2
	secondary := &config.ServiceConfig{Image: "busybox:1.37.0", Environment: yaml.MaporEqualSlice{"Q=q"}, Labels: yaml.SliceorMap{}}
	secondary.HealthCheck = &client.InstanceHealthCheck{Port: 81}
	if sidekick {
		primary.Labels["io.rancher.sidekicks"] = "helper"
		primary.Secrets = config.SecretReferences{{Source: "chosen", Target: "target", Uid: "1", Gid: "2", Mode: yaml.StringorOctalInt("0444")}}
	}
	ctx := &Context{Client: c, Stack: &client.Stack{Resource: client.Resource{Id: "$createdStack"}}, CatalogLaunchDelegation: "event-proof"}
	if !sidekick {
		ctx.CatalogLaunchDelegation = ""
	}
	p := project.NewProject(&ctx.Context)
	ctx.Project = p
	p.ServiceConfigs.Add("app", primary)
	if sidekick {
		p.ServiceConfigs.Add("helper", secondary)
	}
	ctx.SidekickInfo, err = NewSidekickInfo(p)
	if err != nil {
		t.Fatal("sidekick owner failed")
	}
	if err := (&NormalFactory{}).Create(NewService("app", primary, ctx)); err != nil {
		t.Fatal("ordinary owner failed")
	}
	parts := []CatalogLaunchPart{}
	resolved := map[string]string{}
	if sidekick {
		parts = append(parts, CatalogLaunchPart{Name: "helper", Config: secondary, Transform: transformed["helper"]})
		resolved["chosen"] = "1se7"
	}
	kind, body, err := CatalogFinalizeService(CatalogLaunchPart{Name: "app", Config: primary, Transform: transformed["app"], VolumeDriver: "driver"}, parts, resolved)
	if err != nil {
		t.Fatal("private owner finalize failed")
	}
	raw, _ := json.Marshal(CatalogServiceRequestPayload(kind, body))
	var expected map[string]interface{}
	_ = json.Unmarshal(raw, &expected)
	if !reflect.DeepEqual(posted, expected) {
		t.Fatal("final graph differs from ordinary POST (private bodies omitted)")
	}
	if posted["unexpectedCoreField"] != nil || kind != "service" || len(posted) == 0 {
		t.Fatal("transform projection or service type changed")
	}
}

func TestCatalogFinalizeRequiresResolvedSecretAndPreservesDelegationInContextClient(t *testing.T) {
	config := &config.ServiceConfig{Secrets: config.SecretReferences{{Source: "chosen"}}}
	if _, _, err := CatalogFinalizeService(CatalogLaunchPart{Name: "app", Config: config, Transform: json.RawMessage(`{}`)}, nil, nil); err == nil {
		t.Fatal("missing verified reference admitted")
	}
	seen := false
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get(client.CatalogLaunchDelegationHeader) == "private-proof"
		w.Header().Set("X-API-Schemas", server.URL+"/schemas")
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer server.Close()
	ctx := &Context{Url: server.URL + "/schemas", CatalogLaunchDelegation: "private-proof"}
	c, err := ctx.loadClient()
	if err != nil || !seen || c.GetOpts().CatalogLaunchDelegation != "private-proof" {
		t.Fatal("normal project context lost event ceiling proof")
	}
	ctx.AccessKey = "private-access-key-marker"
	ctx.ProjectName = "chosen"
	if err := ctx.open(); err == nil || strings.Contains(err.Error(), ctx.AccessKey) {
		t.Fatal("delegated local permission failure leaked service credential")
	}
}
