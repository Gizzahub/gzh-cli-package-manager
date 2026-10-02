package mise

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const testNodeTool = "node"

func TestUpdateMinorPlansBoundedVersionsBeforeMutation(t *testing.T) {
	var calls []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		calls = append(calls, call)
		switch {
		case strings.Contains(call, "ls --current --json"):
			return testutil.SuccessResult(`[{"name":"node","version":"20.10.0","requested_version":"20","installed":true,"active":true}]`), nil
		case strings.Contains(call, "ls-remote node"):
			return testutil.SuccessResult(`["20.11.0", "21.0.0", "20.12.0-rc.1"]`), nil
		case strings.Contains(call, "upgrade node@20.11.0"):
			return testutil.SuccessResult("updated node"), nil
		}
		return nil, errors.New("unexpected command: " + call)
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyMinor})
	if err != nil || !result.Success || len(result.UpdatedPackages) != 1 || result.UpdatedPackages[0] != testNodeTool {
		t.Fatalf("Update() = %#v, %v", result, err)
	}
	if strings.Join(calls, "\n") != "mise ls --current --json\nmise ls-remote node --json\nmise upgrade node@20.11.0 --no-prune" {
		t.Fatalf("calls = %q", calls)
	}
}

func TestUpdateMicroHonorsRequestPinAndDryRunDoesNotUpgrade(t *testing.T) {
	var calls []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`[{"name":"jq","version":"1.8.1","requested_version":"1.8","installed":true,"active":true}]`), nil
		}
		if strings.Contains(call, "ls-remote") {
			return testutil.SuccessResult(`["1.8.2", "1.9.0"]`), nil
		}
		if strings.Contains(call, "--dry-run") {
			return testutil.SuccessResult("would update jq"), nil
		}
		return nil, errors.New("mutation in dry run")
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyMicro, DryRun: true})
	if err != nil || !result.Success || len(result.UpdatedPackages) != 0 {
		t.Fatalf("Update() = %#v, %v", result, err)
	}
	if len(calls) != 3 || !strings.Contains(calls[2], "upgrade jq@1.8.2 --no-prune --dry-run") {
		t.Fatalf("calls = %q", calls)
	}
}

func TestUpdateLatestIncludesPrereleaseAndBump(t *testing.T) {
	var upgrade string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`[{"name":"go","version":"1.22.0","requested_version":"1.22","installed":true,"active":true}]`), nil
		}
		if strings.Contains(call, "ls-remote") {
			return testutil.SuccessResult(`["1.23.0", "1.24.0-rc.1"]`), nil
		}
		upgrade = call
		return testutil.SuccessResult("ok"), nil
	}), testutil.NewMockLogger())
	_, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest, Bump: true})
	if err != nil || upgrade != "mise upgrade go@1.24.0-rc.1 --no-prune --bump" {
		t.Fatalf("err=%v upgrade=%q", err, upgrade)
	}
}

func TestStableNativeFailureAndFixedBumpAreErrors(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return testutil.FailureResult(1, "broken"), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyStable})
	if err == nil || result.Success {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	result, err = adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyFixed, Bump: true})
	if err == nil || result.Success {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUpdateLatestSelectsSemanticVersionAndLocalScope(t *testing.T) {
	var calls []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if strings.Contains(call, "ls --local --json") {
			return testutil.SuccessResult(`[{"name":"tool","version":"1.8.0","requested_version":"*","installed":true,"active":true}]`), nil
		}
		if strings.Contains(call, "ls-remote") {
			return testutil.SuccessResult(`["1.9.0", "1.10.0"]`), nil
		}
		return testutil.SuccessResult("ok"), nil
	}), testutil.NewMockLogger())
	_, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest, MiseLocal: true, MiseDir: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	want := "mise -C /repo ls --local --json\nmise -C /repo ls-remote tool --json --prerelease\nmise -C /repo upgrade tool@1.10.0 --no-prune --local"
	if got := strings.Join(calls, "\n"); got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestDecodeToolsAcceptsMiseGroupedJSON(t *testing.T) {
	tools, err := decodeTools(`{"node":[{"version":"22.14.0","requested_version":"22","installed":true,"active":true}]}`)
	if err != nil || len(tools) != 1 || tools[0].Name != testNodeTool {
		t.Fatalf("tools=%#v err=%v", tools, err)
	}
}

