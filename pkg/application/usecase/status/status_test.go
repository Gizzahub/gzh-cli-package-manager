package status

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

// mockRepository implements manager.Repository for testing.
type mockRepository struct {
	findAllFunc  func(ctx context.Context) ([]*manager.Manager, error)
	findByIDFunc func(ctx context.Context, id manager.ManagerID) (*manager.Manager, error)
}

func (m *mockRepository) FindAll(ctx context.Context) ([]*manager.Manager, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockRepository) FindInstalled(_ context.Context) ([]*manager.Manager, error) {
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
}

func (m *mockLogger) Debug(_ context.Context, _ string, _ ...output.Field) {}

func (m *mockLogger) Info(_ context.Context, msg string, _ ...output.Field) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(_ context.Context, _ string, _ ...output.Field) {}

func (m *mockLogger) Error(_ context.Context, msg string, _ error, _ ...output.Field) {
	m.errorMessages = append(m.errorMessages, msg)
}

func TestNewUseCase(t *testing.T) {
	repo := &mockRepository{}
	logger := &mockLogger{}

	uc := NewUseCase(repo, logger)

	if uc == nil {
		t.Fatal("NewUseCase() returned nil")
	}
	if uc.managerRepo != repo {
		t.Error("UseCase.managerRepo not set correctly")
	}
	if uc.logger != logger {
		t.Error("UseCase.logger not set correctly")
	}
}

func TestUseCase_GetStatus_AllManagers(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		mockManagers    []*manager.Manager
		mockError       error
		wantManagersLen int
		wantInstalled   int
		wantHealthy     int
		wantPackages    int
		wantUpdatable   int
		wantErr         bool
	}{
		{
			name: "no managers",
			mockManagers: []*manager.Manager{},
			wantManagersLen: 0,
			wantInstalled:   0,
			wantHealthy:     0,
			wantPackages:    0,
			wantUpdatable:   0,
			wantErr:         false,
		},
		{
			name: "single healthy manager with packages",
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        "Homebrew",
					Type:        manager.TypeSystem,
					Platform:    manager.PlatformDarwin,
					Installed:   true,
					Version:     "4.2.1",
					Status:      manager.StatusHealthy,
					BinaryPath:  "/usr/local/bin/brew",
					ConfigPath:  "/usr/local",
					LastChecked: now,
					Packages: []manager.Package{
						{
							Name:             "git",
							CurrentVersion:   "2.43.0",
							AvailableVersion: "2.44.0",
							UpdateType:       manager.UpdateMinor,
						},
						{
							Name:           "wget",
							CurrentVersion: "1.21.0",
							UpdateType:     manager.UpdateNone,
						},
					},
				},
			},
			wantManagersLen: 1,
			wantInstalled:   1,
			wantHealthy:     1,
			wantPackages:    2,
			wantUpdatable:   1,
			wantErr:         false,
		},
		{
			name: "multiple managers mixed states",
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerHomebrew,
					Name:        "Homebrew",
					Type:        manager.TypeSystem,
					Installed:   true,
					Version:     "4.2.1",
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: "git", CurrentVersion: "2.43.0", UpdateType: manager.UpdateNone},
					},
				},
				{
					ID:          manager.ManagerPacman,
					Name:        "Pacman",
					Type:        manager.TypeSystem,
					Installed:   false,
					Status:      manager.StatusUnavailable,
					LastChecked: now,
					Packages:    []manager.Package{},
				},
				{
					ID:          manager.ManagerNPM,
					Name:        "NPM",
					Type:        manager.TypeLanguage,
					Installed:   true,
					Version:     "10.5.0",
					Status:      manager.StatusDegraded,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: "react", CurrentVersion: "18.0.0", AvailableVersion: "18.2.0", UpdateType: manager.UpdateMinor},
						{Name: "vue", CurrentVersion: "3.0.0", AvailableVersion: "3.4.0", UpdateType: manager.UpdateMinor},
					},
				},
			},
			wantManagersLen: 3,
			wantInstalled:   2,
			wantHealthy:     1, // Only Homebrew is healthy
			wantPackages:    3,
			wantUpdatable:   2,
			wantErr:         false,
		},
		{
			name:            "repository error",
			mockError:       errors.New("database connection failed"),
			wantManagersLen: 0,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{
				findAllFunc: func(_ context.Context) ([]*manager.Manager, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockManagers, nil
				},
			}
			logger := &mockLogger{}
			uc := NewUseCase(repo, logger)

			req := &dto.StatusRequest{
				Verbose: false,
				Refresh: false,
			}

			resp, err := uc.GetStatus(context.Background(), req)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // No need to check response if we expected an error
			}

			// Validate response
			if resp == nil {
				t.Fatal("GetStatus() returned nil response")
			}

			if len(resp.Managers) != tt.wantManagersLen {
				t.Errorf("GetStatus() returned %d managers, want %d", len(resp.Managers), tt.wantManagersLen)
			}

			if resp.Summary.TotalManagers != tt.wantManagersLen {
				t.Errorf("Summary.TotalManagers = %d, want %d", resp.Summary.TotalManagers, tt.wantManagersLen)
			}

			if resp.Summary.InstalledManagers != tt.wantInstalled {
				t.Errorf("Summary.InstalledManagers = %d, want %d", resp.Summary.InstalledManagers, tt.wantInstalled)
			}

			if resp.Summary.HealthyManagers != tt.wantHealthy {
				t.Errorf("Summary.HealthyManagers = %d, want %d", resp.Summary.HealthyManagers, tt.wantHealthy)
			}

			if resp.Summary.TotalPackages != tt.wantPackages {
				t.Errorf("Summary.TotalPackages = %d, want %d", resp.Summary.TotalPackages, tt.wantPackages)
			}

			if resp.Summary.UpdatablePackages != tt.wantUpdatable {
				t.Errorf("Summary.UpdatablePackages = %d, want %d", resp.Summary.UpdatablePackages, tt.wantUpdatable)
			}

			// Verify logger was called
			if len(logger.infoMessages) < 2 {
				t.Error("Expected at least 2 info log messages")
			}
		})
	}
}

