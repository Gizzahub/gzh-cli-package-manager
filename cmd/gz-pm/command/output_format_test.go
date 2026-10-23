package command

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInvalidOutputRejectedBeforeUseCase(t *testing.T) {
	oldStatus, oldBootstrap := statusOutput, bootstrapOutput
	t.Cleanup(func() { statusOutput, bootstrapOutput = oldStatus, oldBootstrap })
	statusOutput, bootstrapOutput = "simple", "simple"
	for _, cmd := range []*cobra.Command{statusCmd, bootstrapCmd} {
		err := cmd.RunE(cmd, nil)
		if err == nil || !strings.Contains(err.Error(), "unknown output format") {
			t.Fatalf("%s: expected output validation before use case, got %v", cmd.Name(), err)
		}
	}
}
