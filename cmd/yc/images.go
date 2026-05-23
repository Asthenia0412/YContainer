package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/image"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List images",
	Long:  `Lists all locally stored images.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		_ = logger

		imageDir := image.YcImagesDir
		entries, err := os.ReadDir(imageDir)
		if err != nil {
			return fmt.Errorf("read images dir: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tTAG\tIMAGE ID\tSIZE")

		for _, repo := range entries {
			if !repo.IsDir() {
				continue
			}
			tagEntries, _ := os.ReadDir(filepath.Join(imageDir, repo.Name()))
			for _, tag := range tagEntries {
				configPath := filepath.Join(imageDir, repo.Name(), tag.Name(), "config.json")
				data, _ := os.ReadFile(configPath)
				id := "-"
				if len(data) > 0 {
					var cfg image.ImageConfig
					if err := json.Unmarshal(data, &cfg); err == nil {
						_ = cfg
					}
				}
				displayID := id
				if len(displayID) > 12 {
					displayID = displayID[:12]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					repo.Name(), tag.Name(), displayID, "-")
			}
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
}