package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
)

type boundedItem struct {
	Manager manager.ManagerID `json:"manager"`
	Package string            `json:"package"`
	Status  string            `json:"status"`
	Error   string            `json:"error,omitempty"`
}

type boundedState struct {
	Fingerprint string        `json:"fingerprint"`
	Items       []boundedItem `json:"items"`
}

const (
	boundedComplete = "complete"
	boundedDeferred = "deferred"
	boundedFailed   = "failed"
	boundedPending  = "pending"
	boundedRunning  = "running"
)

//nolint:gocognit,gocyclo,cyclop // The sequential state machine keeps every mutation boundary visible.
func (uc *UseCase) updateBounded(ctx context.Context, managers []*manager.Manager, req *dto.UpdateRequest) (*dto.UpdateResponse, error) {
	if err := validateBounded(managers, req); err != nil {
		return nil, err
	}
	unlock, err := lockBounded(req.Bounded.StateFile)
	if err != nil {
		return nil, err
	}
	defer unlock()
	state, err := loadBounded(req.Bounded.StateFile, managers, req)
	if err != nil {
		return nil, err
	}
	if req.DryRun {
		return uc.dryRunBounded(ctx, state, req)
	}
	started, deadline := 0, time.Time{}
	var results []*dto.ManagerUpdateResult
	if req.Bounded.MaxDuration > 0 {
		deadline = time.Now().Add(req.Bounded.MaxDuration)
	}
	for index := range state.Items {
		item := &state.Items[index]
		if item.Status == boundedComplete {
			continue
		}
		if started >= req.Bounded.BatchSize || (!deadline.IsZero() && time.Now().After(deadline)) {
			item.Status, item.Error = boundedDeferred, "batch or time limit reached"
			continue
		}
		if err := checkResources(req.Bounded); err != nil {
			item.Status, item.Error = boundedDeferred, err.Error()
			return saveAndRespond(req.Bounded.StateFile, state, results)
		}
		item.Status, item.Error = boundedRunning, ""
		if err := saveBounded(req.Bounded.StateFile, state); err != nil {
			return nil, err
		}
		result := uc.updateOne(ctx, item, req)
		result.Name = string(item.Manager) + " " + item.Package
		results = append(results, result)
		started++
		if !result.Success {
			item.Status, item.Error = boundedFailed, result.Error
			return saveAndRespond(req.Bounded.StateFile, state, results)
		}
		item.Status = boundedComplete
		if err := saveBounded(req.Bounded.StateFile, state); err != nil {
			return nil, err
		}
	}
	if err := saveBounded(req.Bounded.StateFile, state); err != nil {
		return nil, err
	}
	return boundedResponse(state, results, false), nil
}

func (uc *UseCase) dryRunBounded(ctx context.Context, state *boundedState, req *dto.UpdateRequest) (*dto.UpdateResponse, error) {
	preview := *req
	preview.DryRun = true
	var results []*dto.ManagerUpdateResult
	for index := range state.Items {
		item := &state.Items[index]
		if item.Status == boundedComplete {
			continue
		}
		one := uc.updateOne(ctx, item, &preview)
		if !one.Success {
			return nil, fmt.Errorf("preview %s %s: %s", item.Manager, item.Package, one.Error)
		}
		one.Name = string(item.Manager) + " " + item.Package
		results = append(results, one)
	}
	return boundedResponse(state, results, true), nil
}

func validateBounded(managers []*manager.Manager, req *dto.UpdateRequest) error {
	if req.Bounded.BatchSize < 1 || req.Bounded.BatchSize > 3 || req.Bounded.StateFile == "" {
		return fmt.Errorf("bounded update requires a state file and batch size from 1 to 3")
	}
	if !filepath.IsAbs(req.Bounded.StateFile) || req.Bounded.MaxDuration < 0 {
		return fmt.Errorf("bounded update requires an absolute state path and nonnegative duration")
	}
	if len(managers) == 0 || (!req.All && len(managers) != len(req.ManagerIDs)) {
		return fmt.Errorf("bounded update requires every requested manager to be installed")
	}
	for _, mgr := range managers {
		if mgr.ID != manager.ManagerMise && mgr.ID != manager.ManagerHomebrew {
			return fmt.Errorf("bounded update supports only mise and brew (got %s)", mgr.ID)
		}
		if len(boundedPackages(mgr.ID, req)) == 0 {
			return fmt.Errorf("bounded %s update requires an explicit package selection", mgr.ID)
		}
		if !req.DryRun && resolvedBoundedOptions(mgr.ID, req).Strategy == "fixed" {
			return fmt.Errorf("bounded fixed policy is report-only; use --dry-run")
		}
	}
	return nil
}

