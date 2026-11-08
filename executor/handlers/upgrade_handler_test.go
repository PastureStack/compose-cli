package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/PastureStack/compose-cli/internal/events"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func TestRollbackStackCanceledUpgradeUsesNativeRollbackAndWait(t *testing.T) {
	for _, state := range []string{"canceled-upgrade", "upgraded", "active", "upgrading"} {
		t.Run(state, func(t *testing.T) {
			var mu sync.Mutex
			var trace []string
			rolledBack := false
			var server *httptest.Server
			service := func(current string) map[string]interface{} {
				return map[string]interface{}{
					"id": "1s7", "type": "service", "state": current, "transitioning": "no",
					"links":   map[string]string{"self": server.URL + "/services/1s7"},
					"actions": map[string]string{"rollback": server.URL + "/services/1s7?action=rollback"},
				}
			}
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				encode := func(value interface{}) {
					if err := json.NewEncoder(w).Encode(value); err != nil {
						t.Errorf("encode fixture response: %v", err)
					}
				}
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/schemas":
					w.Header().Set("X-API-Schemas", server.URL+"/schemas")
					encode(map[string]interface{}{"data": []interface{}{
						map[string]interface{}{"id": "stack", "resourceMethods": []string{"GET"}, "links": map[string]string{"collection": server.URL + "/stacks"}},
						map[string]interface{}{"id": "service", "resourceMethods": []string{"GET"}, "links": map[string]string{"collection": server.URL + "/services"}},
						map[string]interface{}{"id": "publish", "collectionMethods": []string{"POST"}, "links": map[string]string{"collection": server.URL + "/publish"}},
					}})
				case r.Method == http.MethodGet && r.URL.Path == "/stacks/1st7":
					encode(map[string]interface{}{
						"id": "1st7", "externalId": "current-id", "previousExternalId": "previous-id",
						"environment": map[string]string{"VALUE": "current"}, "previousEnvironment": map[string]string{"VALUE": "previous"},
						"links": map[string]string{"services": server.URL + "/stacks/1st7/services"},
					})
				case r.Method == http.MethodGet && r.URL.Path == "/stacks/1st7/services":
					listed := service(state)
					if state == "upgrading" {
						listed["transitioning"] = "yes"
					}
					encode(map[string]interface{}{"data": []interface{}{listed}})
				case r.Method == http.MethodGet && r.URL.Path == "/services/1s7" && r.URL.RawQuery == "":
					current := state
					if rolledBack {
						current = "active"
					} else if state == "upgrading" {
						current = "upgraded"
					}
					trace = append(trace, "GET "+current)
					encode(service(current))
				case r.Method == http.MethodPost && r.URL.Path == "/services/1s7" && r.URL.Query().Get("action") == "rollback":
					trace = append(trace, "POST rollback")
					if rolledBack || state == "active" {
						t.Error("unexpected duplicate or active-child rollback")
					}
					rolledBack = true
					pending := service("rolling-back")
					pending["transitioning"] = "yes"
					encode(pending)
				case r.Method == http.MethodPost && r.URL.Path == "/publish":
					trace = append(trace, "POST reply")
					var result client.Publish
					if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
						t.Errorf("decode original reply: %v", err)
					} else if result.Name != "reply.one" || result.Data["externalId"] != "previous-id" || result.Data["previousExternalId"] != nil || result.Data["previousEnvironment"] != nil {
						t.Errorf("original rollback reply changed: %#v", result)
					} else if environment, ok := result.Data["environment"].(map[string]interface{}); !ok || environment["VALUE"] != "previous" {
						t.Errorf("previous environment not restored: %#v", result.Data)
					}
					encode(map[string]interface{}{})
				default:
					t.Errorf("unexpected native route: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
			if err != nil {
				t.Fatal(err)
			}
			event := &events.Event{ID: "rollback-event", ResourceID: "1st7", ReplyTo: "reply.one"}
			if err := RollbackStack(event, c); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			first := state
			if first == "upgrading" {
				first = "upgraded"
			}
			want := []string{"GET " + first, "POST rollback", "GET active", "POST reply"}
			if state == "active" {
				want = []string{"GET active", "POST reply"}
			}
			if !reflect.DeepEqual(trace, want) {
				t.Fatalf("native rollback/readiness order = %v, want %v", trace, want)
			}
		})
	}
}
