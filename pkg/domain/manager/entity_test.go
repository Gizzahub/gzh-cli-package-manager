package manager

import (
	"testing"
	"time"
)

const (
	fixtureVersion100 = "1.0.0"
	fixtureVersion200 = "2.0.0"
	fixtureVersion101 = "1.0.1"
)

func TestManager_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "healthy manager",
			status: StatusHealthy,
			want:   true,
		},
		{
			name:   "degraded manager",
			status: StatusDegraded,
			want:   false,
		},
		{
			name:   "unavailable manager",
			status: StatusUnavailable,
			want:   false,
		},
		{
			name:   "error manager",
			status: StatusError,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Manager{Status: tt.status}
			if got := m.IsHealthy(); got != tt.want {
				t.Errorf("Manager.IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_PackageCount(t *testing.T) {
	tests := []struct {
		name     string
		packages []Package
		want     int
	}{
		{
			name:     "no packages",
			packages: []Package{},
			want:     0,
		},
		{
			name: "single package",
			packages: []Package{
				{Name: "package1"},
			},
			want: 1,
		},
		{
			name: "multiple packages",
			packages: []Package{
				{Name: "package1"},
				{Name: "package2"},
				{Name: "package3"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Manager{Packages: tt.packages}
			if got := m.PackageCount(); got != tt.want {
				t.Errorf("Manager.PackageCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_UpdatableCount(t *testing.T) {
	tests := []struct {
		name     string
		packages []Package
		want     int
	}{
		{
			name:     "no packages",
			packages: []Package{},
			want:     0,
		},
		{
			name: "no updates available",
			packages: []Package{
				{Name: "pkg1", CurrentVersion: fixtureVersion100, AvailableVersion: fixtureVersion100, UpdateType: UpdateNone},
				{Name: "pkg2", CurrentVersion: fixtureVersion200, AvailableVersion: fixtureVersion200, UpdateType: UpdateNone},
			},
			want: 0,
		},
		{
			name: "all packages have updates",
			packages: []Package{
				{Name: "pkg1", CurrentVersion: fixtureVersion100, AvailableVersion: fixtureVersion101, UpdateType: UpdatePatch},
				{Name: "pkg2", CurrentVersion: fixtureVersion100, AvailableVersion: "1.1.0", UpdateType: UpdateMinor},
			},
			want: 2,
		},
		{
			name: "mixed update availability",
			packages: []Package{
				{Name: "pkg1", CurrentVersion: fixtureVersion100, AvailableVersion: fixtureVersion101, UpdateType: UpdatePatch},
				{Name: "pkg2", CurrentVersion: fixtureVersion200, AvailableVersion: fixtureVersion200, UpdateType: UpdateNone},
				{Name: "pkg3", CurrentVersion: fixtureVersion100, AvailableVersion: fixtureVersion200, UpdateType: UpdateMajor},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Manager{Packages: tt.packages}
			if got := m.UpdatableCount(); got != tt.want {
				t.Errorf("Manager.UpdatableCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPackage_IsUpdateAvailable(t *testing.T) {
	tests := []struct {
		name             string
		currentVersion   string
		availableVersion string
		updateType       UpdateType
		want             bool
	}{
		{
			name:             "no update - same version",
			currentVersion:   fixtureVersion100,
			availableVersion: fixtureVersion100,
			updateType:       UpdateNone,
			want:             false,
		},
		{
			name:             "patch update available",
			currentVersion:   fixtureVersion100,
			availableVersion: fixtureVersion101,
			updateType:       UpdatePatch,
			want:             true,
		},
		{
			name:             "minor update available",
			currentVersion:   fixtureVersion100,
			availableVersion: "1.1.0",
			updateType:       UpdateMinor,
			want:             true,
		},
		{
			name:             "major update available",
			currentVersion:   fixtureVersion100,
			availableVersion: fixtureVersion200,
			updateType:       UpdateMajor,
			want:             true,
		},
		{
			name:             "no update - empty available version",
			currentVersion:   fixtureVersion100,
			availableVersion: "",
			updateType:       UpdateNone,
			want:             false,
		},
		{
			name:             "update type set but same version",
			currentVersion:   fixtureVersion100,
			availableVersion: fixtureVersion100,
			updateType:       UpdatePatch,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Package{
				CurrentVersion:   tt.currentVersion,
				AvailableVersion: tt.availableVersion,
				UpdateType:       tt.updateType,
			}
			if got := p.IsUpdateAvailable(); got != tt.want {
				t.Errorf("Package.IsUpdateAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_CompleteEntity(t *testing.T) {
	// Test that we can create a complete Manager entity with all fields
	now := time.Now()
	m := Manager{
		ID:          ManagerHomebrew,
		Name:        "Homebrew",
		Type:        TypeSystem,
		Platform:    PlatformDarwin,
		Installed:   true,
		Version:     "4.0.0",
		Status:      StatusHealthy,
		ConfigPath:  "/opt/homebrew/.homebrew",
		BinaryPath:  "/opt/homebrew/bin/brew",
		LastChecked: now,
		Packages: []Package{
			{
				Name:             "git",
				CurrentVersion:   "2.42.0",
				AvailableVersion: "2.43.0",
				Manager:          ManagerHomebrew,
				Description:      "Distributed version control system",
				SizeMB:           45.2,
				UpdateType:       UpdateMinor,
				IsGlobal:         true,
			},
		},
	}

	// Verify all fields are set
	if m.ID != ManagerHomebrew {
		t.Errorf("ID = %v, want %v", m.ID, ManagerHomebrew)
	}
	if m.PackageCount() != 1 {
		t.Errorf("PackageCount() = %v, want 1", m.PackageCount())
	}
	if m.UpdatableCount() != 1 {
		t.Errorf("UpdatableCount() = %v, want 1", m.UpdatableCount())
	}
	if !m.IsHealthy() {
		t.Error("IsHealthy() = false, want true")
	}
}

func TestPackage_CompleteEntity(t *testing.T) {
	// Test that we can create a complete Package entity with all fields
	p := Package{
		Name:             "typescript",
		CurrentVersion:   "5.3.2",
		AvailableVersion: "5.3.3",
		Manager:          ManagerNPM,
		Description:      "TypeScript language compiler",
		SizeMB:           32.5,
		UpdateType:       UpdatePatch,
		IsGlobal:         true,
	}

	// Verify all fields are accessible
	if p.Name != "typescript" {
		t.Errorf("Name = %v, want typescript", p.Name)
	}
	if !p.IsUpdateAvailable() {
		t.Error("IsUpdateAvailable() = false, want true")
	}
	if p.Manager != ManagerNPM {
		t.Errorf("Manager = %v, want %v", p.Manager, ManagerNPM)
	}
}
