package update

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/detector"
)

const (
	testHomebrewManagerName = "Homebrew"
	testNPMManagerName      = "NPM"
	testPipManagerName      = "Pip"
	testGitPackageName      = "git"
)

// mockRepository implements manager.Repository for testing.
type mockRepository struct {
	findAllFunc       func(ctx context.Context) ([]*manager.Manager, error)
	findInstalledFunc func(ctx context.Context) ([]*manager.Manager, error)
	findByIDFunc      func(ctx context.Context, id manager.ManagerID) (*manager.Manager, error)
}

func (m *mockRepository) FindAll(ctx context.Context) ([]*manager.Manager, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockRepository) FindInstalled(ctx context.Context) ([]*manager.Manager, error) {
	if m.findInstalledFunc != nil {
		return m.findInstalledFunc(ctx)
	}
	return nil, nil
}

func (m *mockRepository) FindByID(ctx context.Context, id manager.ManagerID) (*manager.Manager, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockRepository) Save(_ context.Context, _ *manager.Manager) error {
	return nil
}

func (m *mockRepository) Delete(_ context.Context, _ manager.ManagerID) error {
	return nil
}

// mockLogger implements output.Logger for testing.
type mockLogger struct {
	infoMessages  []string
	errorMessages []string
	warnMessages  []string
}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field) {}

func (m *mockLogger) Info(_ context.Context, msg string, _ ...output.Field) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(_ context.Context, msg string, _ ...output.Field) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *mockLogger) Error(_ context.Context, msg string, _ error, _ ...output.Field) {
	m.errorMessages = append(m.errorMessages, msg)
}

// mockAdapter implements adapterm.Adapter for testing.
type mockAdapter struct {
	updateFunc func(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error)
}

type updateAllManagersCase struct {
	name           string
	mockManagers   []*manager.Manager
	mockError      error
	mockAdapters   map[manager.ManagerID]*mockAdapter
	wantSuccessful int
	wantFailed     int
	wantErr        bool
}

type updateSpecificManagersCase struct {
	name           string
	requestIDs     []manager.ManagerID
	mockManagers   map[manager.ManagerID]*manager.Manager
	mockAdapters   map[manager.ManagerID]*mockAdapter
	wantSuccessful int
	wantFailed     int
	wantErr        bool
}

func (m *mockAdapter) Detect(_ context.Context) (bool, error) {
	return true, nil
}

func (m *mockAdapter) GetVersion(_ context.Context) (string, error) {
	return "1.0.0", nil
}

func (m *mockAdapter) GetBinaryPath(_ context.Context) (string, error) {
	return "/usr/bin/mock", nil
}

func (m *mockAdapter) GetConfigPath(_ context.Context) (string, error) {
	return "/etc/mock", nil
}

func (m *mockAdapter) ListPackages(_ context.Context) ([]manager.Package, error) {
	return nil, nil
}

func (m *mockAdapter) CheckHealth(_ context.Context) (manager.Status, error) {
	return manager.StatusHealthy, nil
}

func (m *mockAdapter) Update(ctx context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, opts)
	}
	return &adapterm.UpdateResult{Success: true}, nil
}

func makeAdapterMap(mockAdapters map[manager.ManagerID]*mockAdapter) map[manager.ManagerID]adapterm.Adapter {
	adapters := make(map[manager.ManagerID]adapterm.Adapter, len(mockAdapters))
	for id, adapter := range mockAdapters {
		adapters[id] = adapter
	}
	return adapters
}

func assertUpdateSummary(t *testing.T, resp *dto.UpdateResponse, total, successful, failed int) {
	t.Helper()
	if resp == nil {
		t.Fatal("Update() returned nil response")
	}
	if resp.Summary.TotalManagers != total {
		t.Errorf("Summary.TotalManagers = %d, want %d", resp.Summary.TotalManagers, total)
	}
	if resp.Summary.SuccessfulManagers != successful {
		t.Errorf("Summary.SuccessfulManagers = %d, want %d", resp.Summary.SuccessfulManagers, successful)
	}
	if resp.Summary.FailedManagers != failed {
		t.Errorf("Summary.FailedManagers = %d, want %d", resp.Summary.FailedManagers, failed)
	}
	if resp.Summary.SkippedManagers != 0 {
		t.Errorf("Summary.SkippedManagers = %d, want 0", resp.Summary.SkippedManagers)
	}
}

