package pacman

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const (
	pacmanCommand = "pacman"
	whichCommand  = "which"
	queryFlag     = "-Qq"
	queryAllFlag  = "-Q"
	queryUpdates  = "-Qu"

	testPacmanGitPackage      = "git"
	testPacmanFirefoxPackage  = "firefox"
	testPacmanChromiumPackage = "chromium"
)

var errPacmanListPackages = errors.New("pacman list executor failed")

type pacmanListPackagesCall struct {
	args   []string
	result *output.ExecutionResult
	err    error
}

type pacmanListPackagesExpectation struct {
	packages []manager.Package
	err      error
	wantErr  bool
}

type pacmanHealthCall struct {
	command string
	args    []string
	result  *output.ExecutionResult
	err     error
}

type pacmanHealthWarning struct {
	message string
	fields  []output.Field
}

type pacmanHealthExpectation struct {
	status   manager.Status
	warnings []pacmanHealthWarning
}

type pacmanHealthLogger struct {
	warnings []pacmanHealthWarning
}

func (l *pacmanHealthLogger) Debug(_ context.Context, _ string, _ ...output.Field) {}
func (l *pacmanHealthLogger) Info(_ context.Context, _ string, _ ...output.Field)  {}
func (l *pacmanHealthLogger) Error(_ context.Context, _ string, _ error, _ ...output.Field) {
}

func (l *pacmanHealthLogger) Warn(_ context.Context, message string, fields ...output.Field) {
	l.warnings = append(l.warnings, pacmanHealthWarning{
		message: message,
		fields:  append([]output.Field(nil), fields...),
	})
}

func newPacmanHealthExecutor(t *testing.T, calls []pacmanHealthCall) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()

	callIndex := 0
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		if callIndex == len(calls) {
			t.Fatalf("unexpected command %q %v", command, args)
		}

		call := calls[callIndex]
		callIndex++
		if command != call.command || !slices.Equal(args, call.args) {
			t.Errorf("command = %q %v, want %q %v", command, args, call.command, call.args)
		}
		return call.result, call.err
	}
	verify = func() {
		t.Helper()
		if callIndex != len(calls) {
			t.Errorf("command calls = %d, want %d", callIndex, len(calls))
		}
	}
	return executor, verify
}

func pacmanLockFileCall(result *output.ExecutionResult, err error) pacmanHealthCall {
	return pacmanHealthCall{
		command: "test",
		args:    []string{"-f", "/var/lib/pacman/db.lck"},
		result:  result,
		err:     err,
	}
}

func assertPacmanHealth(t *testing.T, status manager.Status, err error, logger *pacmanHealthLogger, want pacmanHealthExpectation) {
	t.Helper()
	if err != nil {
		t.Fatalf("CheckHealth() error = %v, want nil", err)
	}
	if status != want.status {
		t.Errorf("CheckHealth() status = %v, want %v", status, want.status)
	}
	if !reflect.DeepEqual(logger.warnings, want.warnings) {
		t.Errorf("CheckHealth() warnings = %#v, want %#v", logger.warnings, want.warnings)
	}
}

func newPacmanListPackagesExecutor(t *testing.T, calls []pacmanListPackagesCall) (executor testutil.ExecutorFunc, verify func()) {
	t.Helper()

	callIndex := 0
	executor = func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		t.Helper()
		if callIndex == len(calls) {
			t.Fatalf("unexpected command %q %v", command, args)
		}

		call := calls[callIndex]
		callIndex++
		if command != pacmanCommand || !slices.Equal(args, call.args) {
			t.Errorf("command = %q %v, want %q %v", command, args, pacmanCommand, call.args)
		}

		return call.result, call.err
	}
	verify = func() {
		t.Helper()
		if callIndex != len(calls) {
			t.Errorf("command calls = %d, want %d", callIndex, len(calls))
		}
	}
	return executor, verify
}

func assertPacmanListPackages(t *testing.T, packages []manager.Package, err error, want pacmanListPackagesExpectation) {
	t.Helper()

	switch {
	case want.err != nil:
		if !errors.Is(err, want.err) {
			t.Fatalf("ListPackages() error = %v, want errors.Is(..., %v)", err, want.err)
		}
		return
	case want.wantErr:
		if err == nil {
			t.Fatal("ListPackages() error = nil, want an error")
		}
		return
	case err != nil:
		t.Fatalf("ListPackages() error = %v, want nil", err)
	}

	if packages == nil {
		t.Fatal("ListPackages() packages = nil")
	}
	if !slices.Equal(packages, want.packages) {
		t.Errorf("ListPackages() packages = %#v, want %#v", packages, want.packages)
	}
}

