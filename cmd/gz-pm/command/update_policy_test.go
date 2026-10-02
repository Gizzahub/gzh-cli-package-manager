package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

const managersFlag = "--managers"

const miseManagerName = "mise"

const configFlag = "--config"

func policyTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	all, dry, bump, local := updateAll, updateDryRun, updateBump, updateMiseLocal
	managers, strategy, output, dir, path := updateManagers, updateStrategy, updateOutput, updateMiseDir, updateConfigPath
	tools := updateMiseTools
	t.Cleanup(func() {
		updateAll, updateDryRun, updateBump, updateMiseLocal = all, dry, bump, local
		updateManagers, updateStrategy, updateOutput, updateMiseDir, updateConfigPath = managers, strategy, output, dir, path
		updateMiseTools = tools
	})
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cmd := &cobra.Command{}
	f := cmd.Flags()
	f.BoolVar(&updateAll, "all", false, "")
	f.BoolVar(&updateDryRun, "dry-run", false, "")
	f.BoolVar(&updateBump, "bump", false, "")
	f.BoolVar(&updateMiseLocal, "mise-local", false, "")
	f.StringSliceVar(&updateMiseTools, "mise-tools", nil, "")
	f.StringVar(&updateManagers, "managers", "", "")
	f.StringVar(&updateStrategy, "strategy", "stable", "")
	f.StringVar(&updateOutput, "output", "text", "")
	f.StringVar(&updateMiseDir, "mise-dir", "", "")
	f.StringVar(&updateConfigPath, "config", "", "")
	return cmd
}

func TestUpdateCLIOverridesManagerConfigIncludingFalse(t *testing.T) {
	cmd := policyTestCommand(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "version: 1\ndefaults:\n  strategy: stable\nmanagers:\n  mise:\n    strategy: minor\n    bump: true\n    directory: /configured\n    local: true\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{configFlag, path, managersFlag, miseManagerName, "--strategy", "micro", "--bump=false", "--mise-local=false", "--mise-dir", "/explicit"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	req, err := buildUpdateRequest(cmd)
	if err != nil {
		t.Fatal(err)
	}
	policy := req.Policies[manager.ManagerMise]
	if policy.Strategy != dto.StrategyMicro || policy.Bump || policy.MiseLocal || policy.MiseDir != "/explicit" {
		t.Fatalf("policy=%#v", policy)
	}
}

func TestUpdateFlagsRejectInvalidInputsBeforeExecution(t *testing.T) {
	for _, args := range [][]string{
		{managersFlag, miseManagerName, "--strategy", "typo"},
		{"--all", "--bump"},
		{managersFlag, "mise,brew", "--bump"},
		{managersFlag, miseManagerName, "--output", "yaml"},
		{managersFlag, "mise,"},
		{managersFlag, miseManagerName, configFlag, "/missing/update-config.yaml"},
		{managersFlag, miseManagerName, "--mise-tools="},
		{},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := policyTestCommand(t)
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}
			if _, err := buildUpdateRequest(cmd); err == nil {
				t.Fatal("invalid flags accepted")
			}
		})
	}
}

func TestDefaultUpdateDoesNotImplicitlyBump(t *testing.T) {
	cmd := policyTestCommand(t)
	if err := cmd.ParseFlags([]string{managersFlag, miseManagerName}); err != nil {
		t.Fatal(err)
	}
	req, err := buildUpdateRequest(cmd)
	if err != nil || req.Bump || req.Strategy != dto.StrategyStable {
		t.Fatalf("request=%#v err=%v", req, err)
	}
}

func TestMiseScopeCLIWithOtherManagerOverride(t *testing.T) {
	cmd := policyTestCommand(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "version: 1\nmanagers:\n  brew:\n    strategy: fixed\n  mise:\n    strategy: micro\n    tools: [jq]\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.ParseFlags([]string{configFlag, path, "--all", "--mise-local", "--mise-tools", "go,pnpm"}); err != nil {
		t.Fatal(err)
	}
	req, err := buildUpdateRequest(cmd)
	if err != nil {
		t.Fatal(err)
	}
	brew := req.Policies[manager.ManagerHomebrew]
	if brew.MiseLocal || brew.MiseDir != "" || len(brew.MiseTools) != 0 || brew.Strategy != dto.StrategyFixed {
		t.Fatalf("mise scope leaked into brew config: %#v", brew)
	}
	mise := req.Policies[manager.ManagerMise]
	if !mise.MiseLocal || strings.Join(mise.MiseTools, ",") != "go,pnpm" {
		t.Fatalf("mise scope missing: %#v", mise)
	}
}
