package main

import (
	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
	"github.com/Yancy/YContainer/pkg/types"
)

var runConfig struct {
	name      string
	memory    int64
	cpus      float64
	pidsMax   int64
	env       []string
}

var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE COMMAND [ARG...]",
	Short: "Create and run a new container",
	Long:  `Compatible with Docker run semantics. Creates a new container from an image and starts it.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		image := args[0]
		command := args[1:]

		mgr := container.NewManager(logger)
		lifecycle := container.NewLifecycleManager(mgr, logger)

		cpuQuota := int64(runConfig.cpus * 100000)

		config := container.RunConfig{
			Name:  runConfig.name,
			Image: image,
			Cmd:   command,
			Envs:  runConfig.env,
			Resources: types.ResourceLimit{
				MemoryLimit: runConfig.memory,
				CPUPeriod:   100000,
				CPUQuota:    cpuQuota,
				PidsMax:     runConfig.pidsMax,
			},
		}

		c, err := lifecycle.Create(config)
		if err != nil {
			return err
		}

		if err := lifecycle.Start(c, ""); err != nil {
			return err
		}

		logger.Info("Container %s started", c.ID[:12])
		return nil
	},
}

func init() {
	runCmd.Flags().StringVarP(&runConfig.name, "name", "n", "", "Assign a name to the container")
	runCmd.Flags().Int64VarP(&runConfig.memory, "memory", "m", 0, "Memory limit in bytes")
	runCmd.Flags().Float64VarP(&runConfig.cpus, "cpus", "", 0, "Number of CPUs")
	runCmd.Flags().Int64VarP(&runConfig.pidsMax, "pids-limit", "", 0, "Max number of processes")
	runCmd.Flags().StringArrayVarP(&runConfig.env, "env", "e", nil, "Set environment variables")
	rootCmd.AddCommand(runCmd)
}