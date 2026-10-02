package diagnostics

import (
	"strings"
	"testing"
)

const (
	brewNode      = "/opt/homebrew/bin/node"
	miseNode20    = "/home/user/.local/share/mise/installs/node/20/bin/node"
	brewCodex     = "/opt/homebrew/bin/codex"
	npmCodex      = "/usr/local/lib/node_modules/codex/bin/codex"
	providerBrew  = "brew"
	providerMise  = "mise"
	providerNpm   = "npm"
	version1      = "1.0.0"
	version20     = "20.0.0"
	version22     = "22.0.0"
	commandNode   = "node"
	commandCodex  = "codex"
	commandClaude = "claude"
)

// TestClassifyCommandIdentity covers the command identity labels from the
// card: when two providers duplicate a command, when a duplicate is not a
// duplicate, and what the owner policy adds.
func TestClassifyCommandIdentity(t *testing.T) {
	tests := []struct {
		name    string
		policy  OwnerPolicy
		records []InstallRecord
		check   func(t *testing.T, findings []CommandFinding)
	}{
		{
			name: "same-realpath",
			records: []InstallRecord{
				{Command: commandNode, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: version22, RealPath: brewNode, Active: true, Executed: true},
				{Command: commandNode, ProviderID: providerMise, Kind: KindVersionManager, Version: version22, RealPath: brewNode, Active: true},
			},
			check: checkSameRealpath,
		},
		{
			name: "inactive-provider",
			records: []InstallRecord{
				{Command: commandNode, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: version22, RealPath: brewNode, Active: true},
				{Command: commandNode, ProviderID: providerMise, Kind: KindVersionManager, Version: version20, RealPath: miseNode20},
			},
			check: checkInactiveProvider,
		},
		{
			name:    "shadowed-same-version",
			records: codexRecords(),
			check:   checkShadowedSameVersion,
		},
		{
			name: "older-executed",
			records: []InstallRecord{
				{Command: commandNode, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: "18.0.0", RealPath: "/usr/bin/node", Active: true, Executed: true},
				{Command: commandNode, ProviderID: providerNpm, Kind: KindSystemOrLanguage, Version: version20, RealPath: "/usr/local/lib/node_modules/node/bin/node", Active: true},
			},
			check: checkOlderExecuted,
		},
		{
			name: "single-provider",
			records: []InstallRecord{
				{Command: "grok", ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: version1, RealPath: "/opt/homebrew/bin/grok", Active: true, Executed: true},
			},
			check: checkSingleProvider,
		},
		{
			name: "version-manager-note",
			records: []InstallRecord{
				{Command: commandNode, ProviderID: providerMise, Kind: KindVersionManager, Version: version20, RealPath: miseNode20, Active: true, Executed: true},
				{Command: commandNode, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: "21.0.0", RealPath: brewNode, Active: true},
			},
			check: checkVersionManagerNote,
		},
		{
			name: "incomparable-version",
			records: []InstallRecord{
				{Command: commandClaude, ProviderID: "native", Kind: KindSystemOrLanguage, Version: "abc", RealPath: "/usr/local/bin/claude", Active: true, Executed: true},
				{Command: commandClaude, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: "1.2.3", RealPath: "/opt/homebrew/bin/claude", Active: true},
			},
			check: checkIncomparableVersion,
		},
		{
			name:    "owner-policy-facts",
			policy:  OwnerPolicy{commandCodex: providerBrew},
			records: codexRecords(),
			check:   checkOwnerPolicyFacts,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, ClassifyCommands(tt.records, tt.policy))
		})
	}
}

func checkSameRealpath(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandNode)
	if finding.Kind != FindingNone {
		t.Errorf("Kind = %q, want %q: one real path is one install", finding.Kind, FindingNone)
	}
	if len(finding.Providers) != 1 {
		t.Fatalf("len(Providers) = %d, want 1 after the real-path collapse", len(finding.Providers))
	}
	if got := finding.Providers[0].ProviderID; got != providerBrew {
		t.Errorf("Providers[0].ProviderID = %q, want the executed provider brew", got)
	}
}

func checkInactiveProvider(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandNode)
	if finding.Kind != FindingWarning {
		t.Fatalf("Kind = %q, want %q: an inactive second provider warns", finding.Kind, FindingWarning)
	}
	assertReasonContains(t, finding.Reasons, `provider "mise" is inactive`)
	if mise := providerFact(t, finding, providerMise); mise.Active {
		t.Errorf("mise Active = true, want false")
	}
}