func TestNewUseCase(t *testing.T) {
	repo := &mockRepository{}
	logger := &mockLogger{}
	adapters := map[manager.ManagerID]adapterm.Adapter{}

	uc := NewUseCase(repo, logger, adapters, nil)

	if uc == nil {
		t.Fatal("NewUseCase() returned nil")
	}
	if uc.managerRepo != repo {
		t.Error("UseCase.managerRepo not set correctly")
	}
	if uc.logger != logger {
		t.Error("UseCase.logger not set correctly")
	}
	if uc.adapters == nil {
		t.Error("UseCase.adapters not set correctly")
	}
}

func TestUseCase_Update_AllManagers(t *testing.T) {
	now := time.Now()

	tests := []updateAllManagersCase{
		{
			name:           "no managers installed",
			mockManagers:   []*manager.Manager{},
			wantSuccessful: 0,
			wantFailed:     0,
			wantErr:        false,
		},
		{
			name: "single manager success",
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Type:        manager.TypeSystem,
					Installed:   true,
					Version:     "4.2.1",
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages:    []manager.Package{},
				},
			},
			mockAdapters: map[manager.ManagerID]*mockAdapter{
				manager.ManagerHomebrew: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{
							Success:         true,
							UpdatedPackages: []string{testGitPackageName, "wget"},
							Message:         "Updated successfully",
						}, nil
					},
				},
			},
			wantSuccessful: 1,
			wantFailed:     0,
			wantErr:        false,
		},
		{
			name: "single manager failure",
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   true,
					LastChecked: now,
				},
			},
			mockAdapters: map[manager.ManagerID]*mockAdapter{
				manager.ManagerHomebrew: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return nil, errors.New("update failed")
					},
				},
			},
			wantSuccessful: 0,
			wantFailed:     1,
			wantErr:        false,
		},
		{
			name: "multiple managers mixed results",
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   true,
					LastChecked: now,
				},
				{
					ID:          manager.ManagerNPM,
					Name:        testNPMManagerName,
					Installed:   true,
					LastChecked: now,
				},
				{
					ID:          manager.ManagerPip,
					Name:        testPipManagerName,
					Installed:   true,
					LastChecked: now,
				},
			},
			mockAdapters: map[manager.ManagerID]*mockAdapter{
				manager.ManagerHomebrew: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{testGitPackageName}}, nil
					},
				},
				manager.ManagerNPM: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return nil, errors.New("npm update failed")
					},
				},
				manager.ManagerPip: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{Success: true}, nil
					},
				},
			},
			wantSuccessful: 2,
			wantFailed:     1,
			wantErr:        false,
		},
		{
			name:         "repository error",
			mockError:    errors.New("database connection failed"),
			mockAdapters: map[manager.ManagerID]*mockAdapter{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testUpdateAllManagers(t, &tt)
		})
	}
}

func testUpdateAllManagers(t *testing.T, tt *updateAllManagersCase) {
	t.Helper()
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			if tt.mockError != nil {
				return nil, tt.mockError
			}
			return tt.mockManagers, nil
		},
	}
	logger := &mockLogger{}
	adapters := makeAdapterMap(tt.mockAdapters)

	resp, err := NewUseCase(repo, logger, adapters, nil).Update(context.Background(), &dto.UpdateRequest{
		All:      true,
		Strategy: dto.StrategyStable,
	})
	if (err != nil) != tt.wantErr {
		t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
		return
	}
	if tt.wantErr {
		return
	}
	assertUpdateSummary(t, resp, len(tt.mockManagers), tt.wantSuccessful, tt.wantFailed)
	if len(logger.infoMessages) < 2 {
		t.Error("Expected at least 2 info log messages")
	}
}

