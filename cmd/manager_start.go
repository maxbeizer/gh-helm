package cmd

import (
	"fmt"
	"log/slog"

	"github.com/maxbeizer/gh-helm/internal/manager"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

var managerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run scheduled manager daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonFlag, _ := cmd.Flags().GetBool("json")
		jqExpr, _ := cmd.Flags().GetString("jq")
		if jsonFlag || jqExpr != "" {
			out := output.New(cmd)
			logger := slog.New(newOutputHandler(out))
			return manager.RunManagerDaemon(cmd.Context(), "helm-manager.toml", logger)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "manager daemon started")
		return manager.RunManagerDaemon(cmd.Context(), "helm-manager.toml", nil)
	},
}

func init() {
	managerCmd.AddCommand(managerStartCmd)
}
