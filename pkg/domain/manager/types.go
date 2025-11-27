package manager

// ManagerID is a unique identifier for a package manager.
type ManagerID string

// Common package manager identifiers.
const (
	ManagerHomebrew ManagerID = "brew"
	ManagerASDF     ManagerID = "asdf"
	ManagerNPM      ManagerID = "npm"
	ManagerPip      ManagerID = "pip"
	ManagerCargo    ManagerID = "cargo"
	ManagerSDKMan   ManagerID = "sdkman"
	ManagerApt      ManagerID = "apt"
	ManagerPacman   ManagerID = "pacman"
	ManagerYay      ManagerID = "yay"
)

// ManagerType represents the category of package manager.
type ManagerType string

const (
	// TypeSystem manages system-level packages (brew, apt, pacman).
	TypeSystem ManagerType = "system"

	// TypeVersion manages multiple versions of tools (asdf, nvm, rbenv).
	TypeVersion ManagerType = "version"

	// TypeLanguage manages language-specific packages (npm, pip, cargo, gem).
	TypeLanguage ManagerType = "language"
)

// Platform represents the operating system platform.
type Platform string

// Supported platforms.
const (
	PlatformDarwin  Platform = "darwin"  // macOS
	PlatformLinux   Platform = "linux"   // Linux
	PlatformWindows Platform = "windows" // Windows
)

// Status represents the health status of a package manager.
type Status string

// Health status values.
const (
	StatusHealthy     Status = "healthy"      // Manager is installed and working
	StatusDegraded    Status = "degraded"     // Manager works but has issues
	StatusUnavailable Status = "unavailable"  // Manager is not installed
	StatusError       Status = "error"        // Manager has errors
)
