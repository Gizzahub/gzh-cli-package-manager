package homebrew

import (
	"context"
	"encoding/json"
	"path"
	"slices"
	"strings"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

// The Homebrew adapter provides the optional CommandInventory capability.
var _ adapterm.CommandInventory = (*Adapter)(nil)

// brewProviderID is the provider id the inventory records, taken from the
// domain ManagerID for Homebrew so the two never drift apart.
const brewProviderID = string(manager.ManagerHomebrew)

const (
	// brewBinSegment is the directory segment a brew list line must end its
	// path with to name a command file.
	brewBinSegment = "bin"

	// brewFormulaLineParts is the least number of slash-separated parts of a
	// brew list line of the form <prefix>/<name>/<version>/bin/<cmd>: the
	// empty part before the leading slash, one prefix part, and the four
	// trailing name, version, bin, and command parts.
	brewFormulaLineParts = 6
)

// brewInfoDocument is the brew info --json=v2 --installed document the
// command inventory parses.
type brewInfoDocument struct {
	Formulae []brewFormulaInfo `json:"formulae"`
	Casks    []brewCaskInfo    `json:"casks"`
}

// brewFormulaInfo is one formula entry the inventory reads. A real formula
// object carries no binary list, so a formula's commands come from the brew
// list query rather than from this entry.
type brewFormulaInfo struct {
	Name      string           `json:"name"`
	Installed []brewKegInstall `json:"installed"`
	LinkedKeg string           `json:"linked_keg"`
}

// brewKegInstall is one installed keg of a formula.
type brewKegInstall struct {
	Version string `json:"version"`
}

// brewCaskInfo is one cask entry the inventory reads. The token and display
// names are package identity, never commands. Artifacts stay raw so an
// artifact shape this adapter does not know cannot fail the whole inventory.
type brewCaskInfo struct {
	Installed string            `json:"installed"`
	Artifacts []json.RawMessage `json:"artifacts"`
}

// brewCaskArtifact is one artifact entry of a cask, decoded only into its
// binary stanza and its artifact-level link target. Every other artifact
// kind, app and uninstall and plain strings, provides no command.
type brewCaskArtifact struct {
	Binary []json.RawMessage `json:"binary"`
	Target string            `json:"target"`
}

// brewBinaryOptions is the options object a binary stanza may carry after
// its source path; target renames the linked command.
type brewBinaryOptions struct {
	Target string `json:"target"`
}

// caskBinary is one binary artifact of a cask: the source path the cask
// ships, the rename target its options object may carry, and the
// artifact-level target the linked file gets.
type caskBinary struct {
	source string
	target string
	link   string
}

// commandName resolves the command basename by precedence: the rename
// target, the artifact-level link target, then the source path.
func (b caskBinary) commandName() string {
	switch {
	case b.target != "":
		return path.Base(b.target)
	case b.link != "":
		return path.Base(b.link)
	default:
		return path.Base(b.source)
	}
}

// isCommandBasename reports whether a decoded basename can name a command.
// A source of "" or "/" resolves to "." or "/" and provides nothing.
func isCommandBasename(base string) bool {
	return base != "" && base != "." && base != "/"
}

// Commands reports one install record per command binary the installed
// Homebrew packages provide, implementing the optional CommandInventory
// capability through two read-only queries: brew info --json=v2 --installed
// carries the cask artifacts and the installed formulae, and one brew list
// --formula call resolves the formula binaries. The inventory never calls
// LookPath, never scans the Homebrew prefix, and never installs, updates,
// or removes anything.
//
// A command is always a binary basename, never a formula name, a cask
// token, or a cask display name. A cask contributes one record per binary
// artifact, and a formula one record per bin file of an installed keg. A
// cask record is Active because an installed cask's binaries are linked; a
// formula record is Active only when its keg is the linked one. Executed
// stays false: PATH resolution is a separate capability.
func (a *Adapter) Commands(ctx context.Context) ([]diagnostics.InstallRecord, error) {
	result, err := a.executor.Execute(ctx, "brew", "info", "--json=v2", "--installed")
	if resultErr := cmdutil.CheckResult(result, err, "list brew commands"); resultErr != nil {
		return nil, resultErr
	}

	var document brewInfoDocument
	if err := cmdutil.UnmarshalJSON(result, &document, "parse brew commands"); err != nil {
		return nil, err
	}

	records := make([]diagnostics.InstallRecord, 0, len(document.Casks)+len(document.Formulae))
	records = appendCaskCommands(records, document.Casks)
	return a.appendFormulaCommands(ctx, records, document.Formulae)
}

// appendCaskCommands appends one record per binary artifact of every cask,
// keeping the records the caller already collected. An artifact that cannot
// be decoded, or that installs no binary, is skipped so one unknown
// artifact never fails the whole inventory.
func appendCaskCommands(records []diagnostics.InstallRecord, casks []brewCaskInfo) []diagnostics.InstallRecord {
	for i := range casks {
		for j := range casks[i].Artifacts {
			binary, ok := caskBinaryArtifact(casks[i].Artifacts[j])
			if !ok {
				continue
			}
			command := binary.commandName()
			if !isCommandBasename(command) {
				continue
			}
			records = append(records, diagnostics.InstallRecord{
				Command:    command,
				ProviderID: brewProviderID,
				Kind:       diagnostics.KindSystemOrLanguage,
				Version:    casks[i].Installed,
				RealPath:   binary.link,
				Active:     true,
			})
		}
	}
	return records
}

// caskBinaryArtifact decodes one raw cask artifact into the binary it
// installs. It reports false for artifacts a cask does not install a binary
// through: entries that are not objects, entries without a binary stanza,
// and stanzas whose first element is not a source string.
func caskBinaryArtifact(raw json.RawMessage) (caskBinary, bool) {
	var artifact brewCaskArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return caskBinary{}, false
	}
	binary, ok := decodeBinaryStanza(artifact.Binary)
	if !ok {
		return caskBinary{}, false
	}
	binary.link = artifact.Target
	return binary, true
}

