package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
)

var rmCmd = &cobra.Command{
	Use:   "rm CONTAINER [CONTAINER...]",
	Short: "Remove one or more containers",
	Long:  `Removes stopped containers and their data.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		mgr := container.NewManager(logger)

		for _, containerID := range args {
			if !mgr.Exists(containerID) {
				logger.Warn("Container %s does not exist", containerID)
				continue
			}

			if err := mgr.Delete(containerID); err != nil {
				return fmt.Errorf("remove container %s: %w", containerID, err)
			}
			logger.Info("Removed container %s", containerID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}