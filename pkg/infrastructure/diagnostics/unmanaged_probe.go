package diagnostics

import (
	"os"
	"path/filepath"
	"strings"

	domaindiagnostics "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
)

// KindUnmanaged marks installers that are not one of the eleven managed
// adapters. It stays distinct from the version-manager and
// system-or-language kinds, so the version-manager note never hides an
// unmanaged install.
const KindUnmanaged domaindiagnostics.ProviderKind = "unmanaged"

// versionsSegment is the path segment that precedes an extracted version in
// a resolved install path.
const versionsSegment = "versions"

// unmanagedRow is one row of the explicit unmanaged-installer table: a
// relative path under the supplied home root plus the install facts to
// report when that exact file exists.
type unmanagedRow struct {
	relativePath string                         // relativePath is the row's file, relative to the home root.
	command      string                         // command is the command basename the file provides.
	providerID   string                         // providerID names the unmanaged installer.
	kind         domaindiagnostics.ProviderKind // kind is the provider kind reported for the row.
}

// unmanagedTable is the explicit table of installers that are not one of the
// eleven adapters in the registry. It is data, not a home-directory walk: the
// probe stats only the relative file each row names, and fixed binary paths
// such as /usr/local/bin or ~/.asdf/shims never become table roots.
var unmanagedTable = []unmanagedRow{
	{
		relativePath: ".local/bin/claude",
		command:      "claude",
		providerID:   "claude-native",
		kind:         KindUnmanaged,
	},
}

// ProbeUnmanagedInstalls checks the unmanaged-installer table under the
// supplied home root and returns the resolved install records of the rows
// whose file exists. The caller supplies the home root and the search paths
// explicitly: the probe never calls os.UserHomeDir, never reads the process
// environment, and never walks the root.
//
// Every found row's path goes through the path-identity resolver, so a
// launcher symlink and its target are one install, and the caller's search
// paths decide which record is executed. A missing root or a missing file
// contributes no record and no error. A home root that is not absolute is
// treated as missing, so a relative or empty root never probes the working
// directory. The file is never executed.
func ProbeUnmanagedInstalls(homeRoot string, searchPaths []string) []domaindiagnostics.InstallRecord {
	if !filepath.IsAbs(homeRoot) {
		return nil
	}
	found := presentTableRows(homeRoot, unmanagedTable)
	if len(found) == 0 {
		return nil
	}
	resolved := ResolvePathIdentity(found, searchPaths)
	applyUnmanagedVersions(resolved)
	return resolved
}

// presentTableRows stats each row's exact relative path under the home root
// and returns a record for every row whose file exists. A row whose file is
// absent — including every row under a missing root — contributes nothing.
func presentTableRows(homeRoot string, rows []unmanagedRow) []domaindiagnostics.InstallRecord {
	records := make([]domaindiagnostics.InstallRecord, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		installPath := filepath.Join(homeRoot, row.relativePath)
		if !filePresent(installPath) {
			continue
		}
		records = append(records, domaindiagnostics.InstallRecord{
			Command:    row.command,
			ProviderID: row.providerID,
			Kind:       row.kind,
			RealPath:   installPath,
			Active:     true,
		})
	}
	return records
}

// filePresent reports whether the exact path names an existing file, following
// a launcher symlink to its target. A dangling link is a missing file, as it is
// for the resolver's search-path walk. Only the row's own path is stat'ed; the
// probe never searches the directories around it.
func filePresent(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// applyUnmanagedVersions fills each resolved record's version from the
// versions segment of its resolved real path. A path without a
// versions/<version>/ segment keeps an empty version and still reports the
// command.
func applyUnmanagedVersions(records []domaindiagnostics.InstallRecord) {
	for i := range records {
		records[i].Version = versionFromRealPath(records[i].RealPath)
	}
}

// versionFromRealPath returns the version directory of a resolved install
// path of the form ".../versions/<version>/<file>". The rightmost versions
// directory wins, so a home root that itself sits under a directory named
// versions does not leak into the version. A path whose versions element is
// not followed by a directory and a file has no extracted version.
func versionFromRealPath(realPath string) string {
	segments := strings.Split(realPath, string(filepath.Separator))
	for i := len(segments) - 3; i >= 0; i-- {
		if segments[i] == versionsSegment {
			return segments[i+1]
		}
	}
	return ""
}