func TestUseCase_Update_AllTakesPrecedenceOverManagerIDs(t *testing.T) {
	findInstalledCalls := 0
	findByIDCalls := 0
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			findInstalledCalls++
			return []*manager.Manager{{
				ID:        manager.ManagerHomebrew,
				Name:      testHomebrewManagerName,
				Installed: true,
			}}, nil
		},
		findByIDFunc: func(_ context.Context, _ manager.ManagerID) (*manager.Manager, error) {
			findByIDCalls++
			return nil, errors.New("unexpected FindByID call")
		},
	}
	uc := NewUseCase(repo, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: &mockAdapter{},
	}, nil)

	resp, err := uc.Update(context.Background(), &dto.UpdateRequest{
		All:        true,
		ManagerIDs: []manager.ManagerID{"missing"},
	})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if findInstalledCalls != 1 || findByIDCalls != 0 {
		t.Fatalf("repository calls = FindInstalled %d, FindByID %d; want 1, 0", findInstalledCalls, findByIDCalls)
	}
	if resp.Summary.SuccessfulManagers != 1 || resp.Summary.FailedManagers != 0 {
		t.Errorf("summary = successful %d failed %d, want 1 and 0", resp.Summary.SuccessfulManagers, resp.Summary.FailedManagers)
	}
}

func TestUseCase_Update_SpecificManagers(t *testing.T) {
	now := time.Now()

	tests := []updateSpecificManagersCase{
		{
			name:       "single specific manager",
			requestIDs: []manager.ManagerID{manager.ManagerHomebrew},
			mockManagers: map[manager.ManagerID]*manager.Manager{
				manager.ManagerHomebrew: {
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   true,
					LastChecked: now,
				},
			},
			mockAdapters: map[manager.ManagerID]*mockAdapter{
				manager.ManagerHomebrew: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{Success: true}, nil
					},
				},
			},
			wantSuccessful: 1,
			wantFailed:     0,
			wantErr:        false,
		},
		{
			name:       "multiple specific managers",
			requestIDs: []manager.ManagerID{manager.ManagerHomebrew, manager.ManagerNPM},
			mockManagers: map[manager.ManagerID]*manager.Manager{
				manager.ManagerHomebrew: {
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   true,
					LastChecked: now,
				},
				manager.ManagerNPM: {
					ID:          manager.ManagerNPM,
					Name:        testNPMManagerName,
					Installed:   true,
					LastChecked: now,
				},
			},
			mockAdapters: map[manager.ManagerID]*mockAdapter{
				manager.ManagerHomebrew: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{Success: true}, nil
					},
				},
				manager.ManagerNPM: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{Success: true}, nil
					},
				},
			},
			wantSuccessful: 2,
			wantFailed:     0,
			wantErr:        false,
		},
		{
			name:         "manager not found",
			requestIDs:   []manager.ManagerID{manager.ManagerHomebrew},
			mockManagers: map[manager.ManagerID]*manager.Manager{},
			mockAdapters: map[manager.ManagerID]*mockAdapter{},
			wantErr:      true,
		},
		{
			name:       "manager not installed (skipped)",
			requestIDs: []manager.ManagerID{manager.ManagerHomebrew},
			mockManagers: map[manager.ManagerID]*manager.Manager{
				manager.ManagerHomebrew: {
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   false, // Not installed
					LastChecked: time.Now(),
				},
			},
			mockAdapters:   map[manager.ManagerID]*mockAdapter{},
			wantSuccessful: 0,
			wantFailed:     0,
			wantErr:        false, // No error, just skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testUpdateSpecificManagers(t, &tt)
		})
	}
}

