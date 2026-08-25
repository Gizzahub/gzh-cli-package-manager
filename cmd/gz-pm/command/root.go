package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "gz-pm",
	Short: "Package Manager Control - Unified package manager orchestration",
	Long: `gz-pm (Package Manager Control) is a CLI tool that orchestrates multiple
package managers (Homebrew, asdf, npm, pip, etc.) through a unified interface.

It provides a single, consistent interface to manage all your package managers,
making it easy to update, configure, and maintain your development environment.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewRootCmd returns a copy of the root command for embedding in other CLIs.
// This allows other projects to integrate gz-pm as a subcommand.
//
// Example usage:
//
//	pmCmd := command.NewRootCmd()
//	pmCmd.Use = "pm"  // Customize the command name
//	rootCmd.AddCommand(pmCmd)
func NewRootCmd() *cobra.Command {
	// Return a copy of the configured root command
	return rootCmd
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gz-pm.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