func checkShadowedSameVersion(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandCodex)
	if finding.Kind != FindingWarning {
		t.Fatalf("Kind = %q, want %q: a shadowed same-version duplicate warns", finding.Kind, FindingWarning)
	}
	fact := versionFact(t, finding, providerNpm)
	if fact.Comparison != ComparisonSame {
		t.Errorf("Comparison = %q, want %q", fact.Comparison, ComparisonSame)
	}
	if fact.ExecutedProviderID != providerBrew || fact.ExecutedVersion != version1 {
		t.Errorf("fact = %+v, want brew 1.0.0 as the executed side", fact)
	}
	assertReasonContains(t, finding.Reasons, `provider "npm" is shadowed by the executed provider "brew"`)
	assertReasonContains(t, finding.Reasons, `provider "npm" reports the same version`)
}

func checkOlderExecuted(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandNode)
	if finding.Kind != FindingWarning {
		t.Fatalf("Kind = %q, want %q: an older executed version warns", finding.Kind, FindingWarning)
	}
	fact := versionFact(t, finding, providerNpm)
	if fact.Comparison != ComparisonExecutedOlder {
		t.Errorf("Comparison = %q, want %q", fact.Comparison, ComparisonExecutedOlder)
	}
	if fact.OtherVersion != version20 {
		t.Errorf("OtherVersion = %q, want 20.0.0", fact.OtherVersion)
	}
	assertReasonContains(t, finding.Reasons, `version 18.0.0 is older than provider "npm" version 20.0.0`)
}

func checkSingleProvider(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, "grok")
	if finding.Kind != FindingNone {
		t.Errorf("Kind = %q, want %q: one provider produces no finding", finding.Kind, FindingNone)
	}
	if len(finding.Providers) != 1 || len(finding.Versions) != 0 || len(finding.Reasons) != 0 || len(finding.Policy) != 0 {
		t.Errorf("finding = %+v, want one provider and no versions, reasons, or policy facts", finding)
	}
}

func checkVersionManagerNote(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandNode)
	if finding.Kind != FindingNote {
		t.Fatalf("Kind = %q, want %q: the version-manager split is a note, not a warning", finding.Kind, FindingNote)
	}
	if len(finding.Providers) != 2 {
		t.Errorf("len(Providers) = %d, want 2", len(finding.Providers))
	}
	assertNoRemovalText(t, finding.Reasons)
	assertReasonContains(t, finding.Reasons, `the executed provider "mise" is a version manager`)
}

func checkIncomparableVersion(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandClaude)
	if finding.Kind != FindingWarning {
		t.Fatalf("Kind = %q, want %q: an incomparable duplicate is still reported", finding.Kind, FindingWarning)
	}
	fact := versionFact(t, finding, providerBrew)
	if fact.Comparison != ComparisonIncomparable {
		t.Errorf("Comparison = %q, want %q, never %q", fact.Comparison, ComparisonIncomparable, ComparisonSame)
	}
	if len(finding.Providers) != 2 {
		t.Errorf("len(Providers) = %d, want 2: the duplicate still reports", len(finding.Providers))
	}
	assertReasonContains(t, finding.Reasons, `provider "brew" versions [1.2.3] are not comparable with the executed version "abc"`)
}

func checkOwnerPolicyFacts(t *testing.T, findings []CommandFinding) {
	t.Helper()
	finding := findingForCommand(t, findings, commandCodex)
	if len(finding.Policy) != 2 {
		t.Fatalf("len(Policy) = %d, want 2 policy facts", len(finding.Policy))
	}
	if fact := finding.Policy[0]; fact.ProviderID != providerBrew || !fact.Declared {
		t.Errorf("Policy[0] = %+v, want brew declared", fact)
	}
	if fact := finding.Policy[1]; fact.ProviderID != providerNpm || fact.Declared {
		t.Errorf("Policy[1] = %+v, want npm undeclared", fact)
	}
	assertReasonContains(t, finding.Reasons, `policy declares provider "brew" as the owner of command "codex"`)
	assertReasonContains(t, finding.Reasons, `policy does not declare provider "npm"`)

	noPolicy := findingForCommand(t, ClassifyCommands(codexRecords(), nil), commandCodex)
	if len(noPolicy.Policy) != 0 {
		t.Errorf("len(Policy) = %d without a policy, want 0", len(noPolicy.Policy))
	}
	for _, reason := range noPolicy.Reasons {
		if strings.Contains(reason, "policy") {
			t.Errorf("reason %q mentions a policy that was not supplied", reason)
		}
	}
	assertNoRemovalText(t, noPolicy.Reasons)
}