func testUpdateSpecificManagers(t *testing.T, tt *updateSpecificManagersCase) {
	t.Helper()
	repo := &mockRepository{
		findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
			if mgr, exists := tt.mockManagers[id]; exists {
				return mgr, nil
			}
			return nil, errors.New("manager not found")
		},
	}
	resp, err := NewUseCase(repo, &mockLogger{}, makeAdapterMap(tt.mockAdapters), nil).Update(context.Background(), &dto.UpdateRequest{
		ManagerIDs: tt.requestIDs,
		Strategy:   dto.StrategyStable,
	})
	if (err != nil) != tt.wantErr {
		t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
		return
	}
	if !tt.wantErr {
		assertUpdateSummary(t, resp, tt.wantSuccessful+tt.wantFailed, tt.wantSuccessful, tt.wantFailed)
	}
}

func TestUseCase_SelectManagers_PreservesSpecificRequestOrder(t *testing.T) {
	requested := []manager.ManagerID{manager.ManagerNPM, manager.ManagerHomebrew, manager.ManagerPip}
	calls := make([]manager.ManagerID, 0, len(requested))
	repo := &mockRepository{
		findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
			calls = append(calls, id)
			return &manager.Manager{ID: id, Installed: id != manager.ManagerHomebrew}, nil
		},
	}

	managers, err := NewUseCase(repo, &mockLogger{}, nil, nil).selectManagers(context.Background(), &dto.UpdateRequest{
		ManagerIDs: requested,
	})
	if err != nil {
		t.Fatalf("selectManagers() unexpected error: %v", err)
	}
	if !slices.Equal(calls, requested) {
		t.Errorf("FindByID calls = %v, want %v", calls, requested)
	}
	if len(managers) != 2 || managers[0].ID != manager.ManagerNPM || managers[1].ID != manager.ManagerPip {
		t.Errorf("selected managers = %#v, want NPM then Pip", managers)
	}
}

func TestUseCase_SelectManagers_StopsOnFirstLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	requested := []manager.ManagerID{manager.ManagerHomebrew, manager.ManagerNPM, manager.ManagerPip}
	calls := make([]manager.ManagerID, 0, len(requested))
	repo := &mockRepository{
		findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
			calls = append(calls, id)
			if id == manager.ManagerNPM {
				return nil, lookupErr
			}
			return &manager.Manager{ID: id, Installed: true}, nil
		},
	}

	_, err := NewUseCase(repo, &mockLogger{}, nil, nil).selectManagers(context.Background(), &dto.UpdateRequest{
		ManagerIDs: requested,
	})
	if !errors.Is(err, lookupErr) {
		t.Fatalf("selectManagers() error = %v, want wrapped %v", err, lookupErr)
	}
	if err.Error() != "failed to fetch manager npm: lookup failed" {
		t.Errorf("selectManagers() error = %q, want exact context", err)
	}
	if !slices.Equal(calls, requested[:2]) {
		t.Errorf("FindByID calls = %v, want %v", calls, requested[:2])
	}
}

func TestUseCase_Update_AllowsPipInCondaWhenRequested(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "test-conda")
	t.Setenv("CONDA_PREFIX", "/tmp/test-conda")
	pipAdapterCalls := 0
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{{ID: manager.ManagerPip, Name: testPipManagerName, Installed: true}}, nil
		},
	}
	uc := NewUseCase(repo, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerPip: &mockAdapter{
			updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
				pipAdapterCalls++
				return &adapterm.UpdateResult{Success: true}, nil
			},
		},
	}, detector.NewDetector(nil, &mockLogger{}))

	resp, err := uc.Update(context.Background(), &dto.UpdateRequest{
		All:           true,
		PipAllowConda: true,
		Strategy:      dto.StrategyStable,
	})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("result count = %d, want 1", len(resp.Results))
	}
	if pipAdapterCalls != 1 || resp.Results[0].Skipped {
		t.Errorf("pip calls/skipped = %d/%t, want 1/false", pipAdapterCalls, resp.Results[0].Skipped)
	}
	assertUpdateSummary(t, resp, 1, 1, 0)
}

