package release_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseTargetSetsStayAligned(t *testing.T) {
	out := runReleaseScript(t, "targets-check.sh")
	for _, target := range []string{
		"linux/amd64",
		"linux/arm64",
		"darwin/amd64",
		"darwin/arm64",
		"windows/amd64",
	} {
		if !strings.Contains(out, target) {
			t.Errorf("targets-check output missing %s\n%s", target, out)
		}
	}
}

func TestReleasePayloadExcludesTestOnlyModules(t *testing.T) {
	out := runReleaseScript(t, "payload-check.sh")
	if !strings.Contains(out, "ok") {
		t.Fatalf("payload-check unexpected output: %s", out)
	}
	for _, row := range loadPayloadRows(t) {
		if strings.Contains(row.module, "testify") || strings.Contains(row.module, "go.yaml.in") {
			t.Errorf("payload includes test-only module %s", row.module)
		}
		if strings.Contains(row.archive, "testify") || strings.Contains(row.archive, "go.yaml.in") {
			t.Errorf("payload archive path includes test-only module %s", row.archive)
		}
	}
}

func TestRuntimeDependencyInventoryStillMatches(t *testing.T) {
	runReleaseScript(t, "runtime-deps.sh", "--check")
}

func TestReleaseVersionValidator(t *testing.T) {
	valid := []string{
		"0.1.0",
		"1.2.3-rc.1",
		"1.2.3+build.7",
		"1.2.3-alpha.1+exp.sha",
	}
	for _, version := range valid {
		t.Run("valid/"+version, func(t *testing.T) {
			output, valid := runVersionValidator(t, version)
			if !valid {
				t.Fatalf("validate-version.sh rejected %q: %s", version, output)
			}
			if strings.TrimSpace(string(output)) != version {
				t.Fatalf("validate-version.sh %q output %q", version, output)
			}
		})
	}

	invalid := []string{
		"v1.2.3",
		"1.2",
		"1.02.3",
		"1.2.3-01",
		"1.2.3-",
		"1.2.3+",
		"1.2.3-foo_bar",
		"1.2.3;touch",
	}
	for _, version := range invalid {
		t.Run("invalid/"+version, func(t *testing.T) {
			output, valid := runVersionValidator(t, version)
			if valid {
				t.Fatalf("validate-version.sh accepted %q: %s", version, output)
			}
		})
	}
}

func runVersionValidator(t *testing.T, version string) ([]byte, bool) {
	t.Helper()
	cmd := exec.Command(filepath.Join(repoRoot(t), "scripts", "release", "validate-version.sh"), version)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	return output, err == nil
}
