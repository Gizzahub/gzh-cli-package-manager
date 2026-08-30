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

const (
	testHomebrewName    = "Homebrew"
	testHomebrewVersion = "4.2.1"
	testGitName         = "git"
	testGitVersion      = "2.43.0"
	testNPMName         = "NPM"
	testNPMVersion      = "10.5.0"
	testReactName       = "react"
	testReactVersion    = "18.0.0"
	testVersion300      = "3.0.0"
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
			name:            "no managers",
			mockManagers:    []*manager.Manager{},
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
					Name:        testHomebrewName,
					Type:        manager.TypeSystem,
					Platform:    manager.PlatformDarwin,
					Installed:   true,
					Version:     testHomebrewVersion,
					Status:      manager.StatusHealthy,
					BinaryPath:  "/usr/local/bin/brew",
					ConfigPath:  "/usr/local",
					LastChecked: now,
					Packages: []manager.Package{
						{
							Name:             testGitName,
							CurrentVersion:   testGitVersion,
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
					Name:        testHomebrewName,
					Type:        manager.TypeSystem,
					Installed:   true,
					Version:     testHomebrewVersion,
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: testGitName, CurrentVersion: testGitVersion, UpdateType: manager.UpdateNone},
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
					Name:        testNPMName,
					Type:        manager.TypeLanguage,
					Installed:   true,
					Version:     testNPMVersion,
					Status:      manager.StatusDegraded,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: testReactName, CurrentVersion: testReactVersion, AvailableVersion: "18.2.0", UpdateType: manager.UpdateMinor},
						{Name: "vue", CurrentVersion: testVersion300, AvailableVersion: "3.4.0", UpdateType: manager.UpdateMinor},
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
					Name:        testHomebrewName,
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
					Name:        testHomebrewName,
					Type:        manager.TypeSystem,
					Installed:   true,
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages:    []manager.Package{},
				},
				manager.ManagerNPM: {
					ID:          manager.ManagerNPM,
					Name:        testNPMName,
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
		Name:        testHomebrewName,
		Type:        manager.TypeSystem,
		Platform:    manager.PlatformDarwin,
		Installed:   true,
		Version:     testHomebrewVersion,
		Status:      manager.StatusHealthy,
		BinaryPath:  "/usr/local/bin/brew",
		ConfigPath:  "/usr/local",
		LastChecked: now,
		Packages: []manager.Package{
			{
				Name:             testGitName,
				CurrentVersion:   testGitVersion,
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

func TestUseCase_GetStatus_VerboseMode(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		verbose         bool
		mockManagers    []*manager.Manager
		wantPackagesLen map[manager.ManagerID]int
	}{
		{
			name:    "verbose mode populates packages",
			verbose: true,
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerNPM,
					Name:        testNPMName,
					Type:        manager.TypeLanguage,
					Installed:   true,
					Version:     testNPMVersion,
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages: []manager.Package{
						{
							Name:             testReactName,
							CurrentVersion:   testReactVersion,
							AvailableVersion: "18.2.0",
							UpdateType:       manager.UpdateMinor,
							Description:      "React library",
						},
						{
							Name:           "vue",
							CurrentVersion: testVersion300,
							UpdateType:     manager.UpdateNone,
							Description:    "Vue framework",
						},
					},
				},
			},
			wantPackagesLen: map[manager.ManagerID]int{
				manager.ManagerNPM: 2,
			},
		},
		{
			name:    "non-verbose mode does not populate packages",
			verbose: false,
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerNPM,
					Name:        testNPMName,
					Type:        manager.TypeLanguage,
					Installed:   true,
					Version:     testNPMVersion,
					Status:      manager.StatusHealthy,
					LastChecked: now,
					Packages: []manager.Package{
						{
							Name:           testReactName,
							CurrentVersion: testReactVersion,
							UpdateType:     manager.UpdateNone,
						},
					},
				},
			},
			wantPackagesLen: map[manager.ManagerID]int{
				manager.ManagerNPM: 0, // Should be empty in non-verbose mode
			},
		},
		{
			name:    "verbose mode only populates installed managers",
			verbose: true,
			mockManagers: []*manager.Manager{
				{
					ID:          manager.ManagerNPM,
					Name:        testNPMName,
					Installed:   true,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: "pkg1", CurrentVersion: "1.0.0"},
					},
				},
				{
					ID:          manager.ManagerPip,
					Name:        "Pip",
					Installed:   false, // Not installed
					LastChecked: now,
					Packages:    []manager.Package{}, // Should not be populated even if present
				},
			},
			wantPackagesLen: map[manager.ManagerID]int{
				manager.ManagerNPM: 1,
				manager.ManagerPip: 0, // Not installed, so no packages
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{
				findAllFunc: func(_ context.Context) ([]*manager.Manager, error) {
					return tt.mockManagers, nil
				},
			}
			logger := &mockLogger{}
			uc := NewUseCase(repo, logger)

			req := &dto.StatusRequest{
				Verbose: tt.verbose,
			}

			resp, err := uc.GetStatus(context.Background(), req)
			if err != nil {
				t.Fatalf("GetStatus() unexpected error: %v", err)
			}

			// Verify package population for each manager
			for _, mgrStatus := range resp.Managers {
				expectedLen, exists := tt.wantPackagesLen[mgrStatus.ID]
				if !exists {
					continue
				}

				if len(mgrStatus.Packages) != expectedLen {
					t.Errorf("Manager %s: got %d packages, want %d",
						mgrStatus.ID, len(mgrStatus.Packages), expectedLen)
				}
			}
		})
	}
}

