//go:build windows

package update

import (
	"fmt"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
)

func lockBounded(string) (func(), error) {
	return nil, fmt.Errorf("bounded updates are not supported on Windows")
}

func checkResources(options dto.BoundedUpdateOptions) error {
	if options.MinFreeGiB != 0 || options.MaxSwapGiB != 0 {
		return fmt.Errorf("bounded resource checks are not supported on Windows")
	}
	return nil
}
