package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
)

var buildCmd = &cobra.Command{
	Use:   "build -t IMAGE:TAG PATH",
	Short: "Build an image from a Dockerfile",
	Long:  `Builds a container image from a Dockerfile (simplified version).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		logger.Warn("Build is not yet fully implemented")
		return fmt.Errorf("build command is under development")
	},
}

func init() {
	buildCmd.Flags().StringP("tag", "t", "", "Name and optionally a tag")
	rootCmd.AddCommand(buildCmd)
}