// TestClassifyCommandsAreSortedByCommand locks the deterministic output order.
func TestClassifyCommandsAreSortedByCommand(t *testing.T) {
	findings := ClassifyCommands([]InstallRecord{
		{Command: "zsh", ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: "5.9", RealPath: "/bin/zsh", Active: true, Executed: true},
		{Command: commandNode, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: version22, RealPath: brewNode, Active: true, Executed: true},
	}, nil)
	if len(findings) != 2 {
		t.Fatalf("len(findings) = %d, want 2", len(findings))
	}
	if findings[0].Command != commandNode || findings[1].Command != "zsh" {
		t.Errorf("commands = [%s, %s], want sorted [node, zsh]", findings[0].Command, findings[1].Command)
	}
}

// TestUnparsableVersionDoesNotHideComparableOne keeps an unparsable version
// that sorts first from masking a comparable version of the same provider.
func TestUnparsableVersionDoesNotHideComparableOne(t *testing.T) {
	findings := ClassifyCommands([]InstallRecord{
		{Command: commandNode, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: "18.0.0", RealPath: brewNode, Active: true, Executed: true},
		{Command: commandNode, ProviderID: providerMise, Kind: KindVersionManager, Version: "abc", RealPath: "/mise/abc/node", Active: true},
		{Command: commandNode, ProviderID: providerMise, Kind: KindVersionManager, Version: "v20.0.0", RealPath: "/mise/20/node", Active: true},
	}, nil)
	finding := findingForCommand(t, findings, commandNode)
	fact := versionFact(t, finding, providerMise)
	if fact.Comparison != ComparisonExecutedOlder || fact.OtherVersion != "v20.0.0" {
		t.Errorf("fact = %+v, want executed-older against v20.0.0", fact)
	}
}

// codexRecords is the shadowed same-version layout: Homebrew codex on PATH
// and npm codex at the same version behind it.
func codexRecords() []InstallRecord {
	return []InstallRecord{
		{Command: commandCodex, ProviderID: providerBrew, Kind: KindSystemOrLanguage, Version: version1, RealPath: brewCodex, Active: true, Executed: true},
		{Command: commandCodex, ProviderID: providerNpm, Kind: KindSystemOrLanguage, Version: version1, RealPath: npmCodex, Active: true},
	}
}

// findingForCommand returns the finding for one command.
func findingForCommand(t *testing.T, findings []CommandFinding, command string) *CommandFinding {
	t.Helper()
	for i := range findings {
		if findings[i].Command == command {
			return &findings[i]
		}
	}
	t.Fatalf("no finding for command %q", command)
	return nil
}

// providerFact returns the provider facts for one provider id.
func providerFact(t *testing.T, finding *CommandFinding, providerID string) ProviderFacts {
	t.Helper()
	for _, fact := range finding.Providers {
		if fact.ProviderID == providerID {
			return fact
		}
	}
	t.Fatalf("no provider fact for %q", providerID)
	return ProviderFacts{}
}

// versionFact returns the version fact comparing against one provider.
func versionFact(t *testing.T, finding *CommandFinding, otherProviderID string) VersionFact {
	t.Helper()
	for _, fact := range finding.Versions {
		if fact.OtherProviderID == otherProviderID {
			return fact
		}
	}
	t.Fatalf("no version fact for provider %q", otherProviderID)
	return VersionFact{}
}

// assertReasonContains fails when no reason carries the expected fact.
func assertReasonContains(t *testing.T, reasons []string, want string) {
	t.Helper()
	for _, reason := range reasons {
		if strings.Contains(reason, want) {
			return
		}
	}
	t.Errorf("no reason contains %q; reasons = %q", want, reasons)
}

// assertNoRemovalText fails when a reason recommends removing an install.
func assertNoRemovalText(t *testing.T, reasons []string) {
	t.Helper()
	for _, reason := range reasons {
		lower := strings.ToLower(reason)
		if strings.Contains(lower, "uninstall") || strings.Contains(lower, "remove") {
			t.Errorf("reason %q recommends removal", reason)
		}
	}
}
