package platformapi

import "testing"

func TestSecretUsesFirstComposeFileAsResourceBase(t *testing.T) {
	composeFiles := []string{"config/compose.yml", "config/override.yml"}
	if got := secretResourceBase(composeFiles); got != composeFiles[0] {
		t.Fatalf("resource base %q, want first Compose file", got)
	}
}

func TestSecretWithoutComposeFileKeepsWorkingDirectoryBase(t *testing.T) {
	if got := secretResourceBase(nil); got != "./" {
		t.Fatalf("resource base %q, want working directory", got)
	}
}
