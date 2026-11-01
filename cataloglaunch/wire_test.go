package cataloglaunch

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/platformapi"
)

func wireObject(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]interface{}
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	return object
}

func noResourceEnvelope(t *testing.T, object map[string]interface{}) {
	t.Helper()
	for _, field := range []string{"id", "type", "links", "actions"} {
		if _, found := object[field]; found {
			t.Fatalf("generic Resource envelope field %s entered a mutation DTO", field)
		}
	}
}

func TestCatalogWireEmptyAndNonemptyServiceLinks(t *testing.T) {
	for _, raw := range [][]string{nil, {"worker:read", "worker:write"}} {
		input, err := platformapi.CatalogServiceLinkSkeleton(raw, map[string]string{"worker": ""})
		if err != nil {
			t.Fatal(err)
		}
		legacyBefore, _ := json.Marshal(input)
		object := wireObject(t, catalogLinksWire(input))
		if len(object) != 1 {
			t.Fatal("link action must contain only serviceLinks")
		}
		links, ok := object["serviceLinks"].([]interface{})
		if !ok || len(links) != len(raw) {
			t.Fatal("empty links must be [] and selected links must retain their count")
		}
		for index, value := range links {
			link := value.(map[string]interface{})
			noResourceEnvelope(t, link)
			if len(link) != 2 || link["serviceId"] != input.ServiceLinks[index].ServiceId || link["name"] != input.ServiceLinks[index].Name {
				t.Fatal("native link target or alias changed")
			}
		}
		legacyAfter, _ := json.Marshal(input)
		if !bytes.Equal(legacyBefore, legacyAfter) || !bytes.Contains(legacyAfter, []byte(`"links":null`)) || !bytes.Contains(legacyAfter, []byte(`"actions":null`)) {
			t.Fatal("ordinary client DTO serialization changed")
		}
	}
}

func TestCatalogWirePreservesBusinessMapsAndNestedRequests(t *testing.T) {
	metadata := map[string]interface{}{"id": "business-id", "type": "business-type", "links": map[string]interface{}{"target": "keep"}, "actions": []interface{}{"keep"}}
	header := client.Resource{Id: "server-id", Type: "server-type", Links: map[string]string{"self": "server-link"}, Actions: map[string]string{"update": "server-action"}}
	input := &client.Service{Resource: header, Name: "app", Metadata: metadata,
		LaunchConfig: &client.LaunchConfig{Resource: header, ImageUuid: "docker:busybox:1.37.0", Labels: metadata,
			Environment: map[string]interface{}{"links": "env-value", "actions": "env-action"},
			LogConfig:   &client.LogConfig{Resource: header, Driver: "json-file", Config: metadata}},
		SecondaryLaunchConfigs: []client.SecondaryLaunchConfig{{Resource: header, Name: "side", Labels: metadata}}}
	before, _ := json.Marshal(input)
	wire, err := catalogRequestWire(input)
	if err != nil {
		t.Fatal(err)
	}
	object := wireObject(t, wire)
	noResourceEnvelope(t, object)
	launch := object["launchConfig"].(map[string]interface{})
	noResourceEnvelope(t, launch)
	logConfig := launch["logConfig"].(map[string]interface{})
	noResourceEnvelope(t, logConfig)
	noResourceEnvelope(t, object["secondaryLaunchConfigs"].([]interface{})[0].(map[string]interface{}))
	if !reflect.DeepEqual(object["metadata"], metadata) || !reflect.DeepEqual(launch["labels"], metadata) || !reflect.DeepEqual(logConfig["config"], metadata) {
		t.Fatal("business maps were stripped or changed")
	}
	environment := launch["environment"].(map[string]interface{})
	if environment["links"] != "env-value" || environment["actions"] != "env-action" {
		t.Fatal("environment keys were stripped")
	}
	after, _ := json.Marshal(input)
	if !bytes.Equal(before, after) {
		t.Fatal("projection mutated the ordinary SDK object")
	}
	volume, err := catalogRequestWire(&client.VolumeTemplate{Resource: header, Name: "data", StackId: "$createdStack", DriverOpts: metadata})
	if err != nil {
		t.Fatal(err)
	}
	noResourceEnvelope(t, volume)
	if !reflect.DeepEqual(volume["driverOpts"], metadata) {
		t.Fatal("volume business options changed")
	}
	named := struct {
		client.Resource
		Business client.Resource `json:"business"`
	}{Resource: header, Business: header}
	namedWire, err := catalogRequestWire(named)
	if err != nil {
		t.Fatal(err)
	}
	noResourceEnvelope(t, namedWire)
	if !reflect.DeepEqual(namedWire["business"], wireObject(t, header)) {
		t.Fatal("a named Resource business property was treated as an embedded envelope")
	}
}

func TestCatalogWireFinalizeMarshaledProtocol(t *testing.T) {
	input := exampleInput()
	input.Phase = "finalize"
	input.DockerCompose = "version: '2'\nservices:\n  app:\n    image: busybox:1.37.0\n    links:\n      - worker:read\n    volumes:\n      - proof:/proof\n  worker:\n    image: busybox:1.37.0\nvolumes:\n  proof: {}\n"
	input.TransformedLaunchConfigs = map[string]json.RawMessage{
		"app":    json.RawMessage(`{"imageUuid":"docker:busybox:1.37.0","logConfig":{"config":{"links":"business-link","actions":"business-action"}}}`),
		"worker": json.RawMessage(`{"imageUuid":"docker:busybox:1.37.0"}`),
	}
	output, err := Prepare(input)
	if err != nil {
		t.Fatal(err)
	}
	wire := wireObject(t, output)
	for _, value := range wire["services"].([]interface{}) {
		noResourceEnvelope(t, value.(map[string]interface{})["createSkeleton"].(map[string]interface{}))
	}
	for _, value := range wire["volumeRequests"].([]interface{}) {
		noResourceEnvelope(t, value.(map[string]interface{})["createSkeleton"].(map[string]interface{}))
	}
	for _, value := range wire["serviceRequests"].([]interface{}) {
		request := value.(map[string]interface{})
		body := request["createBody"].(map[string]interface{})
		noResourceEnvelope(t, body)
		launch := body["launchConfig"].(map[string]interface{})
		noResourceEnvelope(t, launch)
		noResourceEnvelope(t, launch["logConfig"].(map[string]interface{}))
		links := request["setServiceLinks"].(map[string]interface{})
		if len(links) != 1 {
			t.Fatal("link action schema is not closed")
		}
		items, ok := links["serviceLinks"].([]interface{})
		if !ok {
			t.Fatal("empty serviceLinks was omitted or became null")
		}
		for _, item := range items {
			noResourceEnvelope(t, item.(map[string]interface{}))
		}
	}
}
