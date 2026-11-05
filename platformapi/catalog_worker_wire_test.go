package platformapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/digest"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/project"
	"github.com/PastureStack/compose-cli/project/options"
	"github.com/PastureStack/compose-cli/utils"
	"github.com/PastureStack/compose-cli/yaml"
)

// No QA, Docker, credentials or external endpoint. This executes the real worker
// PlatformService.Create -> transform -> NormalFactory -> generated SDK POST,
// rather than sending a manually assembled Service directly to the SDK.
func TestCatalogWorkerCreateMatchesPreviewAfterNativeTransformResponse(t *testing.T) {
	for _, envelope := range []bool{false, true} {
		for _, actualStack := range []string{"$createdStack", "1st1600"} {
			for _, changedApproval := range []bool{false, true} {
				if changedApproval && (!envelope || actualStack == "$createdStack") {
					continue
				}
				t.Run(fmt.Sprintf("nativeResourceEnvelope=%v/boundStack=%v/changedApproval=%v", envelope, actualStack != "$createdStack", changedApproval), func(t *testing.T) {
					flat := map[string]interface{}{"imageUuid": "docker:busybox:1.37.0", "labels": map[string]interface{}{"business": "kept"},
						"environment": map[string]interface{}{"id": "business-id", "type": "business-type", "links": "business-links", "actions": "business-actions"},
						"logConfig":   map[string]interface{}{"config": map[string]interface{}{}}, "ports": []string{"80/tcp"}}
					transform := map[string]interface{}{}
					for key, value := range flat {
						transform[key] = value
					}
					if envelope {
						// JsonResponseWriter wraps TransformInspect's InstanceRecord as a
						// Resource; Engine's preview deliberately supplies getFields only.
						transform["type"] = "container"
						transform["links"] = map[string]interface{}{}
						transform["actions"] = map[string]interface{}{}
					}
					flatBytes, _ := json.Marshal(flat)
					newConfig := func() *config.ServiceConfig {
						cfg := &config.ServiceConfig{Image: "busybox:1.37.0", Labels: yaml.SliceorMap{},
							Environment: yaml.MaporEqualSlice{"actions=business-actions", "id=business-id", "links=business-links", "type=business-type"}}
						cfg.Metadata = map[string]interface{}{"id": "business-id", "links": "business-links"}
						return cfg
					}
					kind, preview, err := CatalogFinalizeService(CatalogLaunchPart{Name: "app", Config: newConfig(), Transform: flatBytes}, nil, nil)
					if err != nil {
						t.Fatal("preview owner failed")
					}
					want, err := client.CatalogMutationRequest(CatalogServiceRequestPayload(kind, preview))
					if err != nil {
						t.Fatal("preview wire projection failed")
					}
					want["stackId"] = actualStack // Same sole top-level binding as Engine graphs.bind.
					var normalizedWant map[string]interface{}
					b, _ := json.Marshal(want)
					_ = json.Unmarshal(b, &normalizedWant)
					if changedApproval {
						normalizedWant["launchConfig"].(map[string]interface{})["environment"].(map[string]interface{})["type"] = "other-business-value"
					}
					posted := map[string]interface{}{}
					sequence := []string{}
					created := false
					var server *httptest.Server
					serviceRow := func() map[string]interface{} {
						return map[string]interface{}{"id": "1s1", "name": "app", "state": "inactive", "stackId": actualStack,
							"actions": map[string]string{"setservicelinks": server.URL + "/v2-beta/services/1s1?action=setservicelinks"}}
					}
					server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Header.Get(client.CatalogLaunchDelegationHeader) != "synthetic-event-proof" {
							t.Error("actual worker lost delegation proof")
						}
						sequence = append(sequence, r.Method+" "+r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						switch r.URL.Path {
						case "/v2-beta/schemas":
							w.Header().Set("X-API-Schemas", server.URL+"/v2-beta/schemas")
							fmt.Fprintf(w, `{"links":{"self":%q},"data":[{"id":"service","collectionMethods":["GET","POST"],"links":{"collection":%q}},{"id":"serviceConsumeMap","collectionMethods":["GET"],"links":{"collection":%q}}]}`,
								server.URL+"/v2-beta/schemas", server.URL+"/v2-beta/services", server.URL+"/v2-beta/serviceConsumeMaps")
						case "/v2-beta/scripts/transform":
							if r.Method != "POST" {
								t.Error("transform must be actual POST")
							}
							_ = json.NewEncoder(w).Encode(transform)
						case "/v2-beta/services":
							if r.Method == "POST" {
								_ = json.NewDecoder(r.Body).Decode(&posted)
								if !reflect.DeepEqual(normalizedWant, posted) {
									w.WriteHeader(http.StatusForbidden)
									fmt.Fprint(w, `{"code":"CatalogLaunchDelegationDenied"}`)
									return
								}
								created = true
								w.WriteHeader(http.StatusCreated)
								_ = json.NewEncoder(w).Encode(serviceRow())
							} else if created {
								_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{serviceRow()}})
							} else {
								fmt.Fprint(w, `{"data":[]}`)
							}
						case "/v2-beta/serviceConsumeMaps":
							fmt.Fprint(w, `{"data":[]}`)
						case "/v2-beta/services/1s1":
							if r.Method != "POST" || r.URL.Query().Get("action") != "setservicelinks" {
								t.Error("unexpected worker action")
							}
							_ = json.NewEncoder(w).Encode(serviceRow())
						default:
							t.Error("unexpected closed worker route")
							w.WriteHeader(http.StatusNotFound)
						}
					}))
					defer server.Close()
					c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/v2-beta/schemas", CatalogLaunchDelegation: "synthetic-event-proof"})
					if err != nil {
						t.Fatal("synthetic native schema bootstrap failed")
					}
					ctx := &Context{Client: c, Stack: &client.Stack{Resource: client.Resource{Id: actualStack}}, CatalogLaunchDelegation: "synthetic-event-proof"}
					p := project.NewProject(&ctx.Context)
					ctx.Project = p
					cfg := newConfig()
					p.ServiceConfigs.Add("app", cfg)
					ctx.SidekickInfo, err = NewSidekickInfo(p)
					if err != nil {
						t.Fatal("worker sidekick owner setup failed")
					}
					err = NewService("app", cfg, ctx).Create(context.Background(), options.Create{})
					atTransform, atCreate := -1, -1
					for i, step := range sequence {
						if step == "POST /v2-beta/scripts/transform" {
							atTransform = i
						}
						if step == "POST /v2-beta/services" {
							atCreate = i
						}
					}
					if atTransform < 0 || atCreate != atTransform+1 {
						t.Fatal("next worker request is not the actual service collection POST")
					}
					if changedApproval {
						denied, ok := err.(*client.ApiError)
						if !ok || denied.StatusCode != http.StatusForbidden || created || atCreate != len(sequence)-1 {
							t.Fatal("changed approved business body did not fail closed before create/link")
						}
						return
					}
					if err != nil {
						paths := differingSafePaths(normalizedWant, posted, "")
						sort.Strings(paths)
						t.Fatalf("full worker denied at structural paths only (no private values/digests): %s", strings.Join(paths, ","))
					}
					if !reflect.DeepEqual(normalizedWant, posted) {
						paths := differingSafePaths(normalizedWant, posted, "")
						sort.Strings(paths)
						t.Fatalf("approved-vs-actual structural paths only (no values/digests): %s", strings.Join(paths, ","))
					}
				})
			}
		}
	}
}

