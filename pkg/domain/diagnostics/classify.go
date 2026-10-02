// Package diagnostics classifies resolved install records into per-command
// findings without touching the machine.
//
// The classifier is a pure domain component: it never resolves PATH, runs a
// package manager, or prints a report. Every result is derived entirely from
// the records and the optional owner policy a caller supplies.
//
// Design Principles:
//   - Pure functions over the provided records
//   - Go standard library only
//   - Facts over instructions: findings describe what is installed and
//     never recommend removing an install
package diagnostics

import (
	"fmt"
	"sort"
	"strings"
)

// ProviderKind groups a provider into the categories the classifier reasons
// about.
type ProviderKind string

const (
	// KindVersionManager marks providers that manage several tool versions.
	KindVersionManager ProviderKind = "version-manager"

	// KindSystemOrLanguage marks system and language package managers.
	KindSystemOrLanguage ProviderKind = "system-or-language"
)

// FindingKind is the severity a command classification ends at.
type FindingKind string

const (
	// FindingNone reports no cross-provider duplicate.
	FindingNone FindingKind = "none"

	// FindingWarning reports a cross-provider duplicate worth attention.
	FindingWarning FindingKind = "warning"

	// FindingNote reports an intentional provider split, not a duplicate.
	FindingNote FindingKind = "note"
)

// VersionComparison states how the executed version relates to another
// provider's version.
type VersionComparison string

const (
	// ComparisonSame marks versions that compare numerically equal.
	ComparisonSame VersionComparison = "same"

	// ComparisonExecutedOlder marks the executed version as the older one.
	ComparisonExecutedOlder VersionComparison = "executed-older"

	// ComparisonExecutedNewer marks the executed version as the newer one.
	ComparisonExecutedNewer VersionComparison = "executed-newer"

	// ComparisonIncomparable marks versions the comparator cannot order.
	// Incomparable versions never compare as equal.
	ComparisonIncomparable VersionComparison = "incomparable"
)

// InstallRecord is one already-resolved install of a command.
type InstallRecord struct {
	Command    string       // Command is the command basename this record provides.
	ProviderID string       // ProviderID names the package manager that installed it.
	Kind       ProviderKind // Kind is the provider kind of ProviderID.
	Version    string       // Version is the installed version string, possibly unparsable.
	RealPath   string       // RealPath is the resolved real path of the installed file.
	Active     bool         // Active reports whether the provider selects this install.
	Executed   bool         // Executed reports whether PATH actually runs this file.
}

// ProviderFacts describes one provider involved in a command classification.
type ProviderFacts struct {
	ProviderID string       // ProviderID names the provider.
	Kind       ProviderKind // Kind is the provider kind.
	Versions   []string     // Versions lists the distinct versions, sorted.
	RealPaths  []string     // RealPaths lists the distinct real paths, sorted.
	Active     bool         // Active reports whether any install is active.
	Executed   bool         // Executed reports whether PATH runs an install of this provider.
}

// VersionFact records how the executed provider's version compares with one
// other provider's highest comparable version.
type VersionFact struct {
	ExecutedProviderID string            // ExecutedProviderID is the provider PATH executes.
	OtherProviderID    string            // OtherProviderID is the provider compared against.
	ExecutedVersion    string            // ExecutedVersion is the executed install's version.
	OtherVersion       string            // OtherVersion is the other provider's highest comparable version, empty when none is comparable.
	Comparison         VersionComparison // Comparison is the comparison outcome.
}

// PolicyFact records what a supplied owner policy says about one provider.
type PolicyFact struct {
	ProviderID string // ProviderID is the provider the fact is about.
	Declared   bool   // Declared reports whether the policy names this provider as the owner.
}

// OwnerPolicy maps a command basename to the provider id a caller declares as
// its owner. The classifier never loads one from disk.
type OwnerPolicy map[string]string

// CommandFinding is the classification result for one command basename.
type CommandFinding struct {
	Command   string          // Command is the command basename.
	Kind      FindingKind     // Kind is none, warning, or note.
	Reasons   []string        // Reasons lists the facts behind Kind, in provider id order.
	Providers []ProviderFacts // Providers lists the involved providers, sorted by id.
	Versions  []VersionFact   // Versions compares the executed provider with every other provider.
	Policy    []PolicyFact    // Policy names declared and undeclared providers; empty without a policy entry.
}

