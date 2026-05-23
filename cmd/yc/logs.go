package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
)

var logsCmd = &cobra.Command{
	Use:   "logs CONTAINER",
	Short: "Fetch the logs of a container",
	Long:  `Prints the log output of a container.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		containerID := args[0]

		mgr := container.NewManager(logger)
		if !mgr.Exists(containerID) {
			return fmt.Errorf("container %s not found", containerID)
		}

		logPath := filepath.Join(container.YCDataDir, "containers", containerID, "log", "stdout.log")
		data, err := os.ReadFile(logPath)
		if err != nil {
			return fmt.Errorf("read log: %w", err)
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}