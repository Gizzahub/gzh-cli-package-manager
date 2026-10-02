package update

import (
	"context"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

const policyWorkspace = "/workspace"

func TestUpdatePreflightsAllPoliciesBeforeMutation(t *testing.T) {
	for _, strategy := range []dto.UpdateStrategy{dto.StrategyMinor, dto.StrategyMicro} {
		t.Run(string(strategy), func(t *testing.T) {
			calls := 0
			repo := &mockRepository{findInstalledFunc: func(context.Context) ([]*manager.Manager, error) {
				return []*manager.Manager{{ID: manager.ManagerMise, Installed: true}, {ID: manager.ManagerHomebrew, Installed: true}}, nil
			}}
			adapter := &mockAdapter{updateFunc: func(context.Context, adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
				calls++
				return &adapterm.UpdateResult{Success: true}, nil
			}}
			uc := NewUseCase(repo, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{
				manager.ManagerMise: adapter, manager.ManagerHomebrew: adapter,
			}, nil)
			if _, err := uc.Update(context.Background(), &dto.UpdateRequest{All: true, Strategy: strategy}); err == nil || calls != 0 {
				t.Fatalf("unsupported policy executed updates: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestPerManagerPoliciesPropagateMiseBumpAndScope(t *testing.T) {
	var received adapterm.UpdateOptions
	repo := &mockRepository{findInstalledFunc: func(context.Context) ([]*manager.Manager, error) {
		return []*manager.Manager{{ID: manager.ManagerMise, Installed: true}}, nil
	}}
	adapter := &mockAdapter{updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
		received = opts
		return &adapterm.UpdateResult{Success: true, Message: "native preview"}, nil
	}}
	uc := NewUseCase(repo, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{manager.ManagerMise: adapter}, nil)
	response, err := uc.Update(context.Background(), &dto.UpdateRequest{
		All: true, DryRun: true, Strategy: dto.StrategyStable,
		Policies: map[manager.ManagerID]dto.UpdatePolicy{manager.ManagerMise: {
			Strategy: dto.StrategyMicro, Bump: true, MiseDir: policyWorkspace, MiseLocal: true,
		}},
	})
	if err != nil || !received.Bump || !received.DryRun || !received.MiseLocal || received.MiseDir != policyWorkspace || received.Strategy != adapterm.StrategyMicro {
		t.Fatalf("options=%#v err=%v", received, err)
	}
	if response.Results[0].Message != "native preview" {
		t.Fatal("native plan was lost")
	}
}

func TestBumpRejectsUnsupportedManagersAndFixed(t *testing.T) {
	uc := &UseCase{}
	for _, id := range []manager.ManagerID{manager.ManagerHomebrew, manager.ManagerMise} {
		managers := []*manager.Manager{{ID: id}}
		req := &dto.UpdateRequest{Bump: true, Strategy: dto.StrategyFixed}
		if err := uc.validatePolicies(managers, req); err == nil {
			t.Fatal("fixed+bump must fail")
		}
	}
	if err := uc.validatePolicies([]*manager.Manager{{ID: manager.ManagerNPM}}, &dto.UpdateRequest{Bump: true}); err == nil {
		t.Fatal("npm must reject bump")
	}
}

func TestMiseScopeCannotBeSilentlyIgnored(t *testing.T) {
	uc := &UseCase{}
	brew := []*manager.Manager{{ID: manager.ManagerHomebrew}}
	if err := uc.validatePolicies(brew, &dto.UpdateRequest{MiseLocal: true}); err == nil {
		t.Fatal("mise-local without mise must fail")
	}
	req := &dto.UpdateRequest{Policies: map[manager.ManagerID]dto.UpdatePolicy{
		manager.ManagerHomebrew: {MiseDir: policyWorkspace},
	}}
	if err := uc.validatePolicies(brew, req); err == nil {
		t.Fatal("non-mise policy cannot contain mise scope")
	}
	mixed := []*manager.Manager{{ID: manager.ManagerHomebrew}, {ID: manager.ManagerMise}}
	req = &dto.UpdateRequest{MiseDir: policyWorkspace, MiseLocal: true}
	if err := uc.validatePolicies(mixed, req); err != nil {
		t.Fatal(err)
	}
	if opts := uc.updateOptions(manager.ManagerHomebrew, req); opts.MiseDir != "" || opts.MiseLocal {
		t.Fatalf("mise scope leaked to brew: %#v", opts)
	}
}

func TestInvalidMiseSelectionPreflightsBeforeOtherManagers(t *testing.T) {
	uc := &UseCase{}
	managers := []*manager.Manager{{ID: manager.ManagerHomebrew}, {ID: manager.ManagerMise}}
	for _, tools := range [][]string{{}, {"jq", "jq"}, {" "}, {" jq"}} {
		req := &dto.UpdateRequest{Policies: map[manager.ManagerID]dto.UpdatePolicy{
			manager.ManagerMise: {MiseTools: tools},
		}}
		if err := uc.validatePolicies(managers, req); err == nil {
			t.Fatalf("tools=%q accepted", tools)
		}
	}
}
