package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
)

var childCmd = &cobra.Command{
	Use:   "child CONTAINER_ID",
	Short: "Internal command used to set up a container environment",
	Long: `This is an internal command used by 'yc run' to set up Namespaces, 
mount points, and root filesystem for a container. Not intended for direct use.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		logger := utils.DefaultLogger
		logger.Info("Child process started for container %s", containerID)

		if err := container.ChildProcess(containerID); err != nil {
			logger.Error("Child process error: %v", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(childCmd)
}