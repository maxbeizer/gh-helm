package cmd

import (
	"fmt"
	"time"

	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/spf13/cobra"
)

var managerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Continuous monitoring (observe loop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		intervalText, _ := cmd.Flags().GetString("observe-interval")
		if intervalText == "" {
			intervalText = "24h"
		}
		interval, err := time.ParseDuration(intervalText)
		if err != nil {
			interval = 24 * time.Hour
		}
		mgr, err := manager.Load("manager-ops.yaml")
		if err != nil {
			return err
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		fmt.Fprintf(cmd.OutOrStdout(), "manager observe loop started (every %s)\n", interval)
		for {
			_, err := mgr.Observe(cmd.Context(), manager.ObserveOptions{Since: "7d"})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "observe error: %v\n", err)
			}
			select {
			case <-cmd.Context().Done():
				return nil
			case <-ticker.C:
			}
		}
	},
}

func init() {
	managerStartCmd.Flags().String("observe-interval", "24h", "How often to run observe")
	managerCmd.AddCommand(managerStartCmd)
}
