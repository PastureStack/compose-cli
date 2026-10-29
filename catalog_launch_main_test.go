package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/PastureStack/compose-cli/cataloglaunch"
)

func TestCatalogLaunchCommandOmitsPrivateParserDiagnostics(t *testing.T) {
	if os.Getenv("PST_CATALOG_LAUNCH_TEST_PROCESS") == "1" {
		os.Args = []string{"compose-executor", cataloglaunch.Command}
		main()
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestCatalogLaunchCommandOmitsPrivateParserDiagnostics$")
	command.Env = append(os.Environ(), "PST_CATALOG_LAUNCH_TEST_PROCESS=1")
	command.Stdin = strings.NewReader(`{"name":"chosen","dockerCompose":"private-parser-answer-marker"}`)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if command.Run() == nil {
		t.Fatal("malformed input succeeded")
	}
	if stderr.Len() != 0 || strings.Contains(stdout.String(), "private-parser") || strings.TrimSpace(stdout.String()) != `{"code":"CatalogLaunchUnsupported"}` {
		t.Fatal("private diagnostics reached process output")
	}
}
