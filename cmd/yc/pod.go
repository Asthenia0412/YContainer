package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
	"github.com/Yancy/YContainer/pkg/pod"
)

var podCmd = &cobra.Command{
	Use:   "pod COMMAND",
	Short: "Manage pods (groups of containers)",
	Long:  `Pod management: create, list, and manage multi-container groups.`,
}

var podRunCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE COMMAND [ARG...]",
	Short: "Create and run a pod with a container",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		image := args[0]
		command := args[1:]

		mgr := container.NewManager(logger)
		lifecycle := container.NewLifecycleManager(mgr, logger)
		podMgr := pod.NewManager(logger, mgr)

		p, err := podMgr.CreatePod("")
		if err != nil {
			return fmt.Errorf("create pod: %w", err)
		}

		config := container.RunConfig{
			Image: image,
			Cmd:   command,
		}

		c, err := lifecycle.Create(config)
		if err != nil {
			return fmt.Errorf("create container in pod: %w", err)
		}

		if err := podMgr.AddContainer(p, c); err != nil {
			return fmt.Errorf("add container to pod: %w", err)
		}

		logger.Info("Pod %s created with container %s", p.ID[:12], c.ID[:12])
		return nil
	},
}

var podPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List pods",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "POD ID\tSTATUS\tCONTAINERS")
		w.Flush()
		return nil
	},
}

func init() {
	podCmd.AddCommand(podRunCmd)
	podCmd.AddCommand(podPsCmd)
	rootCmd.AddCommand(podCmd)
}