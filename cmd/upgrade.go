package cmd

import (
	"fmt"

	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/maxbeizer/gh-helm/internal/upgrade"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade a project to the latest gh-helm defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		result, err := upgrade.Run(cmd.Context(), upgrade.Options{DryRun: dryRun})
		if err != nil {
			return err
		}
		jsonFlag, _ := cmd.Flags().GetBool("json")
		jqExpr, _ := cmd.Flags().GetString("jq")
		if jsonFlag || jqExpr != "" {
			out := output.New(cmd)
			return out.Print(result)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "🔄 gh-helm upgrade")
		fmt.Fprintln(cmd.OutOrStdout(), "")
		for _, change := range result.Changes {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", iconForChange(change.Status), change.Message)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "")
		fmt.Fprintf(cmd.OutOrStdout(), "  %d changes applied, %d skipped\n", result.Applied, result.Skipped)
		return nil
	},
}

func iconForChange(status upgrade.ChangeStatus) string {
	switch status {
	case upgrade.StatusApplied:
		return "✅"
	case upgrade.StatusSkipped:
		return "⏭"
	default:
		return "•"
	}
}

func init() {
	upgradeCmd.Flags().Bool("dry-run", false, "Show changes without applying")
}
