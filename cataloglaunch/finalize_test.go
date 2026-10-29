package cataloglaunch

import (
	"encoding/json"
	"testing"
)

func TestCatalogFinalizeRejectsUnknownOrMissingTransformAndUnresolvedReferences(t *testing.T) {
	input := exampleInput()
	input.Phase = "finalize"
	input.TransformedLaunchConfigs = map[string]json.RawMessage{"app": json.RawMessage(`{"imageUuid":"docker:busybox:1.37.0","logConfig":{"config":{}}}`)}
	result, err := Prepare(input)
	if err != nil || result.Phase != "finalize" || len(result.ServiceRequests) != 1 || result.ServiceRequests[0].ResourceType != "service" {
		t.Fatal("finalize failed")
	}
	for _, change := range []func(*Input){
		func(i *Input) { i.TransformedLaunchConfigs["foreign"] = json.RawMessage(`{}`) },
		func(i *Input) { delete(i.TransformedLaunchConfigs, "app") },
		func(i *Input) { i.TransformedLaunchConfigs["app"] = json.RawMessage(`null`) },
		func(i *Input) { i.ResolvedSecrets = map[string]string{"unrequested": "1se7"} },
		func(i *Input) { i.Phase = "inspect" },
	} {
		copy := input
		copy.TransformedLaunchConfigs = map[string]json.RawMessage{"app": input.TransformedLaunchConfigs["app"]}
		change(&copy)
		if _, err := Prepare(copy); err == nil {
			t.Fatal("unbound finalize input admitted")
		}
	}
	input.RancherCompose = "version: '2'\nservices:\n  app:\n    secrets:\n      - chosen\n"
	if _, err := Prepare(input); err == nil {
		t.Fatal("unresolved secret accepted")
	}
}
