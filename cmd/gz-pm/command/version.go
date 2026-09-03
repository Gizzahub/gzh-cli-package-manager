package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gizzahub/gzh-cli-package-manager/internal/version"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display version information including the semantic version,
git commit hash, build date, Go version, and platform.`,
	Run: func(_ *cobra.Command, _ []string) {
		info := version.Get()
		fmt.Println(info.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
