package cmd

import (
	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var managerObserveCmd = &cobra.Command{
	Use:   "observe",
	Short: "Generate observations and post to 1-1 repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		handle, _ := cmd.Flags().GetString("handle")

		mgr, err := manager.Load("manager-ops.yaml")
		if err != nil {
			return err
		}

		results, err := mgr.Observe(cmd.Context(), manager.ObserveOptions{Since: since, DryRun: dryRun, Handle: handle})
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(results)
	},
}

func init() {
	managerObserveCmd.Flags().String("since", "7d", "How far back to look (e.g. 7d, 168h)")
	managerObserveCmd.Flags().Bool("dry-run", false, "Show observations without posting")
	managerObserveCmd.Flags().String("handle", "", "Observe a single handle")
	managerObserveCmd.Flags().Bool("json", false, "Output JSON")
	managerObserveCmd.Flags().String("jq", "", "Filter JSON output with jq")
	managerCmd.AddCommand(managerObserveCmd)
}
