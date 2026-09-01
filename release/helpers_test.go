package release_test

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	goosWindows   = "windows"
	archAMD64     = "amd64"
	kindELF       = "elf"
	kindMachO     = "macho"
	kindPE        = "pe"
	binaryUnix    = "gz-pm"
	binaryWindows = "gz-pm.exe"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func runReleaseScript(t *testing.T, name string, args ...string) string {
	t.Helper()
	root := repoRoot(t)
	cmd := exec.Command(filepath.Join(root, "scripts", "release", name), args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

type payloadRow struct {
	goosFilter string
	module     string
	archive    string
}

func loadPayloadRows(t *testing.T) []payloadRow {
	t.Helper()
	file, err := os.Open(filepath.Join(repoRoot(t), "release", "payload-files.tsv"))
	if err != nil {
		t.Fatalf("open payload-files.tsv: %v", err)
	}
	defer file.Close()

	var rows []payloadRow
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			t.Fatalf("payload row has %d fields, want 7: %q", len(fields), line)
		}
		rows = append(rows, payloadRow{
			goosFilter: fields[1],
			module:     fields[2],
			archive:    fields[5],
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan payload-files.tsv: %v", err)
	}
	return rows
}

func payloadApplies(row payloadRow, goos string) bool {
	return row.goosFilter == "*" || row.goosFilter == goos
}

func archiveMemberNames(t *testing.T, path string) []string {
	t.Helper()
	if strings.HasSuffix(path, ".zip") {
		return zipMemberNames(t, path)
	}
	return tarMemberNames(t, path)
}

func zipMemberNames(t *testing.T, path string) []string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("zip %s: %v", path, err)
	}
	defer reader.Close()
	names := make([]string, 0, len(reader.File))
	for _, member := range reader.File {
		names = append(names, filepath.ToSlash(member.Name))
	}
	return names
}

func tarMemberNames(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("gzip %s: %v", path, err)
	}
	defer gz.Close()
	var names []string
	tr := tar.NewReader(gz)
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			return names
		}
		if nextErr != nil {
			t.Fatalf("tar %s: %v", path, nextErr)
		}
		names = append(names, filepath.ToSlash(hdr.Name))
	}
}

func extractFile(t *testing.T, archivePath, member, dest string) {
	t.Helper()
	if strings.HasSuffix(archivePath, ".zip") {
		extractZipMember(t, archivePath, member, dest)
		return
	}
	extractTarMember(t, archivePath, member, dest)
}

func extractZipMember(t *testing.T, archivePath, member, dest string) {
	t.Helper()
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("zip %s: %v", archivePath, err)
	}
	defer reader.Close()
	for _, memberFile := range reader.File {
		if filepath.ToSlash(memberFile.Name) != member {
			continue
		}
		src, openErr := memberFile.Open()
		if openErr != nil {
			t.Fatalf("open zip member %s: %v", member, openErr)
		}
		writeExtracted(t, dest, src)
		_ = src.Close()
		return
	}
	t.Fatalf("zip member %s not found in %s", member, archivePath)
}

func extractTarMember(t *testing.T, archivePath, member, dest string) {
	t.Helper()
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open %s: %v", archivePath, err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("gzip %s: %v", archivePath, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			t.Fatalf("tar %s: %v", archivePath, nextErr)
		}
		if filepath.ToSlash(hdr.Name) != member {
			continue
		}
		writeExtracted(t, dest, tr)
		return
	}
	t.Fatalf("tar member %s not found in %s", member, archivePath)
}

func writeExtracted(t *testing.T, dest string, src io.Reader) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dest), err)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create %s: %v", dest, err)
	}
	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		t.Fatalf("write %s: %v", dest, copyErr)
	}
	if closeErr != nil {
		t.Fatalf("close %s: %v", dest, closeErr)
	}
}

func binaryKind(data []byte) string {
	switch {
	case len(data) >= 2 && data[0] == 'M' && data[1] == 'Z':
		return kindPE
	case len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F':
		return kindELF
	case isMachO(data):
		return kindMachO
	default:
		return "unknown"
	}
}

func isMachO(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	magic := [4]byte{data[0], data[1], data[2], data[3]}
	switch magic {
	case [4]byte{0xfe, 0xed, 0xfa, 0xce},
		[4]byte{0xfe, 0xed, 0xfa, 0xcf},
		[4]byte{0xce, 0xfa, 0xed, 0xfe},
		[4]byte{0xcf, 0xfa, 0xed, 0xfe},
		[4]byte{0xca, 0xfe, 0xba, 0xbe},
		[4]byte{0xbe, 0xba, 0xfe, 0xca}:
		return true
	default:
		return false
	}
}
