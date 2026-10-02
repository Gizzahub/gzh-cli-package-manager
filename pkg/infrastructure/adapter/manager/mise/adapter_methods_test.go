package mise

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

const testNodeName = "node"

func TestFixedPolicyDoesNotExecuteMise(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(context.Context, string, ...string) (*output.ExecutionResult, error) {
		t.Fatal("fixed policy must not execute any command")
		return nil, nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyFixed})
	if err != nil || !result.Success || len(result.UpdatedPackages) != 0 {
		t.Fatalf("fixed result=%#v err=%v", result, err)
	}
}

func TestMetadataMethodsAndHealth(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		switch command {
		case "mise":
			return testutil.SuccessResult("mise 2026.9.15 macos-arm64"), nil
		case locator():
			return testutil.SuccessResult("/usr/local/bin/mise\n"), nil
		default:
			return nil, errors.New("unexpected command")
		}
	}), testutil.NewMockLogger())
	version, err := adapter.GetVersion(context.Background())
	if err != nil || version != "2026.9.15" {
		t.Fatalf("version=%q err=%v", version, err)
	}
	path, err := adapter.GetBinaryPath(context.Background())
	if err != nil || path != "/usr/local/bin/mise" {
		t.Fatalf("path=%q err=%v", path, err)
	}
	detected, err := adapter.Detect(context.Background())
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v", detected, err)
	}
	status, err := adapter.CheckHealth(context.Background())
	if err != nil || status != manager.StatusHealthy {
		t.Fatalf("status=%v err=%v", status, err)
	}
}

func TestMetadataFailuresAndConfigPath(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return nil, errors.New("missing")
	}), testutil.NewMockLogger())
	if detected, _ := adapter.Detect(context.Background()); detected {
		t.Fatal("detected missing mise")
	}
	if _, err := adapter.GetVersion(context.Background()); err == nil {
		t.Fatal("version failure accepted")
	}
	if _, err := adapter.GetBinaryPath(context.Background()); err == nil {
		t.Fatal("binary failure accepted")
	}
	status, err := adapter.CheckHealth(context.Background())
	if err != nil || status != manager.StatusDegraded {
		t.Fatalf("status=%v err=%v", status, err)
	}
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	path, err := adapter.GetConfigPath(context.Background())
	if err != nil || path != filepath.Join(xdgDir, "mise", "config.toml") {
		t.Fatalf("path=%q err=%v", path, err)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	homeDir := t.TempDir()
	adapter.home = func() (string, error) { return homeDir, nil }
	path, err = adapter.GetConfigPath(context.Background())
	if err != nil || path != filepath.Join(homeDir, ".config", "mise", "config.toml") {
		t.Fatalf("path=%q err=%v", path, err)
	}
	adapter.home = func() (string, error) { return "", errors.New("no home") }
	if _, err = adapter.GetConfigPath(context.Background()); err == nil {
		t.Fatal("home error accepted")
	}
}

func TestStableDryRunAndMalformedMetadata(t *testing.T) {
	var args []string
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, values ...string) (*output.ExecutionResult, error) {
		args = append([]string{command}, values...)
		return testutil.SuccessResult("would upgrade"), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyStable, DryRun: true})
	if err != nil || len(result.UpdatedPackages) != 0 || result.Message != "would upgrade" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if got := strings.Join(args, " "); got != "mise upgrade --no-prune --dry-run" {
		t.Fatalf("args=%q", got)
	}
	bad := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return testutil.SuccessResult("not json"), nil
	}), testutil.NewMockLogger())
	if _, err := bad.ListPackages(context.Background()); err == nil {
		t.Fatal("malformed metadata accepted")
	}
}

