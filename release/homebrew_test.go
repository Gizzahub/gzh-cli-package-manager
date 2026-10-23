package release_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseCoverageFloor(t *testing.T) {
	for _, covered := range []int{84, 85, 86} {
		t.Run(fmt.Sprint(covered), func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "coverage.out")
			source := filepath.Join(repoRoot(t), "internal", "version", "version.go")
			data := fmt.Sprintf("mode: set\n%s:32.1,33.2 %d 1\n%s:43.1,44.2 %d 0\n", source, covered, source, 100-covered)
			if err := os.WriteFile(profile, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(filepath.Join(repoRoot(t), "scripts", "release", "coverage-check.sh"), profile)
			out, err := cmd.CombinedOutput()
			if (err == nil) != (covered >= 85) {
				t.Fatalf("coverage %d: error=%v, output=%s", covered, err, out)
			}
		})
	}
}

func TestHomebrewCaskRequiresBothUniqueChecksums(t *testing.T) {
	const candidateVersion = "0.1.0"
	manifest := filepath.Join(t.TempDir(), "checksums.txt")
	arm, intel := strings.Repeat("a", 64), strings.Repeat("b", 64)
	valid := arm + "  gz-pm-darwin-arm64.tar.gz\n" + intel + "  gz-pm-darwin-amd64.tar.gz\n"
	for _, tc := range []struct {
		name, version, checksums string
		valid                    bool
	}{
		{"valid", candidateVersion, valid, true},
		{"missing", candidateVersion, arm + "  gz-pm-darwin-arm64.tar.gz\n", false},
		{"duplicate", candidateVersion, valid + valid, false},
		{"invalid hash", candidateVersion, strings.ReplaceAll(valid, "a", "z"), false},
		{"injection", "0.1.0\";raise", valid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(manifest, []byte(tc.checksums), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(filepath.Join(repoRoot(t), "scripts", "release", "homebrew-cask.sh"), tc.version, manifest)
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v, output=%s", tc.valid, err, out)
			}
			if tc.valid {
				for _, want := range []string{arm, intel, `version "0.1.0"`, "/releases/download/v#{version}/", `binary "gz-pm-darwin-#{arch}/gz-pm"`} {
					if !strings.Contains(string(out), want) {
						t.Errorf("cask missing %s", want)
					}
				}
			}
		})
	}
}
