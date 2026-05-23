package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/image"
)

var pullCmd = &cobra.Command{
	Use:   "pull IMAGE[:TAG]",
	Short: "Pull an image from a registry",
	Long:  `Pulls an image from Docker Hub or other OCI-compatible registry.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		puller := image.NewPuller(logger)

		img, err := puller.Pull(args[0])
		if err != nil {
			return fmt.Errorf("pull image: %w", err)
		}

		logger.Info("Image %s:%s pulled successfully (%d layers, %d bytes)",
			img.Name, img.Tag, len(img.Layers), img.Size)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}