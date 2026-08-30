// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package scoop

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	testScoopStatusCommand = "status"
	testScoopVersionFlag   = "--version"
)

func TestNewAdapter(t *testing.T) {
	executor := testutil.NewMockExecutor(nil)
	logger := testutil.NewMockLogger()
	adapter := NewAdapter(executor, logger)

	if adapter == nil {
		t.Fatal("NewAdapter() returned nil")
	}
	if adapter.executor == nil {
		t.Error("executor should not be nil")
	}
	if adapter.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc testutil.ExecutorFunc
		want     bool
		wantErr  bool
	}{
		{
			name: "scoop installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) == 1 && args[0] == testScoopVersionFlag {
					return testutil.SuccessResult("Current Scoop version:\nv0.3.1\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "scoop not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) == 1 && args[0] == testScoopVersionFlag {
					return testutil.FailureResult(1, "scoop: command not found"), errors.New("command not found")
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
			name: "version with prefix",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) == 1 && args[0] == testScoopVersionFlag {
					return testutil.SuccessResult("Current Scoop version:\nv0.3.1\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "0.3.1",
			wantErr: false,
		},
		{
			name: "simple version",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) == 1 && args[0] == testScoopVersionFlag {
					return testutil.SuccessResult("v0.4.0\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "0.4.0",
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
				if command == "where" && len(args) == 1 && args[0] == scoopCommand {
					return testutil.SuccessResult("C:\\Users\\test\\scoop\\shims\\scoop.ps1\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "C:\\Users\\test\\scoop\\shims\\scoop.ps1",
			wantErr: false,
		},
		{
			name: "multiple paths returned",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == "where" && len(args) == 1 && args[0] == scoopCommand {
					return testutil.SuccessResult("C:\\First\\scoop.ps1\nC:\\Second\\scoop.ps1\n"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "C:\\First\\scoop.ps1",
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
	if got == "" {
		t.Error("GetConfigPath() returned empty path")
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	listOutput := `Name     Version   Source  Updated     Info
----     -------   ------  -------     ----
git      2.43.0    main    2024-01-01
nodejs   20.10.0   main    2024-01-02  Update available
`

	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == scoopCommand && len(args) > 0 && args[0] == "list" {
			return testutil.SuccessResult(listOutput), nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil {
		t.Fatalf("ListPackages() error = %v", err)
	}

	if len(packages) != 2 {
		t.Errorf("ListPackages() returned %d packages, want 2", len(packages))
		return
	}

	if packages[0].Name != "git" {
		t.Errorf("packages[0].Name = %q, want %q", packages[0].Name, "git")
	}
	if packages[0].CurrentVersion != "2.43.0" {
		t.Errorf("packages[0].CurrentVersion = %q, want %q", packages[0].CurrentVersion, "2.43.0")
	}
}

func TestAdapter_ListPackages_Empty(t *testing.T) {
	execFunc := func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == scoopCommand && len(args) > 0 && args[0] == "list" {
			return testutil.SuccessResult("Name     Version   Source\n----     -------   ------\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil {
		t.Fatalf("ListPackages() error = %v", err)
	}

	if len(packages) != 0 {
		t.Errorf("ListPackages() returned %d packages, want 0", len(packages))
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
				if command == scoopCommand && len(args) > 0 && args[0] == testScoopStatusCommand {
					return testutil.SuccessResult("Everything is ok!"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: manager.StatusHealthy,
		},
		{
			name: "degraded with warnings",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) > 0 && args[0] == testScoopStatusCommand {
					return testutil.SuccessResult("Some apps are outdated"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: manager.StatusDegraded,
		},
		{
			name: "error status",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) > 0 && args[0] == testScoopStatusCommand {
					return testutil.FailureResult(1, "Error occurred"), nil
				}
				return nil, errors.New("unexpected command")
			},
			want: manager.StatusError,
		},
		{
			name: "command fails",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command failed")
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
		name         string
		opts         adapterm.UpdateOptions
		execFunc     testutil.ExecutorFunc
		wantSuccess  bool
		wantUpdated  int
		wantContains string
		wantErr      bool
	}{
		{
			name: "dry run",
			opts: adapterm.UpdateOptions{DryRun: true},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			wantSuccess:  true,
			wantContains: "Dry-run",
		},
		{
			name: "fixed strategy",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyFixed},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				return testutil.SuccessResult(""), nil
			},
			wantSuccess:  true,
			wantContains: "skipped",
		},
		{
			name: "successful upgrade",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) > 0 {
					if args[0] == "update" && len(args) == 1 {
						return testutil.SuccessResult("Scoop was updated successfully"), nil
					}
					if args[0] == "update" && len(args) == 2 && args[1] == "*" {
						return testutil.SuccessResult("Updating 'git' (2.42.0 -> 2.43.0)"), nil
					}
				}
				return nil, errors.New("unexpected command")
			},
			wantSuccess: true,
			wantUpdated: 1,
		},
		{
			name: "no updates available",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) > 0 {
					if args[0] == "update" && len(args) == 1 {
						return testutil.SuccessResult(""), nil
					}
					if args[0] == "update" && len(args) == 2 && args[1] == "*" {
						return &output.ExecutionResult{
							ExitCode: 1,
							Stdout:   "All packages are up to date",
						}, nil
					}
				}
				return nil, errors.New("unexpected command")
			},
			wantSuccess:  true,
			wantContains: "up to date",
		},
		{
			name: "upgrade fails",
			opts: adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest},
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) > 0 {
					if args[0] == "update" && len(args) == 1 {
						return testutil.SuccessResult(""), nil
					}
					if args[0] == "update" && len(args) == 2 && args[1] == "*" {
						return nil, errors.New("update failed")
					}
				}
				return nil, errors.New("unexpected command")
			},
			wantSuccess: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewAdapter(testutil.NewMockExecutor(tt.execFunc), testutil.NewMockLogger())

			result, err := adapter.Update(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				if !tt.wantErr {
					t.Error("Update() returned nil result")
				}
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("Update().Success = %v, want %v", result.Success, tt.wantSuccess)
			}
			if tt.wantUpdated > 0 && len(result.UpdatedPackages) != tt.wantUpdated {
				t.Errorf("Update().UpdatedPackages = %d, want %d", len(result.UpdatedPackages), tt.wantUpdated)
			}
			if tt.wantContains != "" && !strings.Contains(result.Message, tt.wantContains) {
				t.Errorf("Update().Message = %q, want to contain %q", result.Message, tt.wantContains)
			}
		})
	}
}

func TestAdapter_Search(t *testing.T) {
	tableOutput := `Name   Version Source
----   ------- ------
git    2.43.0  main
github 2.40.0  main
`

	quotedOutput := `Results from other known buckets...
'git' (2.43.0) main
'gh' (2.40.0) extras
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
			name:  "table results",
			query: "git",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) == 2 && args[0] == "search" && args[1] == "git" {
					return testutil.SuccessResult(tableOutput), nil
				}
				return nil, errors.New("unexpected command")
			},
			wantCount: 2,
			wantFirst: "git",
		},
		{
			name:  "quoted results",
			query: "git",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == scoopCommand && len(args) == 2 && args[0] == "search" {
					return testutil.SuccessResult(quotedOutput), nil
				}
				return nil, errors.New("unexpected command")
			},
			wantCount: 2,
			wantFirst: "git",
		},
		{
			name:  "empty query",
			query: "",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				t.Fatal("executor should not be called")
				return nil, nil
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
			if packages[0].Name != tt.wantFirst {
				t.Errorf("Search() first = %q, want %q", packages[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestAdapter_Install(t *testing.T) {
	t.Run("dry-run", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return nil, errors.New("should not be called")
		}), testutil.NewMockLogger())
		if err := adapter.Install(context.Background(), "git", true); err != nil {
			t.Fatalf("Install dry-run: %v", err)
		}
	})

	t.Run("install success", func(t *testing.T) {
		var gotArgs []string
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
			if command != scoopCommand {
				return nil, errors.New("unexpected command")
			}
			gotArgs = args
			return testutil.SuccessResult("Installing 'git'"), nil
		}), testutil.NewMockLogger())
		if err := adapter.Install(context.Background(), "git", false); err != nil {
			t.Fatalf("Install: %v", err)
		}
		if len(gotArgs) != 2 || gotArgs[0] != "install" || gotArgs[1] != "git" {
			t.Fatalf("args = %v", gotArgs)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		adapter := NewAdapter(testutil.NewMockExecutor(nil), testutil.NewMockLogger())
		if err := adapter.Install(context.Background(), "", false); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAdapter_Uninstall(t *testing.T) {
	var gotArgs []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, args ...string) (*output.ExecutionResult, error) {
		gotArgs = args
		return testutil.SuccessResult("Uninstalling 'git'"), nil
	}), testutil.NewMockLogger())
	if err := adapter.Uninstall(context.Background(), "git", false); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if gotArgs[0] != "uninstall" || gotArgs[1] != "git" {
		t.Fatalf("args = %v", gotArgs)
	}
	if err := adapter.Uninstall(context.Background(), "git", true); err != nil {
		t.Fatalf("Uninstall dry-run: %v", err)
	}
}

func TestAdapter_ListBuckets(t *testing.T) {
	bucketOutput := `Name   Source                                  Updated             Manifests
----   ------                                  -------             ---------
main   https://github.com/ScoopInstaller/Main  2024-01-01 00:00:00 1234
extras https://github.com/ScoopInstaller/Extras 2024-01-01 00:00:00 567
`

	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		if command == scoopCommand && len(args) == 2 && args[0] == "bucket" && args[1] == "list" {
			return testutil.SuccessResult(bucketOutput), nil
		}
		return nil, errors.New("unexpected")
	}), testutil.NewMockLogger())

	buckets, err := adapter.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("count = %d, want 2", len(buckets))
	}
	if buckets[0].Name != "main" || buckets[1].Name != "extras" {
		t.Errorf("buckets = %+v", buckets)
	}
}

func TestAdapter_AddRemoveBucket(t *testing.T) {
	var calls [][]string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, args ...string) (*output.ExecutionResult, error) {
		cp := append([]string{}, args...)
		calls = append(calls, cp)
		return testutil.SuccessResult("ok"), nil
	}), testutil.NewMockLogger())

	if err := adapter.AddBucket(context.Background(), "extras", ""); err != nil {
		t.Fatalf("AddBucket: %v", err)
	}
	if err := adapter.AddBucket(context.Background(), "custom", "https://example.com/bucket"); err != nil {
		t.Fatalf("AddBucket with url: %v", err)
	}
	if err := adapter.RemoveBucket(context.Background(), "extras"); err != nil {
		t.Fatalf("RemoveBucket: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(calls))
	}
	if calls[0][0] != "bucket" || calls[0][1] != "add" || calls[0][2] != "extras" {
		t.Errorf("add known = %v", calls[0])
	}
	if len(calls[1]) != 4 || calls[1][3] != "https://example.com/bucket" {
		t.Errorf("add custom = %v", calls[1])
	}
	if calls[2][1] != "rm" || calls[2][2] != "extras" {
		t.Errorf("rm = %v", calls[2])
	}

	if err := adapter.AddBucket(context.Background(), "", ""); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := adapter.RemoveBucket(context.Background(), "  "); err == nil {
		t.Fatal("expected empty name error on remove")
	}
}
