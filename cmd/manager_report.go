package cmd

import (
	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var managerReportCmd = &cobra.Command{
	Use:   "report <handle>",
	Short: "Generate a report card for a team member",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		mgr, err := manager.Load("manager-ops.yaml")
		if err != nil {
			return err
		}
		result, err := mgr.Report(cmd.Context(), manager.ReportOptions{Since: since, Handle: args[0]})
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(result)
	},
}

func init() {
	managerReportCmd.Flags().String("since", "90d", "How far back to look (e.g. 90d, 2160h)")
	managerReportCmd.Flags().Bool("json", false, "Output JSON")
	managerReportCmd.Flags().String("jq", "", "Filter JSON output with jq")
	managerCmd.AddCommand(managerReportCmd)
}