// decodeBinaryStanza reads one binary stanza: element 0 is the source path
// the cask ships and any object element renames the linked command.
// Elements that are neither strings nor objects carry nothing to read.
func decodeBinaryStanza(elements []json.RawMessage) (caskBinary, bool) {
	if len(elements) == 0 {
		return caskBinary{}, false
	}
	var binary caskBinary
	if err := json.Unmarshal(elements[0], &binary.source); err != nil {
		return caskBinary{}, false
	}
	for _, element := range elements[1:] {
		var options brewBinaryOptions
		if err := json.Unmarshal(element, &options); err != nil {
			continue
		}
		if options.Target != "" {
			binary.target = options.Target
		}
	}
	return binary, true
}

// formulaIndex holds what the brew list query needs from the info document:
// the formula names to ask about, the installed keg versions a list line's
// version must match, and the linked keg a record's Active flag derives
// from.
type formulaIndex struct {
	names    []string
	versions map[string][]string
	linked   map[string]string
}

// newFormulaIndex indexes the installed formulae by name.
func newFormulaIndex(formulae []brewFormulaInfo) formulaIndex {
	index := formulaIndex{
		names:    make([]string, 0, len(formulae)),
		versions: make(map[string][]string, len(formulae)),
		linked:   make(map[string]string, len(formulae)),
	}
	for i := range formulae {
		formula := &formulae[i]
		index.names = append(index.names, formula.Name)
		kegVersions := make([]string, 0, len(formula.Installed))
		for j := range formula.Installed {
			kegVersions = append(kegVersions, formula.Installed[j].Version)
		}
		index.versions[formula.Name] = kegVersions
		index.linked[formula.Name] = formula.LinkedKeg
	}
	return index
}

// appendFormulaCommands runs the single brew list --formula query for every
// installed formula and appends one record per command binary it lists,
// keeping the records the caller already collected. Without formulae the
// query does not run at all.
func (a *Adapter) appendFormulaCommands(ctx context.Context, records []diagnostics.InstallRecord, formulae []brewFormulaInfo) ([]diagnostics.InstallRecord, error) {
	if len(formulae) == 0 {
		return records, nil
	}

	index := newFormulaIndex(formulae)
	args := slices.Concat([]string{"list", "--formula"}, index.names)
	result, err := a.executor.Execute(ctx, "brew", args...)
	if resultErr := cmdutil.CheckResult(result, err, "list brew formula binaries"); resultErr != nil {
		return nil, resultErr
	}

	for line := range strings.SplitSeq(cmdutil.ExtractStdout(result), "\n") {
		if record, ok := index.commandRecord(line); ok {
			records = append(records, record)
		}
	}
	return records, nil
}

// commandRecord turns one brew list line into an install record. Lines that
// are not the bin file of an installed keg, library files, man pages, and
// nested bin paths, report false, so a formula name never becomes a
// command.
func (f formulaIndex) commandRecord(line string) (diagnostics.InstallRecord, bool) {
	name, version, command, ok := parseFormulaListLine(line)
	if !ok || !slices.Contains(f.versions[name], version) {
		return diagnostics.InstallRecord{}, false
	}
	return diagnostics.InstallRecord{
		Command:    command,
		ProviderID: brewProviderID,
		Kind:       diagnostics.KindSystemOrLanguage,
		Version:    version,
		RealPath:   line,
		Active:     f.linked[name] == version,
	}, true
}

// parseFormulaListLine splits one brew list line of the exact form
// <prefix>/<name>/<version>/bin/<cmd> into its formula name, keg version,
// and command basename. Every other line, relative paths, deeper or
// shallower layouts, and nested bin paths like libexec/bin/x or bin/sub/x,
// reports false.
func parseFormulaListLine(line string) (name, version, command string, ok bool) {
	if !strings.HasPrefix(line, "/") {
		return "", "", "", false
	}
	parts := strings.Split(line, "/")
	if len(parts) < brewFormulaLineParts {
		return "", "", "", false
	}
	last := len(parts) - 1
	if parts[last-1] != brewBinSegment || parts[last] == "" {
		return "", "", "", false
	}
	return parts[last-3], parts[last-2], parts[last], true
}
