package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/usecase/update"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

func TestNewManagerAdaptersRegistersSupportedManagers(t *testing.T) {
	adapters := newManagerAdapters(nil, nil)

	expected := []manager.ManagerID{
		manager.ManagerApt,
		manager.ManagerHomebrew,
		manager.ManagerASDF,
		manager.ManagerNPM,
		manager.ManagerCargo,
		manager.ManagerPip,
		manager.ManagerWinget,
		manager.ManagerScoop,
		manager.ManagerChocolatey,
		manager.ManagerPacman,
	}

	if len(adapters) != len(expected) {
		t.Fatalf("adapter registry has %d entries, want %d", len(adapters), len(expected))
	}

	for _, id := range expected {
		if adapters[id] == nil {
			t.Errorf("adapter registry is missing %q", id)
		}
	}
}

func TestNewManagerAdaptersUpdateUseCaseSupportsAptAndPacmanDryRun(t *testing.T) {
	var executorCalls int
	executor := testutil.NewMockExecutor(func(context.Context, string, ...string) (*output.ExecutionResult, error) {
		executorCalls++
		return testutil.SuccessResult(""), nil
	})
	adapters := newManagerAdapters(executor, testutil.NewMockLogger())
	repository := &adapterRegistryRepository{managers: map[manager.ManagerID]*manager.Manager{
		manager.ManagerApt:    {ID: manager.ManagerApt, Name: "APT", Installed: true},
		manager.ManagerPacman: {ID: manager.ManagerPacman, Name: "Pacman", Installed: true},
	}}
	useCase := update.NewUseCase(repository, testutil.NewMockLogger(), adapters, nil)

	response, err := useCase.Update(context.Background(), &dto.UpdateRequest{
		ManagerIDs: []manager.ManagerID{manager.ManagerApt, manager.ManagerPacman},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("dry-run update unexpected error: %v", err)
	}
	if len(response.Results) != 2 {
		t.Fatalf("dry-run update returned %d results, want 2", len(response.Results))
	}
	for _, result := range response.Results {
		if !result.Success {
			t.Errorf("%s dry-run failed: %s", result.ID, result.Error)
		}
		if result.Error == "no adapter found for manager: "+string(result.ID) {
			t.Errorf("%s was not registered in the adapter factory", result.ID)
		}
	}
	if executorCalls != 0 {
		t.Fatalf("dry-run update executed %d commands, want 0", executorCalls)
	}
}

type adapterRegistryRepository struct {
	managers map[manager.ManagerID]*manager.Manager
}

func (r *adapterRegistryRepository) FindAll(context.Context) ([]*manager.Manager, error) {
	return nil, errors.New("not implemented")
}

func (r *adapterRegistryRepository) FindInstalled(context.Context) ([]*manager.Manager, error) {
	return nil, errors.New("not implemented")
}

func (r *adapterRegistryRepository) FindByID(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
	result, ok := r.managers[id]
	if !ok {
		return nil, errors.New("manager not found")
	}
	return result, nil
}

func (r *adapterRegistryRepository) Save(context.Context, *manager.Manager) error {
	return nil
}

func (r *adapterRegistryRepository) Delete(context.Context, manager.ManagerID) error {
	return nil
}