func TestListPackagesAndParsingFailures(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, _ string, _ ...string) (*output.ExecutionResult, error) {
		return testutil.SuccessResult(`{"node":[{"version":"22.14.0","requested_version":"22","installed":true,"active":true}],"missing":[{"version":"1.0.0","installed":false,"active":true}]}`), nil
	}), testutil.NewMockLogger())
	packages, err := adapter.ListPackages(context.Background())
	if err != nil || len(packages) != 1 || packages[0].Name != testNodeName {
		t.Fatalf("packages=%#v err=%v", packages, err)
	}
	if _, err := decodeVersions("["); err == nil {
		t.Fatal("bad remote JSON accepted")
	}
	if _, err := decodeVersions(`[{}]`); err != nil {
		t.Fatal(err)
	}
}

func TestSemverValidationAndPrereleaseOrder(t *testing.T) {
	for _, value := range []string{"1.2", "01.2.3", "1.2.3-alpha..1", "1.2.3-alpha.01"} {
		if _, ok := parseSemver(value); ok {
			t.Fatalf("invalid semver accepted: %s", value)
		}
	}
	for _, pair := range [][2]string{{"1.2.3-alpha", "1.2.3-beta"}, {"1.2.3-1", "1.2.3-alpha"}, {"1.2.3-rc.2", "1.2.3-rc.10"}} {
		left, _ := parseSemver(pair[0])
		right, _ := parseSemver(pair[1])
		if left.compare(right) >= 0 {
			t.Fatalf("expected %s < %s", pair[0], pair[1])
		}
	}
	for _, value := range []string{"^20", "node", "20.1.2.3"} {
		if _, _, err := parseRequest(value); err == nil {
			t.Fatalf("request %q accepted", value)
		}
	}
}

func TestPolicyUpgradeFailureReportsSelectedTool(t *testing.T) {
	adapter := NewAdapter(testutil.NewMockExecutor(func(_ context.Context, command string, args ...string) (*output.ExecutionResult, error) {
		call := command + " " + strings.Join(args, " ")
		if strings.Contains(call, " ls ") {
			return testutil.SuccessResult(`{"jq":[{"version":"1.8.1","requested_version":"1.8","installed":true,"active":true}]}`), nil
		}
		if strings.Contains(call, "ls-remote") {
			return testutil.SuccessResult(`["1.8.2"]`), nil
		}
		return testutil.FailureResult(1, "install failed"), nil
	}), testutil.NewMockLogger())
	result, err := adapter.Update(context.Background(), adapterm.UpdateOptions{Strategy: adapterm.StrategyMicro})
	if err == nil || result.Success || len(result.FailedPackages) != 1 || result.FailedPackages[0] != "jq" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestRequestMatchingAndPreview(t *testing.T) {
	version, ok := parseSemver("1.2.3+build.7")
	if !ok || !(request{major: 1, parts: 1}).matches(version) || (request{major: 1, minor: 3, parts: 2}).matches(version) {
		t.Fatal("numeric prefix matching failed")
	}
	if !(request{major: 1, minor: 2, patch: 3, parts: 3}).matches(version) {
		t.Fatal("exact stable match failed")
	}
	pre, _ := parseSemver("1.2.3-rc.1")
	if (request{major: 1, minor: 2, patch: 3, parts: 3}).matches(pre) {
		t.Fatal("exact request accepted prerelease")
	}
	if preview("", " stderr ") != "stderr" || preview(" stdout ", "stderr") != "stdout" {
		t.Fatal("preview selection failed")
	}
	latest, wildcard, err := parseRequest("latest")
	if err != nil || !wildcard || latest.parts != 0 {
		t.Fatalf("latest request=%#v wildcard=%v err=%v", latest, wildcard, err)
	}
	if !withinStrategy(version, version, adapterm.StrategyLatest) || withinStrategy(version, version, adapterm.StrategyFixed) {
		t.Fatal("strategy boundaries failed")
	}
	if !allowedPrerelease(pre, adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest, Bump: true}) || allowedPrerelease(pre, adapterm.UpdateOptions{Strategy: adapterm.StrategyLatest}) {
		t.Fatal("prerelease policy failed")
	}
}
