package cataloglaunch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/convert"
	"github.com/PastureStack/compose-cli/lookup"
	"github.com/PastureStack/compose-cli/project"
	"github.com/PastureStack/compose-cli/template"
)

func exampleInput() Input {
	return Input{Name: "chosen-stack", EnvironmentName: "project", DockerCompose: `version: '2'
services:
  app:
    image: busybox:1.37.0
    command: ['sh', '-c', 'sleep 3600']
    environment:
      FLAG: '${flag}'
      COUNT: '${count}'
      PRIVATE: '{{ .Values.password }}'
    volumes:
      - proof:/proof
    ports:
      - '18080:80'
volumes:
  proof: {}
`, RancherCompose: `version: '2'
services:
  app:
    scale: 2
`, Variables: map[string]interface{}{"flag": false, "count": json.Number("0"), "password": "private-answer-marker"}}
}

func TestCatalogPrepareBoundsInterpolationBeforeGraphAllocation(t *testing.T) {
	var source strings.Builder
	source.WriteString("version: '2'\nservices:\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&source, "  app%d:\n    image: busybox:1.37.0\n    environment:\n      TEXT: '${text}'\n", i)
	}
	input := Input{Name: "chosen", DockerCompose: source.String(), Variables: map[string]interface{}{"text": strings.Repeat("x", 16384)}}
	if _, err := Prepare(input); err == nil {
		t.Fatal("unbounded interpolation accepted before graph allocation")
	}
	input.DockerCompose = "version: '2'\nservices:\n  app:\n    image: busybox\n    environment:\n      TEXT: '${undeclared}'\n"
	input.Variables = nil
	if _, err := Prepare(input); err == nil {
		t.Fatal("undeclared interpolation could reach ordinary warning logs")
	}
}

func TestCatalogPrepareUsesNormalMergeConvertAndPreservesPrivateVariables(t *testing.T) {
	input := exampleInput()
	result, err := Prepare(input)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	if result.FormatVersion != 1 || len(result.Services) != 1 || len(result.Volumes) != 1 || result.Services[0].Config.Scale != 2 {
		t.Fatal("normal graph changed")
	}
	if result.Variables["flag"] != "false" || result.Variables["count"] != "0" {
		t.Fatal("variables differ from normal Stack map[string] schema")
	}
	service := result.Services[0]
	for _, entry := range []string{"FLAG=false", "COUNT=0", "PRIVATE=private-answer-marker"} {
		found := false
		for _, value := range service.Inspect.Config.Env {
			if value == entry {
				found = true
			}
		}
		if !found {
			t.Fatal("explicit private/false/zero value lost")
		}
	}
	// Parse the frozen result through the exact ordinary owner, then compare
	// Docker conversion. A second template context cannot change the config.
	env := &lookup.MapEnvLookup{Env: result.Variables}
	ctx := project.Context{ProjectName: input.Name, EnvironmentLookup: env, ResourceLookup: lookup.NewFileResourceLookup()}
	merged, err := config.Merge(config.NewServiceConfigs(), env, ctx.ResourceLookup, template.StackInfo{Name: "different"}, template.EnvironmentInfo{}, "", []byte(result.DockerCompose))
	if err != nil {
		t.Fatal("frozen normal parse failed")
	}
	c, _, err := convert.Convert(merged.Services["app"], ctx)
	if err != nil {
		t.Fatal("normal conversion failed")
	}
	sort.Strings(c.Env)
	sort.Strings(service.Inspect.Config.Env)
	got, _ := json.Marshal(c)
	want, _ := json.Marshal(service.Inspect.Config)
	if !bytes.Equal(got, want) {
		t.Fatal("preview and normal owner Docker config differ")
	}
}

func TestCatalogPrepareRejectsHostFileProviderKVMAndUnboundedGraphs(t *testing.T) {
	for index, source := range []string{
		"version: '2'\nservices:\n  app:\n    image: busybox\n    env_file: /etc/private\n",
		"version: '2'\nservices:\n  app:\n    extends:\n      file: /etc/private\n      service: other\n",
		"version: '2'\nservices:\n  app:\n    image: busybox\n    type: virtual_machine\n",
		"version: '2'\nservices: {}\n",
		"version: '2'\nservices:\n  app:\n    image: '{{ env \"HOST_SECRET\" }}'\n",
	} {
		input := exampleInput()
		input.DockerCompose = source
		if _, err := Prepare(input); err == nil {
			t.Fatalf("unsupported or host input accepted at fixture %d", index)
		}
	}
	input := exampleInput()
	input.Variables["rawNested"] = map[string]interface{}{"password": "never-echo"}
	if _, err := Prepare(input); err != ErrInput {
		t.Fatal("nested arbitrary input accepted")
	}
}

func TestCatalogPrepareUsesOrdinaryExecutorProjectAndCatalogVersionContext(t *testing.T) {
	input := exampleInput()
	input.Name = "-Chosen"
	input.DockerCompose = "version: '2'\nservices:\n  app:\n    image: '{{ .Stack.Name }}:{{ .Stack.Version }}'\n"
	input.RancherCompose = "version: '2'\ncatalog:\n  version: '2.3'\n"
	result, err := Prepare(input)
	if err != nil || len(result.Services) != 1 || result.Services[0].Config.Image != "x-chosen:2.3" {
		t.Fatal("preflight used a different context than normal executor")
	}
}

func TestCatalogPrivateProtocolHasClosedCommandBodyAndSafeFailureCodes(t *testing.T) {
	var output bytes.Buffer
	input := exampleInput()
	raw, _ := json.Marshal(input)
	if Run(bytes.NewReader(raw), &output) != 0 || !strings.Contains(output.String(), `"formatVersion":1`) {
		t.Fatal("private owner protocol failed")
	}
	for _, body := range []string{
		`{"name":"chosen","url":"https://remote","dockerCompose":"private-source-marker"}`,
		`{"name":"chosen","dockerCompose":"private-invalid-yaml-marker"} {}`,
		`{"name":"chosen","dockerCompose":"private-invalid-yaml-marker"}`,
		strings.Repeat("x", MaxInputBytes+1),
	} {
		output.Reset()
		if Run(strings.NewReader(body), &output) == 0 || strings.Contains(output.String(), "private-") || output.Len() > 80 {
			t.Fatal("invalid body succeeded or private error leaked")
		}
	}
}
