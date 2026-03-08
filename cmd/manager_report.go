package cmd

import (
	"github.com/maxbeizer/gh-helm/internal/manager"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

var managerReportCmd = &cobra.Command{
	Use:   "report <handle>",
	Short: "Generate a report card for a team member",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		mgr, err := manager.Load("helm-manager.toml")
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
	managerCmd.AddCommand(managerReportCmd)
}
