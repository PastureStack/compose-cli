package events

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/PastureStack/compose-cli/internal/locks"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func TestWorkerPublishesOneOriginalTerminalError(t *testing.T) {
	for _, message := range []string{"Timeout waiting service", "reload failed", "publish failed"} {
		t.Run(message, func(t *testing.T) {
			calls, replies := 0, 0
			var server *httptest.Server
			var mu sync.Mutex
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/schemas" {
					w.Header().Set("X-API-Schemas", server.URL+"/schemas")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": "publish", "collectionMethods": []string{"POST"}, "links": map[string]string{"collection": server.URL + "/publish"}}}})
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != "/publish" {
					t.Error("unexpected request")
					http.NotFound(w, r)
					return
				}
				replies++
				var reply client.Publish
				if err := json.NewDecoder(r.Body).Decode(&reply); err != nil {
					t.Error(err)
				}
				if reply.Name != "reply.original" || len(reply.PreviousIds) != 1 || reply.PreviousIds[0] != "original" || reply.Transitioning != "error" || reply.TransitioningMessage != message {
					t.Errorf("wrong terminal reply: %#v", reply)
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{})
			}))
			defer server.Close()
			c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
			if err != nil {
				t.Fatal(err)
			}
			event := &Event{ID: "original", Name: "stack.rollback", ReplyTo: "reply.original"}
			doWork(event, map[string]EventHandler{"stack.rollback": func(*Event, *client.RancherClient) error { calls++; return errors.New(message) }}, c, locks.NopLocker())
			mu.Lock()
			defer mu.Unlock()
			if calls != 1 || replies != 1 {
				t.Fatalf("calls=%d replies=%d", calls, replies)
			}
		})
	}
}
