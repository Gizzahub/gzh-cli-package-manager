package main

import (
	"fmt"
	"os"

	"github.com/gizzahub/gzh-cli-package-manager/internal/version"
)

func main() {
	// For now, just print version information
	// Full CLI implementation will come later with Cobra
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		info := version.Get()
		fmt.Println(info.String())
		os.Exit(0)
	}

	fmt.Println("pmctl - Package Manager Control")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pmctl --version    Show version information")
	fmt.Println()
	fmt.Println("Full CLI implementation coming soon...")
}
