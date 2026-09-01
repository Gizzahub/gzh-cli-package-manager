package release_test

import (
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
