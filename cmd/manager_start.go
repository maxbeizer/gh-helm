package cmd

import (
	"fmt"

	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/spf13/cobra"
)

var managerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run scheduled manager daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "manager daemon started")
		return manager.RunManagerDaemon(cmd.Context(), "manager-ops.yaml")
	},
}

func init() {
	managerCmd.AddCommand(managerStartCmd)
}
