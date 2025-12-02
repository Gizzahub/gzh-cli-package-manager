// Package detector provides environment detection capabilities.
// It detects special environments like conda, virtualenv, and Docker containers
// to enable appropriate handling during package manager operations.
package detector

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
)

// EnvironmentType represents the type of detected environment.
type EnvironmentType string

const (
	// EnvNormal represents a normal environment without special handling.
	EnvNormal EnvironmentType = "normal"

	// EnvConda represents a conda/mamba environment.
	EnvConda EnvironmentType = "conda"

	// EnvVirtualenv represents a Python virtualenv.
	EnvVirtualenv EnvironmentType = "virtualenv"

	// EnvDocker represents execution inside a Docker container.
	EnvDocker EnvironmentType = "docker"

	// EnvWSL represents Windows Subsystem for Linux.
	EnvWSL EnvironmentType = "wsl"
)

// Environment contains information about the detected environment.
type Environment struct {
	// Type is the primary environment type.
	Type EnvironmentType

	// Name is the name of the environment (e.g., conda env name).
	Name string

	// Path is the path to the environment (e.g., virtualenv path).
	Path string

	// IsPipSafe indicates whether pip operations are safe.
	// For example, pip in conda environments may cause conflicts.
	IsPipSafe bool

	// Warnings contains any warnings about the environment.
	Warnings []string
}

// Detector detects the current runtime environment.
type Detector struct {
	executor output.CommandExecutor
	logger   output.Logger
}

// NewDetector creates a new environment detector.
func NewDetector(executor output.CommandExecutor, logger output.Logger) *Detector {
	return &Detector{
		executor: executor,
		logger:   logger,
	}
}

// Detect detects the current environment.
func (d *Detector) Detect(ctx context.Context) *Environment {
	env := &Environment{
		Type:      EnvNormal,
		IsPipSafe: true,
		Warnings:  []string{},
	}

	// Check for Docker first (most isolated environment)
	if d.isDocker() {
		env.Type = EnvDocker
		d.logger.Debug(ctx, "Detected Docker environment")
	}

	// Check for WSL
	if d.isWSL() {
		env.Type = EnvWSL
		d.logger.Debug(ctx, "Detected WSL environment")
	}

	// Check for Conda (takes precedence for Python environments)
	if condaEnv := d.detectConda(); condaEnv != nil {
		env.Type = EnvConda
		env.Name = condaEnv.Name
		env.Path = condaEnv.Path
		env.IsPipSafe = false
		env.Warnings = append(env.Warnings, "Conda environment detected. Using pip may cause dependency conflicts with conda packages.")
		d.logger.Debug(ctx, "Detected Conda environment",
			output.Field{Key: "name", Value: condaEnv.Name},
			output.Field{Key: "path", Value: condaEnv.Path},
		)
	}

	// Check for Virtualenv (if not in conda)
	if env.Type != EnvConda {
		if venv := d.detectVirtualenv(); venv != nil {
			env.Type = EnvVirtualenv
			env.Name = venv.Name
			env.Path = venv.Path
			env.IsPipSafe = true // pip is safe in virtualenv
			d.logger.Debug(ctx, "Detected virtualenv environment",
				output.Field{Key: "name", Value: venv.Name},
				output.Field{Key: "path", Value: venv.Path},
			)
		}
	}

	return env
}

// condaEnv holds conda environment info.
type condaEnv struct {
	Name string
	Path string
}

// detectConda checks for active conda/mamba environment.
func (d *Detector) detectConda() *condaEnv {
	// Check CONDA_DEFAULT_ENV (name of active conda environment)
	condaName := os.Getenv("CONDA_DEFAULT_ENV")
	if condaName == "" {
		return nil
	}

	// Check CONDA_PREFIX (path to active conda environment)
	condaPath := os.Getenv("CONDA_PREFIX")
	if condaPath == "" {
		// Try to derive from CONDA_DEFAULT_ENV
		condaBase := os.Getenv("CONDA_EXE")
		if condaBase != "" {
			condaPath = filepath.Join(filepath.Dir(filepath.Dir(condaBase)), "envs", condaName)
		}
	}

	return &condaEnv{
		Name: condaName,
		Path: condaPath,
	}
}

// venvInfo holds virtualenv info.
type venvInfo struct {
	Name string
	Path string
}

// detectVirtualenv checks for active Python virtualenv.
func (d *Detector) detectVirtualenv() *venvInfo {
	// Check VIRTUAL_ENV environment variable
	venvPath := os.Getenv("VIRTUAL_ENV")
	if venvPath == "" {
		return nil
	}

	// Extract name from path
	venvName := filepath.Base(venvPath)

	return &venvInfo{
		Name: venvName,
		Path: venvPath,
	}
}

// isDocker checks if running inside a Docker container.
func (d *Detector) isDocker() bool {
	// Check for /.dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker
	cgroupFile := "/proc/1/cgroup"
	if data, err := os.ReadFile(cgroupFile); err == nil {
		if strings.Contains(string(data), "docker") ||
			strings.Contains(string(data), "containerd") ||
			strings.Contains(string(data), "kubepods") {
			return true
		}
	}

	// Check for container environment variable
	if os.Getenv("container") != "" {
		return true
	}

	return false
}

// isWSL checks if running inside Windows Subsystem for Linux.
func (d *Detector) isWSL() bool {
	// Check for WSL-specific environment variable
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}

	// Check /proc/version for WSL string
	versionFile := "/proc/version"
	if data, err := os.ReadFile(versionFile); err == nil {
		lowered := strings.ToLower(string(data))
		if strings.Contains(lowered, "microsoft") || strings.Contains(lowered, "wsl") {
			return true
		}
	}

	return false
}

// IsPipUpdateSafe returns true if pip updates are safe in the current environment.
func (d *Detector) IsPipUpdateSafe(ctx context.Context) bool {
	env := d.Detect(ctx)
	return env.IsPipSafe
}

// GetEnvironmentWarnings returns any warnings about the current environment.
func (d *Detector) GetEnvironmentWarnings(ctx context.Context) []string {
	env := d.Detect(ctx)
	return env.Warnings
}
