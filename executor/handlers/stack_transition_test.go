package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PastureStack/compose-cli/internal/events"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func TestStackTransitionWaitsWithoutReplayingActions(t *testing.T) {
	for _, action := range []string{"rollback", "finishupgrade"} {
		t.Run(action, func(t *testing.T) {
			var server *httptest.Server
			var mu sync.Mutex
			actions, reads := map[string]int{}, map[string]int{}
			terminal, progress := 0, 0
			service := func(id, state, transitioning string) map[string]interface{} {
				return map[string]interface{}{
					"id": id, "type": "service", "state": state, "transitioning": transitioning,
					"links":   map[string]string{"self": server.URL + "/services/" + id},
					"actions": map[string]string{action: server.URL + "/services/" + id + "?action=" + action},
				}
			}
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				encode := func(v interface{}) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch {
				case r.URL.Path == "/schemas":
					w.Header().Set("X-API-Schemas", server.URL+"/schemas")
					var schemas []interface{}
					for _, kind := range []string{"stack", "service", "publish"} {
						schemas = append(schemas, map[string]interface{}{"id": kind, "resourceMethods": []string{"GET"}, "collectionMethods": []string{"POST"}, "links": map[string]string{"collection": server.URL + "/" + kind + "s"}})
					}
					encode(map[string]interface{}{"data": schemas})
				case r.URL.Path == "/stacks/1st7":
					encode(map[string]interface{}{"id": "1st7", "previousExternalId": "before", "links": map[string]string{"services": server.URL + "/stacks/1st7/services"}})
				case r.URL.Path == "/stacks/1st7/services":
					encode(map[string]interface{}{"data": []interface{}{service("1s1", "upgrading", "yes"), service("1s2", "active", "no"), service("1s3", "upgraded", "no")}})
				case strings.HasPrefix(r.URL.Path, "/services/"):
					id := strings.TrimPrefix(r.URL.Path, "/services/")
					if r.Method == http.MethodPost {
						actions[id]++
						if id == "1s2" || actions[id] != 1 || r.URL.Query().Get("action") != action {
							t.Error("replayed or wrong child mutation")
						}
						reads[id] = 0
						encode(service(id, "transitioning", "yes"))
						return
					}
					reads[id]++
					if id == "1s1" && actions[id] == 0 && reads[id] <= 8 {
						encode(service(id, "upgrading", "yes"))
					} else if id == "1s1" && actions[id] == 1 && reads[id] <= 100 {
						encode(service(id, "transitioning", "yes"))
					} else if actions[id] > 0 || id == "1s2" {
						encode(service(id, "active", "no"))
					} else {
						state := "upgraded"
						if id == "1s3" && action == "rollback" {
							state = "canceled-upgrade"
						}
						encode(service(id, state, "no"))
					}
				case r.URL.Path == "/publishs" && r.Method == http.MethodPost:
					var reply client.Publish
					if err := json.NewDecoder(r.Body).Decode(&reply); err != nil {
						t.Error(err)
					}
					if reply.Name != "reply.original" || len(reply.PreviousIds) != 1 || reply.PreviousIds[0] != "original-event" {
						t.Error("lost original event ownership")
					}
					if reply.Transitioning == "yes" {
						progress++
					} else {
						terminal++
						if reads["1s1"] != 101 || actions["1s1"] != 1 || actions["1s3"] != 1 {
							t.Error("premature terminal reply")
						}
					}
					encode(map[string]interface{}{})
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
			if err != nil {
				t.Fatal(err)
			}
			event := &events.Event{ID: "original-event", ResourceID: "1st7", ReplyTo: "reply.original"}
			waiter := newServiceWaiter(event, c)
			now := time.Unix(0, 0)
			waiter.now = func() time.Time { return now }
			waiter.sleep = func(d time.Duration) { now = now.Add(d) }
			if action == "rollback" {
				err = rollbackStack(event, c, waiter.wait)
			} else {
				err = finishUpgradeStack(event, c, waiter.wait)
			}
			if err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if terminal != 1 || progress != 6 || actions["1s2"] != 0 {
				t.Fatalf("terminal=%d progress=%d actions=%v", terminal, progress, actions)
			}
		})
	}
}