func TestUseCase_GetStatus_VerbosePackageMapping(t *testing.T) {
	now := time.Now()

	inputPackages := []manager.Package{
		{
			Name:             "typescript",
			CurrentVersion:   "5.0.0",
			AvailableVersion: "5.3.0",
			UpdateType:       manager.UpdateMinor,
			Description:      "TypeScript compiler",
		},
		{
			Name:             "eslint",
			CurrentVersion:   "8.0.0",
			AvailableVersion: "9.0.0",
			UpdateType:       manager.UpdateMajor,
			Description:      "Linting tool",
		},
	}

	repo := &mockRepository{
		findAllFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:          manager.ManagerNPM,
					Name:        testNPMName,
					Installed:   true,
					LastChecked: now,
					Packages:    inputPackages,
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	uc := NewUseCase(repo, logger)

	req := &dto.StatusRequest{Verbose: true}
	resp, err := uc.GetStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("GetStatus() unexpected error: %v", err)
	}

	if len(resp.Managers) != 1 {
		t.Fatalf("Expected 1 manager, got %d", len(resp.Managers))
	}

	mgrStatus := resp.Managers[0]
	if len(mgrStatus.Packages) != 2 {
		t.Fatalf("Expected 2 packages, got %d", len(mgrStatus.Packages))
	}

	// Verify first package mapping
	pkg1 := mgrStatus.Packages[0]
	if pkg1.Name != "typescript" {
		t.Errorf("Package[0].Name = %v, want typescript", pkg1.Name)
	}
	if pkg1.CurrentVersion != "5.0.0" {
		t.Errorf("Package[0].CurrentVersion = %v, want 5.0.0", pkg1.CurrentVersion)
	}
	if pkg1.AvailableVersion != "5.3.0" {
		t.Errorf("Package[0].AvailableVersion = %v, want 5.3.0", pkg1.AvailableVersion)
	}
	if pkg1.UpdateType != manager.UpdateMinor {
		t.Errorf("Package[0].UpdateType = %v, want %v", pkg1.UpdateType, manager.UpdateMinor)
	}
	if pkg1.Description != "TypeScript compiler" {
		t.Errorf("Package[0].Description = %v, want TypeScript compiler", pkg1.Description)
	}

	// Verify second package mapping
	pkg2 := mgrStatus.Packages[1]
	if pkg2.Name != "eslint" {
		t.Errorf("Package[1].Name = %v, want eslint", pkg2.Name)
	}
	if pkg2.UpdateType != manager.UpdateMajor {
		t.Errorf("Package[1].UpdateType = %v, want %v", pkg2.UpdateType, manager.UpdateMajor)
	}
}

func TestUseCase_GetStatus_VerboseWithMultipleManagers(t *testing.T) {
	now := time.Now()

	repo := &mockRepository{
		findAllFunc: func(_ context.Context) ([]*manager.Manager, error) {
			return []*manager.Manager{
				{
					ID:          manager.ManagerNPM,
					Name:        testNPMName,
					Installed:   true,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: "npm-pkg1", CurrentVersion: "1.0.0"},
						{Name: "npm-pkg2", CurrentVersion: "2.0.0"},
					},
				},
				{
					ID:          manager.ManagerPip,
					Name:        "Pip",
					Installed:   true,
					LastChecked: now,
					Packages: []manager.Package{
						{Name: "pip-pkg1", CurrentVersion: testVersion300},
					},
				},
				{
					ID:          manager.ManagerHomebrew,
					Name:        testHomebrewName,
					Installed:   false, // Not installed
					LastChecked: now,
					Packages:    []manager.Package{},
				},
			}, nil
		},
	}
	logger := &mockLogger{}
	uc := NewUseCase(repo, logger)

	req := &dto.StatusRequest{Verbose: true}
	resp, err := uc.GetStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("GetStatus() unexpected error: %v", err)
	}

	if len(resp.Managers) != 3 {
		t.Fatalf("Expected 3 managers, got %d", len(resp.Managers))
	}

	// Verify NPM has packages
	npmMgr := findManagerByID(resp.Managers, manager.ManagerNPM)
	if npmMgr == nil {
		t.Fatal("NPM manager not found")
	}
	if len(npmMgr.Packages) != 2 {
		t.Errorf("NPM: expected 2 packages, got %d", len(npmMgr.Packages))
	}

	// Verify Pip has packages
	pipMgr := findManagerByID(resp.Managers, manager.ManagerPip)
	if pipMgr == nil {
		t.Fatal("Pip manager not found")
	}
	if len(pipMgr.Packages) != 1 {
		t.Errorf("Pip: expected 1 package, got %d", len(pipMgr.Packages))
	}

	// Verify Homebrew has no packages (not installed)
	brewMgr := findManagerByID(resp.Managers, manager.ManagerHomebrew)
	if brewMgr == nil {
		t.Fatal("Homebrew manager not found")
	}
	if len(brewMgr.Packages) != 0 {
		t.Errorf("Homebrew: expected 0 packages (not installed), got %d", len(brewMgr.Packages))
	}
}

// Helper function to find manager by ID in response.
func findManagerByID(managers []*dto.ManagerStatus, id manager.ManagerID) *dto.ManagerStatus {
	for _, mgr := range managers {
		if mgr.ID == id {
			return mgr
		}
	}
	return nil
}