func TestExactRequestDoesNotUpgradeWithoutBump(t *testing.T) {
	var calls []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		calls = append(calls, cmd+" "+strings.Join(args, " "))
		return testutil.SuccessResult(`{"jq":[{"version":"1.8.1","requested_version":"1.8.2","installed":true,"active":true}]}`), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyMicro})
	if err != nil || !result.Success || len(calls) != 1 {
		t.Fatalf("result=%#v err=%v calls=%q", result, err, calls)
	}
}

func TestPrereleaseUsesNumericIdentifiers(t *testing.T) {
	rc10, ok10 := parseSemver("1.2.3-rc.10")
	rc2, ok2 := parseSemver("1.2.3-rc.2")
	if !ok10 || !ok2 || rc10.compare(rc2) <= 0 {
		t.Fatalf("semver prerelease comparison failed: %#v %#v", rc10, rc2)
	}
}

func TestPlanningFailureRunsNoUpgrade(t *testing.T) {
	var upgraded bool
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`{"a":[{"version":"1.0.0","requested_version":"1","installed":true,"active":true}],"b":[{"version":"1.0.0","requested_version":"1","installed":true,"active":true}]}`), nil
		}
		if strings.Contains(call, "ls-remote a") {
			return testutil.SuccessResult(`["1.1.0"]`), nil
		}
		if strings.Contains(call, "ls-remote b") {
			return testutil.FailureResult(1, "offline"), nil
		}
		if strings.Contains(call, "upgrade") {
			upgraded = true
		}
		return testutil.SuccessResult(""), nil
	}), testutil.NewMockLogger())
	if _, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyMinor}); err == nil || upgraded {
		t.Fatalf("err=%v upgraded=%v", err, upgraded)
	}
}

func TestSelectedToolSkipsUnsupportedUnselectedTool(t *testing.T) {
	var calls []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`{"jq":[{"version":"1.8.1","requested_version":"1.8","installed":true,"active":true}],"java":[{"version":"21","requested_version":"lts","installed":true,"active":true}]}`), nil
		}
		if strings.Contains(call, "ls-remote jq") {
			return testutil.SuccessResult(`["1.8.2"]`), nil
		}
		return testutil.SuccessResult("ok"), nil
	}), testutil.NewMockLogger())
	_, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyMicro, Packages: []string{"jq"}})
	if err != nil || len(calls) != 3 || !strings.Contains(calls[2], "jq@1.8.2") {
		t.Fatalf("err=%v calls=%q", err, calls)
	}
}

func TestSelectedToolsValidateAndStableUsesBareNames(t *testing.T) {
	for _, packages := range [][]string{{}, {""}, {"jq", "jq"}, {"missing"}} {
		adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
			return testutil.SuccessResult(`{}`), nil
		}), testutil.NewMockLogger())
		if _, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyStable, Packages: packages}); err == nil {
			t.Fatalf("packages %q accepted", packages)
		}
	}
	var call string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call = cmd + " " + strings.Join(args, " ")
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`{"jq":[{"version":"1.8.1","requested_version":"1.8","installed":true,"active":true}]}`), nil
		}
		return testutil.SuccessResult("would update"), nil
	}), testutil.NewMockLogger())
	_, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyStable, Packages: []string{"jq"}, Bump: true, DryRun: true})
	if err != nil || call != "mise upgrade jq --no-prune --bump --dry-run" {
		t.Fatalf("err=%v call=%q", err, call)
	}
}

func TestLatestWithoutBumpExcludesNewerPrerelease(t *testing.T) {
	var upgrade string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, cmd string, args ...string) (*output.ExecutionResult, error) {
		call := cmd + " " + strings.Join(args, " ")
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`{"node":[{"version":"22.14.0","requested_version":"22","installed":true,"active":true}]}`), nil
		}
		if strings.Contains(call, "ls-remote") {
			return testutil.SuccessResult(`["22.15.0", "23.0.0-rc.1"]`), nil
		}
		upgrade = call
		return testutil.SuccessResult("ok"), nil
	}), testutil.NewMockLogger())
	_, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest})
	if err != nil || !strings.Contains(upgrade, "node@22.15.0") {
		t.Fatalf("err=%v upgrade=%q", err, upgrade)
	}
}
