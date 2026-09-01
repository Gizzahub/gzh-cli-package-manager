package update

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

type metadataFidelityCase struct {
	name             string
	managerID        manager.ManagerID
	managerName      string
	dryRun           bool
	adapter          *mockAdapter
	wantSuccess      bool
	wantPilot        bool
	wantCorrelation  dto.PackageCorrelation
	wantPackages     []dto.PackageUpdate
	wantListCalls    int
	wantWarn         bool
	wantUpdatedCount int
}

func runMetadataFidelityCase(t *testing.T, tt *metadataFidelityCase) {
	t.Helper()
	logger := &mockLogger{}
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{{ID: tt.managerID, Name: tt.managerName, Installed: true}}, nil
		},
	}
	resp, err := NewUseCase(repo, logger, map[manager.ManagerID]adapterm.Adapter{
		tt.managerID: tt.adapter,
	}, nil).Update(context.Background(), &dto.UpdateRequest{
		All:      true,
		DryRun:   tt.dryRun,
		Strategy: dto.StrategyStable,
	})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("result count = %d, want 1", len(resp.Results))
	}
	got := resp.Results[0]
	if got.Success != tt.wantSuccess {
		t.Errorf("Success = %t, want %t", got.Success, tt.wantSuccess)
	}
	if got.MetadataPilot != tt.wantPilot {
		t.Errorf("MetadataPilot = %t, want %t", got.MetadataPilot, tt.wantPilot)
	}
	if got.PackageCorrelation != tt.wantCorrelation {
		t.Errorf("PackageCorrelation = %s, want %s", got.PackageCorrelation, tt.wantCorrelation)
	}
	if !slices.Equal(got.UpdatedPackages, tt.wantPackages) {
		t.Errorf("UpdatedPackages = %#v, want %#v", got.UpdatedPackages, tt.wantPackages)
	}
	if tt.adapter.listPackagesCalls != tt.wantListCalls {
		t.Errorf("ListPackages calls = %d, want %d", tt.adapter.listPackagesCalls, tt.wantListCalls)
	}
	if resp.Summary.TotalPackagesUpdated != tt.wantUpdatedCount {
		t.Errorf("TotalPackagesUpdated = %d, want %d", resp.Summary.TotalPackagesUpdated, tt.wantUpdatedCount)
	}
	if resp.DryRun != tt.dryRun {
		t.Errorf("DryRun = %t, want %t", resp.DryRun, tt.dryRun)
	}
	if tt.wantWarn && !slices.Contains(logger.warnMessages, "Pre-update package snapshot unavailable") {
		t.Errorf("warnings = %q, want snapshot warning", logger.warnMessages)
	}
	assertNoFabricatedMetadata(t, got.UpdatedPackages)
}

func assertUnavailablePackageUpdate(t *testing.T, got *dto.PackageUpdate, name string) {
	t.Helper()
	want := dto.UnavailablePackageUpdate(name)
	if *got != want {
		t.Errorf("package update = %#v, want unavailable %#v", *got, want)
	}
}

func assertNoFabricatedMetadata(t *testing.T, packages []dto.PackageUpdate) {
	t.Helper()
	encoded, err := json.Marshal(packages)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if slices.ContainsFunc(packages, func(pkg dto.PackageUpdate) bool {
		return pkg.OldVersion == "unknown" || pkg.NewVersion == "unknown" ||
			(pkg.UpdateType == manager.UpdateMinor && pkg.UpdateTypePresence != dto.PresenceDerived) ||
			(pkg.SizeBytes != 0 && pkg.SizeBytesPresence != dto.PresenceObserved)
	}) {
		t.Errorf("fabricated metadata in %s", encoded)
	}
}