// ClassifyCommands classifies every command represented in records and
// returns the findings sorted by command name. A nil policy reports no
// policy facts.
func ClassifyCommands(records []InstallRecord, policy OwnerPolicy) []CommandFinding {
	grouped := make(map[string][]InstallRecord, len(records))
	commands := make([]string, 0, len(records))
	for _, record := range records {
		if _, seen := grouped[record.Command]; !seen {
			commands = append(commands, record.Command)
		}
		grouped[record.Command] = append(grouped[record.Command], record)
	}
	sort.Strings(commands)

	findings := make([]CommandFinding, 0, len(commands))
	for _, command := range commands {
		findings = append(findings, classifyCommand(command, grouped[command], policy))
	}
	return findings
}

// classifyCommand collapses one command's records into providers and decides
// the finding kind from the provider layout.
func classifyCommand(command string, records []InstallRecord, policy OwnerPolicy) CommandFinding {
	providers := buildProviders(collapseInstalls(records))
	executed := executedProvider(providers)
	versions := versionFacts(executed, providers)

	kind := FindingNone
	var reasons []string
	switch {
	case len(providers) < 2:
		// One provider is never a cross-provider duplicate.
	case isVersionManagerNote(executed, providers):
		kind = FindingNote
		reasons = append(reasons, noteReason(executed))
	case hasWarningTrigger(executed, providers):
		kind = FindingWarning
		reasons = warningReasons(executed, providers, versions)
	}
	reasons = append(reasons, policyReasons(command, policy, providers)...)

	return CommandFinding{
		Command:   command,
		Kind:      kind,
		Reasons:   reasons,
		Providers: providerFacts(providers),
		Versions:  versions,
		Policy:    policyFacts(command, policy, providers),
	}
}

// install is one collapsed physical file behind a command.
type install struct {
	providerID string
	kind       ProviderKind
	version    string
	realPath   string
	active     bool
	executed   bool
}

// collapseInstalls folds records that share a command and a real path into
// one install, so shims and symlinks that resolve to the same file count
// once. The result is ordered by real path.
func collapseInstalls(records []InstallRecord) []install {
	byPath := make(map[string][]InstallRecord, len(records))
	paths := make([]string, 0, len(records))
	for _, record := range records {
		if _, seen := byPath[record.RealPath]; !seen {
			paths = append(paths, record.RealPath)
		}
		byPath[record.RealPath] = append(byPath[record.RealPath], record)
	}
	sort.Strings(paths)

	installs := make([]install, 0, len(paths))
	for _, path := range paths {
		installs = append(installs, representativeInstall(byPath[path]))
	}
	return installs
}

// representativeInstall collapses the records of one real path. The executed
// record identifies the install so the finding names what PATH actually
// runs; ties break on provider id and version for a deterministic result.
func representativeInstall(records []InstallRecord) install {
	chosen := records[0]
	for _, record := range records[1:] {
		if record.Executed && !chosen.Executed {
			chosen = record
			continue
		}
		if record.Executed == chosen.Executed &&
			(record.ProviderID < chosen.ProviderID ||
				(record.ProviderID == chosen.ProviderID && record.Version < chosen.Version)) {
			chosen = record
		}
	}
	active := false
	for _, record := range records {
		active = active || record.Active
	}
	return install{
		providerID: chosen.ProviderID,
		kind:       chosen.Kind,
		version:    chosen.Version,
		realPath:   chosen.RealPath,
		active:     active,
		executed:   chosen.Executed,
	}
}

// provider is one package manager contributing installs for a command.
type provider struct {
	id        string
	kind      ProviderKind
	versions  []string
	realPaths []string
	version   string // version of the executed install, empty when none is executed
	active    bool
	executed  bool
}

// buildProviders folds installs into providers, collapsing several versions
// of one version manager into a single provider before any comparison. The
// result is ordered by provider id.
func buildProviders(installs []install) []provider {
	byID := make(map[string][]install, len(installs))
	ids := make([]string, 0, len(installs))
	for _, ins := range installs {
		if _, seen := byID[ins.providerID]; !seen {
			ids = append(ids, ins.providerID)
		}
		byID[ins.providerID] = append(byID[ins.providerID], ins)
	}
	sort.Strings(ids)

	providers := make([]provider, 0, len(ids))
	for _, id := range ids {
		providers = append(providers, newProvider(id, byID[id]))
	}
	return providers
}

// newProvider folds one provider's installs into a single provider summary.
func newProvider(id string, installs []install) provider {
	p := provider{id: id, kind: installs[0].kind}
	versions := make(map[string]struct{}, len(installs))
	paths := make(map[string]struct{}, len(installs))
	for _, ins := range installs {
		p.active = p.active || ins.active
		if ins.executed && !p.executed {
			p.executed = true
			p.version = ins.version
		}
		versions[ins.version] = struct{}{}
		paths[ins.realPath] = struct{}{}
	}
	p.versions = sortedValues(versions)
	p.realPaths = sortedValues(paths)
	return p
}

