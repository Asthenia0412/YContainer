package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers",
	Long:  `Lists all containers managed by YContainer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger
		mgr := container.NewManager(logger)

		containers, err := mgr.List()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CONTAINER ID\tNAME\tIMAGE\tSTATUS\tPID\tCREATED")

		for _, c := range containers {
			id := c.ID
			if len(id) > 12 {
				id = id[:12]
			}
			created := c.CreatedAt.Format("2006-01-02 15:04")
			pidStr := fmt.Sprintf("%d", c.PID)
			if c.PID == 0 {
				pidStr = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				id, c.Name, c.Image, c.Status, pidStr, created)
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}