package cmd

import (
	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var managerPulseCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Team health overview",
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		mgr, err := manager.Load("manager-ops.yaml")
		if err != nil {
			return err
		}
		result, err := mgr.Pulse(cmd.Context(), manager.PulseOptions{Since: since})
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(result)
	},
}

func init() {
	managerPulseCmd.Flags().String("since", "30d", "How far back to look (e.g. 30d, 720h)")
	managerPulseCmd.Flags().Bool("json", false, "Output JSON")
	managerPulseCmd.Flags().String("jq", "", "Filter JSON output with jq")
	managerCmd.AddCommand(managerPulseCmd)
}
