package release_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPublishedReleaseUploadGuard(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "gh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s\\n' \"$GUARD_RESPONSE\"\nexit \"$GUARD_EXIT\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, response, exit string
		allow                bool
	}{
		{"draft", "HTTP/2.0 200 OK\n\n{\"draft\":true}", "0", true},
		{"published", "HTTP/2.0 200 OK\n\n{\"draft\":false}", "0", false},
		{"absent", "HTTP/2.0 404 Not Found\n\n{\"message\":\"Not Found\"}", "1", true},
		{"forbidden", "HTTP/2.0 403 Forbidden\n\n{\"message\":\"Forbidden\"}", "1", false},
		{"network", "", "1", false},
		{"invalid JSON", "HTTP/2.0 200 OK\n\nnot JSON", "0", false},
		{"missing state", "HTTP/2.0 200 OK\n\n{}", "0", false},
		{"wrong state type", "HTTP/2.0 200 OK\n\n{\"draft\":\"true\"}", "0", false},
		{"multiple JSON values", "HTTP/2.0 200 OK\n\n{\"draft\":false}\n{\"draft\":true}", "0", false},
		{"proxy absence", "HTTP/2.0 404 Not Found\n\n{\"message\":\"proxy unavailable\"}", "1", false},
		{"HTML absence", "HTTP/2.0 404 Not Found\n\n<html>not found</html>", "1", false},
		{"failed success response", "HTTP/2.0 200 OK\n\n{\"draft\":true}", "1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(filepath.Join(repoRoot(t), "scripts", "release", "published-release-check.sh"), "Gizzahub/gzh-cli-package-manager", "v0.1.0")
			cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "GUARD_RESPONSE="+tc.response, "GUARD_EXIT="+tc.exit)
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.allow {
				t.Fatalf("allow=%v error=%v output=%s", tc.allow, err, out)
			}
		})
	}
}
