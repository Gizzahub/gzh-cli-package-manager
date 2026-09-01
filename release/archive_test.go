package release_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type snapshotTarget struct {
	goos   string
	goarch string
	kind   string
}

func TestSnapshotArchivesContainLicensePayload(t *testing.T) {
	outDir := t.TempDir()
	runReleaseScript(t, "snapshot-archives.sh", "--output-dir", outDir)
	rows := loadPayloadRows(t)
	checksums := readChecksums(t, filepath.Join(outDir, "checksums.txt"))
	targets := []snapshotTarget{
		{"linux", archAMD64, kindELF},
		{"linux", "arm64", kindELF},
		{"darwin", archAMD64, kindMachO},
		{"darwin", "arm64", kindMachO},
		{goosWindows, archAMD64, kindPE},
	}
	if len(checksums) != len(targets) {
		t.Fatalf("checksums.txt has %d entries, want %d", len(checksums), len(targets))
	}
	for _, target := range targets {
		verifySnapshotTarget(t, outDir, target, rows, checksums)
	}
}

func verifySnapshotTarget(t *testing.T, outDir string, target snapshotTarget, rows []payloadRow, checksums map[string]string) {
	t.Helper()
	archiveName, binaryName := archiveNames(target)
	archivePath := filepath.Join(outDir, archiveName)
	assertChecksum(t, archivePath, archiveName, checksums)
	members := memberSet(archiveMemberNames(t, archivePath))
	assertNoTestOnly(t, archiveName, members)
	assertRequiredMembers(t, archiveName, target, rows, members)
	assertBinaryKind(t, archivePath, target, binaryName)
	smokeHostBinary(t, archivePath, target, binaryName)
}

func archivePrefix(target snapshotTarget) string {
	return binaryUnix + "-" + target.goos + "-" + target.goarch
}

func wrapDir(target snapshotTarget) string {
	return archivePrefix(target) + "/"
}

func archiveNames(target snapshotTarget) (archiveName string, binaryName string) {
	name := archivePrefix(target)
	if target.goos == goosWindows {
		return name + ".zip", binaryWindows
	}
	return name + ".tar.gz", binaryUnix
}

func assertChecksum(t *testing.T, archivePath, archiveName string, checksums map[string]string) {
	t.Helper()
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("stat %s: %v", archivePath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("empty archive %s", archivePath)
	}
	digest, ok := checksums[archiveName]
	if !ok {
		t.Fatalf("checksums.txt missing %s", archiveName)
	}
	if got := fileSHA256(t, archivePath); got != digest {
		t.Fatalf("%s checksum %s, want %s", archiveName, got, digest)
	}
}

func memberSet(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[strings.TrimPrefix(name, "./")] = struct{}{}
	}
	return out
}

func assertNoTestOnly(t *testing.T, archiveName string, members map[string]struct{}) {
	t.Helper()
	for member := range members {
		lower := strings.ToLower(member)
		if strings.Contains(lower, "testify") || strings.Contains(lower, "go.yaml.in") {
			t.Errorf("%s contains test-only path %s", archiveName, member)
		}
	}
}

func assertRequiredMembers(t *testing.T, archiveName string, target snapshotTarget, rows []payloadRow, members map[string]struct{}) {
	t.Helper()
	wrap := wrapDir(target)
	binaryName := binaryUnix
	if target.goos == goosWindows {
		binaryName = binaryWindows
	}
	required := []string{wrap + binaryName}
	hasMousetrap := false
	for _, row := range rows {
		if !payloadApplies(row, target.goos) {
			assertAbsent(t, archiveName, wrap+row.archive, members)
			continue
		}
		required = append(required, wrap+row.archive)
		if strings.Contains(row.module, "mousetrap") {
			hasMousetrap = true
		}
	}
	if (target.goos == goosWindows) != hasMousetrap {
		t.Errorf("%s mousetrap mapping mismatch for %s", archiveName, target.goos)
	}
	for _, member := range required {
		if _, found := members[member]; !found {
			t.Errorf("%s missing %s", archiveName, member)
		}
	}
}

func assertAbsent(t *testing.T, archiveName, member string, members map[string]struct{}) {
	t.Helper()
	if _, found := members[member]; found {
		t.Errorf("%s unexpectedly contains %s", archiveName, member)
	}
}

func assertBinaryKind(t *testing.T, archivePath string, target snapshotTarget, binaryName string) {
	t.Helper()
	extracted := filepath.Join(t.TempDir(), binaryName)
	extractFile(t, archivePath, wrapDir(target)+binaryName, extracted)
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if kind := binaryKind(data); kind != target.kind {
		t.Errorf("%s binary kind %s, want %s", filepath.Base(archivePath), kind, target.kind)
	}
}

func smokeHostBinary(t *testing.T, archivePath string, target snapshotTarget, binaryName string) {
	t.Helper()
	if runtime.GOOS != target.goos || runtime.GOARCH != target.goarch {
		return
	}
	extracted := filepath.Join(t.TempDir(), binaryName)
	extractFile(t, archivePath, wrapDir(target)+binaryName, extracted)
	out, err := exec.Command(extracted, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("smoke %s version: %v\n%s", filepath.Base(archivePath), err, out)
	}
	if !strings.Contains(string(out), binaryUnix) {
		t.Fatalf("smoke %s output %q does not contain %s", filepath.Base(archivePath), out, binaryUnix)
	}
}

func TestReleaseWorkflowPublishesArchivesAndChecksums(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "build.yml"))
	if err != nil {
		t.Fatalf("read build.yml: %v", err)
	}
	text := string(data)
	needles := []string{
		"artifacts/**/*.tar.gz",
		"artifacts/**/*.zip",
		"artifacts/**/checksums.txt",
		"dist/gz-pm-${{ matrix.goos }}-${{ matrix.goarch }}.*",
		"./scripts/release/package-archive.sh",
		"./scripts/release/checksums.sh",
		"./scripts/release/targets-check.sh",
	}
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			t.Errorf("build.yml missing %q", needle)
		}
	}
	if strings.Contains(text, "path: build/gz-pm-*") {
		t.Error("build.yml still uploads bare binaries from build/")
	}
	if strings.Contains(text, "files: artifacts/**/*") {
		t.Error("GitHub Release input still points at every downloaded artifact")
	}
}

func readChecksums(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	out := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "  ", 2)
		if len(fields) != 2 {
			t.Fatalf("invalid checksum line %q", scanner.Text())
		}
		out[fields[1]] = fields[0]
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return out
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
