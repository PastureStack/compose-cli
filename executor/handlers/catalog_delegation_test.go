package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PastureStack/compose-cli/internal/events"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/sirupsen/logrus"
)

func catalogEvent() *events.Event {
	return &events.Event{ID: "event-one", ResourceID: "1st7", Data: map[string]interface{}{"catalogLaunchDelegation": map[string]interface{}{
		"planId": "aicataloglaunch-11111111-2222-3333-4444-555555555555", "stackId": "1st7", "eventId": "event-one",
		"proof": "cld1.11111111-2222-3333-4444-555555555555." + base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
	}}}
}

func TestCatalogControlReplyUsesOriginalServiceChannelWithoutOwnerPublishGrant(t *testing.T) {
	readsWithProof, publishesWithoutProof := 0, 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/schemas":
			w.Header().Set("X-API-Schemas", server.URL+"/schemas")
			fmt.Fprintf(w, `{"data":[{"id":"stack","resourceMethods":["GET"],"links":{"collection":%q}},{"id":"publish","collectionMethods":["POST"],"links":{"collection":%q}}]}`, server.URL+"/stacks", server.URL+"/publish")
		case "/stacks/1st7":
			if r.Header.Get(client.CatalogLaunchDelegationHeader) != "" {
				readsWithProof++
			}
			fmt.Fprint(w, `{"id":"1st7","dockerCompose":""}`)
		case "/publish":
			if r.Header.Get(client.CatalogLaunchDelegationHeader) == "" {
				publishesWithoutProof++
			}
			fmt.Fprint(w, `{}`)
		default:
			t.Error("unexpected owner control route")
		}
	}))
	defer server.Close()
	c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
	if err != nil {
		t.Fatal("fake service channel failed")
	}
	event := catalogEvent()
	event.ReplyTo = "reply.one"
	if err := createStack(logrus.NewEntry(logrus.New()), event, c); err != nil {
		t.Fatal("control fixture failed")
	}
	if readsWithProof != 1 || publishesWithoutProof != 1 || c.GetOpts().CatalogLaunchDelegation != "" {
		t.Fatal("domain ceiling or control identity mixed")
	}
}

func TestCatalogDelegationBindsExactEventAndStackBeforeAnyServiceClient(t *testing.T) {
	proof, marked, err := catalogLaunchProof(catalogEvent())
	if err != nil || !marked || !strings.HasPrefix(proof, "cld1.") {
		t.Fatal("valid Engine event rejected")
	}
	for _, change := range []func(*events.Event){
		func(e *events.Event) { e.ResourceID = "1st8" }, func(e *events.Event) { e.ID = "different-event" },
		func(e *events.Event) { e.Data["catalogLaunchDelegation"] = nil },
		func(e *events.Event) { delete(e.Data["catalogLaunchDelegation"].(map[string]interface{}), "proof") },
		func(e *events.Event) { e.Data["catalogLaunchDelegation"].(map[string]interface{})["proof"] = "forged" },
		func(e *events.Event) {
			e.Data["catalogLaunchDelegation"].(map[string]interface{})["url"] = "http://foreign"
		},
	} {
		event := catalogEvent()
		change(event)
		if c, err := catalogLaunchClient(event, nil); err == nil || c != nil {
			t.Fatal("marked event fell back to service client")
		}
	}
	if c, err := catalogLaunchClient(&events.Event{}, nil); err != nil || c != nil {
		t.Fatal("ordinary event changed")
	}
}
