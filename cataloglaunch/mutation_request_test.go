package cataloglaunch

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func mutationClient(t *testing.T, proof string, receive func(*http.Request, []byte) int) *client.RancherClient {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schemas" {
			w.Header().Set("X-API-Schemas", server.URL+"/schemas")
			fmt.Fprintf(w, `{"data":[{"id":"service","pluralName":"services","collectionMethods":["POST"],"links":{"collection":%q}},{"id":"volumeTemplate","pluralName":"volumeTemplates","collectionMethods":["POST"],"links":{"collection":%q}}]}`, server.URL+"/mutation", server.URL+"/mutation")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			t.Error("bounded mutation body read failed")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(receive(r, raw))
		fmt.Fprint(w, `{}`)
	}))
	t.Cleanup(server.Close)
	c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas", CatalogLaunchDelegation: proof})
	if err != nil {
		t.Fatal("SDK bootstrap failed")
	}
	return c
}

func sendMutation(c *client.RancherClient, input interface{}) error {
	switch value := input.(type) {
	case *client.Service:
		_, err := c.Service.Create(value)
		return err
	case *client.VolumeTemplate:
		_, err := c.VolumeTemplate.Create(value)
		return err
	case *client.SetServiceLinksInput:
		target := &client.Service{Resource: client.Resource{Actions: map[string]string{"setservicelinks": c.GetOpts().Url + "/mutation"}}}
		_, err := c.Service.ActionSetservicelinks(target, value)
		return err
	default:
		return c.Post(c.GetOpts().Url+"/mutation", input, &map[string]interface{}{})
	}
}

func TestDelegatedMutationBytesMatchPreviewForServiceVolumeAndEmptyOrNamedLinks(t *testing.T) {
	business := map[string]interface{}{"id": "business-id", "type": "business-type", "links": nil, "actions": map[string]interface{}{"keep": true}}
	header := client.Resource{Id: "response-id", Type: "response-type", Links: map[string]string{"self": "response-link"}}
	service := &client.Service{Resource: header, Name: "app", StackId: "1st7", Metadata: business,
		LaunchConfig: &client.LaunchConfig{Resource: header, ImageUuid: "docker:busybox:1.37.0", Labels: business,
			Environment: map[string]interface{}{"id": "env-id", "type": "env-type", "links": "env-link", "actions": "env-action"},
			DataVolumes: []string{"/data"}, Ports: []string{"8080:80/tcp"}, CapAdd: []string{"NET_BIND_SERVICE"},
			SecurityOpt: []string{"no-new-privileges"}, LogConfig: &client.LogConfig{Resource: header, Driver: "json-file", Config: business}},
		SecondaryLaunchConfigs: []client.SecondaryLaunchConfig{{Resource: header, Name: "side", ImageUuid: "docker:busybox:1.37.0", Labels: business}}}
	for _, input := range []interface{}{&client.Service{Name: "app", StackId: "1st7", LaunchConfig: &client.LaunchConfig{ImageUuid: "docker:busybox:1.37.0"}},
		service, &client.VolumeTemplate{Resource: header, Name: "data", StackId: "1st7", Driver: "local", DriverOpts: business},
		&client.SetServiceLinksInput{Resource: header}, &client.SetServiceLinksInput{Resource: header, ServiceLinks: []client.ServiceLink{{Resource: header, Name: "read", ServiceId: "1s8"}}}} {
		before, _ := json.Marshal(input)
		var expected interface{}
		if links, ok := input.(*client.SetServiceLinksInput); ok {
			expected = catalogLinksWire(links)
		} else {
			var err error
			expected, err = catalogRequestWire(input)
			if err != nil {
				t.Fatal("preview DTO projection failed")
			}
		}
		want, _ := json.Marshal(expected)
		calls := 0
		c := mutationClient(t, "event-proof", func(r *http.Request, raw []byte) int {
			calls++
			if r.Header.Get(client.CatalogLaunchDelegationHeader) != "event-proof" || !bytes.Equal(raw, want) || sha256.Sum256(raw) != sha256.Sum256(want) {
				t.Error("delegated SDK mutation differs from the approved preview bytes")
				return http.StatusForbidden
			}
			return http.StatusCreated
		})
		if err := sendMutation(c, input); err != nil {
			t.Error("exact approved delegated mutation was rejected")
		}
		after, _ := json.Marshal(input)
		if calls != 1 || !bytes.Equal(before, after) {
			t.Fatal("projection mutated the DTO or changed request count")
		}
	}
}

func TestOrdinaryAndLegacyMutationBytesRemainTheOriginalSDKMarshal(t *testing.T) {
	for _, input := range []interface{}{&client.Service{Name: "app", LaunchConfig: &client.LaunchConfig{ImageUuid: "docker:busybox:1.37.0"}},
		&client.VolumeTemplate{Name: "data"}, &client.SetServiceLinksInput{}, map[string]interface{}{"id": "business-id", "links": nil, "actions": []interface{}{"keep"}}} {
		want, _ := json.Marshal(input)
		calls := 0
		c := mutationClient(t, "", func(r *http.Request, raw []byte) int {
			calls++
			if r.Header.Get(client.CatalogLaunchDelegationHeader) != "" || !bytes.Equal(raw, want) {
				t.Error("ordinary SDK bytes or authority changed")
			}
			return http.StatusCreated
		})
		if sendMutation(c, input) != nil || calls != 1 {
			t.Fatal("ordinary mutation failed")
		}
	}
}

func TestDelegatedBusinessMapsRemainExactAndChangedBodiesStillFailClosed(t *testing.T) {
	approved := map[string]interface{}{"name": "app", "metadata": map[string]interface{}{"id": "business-id", "type": "business-type", "links": nil, "actions": []interface{}{"keep"}}}
	want, _ := json.Marshal(approved)
	calls := 0
	c := mutationClient(t, "event-proof", func(_ *http.Request, raw []byte) int {
		calls++
		if sha256.Sum256(raw) != sha256.Sum256(want) {
			return http.StatusForbidden
		}
		return http.StatusCreated
	})
	if c.Post(c.GetOpts().Url+"/mutation", approved, &map[string]interface{}{}) != nil {
		t.Fatal("business-map keys were stripped")
	}
	for _, changed := range []map[string]interface{}{{"name": "other", "metadata": approved["metadata"]}, {"name": "app", "metadata": approved["metadata"], "unknown": true}} {
		err := c.Post(c.GetOpts().Url+"/mutation", changed, &map[string]interface{}{})
		apiError, ok := err.(*client.ApiError)
		if !ok || apiError.StatusCode != http.StatusForbidden {
			t.Fatal("changed delegated body passed the exact approved-body verifier")
		}
	}
	if calls != 3 {
		t.Fatal("unexpected delegated request count")
	}
}
