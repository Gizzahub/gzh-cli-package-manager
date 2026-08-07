package cleanup

import (
	"path/filepath"
	"runtime"
)

// KnownCachePath describes a well-known package-manager cache location.
type KnownCachePath struct {
	// ManagerID is the package manager identifier (e.g. "homebrew", "npm").
	ManagerID string
	// RelPath is the path relative to the home directory, using forward slashes.
	// Empty when AbsEnv expands to an absolute path from an environment variable.
	RelPath string
	// AbsEnv, if set, is an environment variable that holds an absolute cache path
	// (e.g. "npm_config_cache"). When present and non-empty it takes precedence.
	AbsEnv string
	// GOOS limits this entry to a specific OS ("darwin", "linux", "windows").
	// Empty means all platforms.
	GOOS string
}

// DefaultKnownCachePaths returns the built-in list of known cache locations.
func DefaultKnownCachePaths() []KnownCachePath {
	return []KnownCachePath{
		// Homebrew
		{ManagerID: "homebrew", RelPath: "Library/Caches/Homebrew", GOOS: "darwin"},
		{ManagerID: "homebrew", RelPath: ".cache/Homebrew", GOOS: "linux"},
		// npm
		{ManagerID: "npm", RelPath: ".npm", AbsEnv: "npm_config_cache"},
		// pip
		{ManagerID: "pip", RelPath: ".cache/pip"},
		// cargo
		{ManagerID: "cargo", RelPath: ".cargo/registry/cache"},
		// yarn
		{ManagerID: "yarn", RelPath: ".cache/yarn"},
		// go module download cache (under GOPATH or default ~/go)
		{ManagerID: "go", RelPath: "go/pkg/mod/cache/download"},
		// pnpm store (common default)
		{ManagerID: "pnpm", RelPath: "Library/pnpm/store", GOOS: "darwin"},
		{ManagerID: "pnpm", RelPath: ".local/share/pnpm/store", GOOS: "linux"},
	}
}

// ResolveCachePaths expands known cache paths for the current (or provided) home/OS.
// envLookup returns environment variable values (injectable; use os.Getenv in prod).
// goos defaults to runtime.GOOS when empty.
func ResolveCachePaths(home string, goos string, envLookup func(string) string, known []KnownCachePath) map[string]string {
	if goos == "" {
		goos = runtime.GOOS
	}
	if known == nil {
		known = DefaultKnownCachePaths()
	}
	if envLookup == nil {
		envLookup = func(string) string { return "" }
	}

	// managerID -> absolute path (first match wins)
	out := make(map[string]string)
	for _, k := range known {
		if k.GOOS != "" && k.GOOS != goos {
			continue
		}
		if _, exists := out[k.ManagerID]; exists {
			continue
		}
		if k.AbsEnv != "" {
			if v := envLookup(k.AbsEnv); v != "" {
				out[k.ManagerID] = v
				continue
			}
		}
		if k.RelPath == "" || home == "" {
			continue
		}
		out[k.ManagerID] = filepath.Join(home, filepath.FromSlash(k.RelPath))
	}
	return out
}