func TestUseCase_GetStatus_SpecificManagers(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		requestIDs      []manager.ManagerID
		mockManagers    map[manager.ManagerID]*manager.Manager
		wantManagersLen int
		wantErr         bool
	}{
		{
			name:       "single manager by ID",
			requestIDs: []manager.ManagerID{manager.ManagerHomebrew},
			mockManagers: map[manager.ManagerID]*manager.Manager{
				manager.ManagerHomebrew: {
					ID:          manager.ManagerHomebrew,
					Name:        "Homebrew",
					Type:        manager.TypeSystem,
					Installed:   true,
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages:    []manager.Package{},
				},
			},
			wantManagersLen: 1,
			wantErr:         false,
		},
		{
			name: "multiple managers by ID",
			requestIDs: []manager.ManagerID{
				manager.ManagerHomebrew,
				manager.ManagerNPM,
			},
			mockManagers: map[manager.ManagerID]*manager.Manager{
				manager.ManagerHomebrew: {
					ID:          manager.ManagerHomebrew,
					Name:        "Homebrew",
					Type:        manager.TypeSystem,
					Installed:   true,
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages:    []manager.Package{},
				},
				manager.ManagerNPM: {
					ID:          manager.ManagerNPM,
					Name:        "NPM",
					Type:        manager.TypeLanguage,
					Installed:   false,
					Status:      manager.StatusUnavailable,
					LastChecked: now,
					Packages:    []manager.Package{},
				},
			},
			wantManagersLen: 2,
			wantErr:         false,
		},
		{
			name:            "manager not found",
			requestIDs:      []manager.ManagerID{manager.ManagerHomebrew},
			mockManagers:    map[manager.ManagerID]*manager.Manager{},
			wantManagersLen: 0,
			wantErr:         true,
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
			uc := NewUseCase(repo, logger)

			req := &dto.StatusRequest{
				ManagerIDs: tt.requestIDs,
				Verbose:    false,
				Refresh:    false,
			}

			resp, err := uc.GetStatus(context.Background(), req)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				// Verify error was logged
				if len(logger.errorMessages) == 0 {
					t.Error("Expected error to be logged")
				}
				return
			}

			if resp == nil {
				t.Fatal("GetStatus() returned nil response")
			}

			if len(resp.Managers) != tt.wantManagersLen {
				t.Errorf("GetStatus() returned %d managers, want %d", len(resp.Managers), tt.wantManagersLen)
			}
		})
	}
}

func TestUseCase_GetStatus_DTOMapping(t *testing.T) {
	now := time.Now()

	// Create a manager with known values
	inputManager := &manager.Manager{
		ID:          manager.ManagerHomebrew,
		Name:        "Homebrew",
		Type:        manager.TypeSystem,
		Platform:    manager.PlatformDarwin,
		Installed:   true,
		Version:     "4.2.1",
		Status:      manager.StatusHealthy,
		BinaryPath:  "/usr/local/bin/brew",
		ConfigPath:  "/usr/local",
		LastChecked: now,
		Packages: []manager.Package{
			{
				Name:             "git",
				CurrentVersion:   "2.43.0",
				AvailableVersion: "2.44.0",
				UpdateType:       manager.UpdateMinor,
			},
		},
	}

	repo := &mockRepository{
		findAllFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{inputManager}, nil
		},
	}
	logger := &mockLogger{}
	uc := NewUseCase(repo, logger)

	req := &dto.StatusRequest{Verbose: false}
	resp, err := uc.GetStatus(context.Background(), req)

	if err != nil {
		t.Fatalf("GetStatus() unexpected error: %v", err)
	}

	if len(resp.Managers) != 1 {
		t.Fatalf("Expected 1 manager, got %d", len(resp.Managers))
	}

	// Verify DTO mapping
	status := resp.Managers[0]

	if status.ID != inputManager.ID {
		t.Errorf("ID = %v, want %v", status.ID, inputManager.ID)
	}
	if status.Name != inputManager.Name {
		t.Errorf("Name = %v, want %v", status.Name, inputManager.Name)
	}
	if status.Type != inputManager.Type {
		t.Errorf("Type = %v, want %v", status.Type, inputManager.Type)
	}
	if status.Installed != inputManager.Installed {
		t.Errorf("Installed = %v, want %v", status.Installed, inputManager.Installed)
	}
	if status.Version != inputManager.Version {
		t.Errorf("Version = %v, want %v", status.Version, inputManager.Version)
	}
	if status.Status != inputManager.Status {
		t.Errorf("Status = %v, want %v", status.Status, inputManager.Status)
	}
	if status.BinaryPath != inputManager.BinaryPath {
		t.Errorf("BinaryPath = %v, want %v", status.BinaryPath, inputManager.BinaryPath)
	}
	if status.ConfigPath != inputManager.ConfigPath {
		t.Errorf("ConfigPath = %v, want %v", status.ConfigPath, inputManager.ConfigPath)
	}
	if status.PackageCount != 1 {
		t.Errorf("PackageCount = %v, want 1", status.PackageCount)
	}
	if status.UpdatableCount != 1 {
		t.Errorf("UpdatableCount = %v, want 1", status.UpdatableCount)
	}
}
