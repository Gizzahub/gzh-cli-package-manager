// Package diagnostics resolves collected install records to filesystem
// identities and marks the record the supplied search paths execute.
//
// The resolver is an infrastructure component: it touches only the record
// paths and the directory list a caller supplies. It never reads the process
// environment, walks a home directory, runs a package manager, or parses the
// contents of a shim or script.
//
// Design Principles:
//   - Filesystem identity is the join key: two names for one real file are
//     one install, a different file stays its own record
//   - Facts over interpretation: what a shim launches is never guessed
//   - The caller owns the search order; the first existing entry executes
package diagnostics

import (
	"os"
	"path/filepath"
	"sort"

	domaindiagnostics "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
)

// ResolvePathIdentity resolves every record's install path to its real file,
// collapses records whose paths name the same file into one record, and marks
// executed the one record whose file the supplied search paths run.
//
// For each distinct command basename the directory list is walked in the
// given order; the first existing entry of that name is the executed file,
// after symlink resolution. Records keep the provider id, provider kind, and
// version that arrived on them: when several records share one file, the
// record whose own path already is the real file speaks for the group. A shim
// that is a different file stays its own record and stays not executed, and a
// record whose path does not exist keeps its collected path and is never
// executed. The result is sorted by command name, then real path.
func ResolvePathIdentity(records []domaindiagnostics.InstallRecord, searchPaths []string) []domaindiagnostics.InstallRecord {
	identities := resolveIdentities(records)
	executed := executedIdentities(records, searchPaths)
	return collapseRecords(records, identities, executed)
}

// installKey identifies one physical install behind one command: the command
// basename plus the real file the record's path resolves to.
type installKey struct {
	command  string
	identity string
}

// resolveIdentities maps every record index to the real file its path names.
func resolveIdentities(records []domaindiagnostics.InstallRecord) []string {
	identities := make([]string, len(records))
	for i := range records {
		identities[i] = realIdentity(records[i].RealPath)
	}
	return identities
}

// realIdentity resolves one path to the file it names, following symlinks to
// their target. A path that does not resolve keeps its cleaned form, so a
// stale collected path still groups with itself.
func realIdentity(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return resolved
}

// executedIdentities returns, per command basename, the file identity of the
// first search-path entry holding that name. A command with no existing entry
// is absent from the result, so nothing is executed for it.
func executedIdentities(records []domaindiagnostics.InstallRecord, searchPaths []string) map[string]string {
	executed := make(map[string]string, len(records))
	for _, command := range distinctCommands(records) {
		if hit, found := firstSearchHit(command, searchPaths); found {
			executed[command] = realIdentity(hit)
		}
	}
	return executed
}

// distinctCommands lists the command basenames the records carry, in
// first-seen order.
func distinctCommands(records []domaindiagnostics.InstallRecord) []string {
	seen := make(map[string]struct{}, len(records))
	commands := make([]string, 0, len(records))
	for i := range records {
		command := records[i].Command
		if _, dup := seen[command]; dup {
			continue
		}
		seen[command] = struct{}{}
		commands = append(commands, command)
	}
	return commands
}

// firstSearchHit walks the directory list in order and returns the first
// entry holding the command name. A missing or dangling entry is not a hit,
// so the walk continues to the next directory.
func firstSearchHit(command string, searchPaths []string) (string, bool) {
	for _, dir := range searchPaths {
		candidate := filepath.Join(dir, command)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// collapseRecords folds the records of every command and file identity into
// one record each, renames it to the resolved real path, and marks executed
// the identity the search paths run. The result is sorted by command name,
// then real path.
func collapseRecords(records []domaindiagnostics.InstallRecord, identities []string, executed map[string]string) []domaindiagnostics.InstallRecord {
	groups := make(map[installKey][]int, len(records))
	keys := make([]installKey, 0, len(records))
	for i := range records {
		key := installKey{command: records[i].Command, identity: identities[i]}
		if _, seen := groups[key]; !seen {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], i)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].command != keys[j].command {
			return keys[i].command < keys[j].command
		}
		return keys[i].identity < keys[j].identity
	})

	resolved := make([]domaindiagnostics.InstallRecord, 0, len(keys))
	for _, key := range keys {
		record := records[representative(records, identities, groups[key])]
		record.RealPath = key.identity
		record.Executed = executed[key.command] == key.identity
		resolved = append(resolved, record)
	}
	return resolved
}

// representative returns the index of the record that speaks for one identity
// group, so the group keeps deterministic provider facts.
func representative(records []domaindiagnostics.InstallRecord, identities []string, indexes []int) int {
	chosen := indexes[0]
	for _, index := range indexes[1:] {
		if betterRepresentative(records, identities, index, chosen) {
			chosen = index
		}
	}
	return chosen
}

// betterRepresentative reports whether the candidate record should replace
// the current one as the speaker for one identity group. The record whose own
// path already is the real file wins, then the executed one, then the lowest
// provider id and version.
func betterRepresentative(records []domaindiagnostics.InstallRecord, identities []string, candidate, current int) bool {
	candidateRecord, currentRecord := &records[candidate], &records[current]
	candidateDirect, currentDirect := candidateRecord.RealPath == identities[candidate], currentRecord.RealPath == identities[current]
	if candidateDirect != currentDirect {
		return candidateDirect
	}
	if candidateRecord.Executed != currentRecord.Executed {
		return candidateRecord.Executed
	}
	if candidateRecord.ProviderID != currentRecord.ProviderID {
		return candidateRecord.ProviderID < currentRecord.ProviderID
	}
	return candidateRecord.Version < currentRecord.Version
}
