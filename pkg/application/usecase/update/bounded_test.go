package update

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func TestBoundedUpdateRunsOneSelectedPackageAndDefersRemainder(t *testing.T) {
	var calls []string
	adapter := &mockAdapter{updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
		calls = append(calls, opts.Packages...)
		return &adapterm.UpdateResult{Success: true}, nil
	}}
	uc := NewUseCase(&mockRepository{findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
		return &manager.Manager{ID: id, Name: string(id), Installed: true}, nil
	}}, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{manager.ManagerMise: adapter}, nil)
	statePath := filepath.Join(t.TempDir(), "state.json")
	req := &dto.UpdateRequest{
		ManagerIDs: []manager.ManagerID{manager.ManagerMise}, Strategy: dto.StrategyStable,
		Policies: map[manager.ManagerID]dto.UpdatePolicy{manager.ManagerMise: {MiseTools: []string{"go", "jq"}}},
		Bounded:  dto.BoundedUpdateOptions{Enabled: true, BatchSize: 1, StateFile: statePath},
	}
	resp, err := uc.Update(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Deferred || !slices.Equal(calls, []string{"go"}) {
		t.Fatalf("deferred=%v calls=%v", resp.Deferred, calls)
	}
	state, err := loadBounded(statePath, []*manager.Manager{{ID: manager.ManagerMise}}, req)
	if err != nil || state.Items[0].Status != "complete" || state.Items[1].Status != "deferred" {
		t.Fatalf("state=%#v err=%v", state, err)
	}
}

func TestBoundedUpdateRejectsUnselectedManager(t *testing.T) {
	uc := NewUseCase(&mockRepository{findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
		return &manager.Manager{ID: id, Name: string(id), Installed: true}, nil
	}}, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{manager.ManagerHomebrew: &mockAdapter{}}, nil)
	_, err := uc.Update(context.Background(), &dto.UpdateRequest{
		ManagerIDs: []manager.ManagerID{manager.ManagerHomebrew}, Strategy: dto.StrategyStable,
		Bounded: dto.BoundedUpdateOptions{Enabled: true, BatchSize: 1, StateFile: filepath.Join(t.TempDir(), "state.json")},
	})
	if err == nil {
		t.Fatal("unselected bounded brew update was accepted")
	}
}

func TestBoundedStateRejectsChangedSelectionAndFailedItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	managers := []*manager.Manager{{ID: manager.ManagerMise}}
	req := &dto.UpdateRequest{Policies: map[manager.ManagerID]dto.UpdatePolicy{manager.ManagerMise: {MiseTools: []string{"go"}}}}
	fingerprint, err := boundedFingerprint(managers, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveBounded(path, &boundedState{Fingerprint: fingerprint, Items: []boundedItem{{Manager: manager.ManagerMise, Package: "go", Status: boundedFailed}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := loadBounded(path, managers, req); err == nil {
		t.Fatal("failed package state was resumed")
	}
	req.Policies[manager.ManagerMise] = dto.UpdatePolicy{MiseTools: []string{"jq"}}
	if _, err := loadBounded(path, managers, req); err == nil {
		t.Fatal("changed selection state was accepted")
	}
}

func TestBoundedStateRefreshesCompletedPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	managers := []*manager.Manager{{ID: manager.ManagerMise}}
	req := &dto.UpdateRequest{Policies: map[manager.ManagerID]dto.UpdatePolicy{manager.ManagerMise: {MiseTools: []string{"go"}}}}
	fingerprint, err := boundedFingerprint(managers, req)
	if err != nil {
		t.Fatal(err)
	}
	if saveErr := saveBounded(path, &boundedState{Fingerprint: fingerprint, Items: []boundedItem{{Manager: manager.ManagerMise, Package: "go", Status: boundedComplete}}}); saveErr != nil {
		t.Fatal(saveErr)
	}
	state, err := loadBounded(path, managers, req)
	if err != nil || state.Items[0].Status != boundedPending {
		t.Fatalf("state=%#v err=%v", state, err)
	}
}