func TestCatalogWorkerHashBoundariesUseActualNestedYamlMaps(t *testing.T) {
	cfg := &config.ServiceConfig{Labels: yaml.SliceorMap{}}
	symbolic := serviceCreateFields("app", cfg, "$createdStack")
	bound := serviceCreateFields("app", cfg, "1st1600")
	var converted map[interface{}]interface{}
	if utils.Convert(symbolic, &converted) != nil {
		t.Fatal("actual YAML owner conversion failed")
	}
	nested, ok := converted["service"].(map[string]interface{})
	if !ok || nested["stack_id"] != "$createdStack" {
		t.Fatal("actual nested YAML shape differs from hypothesis")
	}
	flat := &client.LaunchConfig{Labels: map[string]interface{}{}}
	withResource := *flat
	withResource.Resource = client.Resource{Type: "container", Links: map[string]string{}, Actions: map[string]string{}}
	a, errA := digest.CreateServiceHash(symbolic, flat, nil)
	b, errB := digest.CreateServiceHash(bound, flat, nil)
	c, errC := digest.CreateServiceHash(symbolic, &withResource, nil)
	if errA != nil || errB != nil || errC != nil {
		t.Fatal("actual digest owner failed")
	}
	if a.Service == b.Service || a.LaunchConfig != b.LaunchConfig {
		t.Fatal("bound Stack independence hypothesis is not reproduced")
	}
	if a.Service != c.Service || a.LaunchConfig == c.LaunchConfig {
		t.Fatal("typed response envelope influence is not reproduced")
	}
}

func differingSafePaths(a, b interface{}, prefix string) []string {
	if reflect.DeepEqual(a, b) {
		return nil
	}
	am, aok := a.(map[string]interface{})
	bm, bok := b.(map[string]interface{})
	if !aok || !bok {
		return []string{prefix}
	}
	keys := map[string]bool{}
	for key := range am {
		keys[key] = true
	}
	for key := range bm {
		keys[key] = true
	}
	result := []string{}
	for key := range keys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		result = append(result, differingSafePaths(am[key], bm[key], path)...)
	}
	return result
}