// sortedValues orders a string set.
func sortedValues(set map[string]struct{}) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

// executedProvider returns the provider whose install PATH executes.
func executedProvider(providers []provider) *provider {
	for i := range providers {
		if providers[i].executed {
			return &providers[i]
		}
	}
	return nil
}

// isVersionManagerNote reports the intentional runtime split: PATH executes
// a version manager, every other provider is a system or language package
// manager, and no inactive extra provider exists.
func isVersionManagerNote(executed *provider, providers []provider) bool {
	if executed == nil || executed.kind != KindVersionManager {
		return false
	}
	for i := range providers {
		other := &providers[i]
		if other.id == executed.id {
			continue
		}
		if other.kind != KindSystemOrLanguage || !other.active {
			return false
		}
	}
	return true
}

// hasWarningTrigger reports whether the layout holds a duplicate trigger: a
// shadowed extra provider, an inactive extra provider, or an executed
// version older than a comparable one. After the real-path collapse two
// providers never share a real path, so an executed provider with another
// provider beside it is always a shadowing layout.
func hasWarningTrigger(executed *provider, providers []provider) bool {
	if executed != nil {
		return true
	}
	for i := range providers {
		if !providers[i].active {
			return true
		}
	}
	return false
}

// versionFacts compares the executed provider against every other provider,
// ordered by provider id.
func versionFacts(executed *provider, providers []provider) []VersionFact {
	if executed == nil {
		return nil
	}
	facts := make([]VersionFact, 0, len(providers))
	for i := range providers {
		if p := &providers[i]; p.id != executed.id {
			facts = append(facts, compareProvider(executed, p))
		}
	}
	return facts
}

// compareProvider compares the executed version with one provider's highest
// comparable version. When no version of that provider is comparable the
// outcome is incomparable, never equal.
func compareProvider(executed, p *provider) VersionFact {
	fact := VersionFact{
		ExecutedProviderID: executed.id,
		OtherProviderID:    p.id,
		ExecutedVersion:    executed.version,
		Comparison:         ComparisonIncomparable,
	}
	highest := ""
	for _, version := range p.versions {
		if moreRecent(version, highest) {
			highest = version
		}
	}
	if highest == "" {
		return fact
	}
	cmp, ok := compareVersions(executed.version, highest)
	if !ok {
		return fact
	}
	fact.OtherVersion = highest
	switch {
	case cmp < 0:
		fact.Comparison = ComparisonExecutedOlder
	case cmp > 0:
		fact.Comparison = ComparisonExecutedNewer
	default:
		fact.Comparison = ComparisonSame
	}
	return fact
}

// moreRecent reports whether a version is comparable and numerically greater
// than the current highest comparable version. An unparsable version never
// becomes the highest, so it cannot hide a comparable one.
func moreRecent(version, highest string) bool {
	if _, ok := parseVersion(version); !ok {
		return false
	}
	if highest == "" {
		return true
	}
	cmp, ok := compareVersions(version, highest)
	return ok && cmp > 0
}

// providerFacts snapshots the collapsed providers in provider id order.
func providerFacts(providers []provider) []ProviderFacts {
	facts := make([]ProviderFacts, 0, len(providers))
	for i := range providers {
		p := &providers[i]
		facts = append(facts, ProviderFacts{
			ProviderID: p.id,
			Kind:       p.kind,
			Versions:   append([]string(nil), p.versions...),
			RealPaths:  append([]string(nil), p.realPaths...),
			Active:     p.active,
			Executed:   p.executed,
		})
	}
	return facts
}

// declaredProvider looks up the declared owner of one command.
func declaredProvider(command string, policy OwnerPolicy) (string, bool) {
	if policy == nil {
		return "", false
	}
	declared, ok := policy[command]
	if !ok || declared == "" {
		return "", false
	}
	return declared, true
}

// policyReasons states the owner-policy facts for one command. A missing
// policy entry contributes nothing; a present one only names providers.
func policyReasons(command string, policy OwnerPolicy, providers []provider) []string {
	declared, ok := declaredProvider(command, policy)
	if !ok {
		return nil
	}
	reasons := []string{
		fmt.Sprintf("policy declares provider %q as the owner of command %q", declared, command),
	}
	for i := range providers {
		if providers[i].id != declared {
			reasons = append(reasons, fmt.Sprintf("policy does not declare provider %q", providers[i].id))
		}
	}
	return reasons
}

