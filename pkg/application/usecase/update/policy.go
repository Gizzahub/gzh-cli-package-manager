package update

import (
	"fmt"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

func (uc *UseCase) updateOptions(id manager.ManagerID, req *dto.UpdateRequest) adapterm.UpdateOptions {
	policy := dto.UpdatePolicy{Strategy: req.Strategy, Bump: req.Bump, MiseDir: req.MiseDir, MiseLocal: req.MiseLocal, MiseTools: req.MiseTools}
	if id != manager.ManagerMise {
		policy.MiseDir, policy.MiseLocal = "", false
		policy.MiseTools = nil
	}
	if override, ok := req.Policies[id]; ok {
		policy = override
	}
	return adapterm.UpdateOptions{
		Strategy:  uc.convertStrategy(policy.Strategy),
		Bump:      policy.Bump,
		DryRun:    req.DryRun,
		MiseDir:   policy.MiseDir,
		MiseLocal: policy.MiseLocal,
		Packages:  policy.MiseTools,
	}
}

// Validate every selected policy before starting any manager update.
func (uc *UseCase) validatePolicies(managers []*manager.Manager, req *dto.UpdateRequest) error {
	if err := validateMiseScope(managers, req); err != nil {
		return err
	}
	for _, mgr := range managers {
		if policy, ok := req.Policies[mgr.ID]; ok {
			if err := dto.ValidateUpdateStrategy(policy.Strategy); err != nil {
				return fmt.Errorf("validate %s policy: %w", mgr.ID, err)
			}
		}
		opts := uc.updateOptions(mgr.ID, req)
		if err := validateManagerOptions(mgr.ID, opts); err != nil {
			return err
		}
	}
	return nil
}

func validateManagerOptions(id manager.ManagerID, opts adapterm.UpdateOptions) error {
	if opts.Bump && opts.Strategy == adapterm.StrategyFixed {
		return fmt.Errorf("%s: bump cannot be combined with fixed strategy", id)
	}
	if id == manager.ManagerMise {
		if err := adapterm.ValidateToolSelection(opts.Packages); err != nil {
			return fmt.Errorf("mise update: %w", err)
		}
		return nil
	}
	if opts.MiseDir != "" || opts.MiseLocal || len(opts.Packages) != 0 {
		return fmt.Errorf("%s: directory, local scope and tools are supported only by mise", id)
	}
	if opts.Bump {
		return fmt.Errorf("%s: bump is supported only by mise", id)
	}
	if opts.Strategy == adapterm.StrategyMinor || opts.Strategy == adapterm.StrategyMicro {
		return fmt.Errorf("%s does not support %s strategy; no updates executed", id, opts.Strategy)
	}
	return nil
}

func validateRequest(req *dto.UpdateRequest) error {
	if req == nil {
		return fmt.Errorf("update request is required")
	}
	if err := dto.ValidateUpdateStrategy(req.Strategy); err != nil {
		return fmt.Errorf("validate update request: %w", err)
	}
	return nil
}

func validateMiseScope(managers []*manager.Manager, req *dto.UpdateRequest) error {
	if req.MiseDir == "" && !req.MiseLocal && len(req.MiseTools) == 0 {
		return nil
	}
	for _, mgr := range managers {
		if mgr.ID == manager.ManagerMise {
			return nil
		}
	}
	return fmt.Errorf("mise scope options require mise among the selected installed managers")
}
