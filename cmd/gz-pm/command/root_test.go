package command

import "testing"

func TestRootCommandUsesPublicBinaryName(t *testing.T) {
	t.Parallel()

	if got := NewRootCmd().Name(); got != "gz-pm" {
		t.Fatalf("root command name = %q, want %q", got, "gz-pm")
	}
}
