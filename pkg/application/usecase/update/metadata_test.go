package update

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func TestUseCase_Update_NPMMetadataFidelity(t *testing.T) {
	listErr := errors.New("list packages failed")
	updateErr := errors.New("npm update failed")

	tests := []metadataFidelityCase{
		{
			name:        "npm success without names is unsupported correlation",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			adapter: &mockAdapter{
				listPackagesFunc: func(_ context.Context) ([]manager.Package, error) {
					return []manager.Package{{
						Name:             "typescript",
						CurrentVersion:   "5.0.0",
						AvailableVersion: "5.1.0",
						UpdateType:       manager.UpdateMinor,
					}}, nil
				},
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true}, nil
				},
			},
			wantSuccess:      true,
			wantPilot:        true,
			wantCorrelation:  dto.CorrelationUnsupported,
			wantPackages:     []dto.PackageUpdate{},
			wantListCalls:    1,
			wantUpdatedCount: 0,
		},
		{
			name:        "npm dry-run without names is unsupported correlation",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			dryRun:      true,
			adapter: &mockAdapter{
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true}, nil
				},
			},
			wantSuccess:      true,
			wantPilot:        true,
			wantCorrelation:  dto.CorrelationUnsupported,
			wantPackages:     []dto.PackageUpdate{},
			wantListCalls:    0,
			wantUpdatedCount: 0,
		},
		{
			name:        "npm joins observed versions and derives update type",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			adapter: &mockAdapter{
				listPackagesFunc: func(_ context.Context) ([]manager.Package, error) {
					return []manager.Package{{
						Name:             "typescript",
						CurrentVersion:   "5.0.0",
						AvailableVersion: "5.1.0",
					}}, nil
				},
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{"typescript"}}, nil
				},
			},
			wantSuccess:     true,
			wantPilot:       true,
			wantCorrelation: dto.CorrelationJoined,
			wantPackages: []dto.PackageUpdate{{
				Name:               "typescript",
				OldVersion:         "5.0.0",
				NewVersion:         "5.1.0",
				UpdateType:         manager.UpdateMinor,
				OldVersionPresence: dto.PresenceObserved,
				NewVersionPresence: dto.PresenceObserved,
				UpdateTypePresence: dto.PresenceDerived,
				SizeBytesPresence:  dto.PresenceUnavailable,
			}},
			wantListCalls:    1,
			wantUpdatedCount: 1,
		},
		{
			name:        "npm partial snapshot keeps missing fields unavailable",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			adapter: &mockAdapter{
				listPackagesFunc: func(_ context.Context) ([]manager.Package, error) {
					return []manager.Package{
						{Name: "typescript", CurrentVersion: "5.0.0", AvailableVersion: "5.1.0"},
						{Name: "react", CurrentVersion: "18.0.0"},
					}, nil
				},
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{"typescript", "react", "vue"}}, nil
				},
			},
			wantSuccess:     true,
			wantPilot:       true,
			wantCorrelation: dto.CorrelationPartial,
			wantPackages: []dto.PackageUpdate{
				{
					Name:               "typescript",
					OldVersion:         "5.0.0",
					NewVersion:         "5.1.0",
					UpdateType:         manager.UpdateMinor,
					OldVersionPresence: dto.PresenceObserved,
					NewVersionPresence: dto.PresenceObserved,
					UpdateTypePresence: dto.PresenceDerived,
					SizeBytesPresence:  dto.PresenceUnavailable,
				},
				{
					Name:               "react",
					OldVersion:         "18.0.0",
					OldVersionPresence: dto.PresenceObserved,
					NewVersionPresence: dto.PresenceUnavailable,
					UpdateTypePresence: dto.PresenceUnavailable,
					SizeBytesPresence:  dto.PresenceUnavailable,
				},
				dto.UnavailablePackageUpdate("vue"),
			},
			wantListCalls:    1,
			wantUpdatedCount: 3,
		},
		{
			name:        "npm names without snapshot metadata stay unobserved",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			adapter: &mockAdapter{
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{"typescript"}}, nil
				},
			},
			wantSuccess:      true,
			wantPilot:        true,
			wantCorrelation:  dto.CorrelationUnobserved,
			wantPackages:     []dto.PackageUpdate{dto.UnavailablePackageUpdate("typescript")},
			wantListCalls:    1,
			wantUpdatedCount: 1,
		},
		{
			name:        "npm list failure does not fabricate versions",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			adapter: &mockAdapter{
				listPackagesFunc: func(_ context.Context) ([]manager.Package, error) {
					return nil, listErr
				},
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{"typescript"}}, nil
				},
			},
			wantSuccess:      true,
			wantPilot:        true,
			wantCorrelation:  dto.CorrelationUnobserved,
			wantPackages:     []dto.PackageUpdate{dto.UnavailablePackageUpdate("typescript")},
			wantListCalls:    1,
			wantWarn:         true,
			wantUpdatedCount: 1,
		},
		{
			name:        "npm command failure does not invent package rows",
			managerID:   manager.ManagerNPM,
			managerName: testNPMManagerName,
			adapter: &mockAdapter{
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: false, UpdatedPackages: []string{"typescript"}}, updateErr
				},
			},
			wantSuccess:      false,
			wantPilot:        true,
			wantCorrelation:  dto.CorrelationNotApplicable,
			wantPackages:     []dto.PackageUpdate{},
			wantListCalls:    1,
			wantUpdatedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runMetadataFidelityCase(t, tt)
		})
	}
}