func TestUseCase_Update_PreservesResultShapeAndOrder(t *testing.T) {
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{ID: manager.ManagerHomebrew, Name: testHomebrewManagerName, Installed: true},
				{ID: manager.ManagerNPM, Name: testNPMManagerName, Installed: true},
			}, nil
		},
	}
	uc := NewUseCase(repo, &mockLogger{}, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: &mockAdapter{
			updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
				return &adapterm.UpdateResult{
					Success:         true,
					UpdatedPackages: []string{testGitPackageName, "curl"},
					FailedPackages:  []string{"openssl"},
				}, nil
			},
		},
		manager.ManagerNPM: &mockAdapter{
			updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
				return &adapterm.UpdateResult{Success: false, UpdatedPackages: []string{"npm"}}, nil
			},
		},
	}, nil)

	resp, err := uc.Update(context.Background(), &dto.UpdateRequest{All: true, Strategy: dto.StrategyStable})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	assertUpdateSummary(t, resp, 2, 1, 1)
	if resp.Summary.TotalPackagesUpdated != 3 {
		t.Errorf("Summary.TotalPackagesUpdated = %d, want 3", resp.Summary.TotalPackagesUpdated)
	}
	if len(resp.Results) != 2 || resp.Results[0].ID != manager.ManagerHomebrew || resp.Results[1].ID != manager.ManagerNPM {
		t.Fatalf("result order = %#v, want Homebrew then NPM", resp.Results)
	}
	first := resp.Results[0]
	if len(first.UpdatedPackages) != 2 {
		t.Fatalf("updated packages = %#v, want git then curl", first.UpdatedPackages)
	}
	if first.UpdatedPackages[0].Name != testGitPackageName || first.UpdatedPackages[1].Name != "curl" {
		t.Errorf("updated package order = %#v, want git then curl", first.UpdatedPackages)
	}
	if first.UpdatedPackages[0].OldVersion != "unknown" || first.UpdatedPackages[0].NewVersion != "unknown" || first.UpdatedPackages[0].UpdateType != manager.UpdateMinor || first.UpdatedPackages[0].SizeBytes != 0 {
		t.Errorf("package update = %#v, want unknown/minor/zero conversion", first.UpdatedPackages[0])
	}
	if !slices.Equal(first.SkippedPackages, []string{"openssl"}) {
		t.Errorf("SkippedPackages = %v, want [openssl]", first.SkippedPackages)
	}
}

func TestUseCase_Update_SkipsPipInCondaEnvironment(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "test-conda")
	t.Setenv("CONDA_PREFIX", "/tmp/test-conda")

	pipAdapterCalls := 0
	homebrewAdapterCalls := 0
	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:        manager.ManagerPip,
					Name:      testPipManagerName,
					Installed: true,
				},
				{
					ID:        manager.ManagerHomebrew,
					Name:      testHomebrewManagerName,
					Installed: true,
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	uc := NewUseCase(repo, logger, map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerPip: &mockAdapter{
			updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
				pipAdapterCalls++
				return &adapterm.UpdateResult{Success: true}, nil
			},
		},
		manager.ManagerHomebrew: &mockAdapter{
			updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
				homebrewAdapterCalls++
				return &adapterm.UpdateResult{Success: true}, nil
			},
		},
	}, detector.NewDetector(nil, logger))

	resp, err := uc.Update(context.Background(), &dto.UpdateRequest{
		All:      true,
		Strategy: dto.StrategyStable,
	})
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if pipAdapterCalls != 0 || homebrewAdapterCalls != 1 {
		t.Errorf("adapter calls = pip %d homebrew %d, want 0 and 1", pipAdapterCalls, homebrewAdapterCalls)
	}
	if len(resp.Results) != 2 || !resp.Results[0].Skipped || !resp.Results[0].Success || resp.Results[1].Skipped || !resp.Results[1].Success {
		t.Errorf("results = %#v, want skipped Pip then successful Homebrew", resp.Results)
	}
	if resp.Summary.SkippedManagers != 1 || resp.Summary.SuccessfulManagers != 1 || resp.Summary.FailedManagers != 0 {
		t.Errorf(
			"summary = skipped %d successful %d failed %d, want 1, 1, 0",
			resp.Summary.SkippedManagers,
			resp.Summary.SuccessfulManagers,
			resp.Summary.FailedManagers,
		)
	}
}

