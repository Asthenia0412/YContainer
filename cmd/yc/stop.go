package main

import (
	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
)

var stopCmd = &cobra.Command{
	Use:   "stop CONTAINER",
	Short: "Stop a running container",
	Long:  `Stops a running container by sending SIGTERM.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		containerID := args[0]

		mgr := container.NewManager(logger)
		lifecycle := container.NewLifecycleManager(mgr, logger)

		c, err := mgr.Load(containerID)
		if err != nil {
			return err
		}

		return lifecycle.Stop(c)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}