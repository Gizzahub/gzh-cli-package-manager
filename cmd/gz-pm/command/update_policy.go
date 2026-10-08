package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/config"
)

func buildUpdateRequest(cmd *cobra.Command) (*dto.UpdateRequest, error) {
	if err := validateUpdateFlags(cmd); err != nil {
		return nil, err
	}
	cfg, err := loadUpdateConfig(cmd)
	if err != nil {
		return nil, fmt.Errorf("load update preferences: %w", err)
	}
	policy := resolveUpdatePolicy(cmd, cfg.Resolve(""))
	if validationErr := dto.ValidateUpdateStrategy(policy.Strategy); validationErr != nil {
		return nil, fmt.Errorf("validate update flags: %w", validationErr)
	}
	req := &dto.UpdateRequest{
		All:             updateAll,
		DryRun:          updateDryRun,
		Strategy:        policy.Strategy,
		Bump:            policy.Bump,
		MiseDir:         policy.MiseDir,
		MiseLocal:       policy.MiseLocal,
		MiseTools:       policy.MiseTools,
		PipAllowConda:   updatePipAllowConda,
		CheckDuplicates: updateCheckDuplicates,
		Policies:        make(map[manager.ManagerID]dto.UpdatePolicy, len(cfg.Managers)),
	}
	req.ManagerIDs, err = parseUpdateManagers(updateManagers)
	if err != nil {
		return nil, err
	}
	if !req.All && len(req.ManagerIDs) == 0 {
		return nil, fmt.Errorf("either --all or --managers is required")
	}
	if policy.Bump && (req.All || len(req.ManagerIDs) != 1 || req.ManagerIDs[0] != manager.ManagerMise) {
		return nil, fmt.Errorf("--bump requires --managers mise; configure managers.mise.bump for --all")
	}
	for id := range cfg.Managers {
		override := resolveUpdatePolicy(cmd, cfg.Resolve(id))
		if id != manager.ManagerMise {
			override.MiseDir, override.MiseLocal = "", false
			override.MiseTools = nil
		}
		req.Policies[id] = override
	}
	return req, nil
}

func validateUpdateFlags(cmd *cobra.Command) error {
	if cmd.Flags().Changed("mise-tools") && len(updateMiseTools) == 0 {
		return fmt.Errorf("--mise-tools must select at least one declared tool")
	}
	if updateOutput != outputFormatText && updateOutput != outputFormatJSON {
		return fmt.Errorf("unknown output format %q (supported: text, json)", updateOutput)
	}
	return nil
}

func loadUpdateConfig(cmd *cobra.Command) (*config.UpdateConfig, error) {
	path := updateConfigPath
	if path == "" {
		var err error
		path, err = config.DefaultPath()
		if err != nil {
			return nil, fmt.Errorf("resolve update config: %w", err)
		}
	}
	cfg, err := config.Load(path, cmd.Flags().Changed("config"))
	if err != nil {
		return nil, fmt.Errorf("load update config: %w", err)
	}
	return cfg, nil
}

func parseUpdateManagers(value string) ([]manager.ManagerID, error) {
	if value == "" {
		return nil, nil
	}
	ids := make([]manager.ManagerID, 0)
	for _, part := range strings.Split(value, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			return nil, fmt.Errorf("manager list contains an empty name")
		}
		ids = append(ids, manager.ManagerID(id))
	}
	return ids, nil
}

func resolveUpdatePolicy(cmd *cobra.Command, policy dto.UpdatePolicy) dto.UpdatePolicy {
	if cmd.Flags().Changed("strategy") {
		policy.Strategy = dto.UpdateStrategy(updateStrategy)
	}
	if cmd.Flags().Changed("bump") {
		policy.Bump = updateBump
	}
	if cmd.Flags().Changed("mise-dir") {
		policy.MiseDir = updateMiseDir
	}
	if cmd.Flags().Changed("mise-local") {
		policy.MiseLocal = updateMiseLocal
	}
	if cmd.Flags().Changed("mise-tools") {
		policy.MiseTools = updateMiseTools
	}
	return policy
}
