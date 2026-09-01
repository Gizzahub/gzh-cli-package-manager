package update

import (
	"context"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/detector"
)

func TestUseCase_Update_PilotAndOutOfPilotScope(t *testing.T) {
	tests := []metadataFidelityCase{
		{
			name:        "pacman joins observed versions from the pre-update snapshot",
			managerID:   manager.ManagerPacman,
			managerName: "Pacman",
			adapter: &mockAdapter{
				listPackagesFunc: func(_ context.Context) ([]manager.Package, error) {
					return []manager.Package{{
						Name:             testFirefoxPackageName,
						CurrentVersion:   "143.0.3-1",
						AvailableVersion: "145.0-1",
					}}, nil
				},
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{testFirefoxPackageName}}, nil
				},
			},
			wantSuccess:     true,
			wantPilot:       true,
			wantCorrelation: dto.CorrelationJoined,
			wantPackages: []dto.PackageUpdate{{
				Name:               testFirefoxPackageName,
				OldVersion:         "143.0.3-1",
				NewVersion:         "145.0-1",
				UpdateType:         manager.UpdateMajor,
				OldVersionPresence: dto.PresenceObserved,
				NewVersionPresence: dto.PresenceObserved,
				UpdateTypePresence: dto.PresenceDerived,
				SizeBytesPresence:  dto.PresenceUnavailable,
			}},
			wantListCalls:    1,
			wantUpdatedCount: 1,
		},
		{
			name:        "non-pilot managers do not use list snapshots",
			managerID:   manager.ManagerHomebrew,
			managerName: testHomebrewManagerName,
			adapter: &mockAdapter{
				listPackagesFunc: func(_ context.Context) ([]manager.Package, error) {
					t.Fatal("ListPackages should not be called for out-of-pilot managers")
					return nil, nil
				},
				updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{testGitPackageName}}, nil
				},
			},
			wantSuccess:      true,
			wantPilot:        false,
			wantCorrelation:  dto.CorrelationOutOfPilot,
			wantPackages:     []dto.PackageUpdate{dto.UnavailablePackageUpdate(testGitPackageName)},
			wantListCalls:    0,
			wantUpdatedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runMetadataFidelityCase(t, &tt)
		})
	}
}

func TestUseCase_Update_SkippedPipMetadataIsNotApplicable(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "test-conda")
	t.Setenv("CONDA_PREFIX", "/tmp/test-conda")
	logger := &mockLogger{}
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{{ID: manager.ManagerPip, Name: testPipManagerName, Installed: true}}, nil
		},
	}
	resp, err := NewUseCase(repo, logger, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerPip: &mockAdapter{},
	}, detector.NewDetector(nil, logger)).Update(context.Background(), &dto.UpdateRequest{
		All:      true,
		Strategy: dto.StrategyStable,
	})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("result count = %d, want 1", len(resp.Results))
	}
	got := resp.Results[0]
	if !got.Skipped || got.PackageCorrelation != dto.CorrelationNotApplicable || got.MetadataPilot {
		t.Errorf("skipped pip metadata = %#v, want not_applicable out of pilot", got)
	}
}
