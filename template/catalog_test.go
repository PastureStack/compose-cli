package template

import (
	"bytes"
	"strings"
	"testing"
)

func TestCatalogRenderingReusesOwnerContextAndFreezesApprovedBytes(t *testing.T) {
	input := []byte("name: {{ .Stack.Name }}\nvalue: {{ .Values.text | quote }}\nflag: {{ if eq .Values.flag \"false\" }}no{{ else }}yes{{ end }}\n")
	variables := map[string]string{"text": "hello", "flag": "false"}
	result, err := ApplyCatalog(input, StackInfo{Name: "chosen"}, EnvironmentInfo{}, variables)
	if err != nil || !strings.Contains(string(result), "name: chosen") || !strings.Contains(string(result), "flag: no") {
		t.Fatalf("render failed: %v", err)
	}
	again, err := Apply(result, StackInfo{Name: "different"}, EnvironmentInfo{}, map[string]string{"text": "changed"})
	if err != nil || !bytes.Equal(result, again) {
		t.Fatal("approved content was reevaluated")
	}
	if text, err := catalogPrintf("[%s]-20261030", "value"); err != nil || text != "[value]-20261030" {
		t.Fatal("literal brackets/version mistaken for format width")
	}
}

func TestCatalogRenderingRejectsHostFunctionsRecursionMissingValuesAndExpansion(t *testing.T) {
	for _, input := range []string{
		`{{ env "PRIVATE_KEY" }}`, `{{ expandenv "$PRIVATE_KEY" }}`, `{{ getHostByName "example.test" }}`,
		`{{ genPrivateKey "rsa" }}`, `{{ randBytes 32 }}`, `{{ repeat 999999999 "a" }}`,
		`{{ printf "%1000000000s" "a" }}`, `{{ printf "%*s" 1000000000 "a" }}`,
		`{{ indent 1000000000 "a" }}`, `{{ nindent 1000000000 "a" }}`,
		`{{ replace "" (printf "%4096s" "x") (printf "%4096s" "a") }}`,
		`{{ join (printf "%4096s" "x") (splitList "" (printf "%4096s" "a")) }}`,
		`{{ define "again" }}{{ template "again" . }}{{ end }}{{ template "again" . }}`,
		`{{ .Values.missing }}`, `{{ .Values.private | invalidFunc }}`,
	} {
		if _, err := ApplyCatalog([]byte(input), StackInfo{}, EnvironmentInfo{}, map[string]string{"private": "never-echo"}); err != ErrCatalogTemplate {
			t.Fatal("unsafe template accepted")
		}
	}
	if _, err := ApplyCatalog([]byte(strings.Repeat("x", MaxCatalogTemplateBytes+1)), StackInfo{}, EnvironmentInfo{}, nil); err != ErrCatalogTemplate {
		t.Fatal("unbounded template accepted")
	}
	variables := map[string]string{}
	for i := 0; i < 100; i++ {
		variables[strings.Repeat("k", i+1)] = strings.Repeat("x", 16384)
	}
	if _, err := ApplyCatalog([]byte(`{{range .Values}}{{.}}{{end}}`), StackInfo{}, EnvironmentInfo{}, variables); err != ErrCatalogTemplate {
		t.Fatal("expanded output exceeded writer boundary")
	}
}
