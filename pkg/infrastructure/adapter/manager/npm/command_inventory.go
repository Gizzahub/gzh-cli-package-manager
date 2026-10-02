package npm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

// The npm adapter satisfies the optional command-inventory capability: it
// reports global package bin names read-only, never installing, updating,
// or removing anything.
var _ adapterm.CommandInventory = (*Adapter)(nil)

const (
	// npmExecutable is the npm command line tool.
	npmExecutable = "npm"
	// npmProviderID is the provider id every npm install record reports.
	npmProviderID = "npm"
	// listSubcommand lists installed packages (npm list).
	listSubcommand = "list"
	// rootSubcommand prints the global node_modules directory (npm root).
	rootSubcommand = "root"
	// globalFlag selects the global install location.
	globalFlag = "-g"
	// depthFlag limits the listing to direct dependencies.
	depthFlag = "--depth=0"
	// jsonFlag switches the output format to JSON.
	jsonFlag = "--json"
	// manifestCommand reads one package manifest through the executor.
	manifestCommand = "cat"
	// manifestFile is the manifest npm reads bin names from.
	manifestFile = "package.json"
)

// Commands implements manager.CommandInventory for the npm adapter. Every
// globally installed package contributes one install record per bin
// basename: a manifest without a bin contributes no record, and a package
// name never becomes a command by itself. The query is read-only: it lists
// the global packages, resolves the global node_modules directory, and reads
// manifest files through the executor. It never looks up PATH and never sees
// project-local packages.
func (a *Adapter) Commands(ctx context.Context) ([]diagnostics.InstallRecord, error) {
	versions, err := a.globalVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("npm command inventory: %w", err)
	}
	if len(versions) == 0 {
		return []diagnostics.InstallRecord{}, nil
	}

	root, err := a.globalRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("npm command inventory: %w", err)
	}

	records := make([]diagnostics.InstallRecord, 0, len(versions))
	for _, name := range slices.Sorted(maps.Keys(versions)) {
		packageRecords, err := a.packageRecords(ctx, root, name, versions[name])
		if err != nil {
			return nil, err
		}
		records = append(records, packageRecords...)
	}
	return records, nil
}

// globalVersions returns the installed version of every global npm package
// by package name, from npm list -g --depth=0 --json.
func (a *Adapter) globalVersions(ctx context.Context) (map[string]string, error) {
	result, err := a.executor.Execute(ctx, npmExecutable, listSubcommand, globalFlag, depthFlag, jsonFlag)
	if resultErr := cmdutil.CheckResult(result, err, "list npm packages"); resultErr != nil {
		return nil, resultErr
	}

	var list struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if resultErr := cmdutil.UnmarshalJSON(result, &list, "parse npm packages"); resultErr != nil {
		return nil, resultErr
	}

	versions := make(map[string]string, len(list.Dependencies))
	for name, dep := range list.Dependencies {
		versions[name] = dep.Version
	}
	return versions, nil
}

// globalRoot returns the global node_modules directory from npm root -g.
func (a *Adapter) globalRoot(ctx context.Context) (string, error) {
	result, err := a.executor.Execute(ctx, npmExecutable, rootSubcommand, globalFlag)
	if resultErr := cmdutil.CheckResult(result, err, "resolve npm global root"); resultErr != nil {
		return "", resultErr
	}
	return cmdutil.ExtractStdout(result), nil
}

// packageRecords reads one global package's manifest and returns one install
// record per bin basename. The record's real path is the manifest file npm
// links the global bin to, so two packages that ship the same bin name stay
// distinct installs. Every record is active: a global npm install is the one
// npm selects. Executed stays false because the adapter never resolves PATH.
func (a *Adapter) packageRecords(ctx context.Context, root, name, installedVersion string) ([]diagnostics.InstallRecord, error) {
	dir := filepath.Join(root, name)
	manifest, err := a.readManifest(ctx, dir)
	if err != nil {
		return nil, err
	}
	bins, err := manifest.binInstalls(name)
	if err != nil {
		return nil, err
	}

	records := make([]diagnostics.InstallRecord, 0, len(bins))
	for _, bin := range bins {
		records = append(records, diagnostics.InstallRecord{
			Command:    bin.command,
			ProviderID: npmProviderID,
			Kind:       diagnostics.KindSystemOrLanguage,
			Version:    installedVersion,
			RealPath:   filepath.Join(dir, bin.file),
			Active:     true,
			Executed:   false,
		})
	}
	return records, nil
}

// readManifest reads one global package's package.json through the executor.
func (a *Adapter) readManifest(ctx context.Context, packageDir string) (npmManifest, error) {
	result, err := a.executor.Execute(ctx, manifestCommand, filepath.Join(packageDir, manifestFile))
	if resultErr := cmdutil.CheckResult(result, err, "read npm package manifest"); resultErr != nil {
		return npmManifest{}, resultErr
	}

	var manifest npmManifest
	if resultErr := cmdutil.UnmarshalJSON(result, &manifest, "parse npm package manifest"); resultErr != nil {
		return npmManifest{}, resultErr
	}
	return manifest, nil
}

// npmManifest is the subset of package.json the command inventory reads.
type npmManifest struct {
	Bin json.RawMessage `json:"bin"`
}

// binInstall is one command a package bin provides.
type binInstall struct {
	command string // the link name npm installs this bin under.
	file    string // the bin target file inside the package directory.
}

// binInstalls returns the commands the manifest's bin field provides. A
// string bin links the package under its name without the scope; a bin map
// links one command per key; an absent or null bin provides nothing.
func (m npmManifest) binInstalls(packageName string) ([]binInstall, error) {
	raw := bytes.TrimSpace(m.Bin)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}

	var file string
	if err := json.Unmarshal(raw, &file); err == nil {
		return []binInstall{{command: unscopedName(packageName), file: file}}, nil
	}

	var bins map[string]string
	if err := json.Unmarshal(raw, &bins); err != nil {
		return nil, fmt.Errorf("npm command inventory: package %q has an unusable bin field: %w", packageName, err)
	}

	installs := make([]binInstall, 0, len(bins))
	for command, binFile := range bins {
		installs = append(installs, binInstall{command: command, file: binFile})
	}
	slices.SortFunc(installs, func(a, b binInstall) int { return strings.Compare(a.command, b.command) })
	return installs, nil
}

// unscopedName returns the npm link name for a single-string bin: the
// package name without its scope.
func unscopedName(packageName string) string {
	if i := strings.LastIndex(packageName, "/"); i >= 0 {
		return packageName[i+1:]
	}
	return packageName
}