// policyFacts names the declared provider and every undeclared provider of
// one command, sorted by provider id. Without a policy entry nothing is
// reported.
func policyFacts(command string, policy OwnerPolicy, providers []provider) []PolicyFact {
	declared, ok := declaredProvider(command, policy)
	if !ok {
		return nil
	}
	facts := []PolicyFact{{ProviderID: declared, Declared: true}}
	for i := range providers {
		if providers[i].id != declared {
			facts = append(facts, PolicyFact{ProviderID: providers[i].id, Declared: false})
		}
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].ProviderID < facts[j].ProviderID })
	return facts
}

// noteReason explains the intentional version-manager split as a fact.
func noteReason(executed *provider) string {
	return fmt.Sprintf("the executed provider %q is a version manager and every other provider is a system or language package manager", executed.id)
}

// versionFactFor finds the comparison fact for one provider pair.
func versionFactFor(versions []VersionFact, executedID, otherID string) *VersionFact {
	for i := range versions {
		if versions[i].ExecutedProviderID == executedID && versions[i].OtherProviderID == otherID {
			return &versions[i]
		}
	}
	return nil
}

// warningReasons collects the duplicate facts behind a warning in provider
// id order.
func warningReasons(executed *provider, providers []provider, versions []VersionFact) []string {
	reasons := []string{}
	for i := range providers {
		p := &providers[i]
		if executed != nil && p.id == executed.id {
			continue
		}
		if !p.active {
			reasons = append(reasons, fmt.Sprintf("provider %q is inactive", p.id))
		}
		if executed == nil {
			continue
		}
		reasons = append(reasons, fmt.Sprintf("provider %q is shadowed by the executed provider %q", p.id, executed.id))
		if fact := versionFactFor(versions, executed.id, p.id); fact != nil && fact.Comparison == ComparisonSame {
			reasons = append(reasons, fmt.Sprintf("provider %q reports the same version as the executed provider %q", p.id, executed.id))
		}
	}
	for _, fact := range versions {
		switch fact.Comparison {
		case ComparisonExecutedOlder:
			reasons = append(reasons, fmt.Sprintf("executed provider %q version %s is older than provider %q version %s",
				fact.ExecutedProviderID, fact.ExecutedVersion, fact.OtherProviderID, fact.OtherVersion))
		case ComparisonIncomparable:
			reasons = append(reasons, fmt.Sprintf("provider %q versions [%s] are not comparable with the executed version %q; the duplicate is still reported",
				fact.OtherProviderID, strings.Join(providerVersions(providers, fact.OtherProviderID), ", "), fact.ExecutedVersion))
		}
	}
	return reasons
}

// providerVersions returns the distinct versions one provider reported.
func providerVersions(providers []provider, id string) []string {
	for i := range providers {
		if providers[i].id == id {
			return providers[i].versions
		}
	}
	return nil
}

// compareVersions orders two dotted numeric version strings, each with an
// optional leading v. The bool result is false when either string is not a
// dotted numeric version; the caller must then treat the pair as
// incomparable rather than equal.
func compareVersions(a, b string) (int, bool) {
	left, ok := parseVersion(a)
	if !ok {
		return 0, false
	}
	right, ok := parseVersion(b)
	if !ok {
		return 0, false
	}
	width := len(left)
	if len(right) > width {
		width = len(right)
	}
	for i := 0; i < width; i++ {
		if cmp := compareSegment(left, right, i); cmp != 0 {
			return cmp, true
		}
	}
	return 0, true
}

// compareSegment compares one numeric component; missing components count
// as zero. Components are zero-stripped, so longer is greater and equal
// length compares lexicographically.
func compareSegment(left, right []string, i int) int {
	a, b := "0", "0"
	if i < len(left) {
		a = left[i]
	}
	if i < len(right) {
		b = right[i]
	}
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	if a != b {
		if a < b {
			return -1
		}
		return 1
	}
	return 0
}

// parseVersion splits a dotted numeric version into zero-stripped numeric
// components. It fails for anything else, including empty strings and
// components with non-digit suffixes.
func parseVersion(version string) ([]string, bool) {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "v")
	trimmed = strings.TrimPrefix(trimmed, "V")
	if trimmed == "" {
		return nil, false
	}
	components := strings.Split(trimmed, ".")
	for i, component := range components {
		if !isNumeric(component) {
			return nil, false
		}
		components[i] = strings.TrimLeft(component, "0")
		if components[i] == "" {
			components[i] = "0"
		}
	}
	return components, true
}

// isNumeric reports whether the string is one or more ASCII digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
