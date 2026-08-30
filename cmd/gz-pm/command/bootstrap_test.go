package command

import (
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
)

const testBootstrapManagerName = "Bootstrap Test Manager"

func TestDisplayBootstrapText_StatusIconPrecedence(t *testing.T) {
	tests := []struct {
		name   string
		result *dto.ManagerBootstrapResult
		header string
	}{
		{
			name: "skipped takes precedence over already installed and success",
			result: &dto.ManagerBootstrapResult{
				Name:             testBootstrapManagerName,
				Skipped:          true,
				AlreadyInstalled: true,
				Success:          true,
			},
			header: "⏭️  " + testBootstrapManagerName,
		},
		{
			name: "already installed takes precedence over success",
			result: &dto.ManagerBootstrapResult{
				Name:             testBootstrapManagerName,
				AlreadyInstalled: true,
				Success:          true,
			},
			header: "✓ " + testBootstrapManagerName,
		},
		{
			name: "success uses success icon",
			result: &dto.ManagerBootstrapResult{
				Name:    testBootstrapManagerName,
				Success: true,
			},
			header: "✅ " + testBootstrapManagerName,
		},
		{
			name: "failure uses failure icon",
			result: &dto.ManagerBootstrapResult{
				Name: testBootstrapManagerName,
			},
			header: "❌ " + testBootstrapManagerName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := captureStdout(t, func() error {
				displayBootstrapText(&dto.BootstrapResponse{
					Results: []*dto.ManagerBootstrapResult{tt.result},
					Summary: &dto.BootstrapSummary{TotalManagers: 1},
				})
				return nil
			})
			if err != nil {
				t.Fatalf("capture stdout: %v", err)
			}
			if !strings.Contains(got, tt.header+"\n") {
				t.Errorf("output missing exact header %q:\n%s", tt.header, got)
			}
		})
	}
}
