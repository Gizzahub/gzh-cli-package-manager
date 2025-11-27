package manager

import "testing"

func TestManagerID_Constants(t *testing.T) {
	// Test that all manager IDs are defined and unique
	ids := []ManagerID{
		ManagerHomebrew,
		ManagerASDF,
		ManagerNPM,
		ManagerPip,
		ManagerCargo,
		ManagerSDKMan,
		ManagerApt,
		ManagerPacman,
		ManagerYay,
	}

	// Check uniqueness
	seen := make(map[ManagerID]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ManagerID found: %s", id)
		}
		seen[id] = true
	}

	// Check expected values
	tests := []struct {
		id   ManagerID
		want string
	}{
		{ManagerHomebrew, "brew"},
		{ManagerASDF, "asdf"},
		{ManagerNPM, "npm"},
		{ManagerPip, "pip"},
		{ManagerCargo, "cargo"},
		{ManagerSDKMan, "sdkman"},
		{ManagerApt, "apt"},
		{ManagerPacman, "pacman"},
		{ManagerYay, "yay"},
	}

	for _, tt := range tests {
		if string(tt.id) != tt.want {
			t.Errorf("ManagerID %v = %q, want %q", tt.id, tt.id, tt.want)
		}
	}
}

func TestManagerType_Constants(t *testing.T) {
	tests := []struct {
		managerType ManagerType
		want        string
	}{
		{TypeSystem, "system"},
		{TypeVersion, "version"},
		{TypeLanguage, "language"},
	}

	for _, tt := range tests {
		if string(tt.managerType) != tt.want {
			t.Errorf("ManagerType %v = %q, want %q", tt.managerType, tt.managerType, tt.want)
		}
	}
}

func TestPlatform_Constants(t *testing.T) {
	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformDarwin, "darwin"},
		{PlatformLinux, "linux"},
		{PlatformWindows, "windows"},
	}

	for _, tt := range tests {
		if string(tt.platform) != tt.want {
			t.Errorf("Platform %v = %q, want %q", tt.platform, tt.platform, tt.want)
		}
	}
}

func TestStatus_Constants(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusHealthy, "healthy"},
		{StatusDegraded, "degraded"},
		{StatusUnavailable, "unavailable"},
		{StatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("Status %v = %q, want %q", tt.status, tt.status, tt.want)
		}
	}
}

func TestManagerID_Coverage(t *testing.T) {
	// Ensure we support the major package managers mentioned in PRD
	requiredManagers := []ManagerID{
		ManagerHomebrew, // macOS/Linux system packages
		ManagerASDF,     // Version manager
		ManagerNPM,      // Node.js packages
		ManagerPip,      // Python packages
		ManagerApt,      // Debian/Ubuntu system packages
	}

	for _, id := range requiredManagers {
		if id == "" {
			t.Errorf("Required manager ID is empty")
		}
	}
}

func TestUpdateType_Constants(t *testing.T) {
	tests := []struct {
		updateType UpdateType
		want       string
	}{
		{UpdateNone, "none"},
		{UpdatePatch, "patch"},
		{UpdateMinor, "minor"},
		{UpdateMajor, "major"},
	}

	for _, tt := range tests {
		if string(tt.updateType) != tt.want {
			t.Errorf("UpdateType %v = %q, want %q", tt.updateType, tt.updateType, tt.want)
		}
	}
}