func testPacmanPackage(name, currentVersion, availableVersion string, updateType manager.UpdateType) manager.Package {
	return manager.Package{
		Name:             name,
		CurrentVersion:   currentVersion,
		AvailableVersion: availableVersion,
		IsGlobal:         true,
		UpdateType:       updateType,
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     bool
		wantErr  bool
	}{
		{
			name: "pacman installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == pacmanCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/bin/pacman\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pacman not installed",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == pacmanCommand {
					return &output.ExecutionResult{
						ExitCode: 1,
						Stderr:   "pacman not found",
					}, errors.New("exit code 1")
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
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "valid version output",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == pacmanCommand && len(args) == 1 && args[0] == "--version" {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout: ` .--.                  Pacman v7.0.0 - libalpm v15.0.0
/ _.-' .-.  .-.  .-.   Copyright (C) 2006-2024 Pacman Development Team
\  '-. '-'  '-'  '-'   Copyright (C) 2002-2006 Judd Vinet
 '--'`,
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "7.0.0",
			wantErr: false,
		},
		{
			name: "command execution error",
			execFunc: func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
				return nil, errors.New("command not found")
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
		execFunc func(ctx context.Context, command string, args ...string) (*output.ExecutionResult, error)
		want     string
		wantErr  bool
	}{
		{
			name: "binary found",
			execFunc: func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
				if command == whichCommand && len(args) == 1 && args[0] == pacmanCommand {
					return &output.ExecutionResult{
						ExitCode: 0,
						Stdout:   "/usr/bin/pacman\n",
					}, nil
				}
				return nil, errors.New("unexpected command")
			},
			want:    "/usr/bin/pacman",
			wantErr: false,
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
		t.Errorf("GetConfigPath() unexpected error = %v", err)
		return
	}

	want := "/etc/pacman.conf"
	if got != want {
		t.Errorf("GetConfigPath() = %v, want %v", got, want)
	}
}

func TestAdapter_ListPackages(t *testing.T) {
	tests := []struct {
		name  string
		calls []pacmanListPackagesCall
		want  pacmanListPackagesExpectation
	}{
		{
			name: "preserves installed order and complete update mapping",
			calls: []pacmanListPackagesCall{
				{
					args:   []string{queryAllFlag},
					result: testutil.SuccessResult("git 2.51.0-1\nincomplete\nfirefox 143.0.3-1\nchromium 142.0.7444.162-1"),
				},
				{
					args:   []string{queryUpdates},
					result: testutil.SuccessResult("firefox 143.0.3-1 -> 145.0-1\nchromium 142.0.7444.162-1 -> 142.0.7444.175-1\nnot-installed 1.0.0 -> 2.0.0\nmalformed update line"),
				},
			},
			want: pacmanListPackagesExpectation{packages: []manager.Package{
				testPacmanPackage(testPacmanGitPackage, "2.51.0-1", "", manager.UpdateNone),
				testPacmanPackage(testPacmanFirefoxPackage, "143.0.3-1", "145.0-1", manager.UpdateMajor),
				testPacmanPackage(testPacmanChromiumPackage, "142.0.7444.162-1", "142.0.7444.175-1", manager.UpdatePatch),
			}},
		},
		{
			name: "ignores an update executor error",
			calls: []pacmanListPackagesCall{
				{
					args:   []string{queryAllFlag},
					result: testutil.SuccessResult("git 2.51.0-1"),
				},
				{
					args: []string{queryUpdates},
					err:  errors.New("pacman update query failed"),
				},
			},
			want: pacmanListPackagesExpectation{packages: []manager.Package{
				testPacmanPackage(testPacmanGitPackage, "2.51.0-1", "", manager.UpdateNone),
			}},
		},
		{
			name: "ignores a non-zero update result",
			calls: []pacmanListPackagesCall{
				{
					args:   []string{queryAllFlag},
					result: testutil.SuccessResult("git 2.51.0-1"),
				},
				{
					args: []string{queryUpdates},
					result: &output.ExecutionResult{
						ExitCode: 2,
						Stdout:   "git 2.51.0-1 -> 2.52.0-1",
					},
				},
			},
			want: pacmanListPackagesExpectation{packages: []manager.Package{
				testPacmanPackage(testPacmanGitPackage, "2.51.0-1", "", manager.UpdateNone),
			}},
		},
		{
			name: "returns a non-nil empty slice for no installed packages",
			calls: []pacmanListPackagesCall{
				{
					args:   []string{queryAllFlag},
					result: testutil.SuccessResult(""),
				},
				{
					args:   []string{queryUpdates},
					result: testutil.SuccessResult(""),
				},
			},
			want: pacmanListPackagesExpectation{packages: []manager.Package{}},
		},
		{
			name: "wraps the installed-list executor error without querying updates",
			calls: []pacmanListPackagesCall{
				{
					args: []string{queryAllFlag},
					err:  errPacmanListPackages,
				},
			},
			want: pacmanListPackagesExpectation{err: errPacmanListPackages},
		},
		{
			name: "rejects a non-zero installed-list result without querying updates",
			calls: []pacmanListPackagesCall{
				{
					args:   []string{queryAllFlag},
					result: testutil.FailureResult(1, "pacman list failed"),
				},
			},
			want: pacmanListPackagesExpectation{wantErr: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, verify := newPacmanListPackagesExecutor(t, tt.calls)
			defer verify()
			adapter := NewAdapter(testutil.NewMockExecutor(executor), testutil.NewMockLogger())

			got, err := adapter.ListPackages(context.Background())
			assertPacmanListPackages(t, got, err, tt.want)
		})
	}
}

func TestAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name  string
		calls []pacmanHealthCall
		want  pacmanHealthExpectation
	}{
		{
			name: "healthy system",
			calls: []pacmanHealthCall{
				{
					command: pacmanCommand,
					args:    []string{queryFlag},
					result:  &output.ExecutionResult{Stdout: "git\nfirefox\n"},
				},
				pacmanLockFileCall(&output.ExecutionResult{ExitCode: 1}, nil),
			},
			want: pacmanHealthExpectation{status: manager.StatusHealthy},
		},
		{
			name: "degraded with a database lock file",
			calls: []pacmanHealthCall{
				{
					command: pacmanCommand,
					args:    []string{queryFlag},
					result:  &output.ExecutionResult{},
				},
				pacmanLockFileCall(&output.ExecutionResult{}, nil),
			},
			want: pacmanHealthExpectation{
				status: manager.StatusDegraded,
				warnings: []pacmanHealthWarning{
					{
						message: "Pacman database lock file exists",
						fields:  []output.Field{{Key: "lockfile", Value: "/var/lib/pacman/db.lck"}},
					},
				},
			},
		},
		{
			name: "reports an inaccessible database as an error without probing the lock file",
			calls: []pacmanHealthCall{
				{
					command: pacmanCommand,
					args:    []string{queryFlag},
					result:  &output.ExecutionResult{ExitCode: 1, Stdout: "ignored", Stderr: "database error"},
				},
			},
			want: pacmanHealthExpectation{status: manager.StatusError},
		},
		{
			name: "reports a database executor error as degraded",
			calls: []pacmanHealthCall{
				{
					command: pacmanCommand,
					args:    []string{queryFlag},
					err:     errPacmanListPackages,
				},
			},
			want: pacmanHealthExpectation{
				status: manager.StatusDegraded,
				warnings: []pacmanHealthWarning{
					{
						message: "Failed to query pacman database",
						fields:  []output.Field{{Key: "error", Value: errPacmanListPackages.Error()}},
					},
				},
			},
		},
		{
			name: "keeps a lock probe execution error healthy",
			calls: []pacmanHealthCall{
				{
					command: pacmanCommand,
					args:    []string{queryFlag},
					result:  &output.ExecutionResult{},
				},
				pacmanLockFileCall(nil, errPacmanListPackages),
			},
			want: pacmanHealthExpectation{status: manager.StatusHealthy},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, verify := newPacmanHealthExecutor(t, tt.calls)
			defer verify()
			logger := &pacmanHealthLogger{}
			adapter := NewAdapter(testutil.NewMockExecutor(executor), logger)

			status, err := adapter.CheckHealth(context.Background())
			assertPacmanHealth(t, status, err, logger, tt.want)
		})
	}
}

func TestAdapter_GetVersion_Error(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_GetVersion_NoVersionFound(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return &output.ExecutionResult{
			Stdout:   "some output without version pattern",
			ExitCode: 0,
		}, nil
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetVersion(context.Background())
	if err == nil {
		t.Error("Expected error when no version pattern found")
	}
}

func TestAdapter_GetBinaryPath_Error(t *testing.T) {
	execFunc := func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("execution failed")
	}
	adapter := NewAdapter(testutil.NewMockExecutor(execFunc), testutil.NewMockLogger())

	_, err := adapter.GetBinaryPath(context.Background())
	if err == nil {
		t.Error("Expected error for executor failure")
	}
}

func TestAdapter_Update(t *testing.T) {
	var calls int
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		calls++
		return testutil.SuccessResult(""), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Update dry-run unexpected error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("Update dry-run expected success result, got %#v", result)
	}
	if calls != 0 {
		t.Fatalf("Update dry-run executed %d commands, want 0", calls)
	}
}
