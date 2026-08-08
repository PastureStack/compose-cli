package lookup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnconfiguredFileLookupRejectsHostRead(t *testing.T) {
	hostFile := filepath.Join(t.TempDir(), "host-secret.env")
	if err := os.WriteFile(hostFile, []byte("TOKEN=must-not-leak\n"), 0600); err != nil {
		t.Fatal(err)
	}

	contents, resolved, err := (&FileResourceLookup{}).Lookup(hostFile, "")
	if err == nil {
		t.Fatalf("unconfigured lookup read %q from %q", string(contents), resolved)
	}
}

func TestConfiguredFileLookupAllowsResourceUnderComposeDirectory(t *testing.T) {
	projectDir := t.TempDir()
	composeFile := filepath.Join(projectDir, "compose.yml")
	resourceFile := filepath.Join(projectDir, "config", "app.env")
	if err := os.MkdirAll(filepath.Dir(resourceFile), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(composeFile, []byte("version: '2'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resourceFile, []byte("TOKEN=allowed\n"), 0600); err != nil {
		t.Fatal(err)
	}

	contents, resolved, err := NewFileResourceLookup(composeFile).Lookup("config/app.env", composeFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "TOKEN=allowed\n" {
		t.Fatalf("unexpected contents %q", contents)
	}
	if filepath.Clean(resolved) != filepath.Clean(resourceFile) {
		t.Fatalf("resolved %q, want %q", resolved, resourceFile)
	}
}

func TestConfiguredFileLookupRejectsEscapes(t *testing.T) {
	parentDir := t.TempDir()
	projectDir := filepath.Join(parentDir, "project")
	if err := os.MkdirAll(projectDir, 0700); err != nil {
		t.Fatal(err)
	}
	composeFile := filepath.Join(projectDir, "compose.yml")
	if err := os.WriteFile(composeFile, []byte("version: '2'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(parentDir, "host-secret.env")
	if err := os.WriteFile(outsideFile, []byte("TOKEN=must-not-leak\n"), 0600); err != nil {
		t.Fatal(err)
	}

	resourceLookup := NewFileResourceLookup(composeFile)
	for name, resource := range map[string]string{
		"parent traversal": "../host-secret.env",
		"absolute path":    outsideFile,
	} {
		t.Run(name, func(t *testing.T) {
			contents, resolved, err := resourceLookup.Lookup(resource, composeFile)
			if err == nil {
				t.Fatalf("escaped to %q and read %q", resolved, contents)
			}
			if strings.Contains(err.Error(), "must-not-leak") {
				t.Fatalf("error disclosed file contents: %v", err)
			}
		})
	}

	symlinkPath := filepath.Join(projectDir, "linked-secret.env")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	contents, resolved, err := resourceLookup.Lookup("linked-secret.env", composeFile)
	if err == nil {
		t.Fatalf("followed escaping symbolic link to %q and read %q", resolved, contents)
	}
}