func TestUseCase_Update_DryRunMode(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		dryRun       bool
		wantDryRun   bool
		verifyUpdate bool // Should the adapter's update method be called?
	}{
		{
			name:         "dry-run enabled",
			dryRun:       true,
			wantDryRun:   true,
			verifyUpdate: true,
		},
		{
			name:         "dry-run disabled",
			dryRun:       false,
			wantDryRun:   false,
			verifyUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalled := false
			dryRunPassed := false

			repo := &mockRepository{
				findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
					return []*manager.Manager{
						{
							ID:          manager.ManagerHomebrew,
							Name:        testHomebrewManagerName,
							Installed:   true,
							LastChecked: now,
						},
					}, nil
				},
			}
			logger := &mockLogger{}

			adapter := &mockAdapter{
				updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					updateCalled = true
					dryRunPassed = opts.DryRun == tt.dryRun
					return &adapterm.UpdateResult{Success: true}, nil
				},
			}

			adapters := map[manager.ManagerID]adapterm.Adapter{
				manager.ManagerHomebrew: adapter,
			}

			uc := NewUseCase(repo, logger, adapters, nil)

			req := &dto.UpdateRequest{
				All:      true,
				DryRun:   tt.dryRun,
				Strategy: dto.StrategyStable,
			}

			resp, err := uc.Update(context.Background(), req)
			if err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}

			if resp.DryRun != tt.wantDryRun {
				t.Errorf("Response.DryRun = %v, want %v", resp.DryRun, tt.wantDryRun)
			}

			if tt.verifyUpdate && !updateCalled {
				t.Error("Adapter Update method was not called")
			}

			if tt.verifyUpdate && !dryRunPassed {
				t.Errorf("Adapter received DryRun=%v, expected %v", !tt.dryRun, tt.dryRun)
			}
		})
	}
}

func TestUseCase_Update_Strategies(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		inputStrategy   dto.UpdateStrategy
		expectedAdapter adapterm.UpdateStrategy
	}{
		{
			name:            "strategy stable",
			inputStrategy:   dto.StrategyStable,
			expectedAdapter: adapterm.StrategyStable,
		},
		{
			name:            "strategy latest",
			inputStrategy:   dto.StrategyLatest,
			expectedAdapter: adapterm.StrategyLatest,
		},
		{
			name:            "strategy minor",
			inputStrategy:   dto.StrategyMinor,
			expectedAdapter: adapterm.StrategyMinor,
		},
		{
			name:            "strategy fixed",
			inputStrategy:   dto.StrategyFixed,
			expectedAdapter: adapterm.StrategyFixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedStrategy adapterm.UpdateStrategy

			repo := &mockRepository{
				findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
					return []*manager.Manager{
						{
							ID:          manager.ManagerHomebrew,
							Name:        testHomebrewManagerName,
							Installed:   true,
							LastChecked: now,
						},
					}, nil
				},
			}
			logger := &mockLogger{}

			adapter := &mockAdapter{
				updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
					receivedStrategy = opts.Strategy
					return &adapterm.UpdateResult{Success: true}, nil
				},
			}

			adapters := map[manager.ManagerID]adapterm.Adapter{
				manager.ManagerHomebrew: adapter,
			}

			uc := NewUseCase(repo, logger, adapters, nil)

			req := &dto.UpdateRequest{
				All:      true,
				Strategy: tt.inputStrategy,
			}

			_, err := uc.Update(context.Background(), req)
			if err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}

			if receivedStrategy != tt.expectedAdapter {
				t.Errorf("Adapter received strategy %v, want %v", receivedStrategy, tt.expectedAdapter)
			}
		})
	}
}

