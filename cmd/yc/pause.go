package main

import (
	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/pod"
)

var pauseCmd = &cobra.Command{
	Use:   "pause POD_ID",
	Short: "Internal: pause container process",
	Long: `Internal command that starts the pause container for a pod. 
Not intended for direct use.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		logger.Info("Pause container started for pod %s", args[0])
		return pod.PauseProcess()
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
}