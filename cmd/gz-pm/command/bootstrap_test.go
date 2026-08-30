package command

import (
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
)

const testBootstrapManagerName = "Bootstrap Test Manager"

const (
	testBootstrapSkippedDetail  = "   Skipped: platform mismatch"
	testBootstrapAlreadyDetail  = "   Already installed"
	testBootstrapDurationDetail = "   Duration: 1.5s"
	testBootstrapNoStepsDetail  = "   Duration: 0.5s"
	testBootstrapDownloadDetail = "   • download"
	testBootstrapErrorDetail    = "   Error: install failed"
)

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

func TestDisplayBootstrapText_DetailPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		result   *dto.ManagerBootstrapResult
		contains []string
		absent   []string
		inOrder  []string
	}{
		{
			name: "skipped takes precedence over already installed and success",
			result: &dto.ManagerBootstrapResult{
				Name:             testBootstrapManagerName,
				Skipped:          true,
				SkipReason:       "platform mismatch",
				AlreadyInstalled: true,
				Success:          true,
			},
			contains: []string{testBootstrapSkippedDetail},
			absent: []string{
				testBootstrapAlreadyDetail,
				testBootstrapDurationDetail,
				testBootstrapErrorDetail,
			},
		},
		{
			name: "already installed takes precedence over success",
			result: &dto.ManagerBootstrapResult{
				Name:             testBootstrapManagerName,
				AlreadyInstalled: true,
				Success:          true,
			},
			contains: []string{testBootstrapAlreadyDetail},
			absent: []string{
				testBootstrapSkippedDetail,
				testBootstrapDurationDetail,
				testBootstrapErrorDetail,
			},
		},
		{
			name: "success shows duration then steps in order",
			result: &dto.ManagerBootstrapResult{
				Name:     testBootstrapManagerName,
				Success:  true,
				Duration: 1.5,
				Steps:    []string{"download", "configure"},
			},
			contains: []string{
				testBootstrapDurationDetail,
				testBootstrapDownloadDetail,
				"   • configure",
			},
			absent: []string{
				testBootstrapSkippedDetail,
				testBootstrapAlreadyDetail,
				testBootstrapErrorDetail,
			},
			inOrder: []string{
				testBootstrapDurationDetail,
				testBootstrapDownloadDetail,
				"   • configure",
			},
		},
		{
			name: "success without steps omits step lines",
			result: &dto.ManagerBootstrapResult{
				Name:     testBootstrapManagerName,
				Success:  true,
				Duration: 0.5,
			},
			contains: []string{testBootstrapNoStepsDetail},
			absent: []string{
				testBootstrapSkippedDetail,
				testBootstrapAlreadyDetail,
				testBootstrapErrorDetail,
				testBootstrapDownloadDetail,
			},
		},
		{
			name: "failure shows error only",
			result: &dto.ManagerBootstrapResult{
				Name:  testBootstrapManagerName,
				Error: "install failed",
			},
			contains: []string{testBootstrapErrorDetail},
			absent: []string{
				testBootstrapSkippedDetail,
				testBootstrapAlreadyDetail,
				testBootstrapDurationDetail,
			},
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
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
			for _, unwanted := range tt.absent {
				if strings.Contains(got, unwanted) {
					t.Errorf("output unexpectedly contains %q:\n%s", unwanted, got)
				}
			}
			assertOutputOrder(t, got, tt.inOrder)
		})
	}
}
