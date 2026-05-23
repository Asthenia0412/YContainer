package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "yc",
	Short: "YContainer - A lightweight container runtime with sidecar support",
	Long: `YContainer is a Docker-compatible container runtime 
that extends into sidecar-based infrastructure capabilities.
Supports container lifecycle, image management, pod management,
and built-in sidecar proxy for rate-limiting, auth, and circuit breaking.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}