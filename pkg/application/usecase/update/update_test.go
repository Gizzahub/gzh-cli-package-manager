package update

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
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

	tests := []struct {
		name           string
		mockManagers   []*manager.Manager
		mockError      error
		mockAdapters   map[manager.ManagerID]*mockAdapter
		wantSuccessful int
		wantFailed     int
		wantErr        bool
	}{
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
					Name:        "Homebrew",
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
							UpdatedPackages: []string{"git", "wget"},
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
					Name:        "Homebrew",
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
					Name:        "Homebrew",
					Installed:   true,
					LastChecked: now,
				},
				{
					ID:          manager.ManagerNPM,
					Name:        "NPM",
					Installed:   true,
					LastChecked: now,
				},
				{
					ID:          manager.ManagerPip,
					Name:        "Pip",
					Installed:   true,
					LastChecked: now,
				},
			},
			mockAdapters: map[manager.ManagerID]*mockAdapter{
				manager.ManagerHomebrew: {
					updateFunc: func(_ context.Context, _ adapterm.UpdateOptions) (*adapterm.UpdateResult, error) {
						return &adapterm.UpdateResult{Success: true, UpdatedPackages: []string{"git"}}, nil
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
			repo := &mockRepository{
				findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockManagers, nil
				},
			}
			logger := &mockLogger{}

			// Convert mock adapters to adapter interface
			adapters := make(map[manager.ManagerID]adapterm.Adapter)
			for id, adapter := range tt.mockAdapters {
				adapters[id] = adapter
			}

			uc := NewUseCase(repo, logger, adapters, nil)

			req := &dto.UpdateRequest{
				All:      true,
				DryRun:   false,
				Strategy: dto.StrategyStable,
			}

			resp, err := uc.Update(context.Background(), req)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // No need to check response if we expected an error
			}

			// Validate response
			if resp == nil {
				t.Fatal("Update() returned nil response")
			}

			if resp.Summary.SuccessfulManagers != tt.wantSuccessful {
				t.Errorf("Summary.SuccessfulManagers = %d, want %d",
					resp.Summary.SuccessfulManagers, tt.wantSuccessful)
			}

			if resp.Summary.FailedManagers != tt.wantFailed {
				t.Errorf("Summary.FailedManagers = %d, want %d",
					resp.Summary.FailedManagers, tt.wantFailed)
			}

			if resp.Summary.TotalManagers != len(tt.mockManagers) {
				t.Errorf("Summary.TotalManagers = %d, want %d",
					resp.Summary.TotalManagers, len(tt.mockManagers))
			}

			// Verify logger was called
			if len(logger.infoMessages) < 2 {
				t.Error("Expected at least 2 info log messages")
			}
		})
	}
}

func TestUseCase_Update_SpecificManagers(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		requestIDs     []manager.ManagerID
		mockManagers   map[manager.ManagerID]*manager.Manager
		mockAdapters   map[manager.ManagerID]*mockAdapter
		wantSuccessful int
		wantFailed     int
		wantErr        bool
	}{
		{
			name:       "single specific manager",
			requestIDs: []manager.ManagerID{manager.ManagerHomebrew},
			mockManagers: map[manager.ManagerID]*manager.Manager{
				manager.ManagerHomebrew: {
					ID:          manager.ManagerHomebrew,
					Name:        "Homebrew",
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
					Name:        "Homebrew",
					Installed:   true,
					LastChecked: now,
				},
				manager.ManagerNPM: {
					ID:          manager.ManagerNPM,
					Name:        "NPM",
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
					Name:        "Homebrew",
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
			repo := &mockRepository{
				findByIDFunc: func(_ context.Context, id manager.ManagerID) (*manager.Manager, error) {
					if mgr, exists := tt.mockManagers[id]; exists {
						return mgr, nil
					}
					return nil, errors.New("manager not found")
				},
			}
			logger := &mockLogger{}

			adapters := make(map[manager.ManagerID]adapterm.Adapter)
			for id, adapter := range tt.mockAdapters {
				adapters[id] = adapter
			}

			uc := NewUseCase(repo, logger, adapters, nil)

			req := &dto.UpdateRequest{
				All:        false,
				ManagerIDs: tt.requestIDs,
				DryRun:     false,
				Strategy:   dto.StrategyStable,
			}

			resp, err := uc.Update(context.Background(), req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if resp == nil {
				t.Fatal("Update() returned nil response")
			}

			if resp.Summary.SuccessfulManagers != tt.wantSuccessful {
				t.Errorf("Summary.SuccessfulManagers = %d, want %d",
					resp.Summary.SuccessfulManagers, tt.wantSuccessful)
			}

			if resp.Summary.FailedManagers != tt.wantFailed {
				t.Errorf("Summary.FailedManagers = %d, want %d",
					resp.Summary.FailedManagers, tt.wantFailed)
			}
		})
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
							Name:        "Homebrew",
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
							Name:        "Homebrew",
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
					Name:        "Homebrew",
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

// mockEnvDetector implements environment detection for testing.
type mockEnvDetector struct {
	envType  string
	envName  string
	pipSafe  bool
	warnings []string
}

func (m *mockEnvDetector) Detect(_ context.Context) *struct {
	Type      string
	Name      string
	Path      string
	IsPipSafe bool
	Warnings  []string
} {
	return &struct {
		Type      string
		Name      string
		Path      string
		IsPipSafe bool
		Warnings  []string
	}{
		Type:      m.envType,
		Name:      m.envName,
		IsPipSafe: m.pipSafe,
		Warnings:  m.warnings,
	}
}

func TestUseCase_Update_WithNilEnvDetector(t *testing.T) {
	now := time.Now()

	repo := &mockRepository{
		findInstalledFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:          manager.ManagerPip,
					Name:        "Pip",
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
					Name:        "Homebrew",
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
