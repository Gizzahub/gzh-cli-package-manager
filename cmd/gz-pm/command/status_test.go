package command

import (
	"fmt"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

const (
	testStatusDetailSeparator  = "   \n"
	testStatusNonVerboseOutput = `📋 Package Manager Status

✅ Homebrew (system)
   Version: 4.0.0
   Status: healthy
   Packages: 2 (Updates: 1)

⛔ Pip (language)

📊 Summary
   Total Managers: 2
   Installed: 1
   Healthy: 1
   Total Packages: 2
   Updates Available: 1`
	testStatusVerboseOutput = `📋 Package Manager Status

✅ Pip (language)
   Version: 24.0
   Status: healthy
   Packages: 2 (Updates: 1)
` + testStatusDetailSeparator + `      requests 2.0
   ⬆️  flask 3.0 → 3.1

📊 Summary
   Total Managers: 1
   Installed: 1
   Healthy: 1
   Total Packages: 2
   Updates Available: 1`
	testStatusEmptyVerboseOutput = `📋 Package Manager Status

✅ Cargo (language)
   Version: 1.80
   Status: degraded
   Packages: 0 (Updates: 0)

📊 Summary
   Total Managers: 1
   Installed: 1
   Healthy: 0
   Total Packages: 0
   Updates Available: 0`
	testStatusTruncatedOutput = `📋 Package Manager Status

✅ NPM (language)
   Version: 10.0
   Status: healthy
   Packages: 11 (Updates: 1)
` + testStatusDetailSeparator + `      package-01 1.0
      package-02 1.0
      package-03 1.0
      package-04 1.0
      package-05 1.0
      package-06 1.0
      package-07 1.0
      package-08 1.0
      package-09 1.0
   ⬆️  package-10 1.0 → 1.1
   ... and 1 more packages

📊 Summary
   Total Managers: 1
   Installed: 1
   Healthy: 1
   Total Packages: 11
   Updates Available: 1`
)

func TestDisplayText(t *testing.T) {
	originalVerbose := statusVerbose
	t.Cleanup(func() { statusVerbose = originalVerbose })

	tests := []struct {
		name     string
		response *dto.StatusResponse
		verbose  bool
		want     string
	}{
		{
			name:    "non verbose preserves manager order and summary",
			verbose: false,
			response: &dto.StatusResponse{
				Managers: []*dto.ManagerStatus{
					{
						Name:           "Homebrew",
						Type:           manager.TypeSystem,
						Installed:      true,
						Version:        "4.0.0",
						Status:         manager.StatusHealthy,
						PackageCount:   2,
						UpdatableCount: 1,
						Packages:       []dto.PackageInfo{{Name: "not displayed", CurrentVersion: testQuarantineVersion}},
					},
					{Name: "Pip", Type: manager.TypeLanguage},
				},
				Summary: &dto.StatusSummary{TotalManagers: 2, InstalledManagers: 1, HealthyManagers: 1, TotalPackages: 2, UpdatablePackages: 1},
			},
			want: testStatusNonVerboseOutput,
		},
		{
			name:    "verbose formats package updates",
			verbose: true,
			response: &dto.StatusResponse{
				Managers: []*dto.ManagerStatus{{
					Name:           "Pip",
					Type:           manager.TypeLanguage,
					Installed:      true,
					Version:        "24.0",
					Status:         manager.StatusHealthy,
					PackageCount:   2,
					UpdatableCount: 1,
					Packages: []dto.PackageInfo{
						{Name: "requests", CurrentVersion: "2.0", UpdateType: manager.UpdateNone},
						{Name: "flask", CurrentVersion: "3.0", AvailableVersion: "3.1", UpdateType: manager.UpdatePatch},
					},
				}},
				Summary: &dto.StatusSummary{TotalManagers: 1, InstalledManagers: 1, HealthyManagers: 1, TotalPackages: 2, UpdatablePackages: 1},
			},
			want: testStatusVerboseOutput,
		},
		{
			name:    "verbose empty packages adds no detail blank line",
			verbose: true,
			response: &dto.StatusResponse{
				Managers: []*dto.ManagerStatus{{
					Name:      "Cargo",
					Type:      manager.TypeLanguage,
					Installed: true,
					Version:   "1.80",
					Status:    manager.StatusDegraded,
				}},
				Summary: &dto.StatusSummary{TotalManagers: 1, InstalledManagers: 1},
			},
			want: testStatusEmptyVerboseOutput,
		},
		{
			name:    "verbose truncates after ten packages",
			verbose: true,
			response: &dto.StatusResponse{
				Managers: []*dto.ManagerStatus{{
					Name:           "NPM",
					Type:           manager.TypeLanguage,
					Installed:      true,
					Version:        "10.0",
					Status:         manager.StatusHealthy,
					PackageCount:   11,
					UpdatableCount: 1,
					Packages:       testStatusPackages(),
				}},
				Summary: &dto.StatusSummary{TotalManagers: 1, InstalledManagers: 1, HealthyManagers: 1, TotalPackages: 11, UpdatablePackages: 1},
			},
			want: testStatusTruncatedOutput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusVerbose = tt.verbose
			got, err := captureStdout(t, func() error {
				displayText(tt.response)
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

func testStatusPackages() []dto.PackageInfo {
	packages := make([]dto.PackageInfo, statusPackageDisplayLimit+1)
	for index := range packages {
		packages[index] = dto.PackageInfo{
			Name:           fmt.Sprintf("package-%02d", index+1),
			CurrentVersion: testQuarantineVersion,
			UpdateType:     manager.UpdateNone,
		}
	}
	packages[statusPackageDisplayLimit-1].AvailableVersion = "1.1"
	packages[statusPackageDisplayLimit-1].UpdateType = manager.UpdatePatch
	return packages
}
