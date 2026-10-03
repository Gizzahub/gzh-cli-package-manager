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

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
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
	// lsSubcommand lists installed packages (npm ls).
	lsSubcommand = "ls"
	// globalFlag selects the global install location.
	globalFlag = "-g"
	// depthFlag limits the listing to direct dependencies.
	depthFlag = "--depth=0"
	// jsonFlag switches the output format to JSON.
	jsonFlag = "--json"
	// longFlag extends every listed package with its install path and bin field.
	longFlag = "--long"
	// inventoryErrTag prefixes every error and warning the inventory reports.
	inventoryErrTag = "npm command inventory"
)

// Commands implements manager.CommandInventory for the npm adapter. Every
// globally installed package contributes one install record per bin
// basename: a package without a bin contributes no record, and a package
// name never becomes a command by itself. One read-only query answers the
// whole inventory: it never looks up PATH, scans a live npm prefix, or sees
// project-local packages.
func (a *Adapter) Commands(ctx context.Context) ([]diagnostics.InstallRecord, error) {
	tree, err := a.globalTree(ctx)
	if err != nil {
		return nil, err
	}
	return a.treeRecords(ctx, tree), nil
}

// globalTree runs the single query the inventory needs, npm ls -g
// --depth=0 --json --long. The --long form embeds every global package's
// install path and bin field, so no second query and no manifest read are
// required.
func (a *Adapter) globalTree(ctx context.Context) (npmGlobalTree, error) {
	result, err := a.executor.Execute(ctx, npmExecutable, lsSubcommand, globalFlag, depthFlag, jsonFlag, longFlag)
	if resultErr := cmdutil.CheckResult(result, err, "list npm packages"); resultErr != nil {
		return npmGlobalTree{}, fmt.Errorf("%s: %w", inventoryErrTag, resultErr)
	}

	var tree npmGlobalTree
	if resultErr := cmdutil.UnmarshalJSON(result, &tree, "parse npm packages"); resultErr != nil {
		return npmGlobalTree{}, fmt.Errorf("%s: %w", inventoryErrTag, resultErr)
	}
	return tree, nil
}

// treeRecords flattens the global tree into install records, one per bin
// basename, sorted by package name and then command. A dependency whose bin
// cannot be decoded is skipped with a warning instead of failing the whole
// inventory.
func (a *Adapter) treeRecords(ctx context.Context, tree npmGlobalTree) []diagnostics.InstallRecord {
	records := make([]diagnostics.InstallRecord, 0, len(tree.Dependencies))
	for _, name := range slices.Sorted(maps.Keys(tree.Dependencies)) {
		dep := tree.Dependencies[name]
		bins, err := dep.binInstalls(name)
		if err != nil {
			a.logger.Warn(ctx, inventoryErrTag+": skipping package with unusable bin",
				output.Field{Key: "package", Value: name},
				output.Field{Key: "error", Value: err.Error()})
			continue
		}
		records = append(records, dep.installRecords(bins)...)
	}
	return records
}

// npmGlobalTree is the subset of npm ls -g --depth=0 --json --long output
// the command inventory reads. The dependencies map key is the package name.
type npmGlobalTree struct {
	Dependencies map[string]npmDependency `json:"dependencies"`
}

// npmDependency is one global package entry of the ls --long tree.
type npmDependency struct {
	Version string          `json:"version"` // Version is the installed version string.
	Path    string          `json:"path"`    // Path is the package's install directory.
	Bin     json.RawMessage `json:"bin"`     // Bin is the raw bin field, in any JSON shape.
}

// binInstall is one command a package bin provides.
type binInstall struct {
	command string // the link name npm installs this bin under.
	file    string // the bin target file inside the package directory.
}

// installRecords turns one package's bins into install records. RealPath is
// the package's bin target file, the file npm links the global bin to, so
// two packages that ship the same bin name stay distinct installs. Every
// record is active: a global npm install is the one npm selects. Executed
// stays false because the adapter never resolves PATH.
func (d npmDependency) installRecords(bins []binInstall) []diagnostics.InstallRecord {
	records := make([]diagnostics.InstallRecord, 0, len(bins))
	for _, bin := range bins {
		records = append(records, diagnostics.InstallRecord{
			Command:    bin.command,
			ProviderID: npmProviderID,
			Kind:       diagnostics.KindSystemOrLanguage,
			Version:    d.Version,
			RealPath:   filepath.Join(d.Path, bin.file),
			Active:     true,
			Executed:   false,
		})
	}
	return records
}

// binInstalls returns the commands the bin field provides, sorted by command
// name. npm --long reports a bin as a map of command name to bin file; a
// string bin, linked under the package name without its scope, is accepted
// defensively, and an absent or null bin provides nothing.
func (d npmDependency) binInstalls(packageName string) ([]binInstall, error) {
	raw := bytes.TrimSpace(d.Bin)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}

	var file string
	if err := json.Unmarshal(raw, &file); err == nil {
		return []binInstall{{command: binBaseName(packageName), file: file}}, nil
	}

	var bins map[string]string
	if err := json.Unmarshal(raw, &bins); err != nil {
		return nil, fmt.Errorf("%s: package %q has an unusable bin field: %w", inventoryErrTag, packageName, err)
	}

	installs := make([]binInstall, 0, len(bins))
	for command, binFile := range bins {
		installs = append(installs, binInstall{command: binBaseName(command), file: binFile})
	}
	slices.SortFunc(installs, func(a, b binInstall) int { return strings.Compare(a.command, b.command) })
	return installs, nil
}

// binBaseName returns the segment after the last "/": npm links a bin map
// key under its basename, and a string-form bin under the package name
// without its scope.
func binBaseName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}