func boundedPackages(id manager.ManagerID, req *dto.UpdateRequest) []string {
	if policy, ok := req.Policies[id]; ok {
		return selectedPackages(id, &policy)
	}
	if id == manager.ManagerMise {
		return req.MiseTools
	}
	return nil
}

//nolint:nestif // The state validation intentionally stops at each unsafe condition.
func loadBounded(path string, managers []*manager.Manager, req *dto.UpdateRequest) (*boundedState, error) {
	if stat, err := os.Lstat(path); err == nil && !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("bounded update state must be a regular file")
	}
	fingerprint, err := boundedFingerprint(managers, req)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		var state boundedState
		if decodeErr := json.Unmarshal(data, &state); decodeErr != nil {
			return nil, fmt.Errorf("read bounded update state: %w", decodeErr)
		}
		if state.Fingerprint != fingerprint {
			return nil, fmt.Errorf("bounded update state does not match selected packages or policy; inspect then reset it")
		}
		if validationErr := validateStateItems(state.Items, managers, req); validationErr != nil {
			return nil, validationErr
		}
		if allComplete(state.Items) {
			return newBoundedState(fingerprint, managers, req), nil
		}
		for _, item := range state.Items {
			if item.Status == boundedRunning || item.Status == boundedFailed {
				return nil, fmt.Errorf("bounded update has %s package %s; inspect and reset state manually", item.Status, item.Package)
			}
		}
		return &state, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read bounded update state: %w", err)
	}
	return newBoundedState(fingerprint, managers, req), nil
}

func validateStateItems(items []boundedItem, managers []*manager.Manager, req *dto.UpdateRequest) error {
	expected := make(map[string]bool)
	for _, mgr := range managers {
		for _, pkg := range boundedPackages(mgr.ID, req) {
			expected[string(mgr.ID)+"/"+pkg] = true
		}
	}
	seen := make(map[string]bool)
	for _, item := range items {
		key := string(item.Manager) + "/" + item.Package
		if !expected[key] || seen[key] {
			return fmt.Errorf("bounded update state contains an unselected or duplicate package")
		}
		switch item.Status {
		case boundedPending, boundedDeferred, boundedComplete, boundedFailed, boundedRunning:
		default:
			return fmt.Errorf("bounded update state has invalid package status")
		}
		seen[key] = true
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("bounded update state does not contain every selected package")
	}
	return nil
}

func newBoundedState(fingerprint string, managers []*manager.Manager, req *dto.UpdateRequest) *boundedState {
	state := &boundedState{Fingerprint: fingerprint}
	for _, mgr := range managers {
		for _, pkg := range boundedPackages(mgr.ID, req) {
			state.Items = append(state.Items, boundedItem{Manager: mgr.ID, Package: pkg, Status: boundedPending})
		}
	}
	return state
}

func allComplete(items []boundedItem) bool {
	for _, item := range items {
		if item.Status != boundedComplete {
			return false
		}
	}
	return true
}

func boundedFingerprint(managers []*manager.Manager, req *dto.UpdateRequest) (string, error) {
	type entry struct {
		ID       manager.ManagerID
		Packages []string
		Policy   adapterm.UpdateOptions
	}
	entries := make([]entry, 0, len(managers))
	for _, mgr := range managers {
		entries = append(entries, entry{mgr.ID, append([]string(nil), boundedPackages(mgr.ID, req)...), resolvedBoundedOptions(mgr.ID, req)})
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("encode bounded update fingerprint: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func (uc *UseCase) updateOne(ctx context.Context, item *boundedItem, req *dto.UpdateRequest) *dto.ManagerUpdateResult {
	mgr := &manager.Manager{ID: item.Manager, Name: string(item.Manager), Installed: true}
	opts := uc.updateOptions(item.Manager, req)
	opts.Packages = []string{item.Package}
	return uc.updateManager(ctx, mgr, opts)
}

func saveAndRespond(path string, state *boundedState, results []*dto.ManagerUpdateResult) (*dto.UpdateResponse, error) {
	if err := saveBounded(path, state); err != nil {
		return nil, err
	}
	return boundedResponse(state, results, false), nil
}

func saveBounded(path string, state *boundedState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create bounded update state directory: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode bounded update state: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".gz-pm-state-*")
	if err != nil {
		return fmt.Errorf("create bounded update state: %w", err)
	}
	temp := file.Name()
	defer func() { _ = os.Remove(temp) }()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write bounded update state: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close bounded update state: %w", err)
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("replace bounded update state: %w", err)
	}
	return nil
}
