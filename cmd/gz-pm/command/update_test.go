package command

import (
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
)

const (
	testUpdateManagerName       = "Test Manager"
	testUpdateErrorPrefix       = "   Error:"
	testUpdateNoPackagesMessage = "   No packages updated"
	testUpdateMixedOutput       = `🧪 Package Manager Update (DRY-RUN)

⚠️ Skipped Manager
   Skipped: conda environment

✅ Updated Manager
   Duration: 1.5s
   Updated: 2 packages
      • alpha
      • beta
   Space freed: 3.0 MB

❌ Failed Manager
   Error: permission denied

📊 Summary
   Total Managers: 3
   Successful: 1
   Failed: 1
   Skipped: 1
   Total Packages Updated: 2
   Total Downloaded: 4.0 MB
   Total Space Freed: 3.0 MB
   Total Duration: 2.5s`
	testUpdateEmptyOutput = `📦 Package Manager Update

📊 Summary
   Total Managers: 0
   Successful: 0
   Failed: 0
   Total Packages Updated: 0
   Total Duration: 0.0s`
)

func TestDisplayUpdateText(t *testing.T) {
	tests := []struct {
		name     string
		response *dto.UpdateResponse
		contains []string
		absent   []string
		inOrder  []string
	}{
		{
			name: "skipped takes precedence over success",
			response: &dto.UpdateResponse{
				Results: []*dto.ManagerUpdateResult{{
					Name:       testUpdateManagerName,
					Success:    true,
					Skipped:    true,
					SkipReason: "conda environment",
				}},
				Summary: &dto.UpdateSummary{TotalManagers: 1, SkippedManagers: 1},
			},
			contains: []string{
				"⚠️ " + testUpdateManagerName,
				"   Skipped: conda environment",
			},
			absent: []string{
				"   Duration:",
				testUpdateErrorPrefix,
			},
		},
		{
			name: "success shows packages and freed space in order",
			response: &dto.UpdateResponse{
				Results: []*dto.ManagerUpdateResult{{
					Name:     testUpdateManagerName,
					Success:  true,
					Duration: 1.2,
					UpdatedPackages: []dto.PackageUpdate{
						{Name: "alpha"},
						{Name: "beta"},
					},
					SpaceFreed: 2 * 1024 * 1024,
				}},
				Summary: &dto.UpdateSummary{TotalManagers: 1, SuccessfulManagers: 1, TotalPackagesUpdated: 2, TotalSpaceFreed: 2 * 1024 * 1024},
			},
			contains: []string{
				"✅ " + testUpdateManagerName,
				"   Duration: 1.2s",
				"   Updated: 2 packages",
				"      • alpha",
				"      • beta",
				"   Space freed: 2.0 MB",
			},
			absent: []string{
				testUpdateNoPackagesMessage,
				testUpdateErrorPrefix,
			},
			inOrder: []string{
				"   Duration: 1.2s",
				"   Updated: 2 packages",
				"      • alpha",
				"      • beta",
				"   Space freed: 2.0 MB",
			},
		},
		{
			name: "success with no updates says so",
			response: &dto.UpdateResponse{
				Results: []*dto.ManagerUpdateResult{{
					Name:     testUpdateManagerName,
					Success:  true,
					Duration: 0.5,
				}},
				Summary: &dto.UpdateSummary{TotalManagers: 1, SuccessfulManagers: 1},
			},
			contains: []string{
				"✅ " + testUpdateManagerName,
				"   Duration: 0.5s",
				testUpdateNoPackagesMessage,
			},
			absent: []string{
				"   Updated:",
				testUpdateErrorPrefix,
			},
		},
		{
			name: "failure shows only error details",
			response: &dto.UpdateResponse{
				Results: []*dto.ManagerUpdateResult{{
					Name:  testUpdateManagerName,
					Error: "permission denied",
				}},
				Summary: &dto.UpdateSummary{TotalManagers: 1, FailedManagers: 1},
			},
			contains: []string{
				"❌ " + testUpdateManagerName,
				"   Error: permission denied",
			},
			absent: []string{
				"   Duration:",
				"   Updated:",
				testUpdateNoPackagesMessage,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := captureStdout(t, func() error {
				displayUpdateText(tt.response)
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

func TestDisplayUpdateText_HeaderAndSummary(t *testing.T) {
	tests := []struct {
		name     string
		response *dto.UpdateResponse
		want     string
	}{
		{
			name: "dry run preserves ordered manager results and complete summary",
			response: &dto.UpdateResponse{
				Results: []*dto.ManagerUpdateResult{
					{Name: "Skipped Manager", Success: true, Skipped: true, SkipReason: "conda environment"},
					{
						Name:            "Updated Manager",
						Success:         true,
						Duration:        1.5,
						UpdatedPackages: []dto.PackageUpdate{{Name: "alpha"}, {Name: "beta"}},
						SpaceFreed:      3 * 1024 * 1024,
					},
					{Name: "Failed Manager", Error: "permission denied"},
				},
				Summary: &dto.UpdateSummary{
					TotalManagers:        3,
					SuccessfulManagers:   1,
					FailedManagers:       1,
					SkippedManagers:      1,
					TotalPackagesUpdated: 2,
					TotalBytesDownloaded: 4 * 1024 * 1024,
					TotalSpaceFreed:      3 * 1024 * 1024,
					TotalDuration:        2.5,
				},
				DryRun: true,
			},
			want: testUpdateMixedOutput,
		},
		{
			name: "normal header omits empty optional summary fields",
			response: &dto.UpdateResponse{
				Summary: &dto.UpdateSummary{},
			},
			want: testUpdateEmptyOutput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := captureStdout(t, func() error {
				displayUpdateText(tt.response)
				return nil
			})
			if err != nil {
				t.Fatalf("capture stdout: %v", err)
			}
			if got != tt.want+"\n" {
				t.Errorf("unexpected output (-want +got):\nwant:\n%s\ngot:\n%s", tt.want, got)
			}
		})
	}
}

func assertOutputOrder(t *testing.T, output string, fragments []string) {
	t.Helper()
	start := 0
	for _, fragment := range fragments {
		offset := strings.Index(output[start:], fragment)
		if offset < 0 {
			t.Errorf("output missing ordered fragment %q:\n%s", fragment, output)
			return
		}
		start += offset + len(fragment)
	}
}
