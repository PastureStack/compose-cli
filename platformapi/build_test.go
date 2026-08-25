package platformapi

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTarHonorsIgnoreAndRetainsDockerfile(t *testing.T) {
	root := t.TempDir()
	for name, contents := range map[string]string{
		"Dockerfile":    "FROM scratch\n",
		".dockerignore": "ignored.txt\nDockerfile\n",
		"ignored.txt":   "secret build input\n",
		"kept.txt":      "required build input\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	stream, err := createTar(root, "")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	entries := map[string]bool{}
	reader := tar.NewReader(stream)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		entries[strings.TrimPrefix(header.Name, "./")] = true
	}
	if !entries["Dockerfile"] || !entries[".dockerignore"] || !entries["kept.txt"] {
		t.Fatalf("required build inputs missing from archive: %#v", entries)
	}
	if entries["ignored.txt"] {
		t.Fatalf("ignored build input was archived: %#v", entries)
	}
}

func TestCreateTarRejectsDockerfileOutsideContext(t *testing.T) {
	_, err := createTar(t.TempDir(), filepath.Join("..", "Dockerfile"))
	if err == nil || !strings.Contains(err.Error(), "escapes build context") {
		t.Fatalf("expected bounded context error, got %v", err)
	}
}
