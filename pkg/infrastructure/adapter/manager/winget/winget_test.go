package winget

import (
	"context"
	"errors"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     bool
		wantErr  bool
	}{
		{
			name: "winget installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand && len(args) == 1 && args[0] == "--version" {
					return testutil.SuccessResult("v1.6.3482\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "winget not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand && len(args) == 1 && args[0] == "--version" {
					return testutil.FailureResult(1, "winget: command not found"), errors.New("exit code 1")
				}
				return nil, errors.New("unexpected command")
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.Detect(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetVersion(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "version with v prefix",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand && len(args) == 1 && args[0] == "--version" {
					return testutil.SuccessResult("v1.6.3482\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "1.6.3482",
			wantErr: false,
		},
		{
			name: "version without v prefix",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand && len(args) == 1 && args[0] == "--version" {
					return testutil.SuccessResult("1.5.1234\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "1.5.1234",
			wantErr: false,
		},
		{
			name: "command fails",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.GetVersion(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetBinaryPath(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     string
		wantErr  bool
	}{
		{
			name: "single path returned",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "where" && len(args) == 1 && args[0] == wingetCommand {
					return testutil.SuccessResult("C:\\Users\\user\\AppData\\Local\\Microsoft\\WindowsApps\\winget.exe\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "C:\\Users\\user\\AppData\\Local\\Microsoft\\WindowsApps\\winget.exe",
			wantErr: false,
		},
		{
			name: "multiple paths returned",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "where" && len(args) == 1 && args[0] == wingetCommand {
					return testutil.SuccessResult("C:\\First\\winget.exe\nC:\\Second\\winget.exe\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "C:\\First\\winget.exe",
			wantErr: false,
		},
		{
			name: "where fails",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, "not found"), errors.New("not found")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.GetBinaryPath(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBinaryPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBinaryPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_GetConfigPath(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())

	got, err := adapter.GetConfigPath(context.Background())
	if err != nil {
		t.Errorf("GetConfigPath() error = %v", err)
		return
	}
	if got != defaultConfigPath {
		t.Errorf("GetConfigPath() = %v, want %v", got, defaultConfigPath)
	}
}

func TestAdapter_ListPackages_JSON(t *testing.T) {
	jsonOutput := `{
		"Sources": [{
			"Packages": [
				{"Id": "Git.Git", "Name": "Git", "Version": "2.43.0", "AvailableVersion": "2.44.0", "Source": "winget"},
				{"Id": "Microsoft.VSCode", "Name": "Visual Studio Code", "Version": "1.85.0", "Source": "winget"}
			]
		}]
	}`

	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == wingetCommand {
			return testutil.SuccessResult(jsonOutput), nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil {
		t.Errorf("ListPackages() error = %v", err)
		return
	}

	if len(packages) != 2 {
		t.Errorf("ListPackages() returned %d packages, want 2", len(packages))
		return
	}

	// Check first package has update available
	if packages[0].Name != "Git" {
		t.Errorf("First package name = %v, want Git", packages[0].Name)
	}
	if packages[0].UpdateType != manager.UpdateMinor {
		t.Errorf("First package UpdateType = %v, want UpdateMinor", packages[0].UpdateType)
	}

	// Check second package has no update
	if packages[1].Name != "Visual Studio Code" {
		t.Errorf("Second package name = %v, want Visual Studio Code", packages[1].Name)
	}
	if packages[1].UpdateType != manager.UpdateNone {
		t.Errorf("Second package UpdateType = %v, want UpdateNone", packages[1].UpdateType)
	}
}

func TestAdapter_ListPackages_TextFallback(t *testing.T) {
	textOutput := `Name                Id                   Version   Available Source
---------------------------------------------------------------------------
Git                 Git.Git              2.43.0    2.44.0    winget
Visual Studio Code  Microsoft.VSCode     1.85.0              winget
`

	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == wingetCommand {
			return testutil.SuccessResult(textOutput), nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil {
		t.Errorf("ListPackages() error = %v", err)
		return
	}

	// Should parse at least some packages from text output
	if len(packages) == 0 {
		t.Error("ListPackages() returned 0 packages from text output")
	}
}

func TestAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     manager.Status
	}{
		{
			name: "healthy",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand && args[0] == "source" && args[1] == "list" {
					return testutil.SuccessResult("Name    Argument\n-----------------\nwinget  https://...\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: manager.StatusHealthy,
		},
		{
			name: "error",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.FailureResult(1, "source error"), nil
			},
			want: manager.StatusError,
		},
		{
			name: "command fails",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("network error")
			},
			want: manager.StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.CheckHealth(context.Background())
			if err != nil {
				t.Errorf("CheckHealth() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("CheckHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdapter_Update(t *testing.T) {
	tests := []struct {
		name     string
		opts     adapterm.UpdateOptions
		execFunc testutil.ExecutorFunc
		want     *adapterm.UpdateResult
		wantErr  bool
	}{
		{
			name: "dry run",
			opts: adapterm.UpdateOptions{DryRun: true},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "Dry-run: would upgrade all winget packages",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
			wantErr: false,
		},
		{
			name: "fixed strategy",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyFixed},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "Strategy 'fixed': winget update skipped",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
			wantErr: false,
		},
		{
			name: "successful upgrade",
			opts: adapterm.UpdateOptions{},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand {
					return testutil.SuccessResult("Found Git [Git.Git]\nSuccessfully installed Git\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "1 packages updated successfully",
				UpdatedPackages: []string{"Git"},
				FailedPackages:  []string{},
			},
			wantErr: false,
		},
		{
			name: "no packages to upgrade",
			opts: adapterm.UpdateOptions{},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "0x8A150010 - No applicable upgrade found",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want: &adapterm.UpdateResult{
				Success:         true,
				Message:         "No packages to upgrade",
				UpdatedPackages: []string{},
				FailedPackages:  []string{},
			},
			wantErr: false,
		},
		{
			name: "upgrade fails",
			opts: adapterm.UpdateOptions{},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("network error")
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			got, err := adapter.Update(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				return
			}

			if got.Success != tt.want.Success {
				t.Errorf("Update() Success = %v, want %v", got.Success, tt.want.Success)
			}
			if got.Message != tt.want.Message {
				t.Errorf("Update() Message = %v, want %v", got.Message, tt.want.Message)
			}
			if len(got.UpdatedPackages) != len(tt.want.UpdatedPackages) {
				t.Errorf("Update() UpdatedPackages len = %v, want %v", len(got.UpdatedPackages), len(tt.want.UpdatedPackages))
			}
		})
	}
}

func TestNewAdapter(t *testing.T) {
	executor := testutil.NewMockExecutor(nil)
	logger := testutil.NewMockLogger()

	adapter := NewAdapter(executor, logger)
	if adapter == nil {
		t.Error("NewAdapter() returned nil")
	}
	if adapter.executor == nil {
		t.Error("NewAdapter() executor is nil")
	}
	if adapter.logger == nil {
		t.Error("NewAdapter() logger is nil")
	}
}

func TestAdapter_Search(t *testing.T) {
	textOutput := `Name                Id                   Version  Match     Source
---------------------------------------------------------------------------
Git                 Git.Git              2.43.0             winget
GitHub CLI          GitHub.cli           2.40.0             winget
`

	tests := []struct {
		name      string
		query     string
		execFunc  testutil.ExecutorFunc
		wantCount int
		wantErr   bool
		wantFirst string
	}{
		{
			name:  "text search results",
			query: "git",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == wingetCommand && len(args) >= 2 && args[0] == "search" && args[1] == "git" {
					return testutil.SuccessResult(textOutput), nil
				}
				return nil, errors.New("unexpected command")
			},
			wantCount: 2,
			wantFirst: "Git",
		},
		{
			name:  "empty query",
			query: "  ",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				t.Fatal("executor should not be called for empty query")
				return nil, nil
			},
			wantErr: true,
		},
		{
			name:  "command fails",
			query: "git",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("search failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())
			packages, err := adapter.Search(context.Background(), tt.query)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(packages) != tt.wantCount {
				t.Fatalf("Search() count = %d, want %d", len(packages), tt.wantCount)
			}
			if tt.wantFirst != "" && packages[0].Name != tt.wantFirst {
				t.Errorf("Search() first = %q, want %q", packages[0].Name, tt.wantFirst)
			}
		})
	}
}
