package main

import (
	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
)

var execCmd = &cobra.Command{
	Use:   "exec CONTAINER COMMAND [ARG...]",
	Short: "Execute a command in a running container",
	Long:  `Runs a command inside a running container, similar to docker exec.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		containerID := args[0]
		cmdArgs := args[1:]

		mgr := container.NewManager(logger)
		lifecycle := container.NewLifecycleManager(mgr, logger)

		return lifecycle.Exec(containerID, cmdArgs)
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}