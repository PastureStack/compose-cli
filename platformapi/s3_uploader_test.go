package platformapi

import "testing"

func TestWorkspaceHashDoesNotDependOnCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "first-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "first-secret-key")
	first := workspaceHash()

	t.Setenv("AWS_ACCESS_KEY_ID", "second-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "second-secret-key")
	if second := workspaceHash(); second != first {
		t.Fatalf("workspace identity changed with credentials: %s != %s", second, first)
	}
}