func TestUseCase_Update_NoAdapterFound(t *testing.T) {
	now := time.Now()

	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   true,
					LastChecked: now,
				},
			}, nil
		},
	}
	logger := &mockLogger{}

	// No adapters registered
	adapters := map[manager.ManagerID]adapterm.Adapter{}

	uc := NewUseCase(repo, logger, adapters, nil)

	req := &dto.UpdateRequest{
		All:      true,
		Strategy: dto.StrategyStable,
	}

	resp, err := uc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if resp.Summary.SuccessfulManagers != 0 {
		t.Errorf("Expected 0 successful managers, got %d", resp.Summary.SuccessfulManagers)
	}

	if resp.Summary.FailedManagers != 1 {
		t.Errorf("Expected 1 failed manager, got %d", resp.Summary.FailedManagers)
	}

	// Verify error was logged
	if len(logger.errorMessages) == 0 {
		t.Error("Expected error to be logged for missing adapter")
	}

	// Verify result has error message
	if len(resp.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(resp.Results))
	}

	if resp.Results[0].Success {
		t.Error("Expected result to be unsuccessful")
	}

	if resp.Results[0].Error == "" {
		t.Error("Expected error message in result")
	}
}

func TestUseCase_Update_NoManagersSpecified(t *testing.T) {
	repo := &mockRepository{}
	logger := &mockLogger{}
	adapters := map[manager.ManagerID]adapterm.Adapter{}

	uc := NewUseCase(repo, logger, adapters, nil)

	req := &dto.UpdateRequest{
		All:        false,
		ManagerIDs: []manager.ManagerID{},
		Strategy:   dto.StrategyStable,
	}

	_, err := uc.Update(context.Background(), req)

	if err == nil {
		t.Error("Expected error when neither --all nor --managers specified")
	}

	if err != nil && err.Error() != "either --all flag or --managers must be specified" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestUseCase_Update_WithNilEnvDetector(t *testing.T) {
	now := time.Now()

	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:          manager.ManagerPip,
					Name:        testPipManagerName,
					Installed:   true,
					LastChecked: now,
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	adapter := &mockAdapter{
		updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
			return &adapterm.UpdateResult{Success: true}, nil
		},
	}
	adapters := map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerPip: adapter,
	}

	// With nil detector, pip updates should proceed normally
	uc := NewUseCase(repo, logger, adapters, nil)

	req := &dto.UpdateRequest{
		All:      true,
		Strategy: dto.StrategyStable,
	}

	resp, err := uc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if resp.Summary.SuccessfulManagers != 1 {
		t.Errorf("Expected 1 successful manager, got %d", resp.Summary.SuccessfulManagers)
	}

	if resp.Summary.SkippedManagers != 0 {
		t.Errorf("Expected 0 skipped managers, got %d", resp.Summary.SkippedManagers)
	}
}

func TestUseCase_Update_UnknownStrategy(t *testing.T) {
	now := time.Now()

	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewManagerName,
					Installed:   true,
					LastChecked: now,
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	adapter := &mockAdapter{
		updateFunc: func(_ context.Context, opts adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
			// Verify that unknown strategy falls back to stable
			if opts.Strategy != adapterm.StrategyStable {
				return &adapterm.UpdateResult{
					Success: false,
					Message: "Expected stable strategy for unknown input",
				}, nil
			}
			return &adapterm.UpdateResult{Success: true}, nil
		},
	}
	adapters := map[manager.ManagerID]adapterm.Adapter{
		manager.ManagerHomebrew: adapter,
	}

	uc := NewUseCase(repo, logger, adapters, nil)

	req := &dto.UpdateRequest{
		All:      true,
		Strategy: dto.UpdateStrategy("unknown"), // Unknown strategy
	}

	resp, err := uc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if !resp.Results[0].Success {
		t.Error("Expected success with unknown strategy falling back to stable")
	}
}
