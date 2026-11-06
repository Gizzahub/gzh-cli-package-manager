//go:build darwin || linux

package update

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func TestBoundedResumeCountsOnlyNativeChangesThisRun(t *testing.T) {
	adapter := &mockAdapter{updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
		return &adapterm.UpdateResult{Success: true, UpdatedPackages: opts.Packages}, nil
	}}
	uc := NewUseCase(&mockRepository{findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
		return &manager.Manager{ID: id, Installed: true}, nil
	}}, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{manager.ManagerMise: adapter}, nil)
	req := &dto.UpdateRequest{ManagerIDs: []manager.ManagerID{manager.ManagerMise}, Strategy: dto.StrategyStable, MiseTools: []string{"go", "jq"}, Bounded: dto.BoundedUpdateOptions{Enabled: true, BatchSize: 1, StateFile: filepath.Join(t.TempDir(), "state.json")}}
	first, err := uc.Update(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Summary.TotalPackagesUpdated != 1 || !first.Deferred {
		t.Fatalf("first: %+v", first)
	}
	second, err := uc.Update(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if second.Summary.TotalPackagesUpdated != 1 || second.Deferred || len(second.Results) != 1 || second.Results[0].UpdatedPackages[0].Name != "jq" {
		t.Fatalf("resumed response counts prior run: %+v", second)
	}
	if len(second.Progress) != 2 || second.Progress[0].Status != boundedComplete {
		t.Fatalf("progress missing: %+v", second.Progress)
	}
}

func TestBoundedStateBindsEffectiveCLIOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	managers := []*manager.Manager{{ID: manager.ManagerMise}}
	req := &dto.UpdateRequest{Strategy: dto.StrategyStable, MiseTools: []string{"go"}, MiseDir: "/initial"}
	fingerprint, err := boundedFingerprint(managers, req)
	if err != nil {
		t.Fatal(err)
	}
	state := newBoundedState(fingerprint, managers, req)
	if err := saveBounded(path, state); err != nil {
		t.Fatal(err)
	}
	req.Strategy = dto.StrategyLatest
	if _, err := loadBounded(path, managers, req); err == nil {
		t.Fatal("changed CLI strategy resumed old plan")
	}
	req.Strategy = dto.StrategyStable
	req.MiseDir = "/different"
	if _, err := loadBounded(path, managers, req); err == nil {
		t.Fatal("changed CLI scope resumed old plan")
	}
}

func TestBoundedDryRunPreservesPendingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	managers := []*manager.Manager{{ID: manager.ManagerMise}}
	req := &dto.UpdateRequest{ManagerIDs: []manager.ManagerID{manager.ManagerMise}, Strategy: dto.StrategyStable, MiseTools: []string{"go"}, DryRun: true, Bounded: dto.BoundedUpdateOptions{Enabled: true, BatchSize: 1, StateFile: path}}
	fingerprint, err := boundedFingerprint(managers, req)
	if err != nil {
		t.Fatal(err)
	}
	if saveErr := saveBounded(path, newBoundedState(fingerprint, managers, req)); saveErr != nil {
		t.Fatal(saveErr)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	adapter := &mockAdapter{updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
		calls++
		if !opts.DryRun || len(opts.Packages) != 1 {
			t.Fatal("preview lost selection or dry-run")
		}
		return &adapterm.UpdateResult{Success: true, Message: "go candidate"}, nil
	}}
	uc := NewUseCase(&mockRepository{findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
		return &manager.Manager{ID: id, Installed: true}, nil
	}}, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{manager.ManagerMise: adapter}, nil)
	if _, updateErr := uc.Update(context.Background(), req); updateErr != nil {
		t.Fatal(updateErr)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || calls != 1 {
		t.Fatal("preview altered durable state or omitted native plan")
	}
}

func TestBoundedLockCreatesParentAndExcludesSecondRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "state.json")
	unlock, err := lockBounded(path)
	if err != nil {
		t.Fatal(err)
	}
	if second, lockErr := lockBounded(path); lockErr == nil {
		second()
		unlock()
		t.Fatal("concurrent run accepted")
	}
	unlock()
	released, err := lockBounded(path)
	if err != nil {
		t.Fatal(err)
	}
	released()
}

func TestDarwinAdmissionFailsClosedAtBoundaries(t *testing.T) {
	for _, test := range []struct {
		output   string
		rejected bool
	}{
		{"total = 10240.00M used = 10239.99M free = 0.01M", false},
		{"used = 10240.00M", true},
		{"used = 11253.76M", true},
		{"garbage", true},
		{"used = 1..2M", true},
	} {
		if rejected := parseDarwinSwap(test.output, 10) != nil; rejected != test.rejected {
			t.Fatalf("swap %q rejected=%v", test.output, rejected)
		}
	}
	for _, test := range []struct {
		output   string
		rejected bool
	}{
		{"System-wide memory free percentage: 10%", false},
		{"System-wide memory free percentage: 9%", true},
		{"System-wide memory free percentage: 101%", true},
		{"garbage", true},
	} {
		if rejected := parseDarwinPressure(test.output) != nil; rejected != test.rejected {
			t.Fatalf("pressure %q rejected=%v", test.output, rejected)
		}
	}
}
