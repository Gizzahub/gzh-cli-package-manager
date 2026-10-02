package mise

import (
	"context"
	"sort"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/diagnostics"
	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

// The mise adapter contributes a read-only command inventory.
var _ adapterm.CommandInventory = (*Adapter)(nil)

const (
	// inventoryQuery is the mise listing that reports every known tool,
	// installed or not and active in the current context or not.
	inventoryQuery = "ls"

	// inventoryAllJSON selects the full listing as JSON.
	inventoryAllJSON = "--json"
)

// Commands reports one install record per installed tool version, including
// tools that are installed but not active. The query is separate from list,
// whose installed-and-active filter serves the update listing. The command
// basename is the tool name, every record stays a version-manager install
// with no resolved path, and nothing is installed, updated, or removed here.
func (a *Adapter) Commands(ctx context.Context) ([]diagnostics.InstallRecord, error) {
	r, execErr := a.executor.Execute(ctx, command, a.args(adapterm.UpdateOptions{}, inventoryQuery, inventoryAllJSON)...)
	if err := cmdutil.CheckResult(r, execErr, "list installed mise tools"); err != nil {
		return nil, err
	}
	tools, err := decodeTools(r.Stdout)
	if err != nil {
		return nil, err
	}
	return inventoryRecords(tools), nil
}

// inventoryRecords folds decoded tools into install records. Only installed
// tools become records, and sorting keeps the result deterministic when the
// grouped JSON form arrives in map order.
func inventoryRecords(tools []miseTool) []diagnostics.InstallRecord {
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Name != tools[j].Name {
			return tools[i].Name < tools[j].Name
		}
		return tools[i].Version < tools[j].Version
	})
	records := make([]diagnostics.InstallRecord, 0, len(tools))
	for _, tool := range tools {
		if !tool.Installed {
			continue
		}
		records = append(records, diagnostics.InstallRecord{
			Command:    tool.Name,
			ProviderID: command,
			Kind:       diagnostics.KindVersionManager,
			Version:    tool.Version,
			Active:     tool.Active,
		})
	}
	return records
}
