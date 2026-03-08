package cmd

import (
	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var managerPrepCmd = &cobra.Command{
	Use:   "prep <handle>",
	Short: "Generate 1-1 prep for one person",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		mgr, err := manager.Load("manager-ops.yaml")
		if err != nil {
			return err
		}
		result, err := mgr.Prep(cmd.Context(), manager.PrepOptions{Since: since, Handle: args[0]})
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(result)
	},
}

func init() {
	managerPrepCmd.Flags().String("since", "14d", "How far back to look (e.g. 14d, 336h)")
	managerPrepCmd.Flags().Bool("json", false, "Output JSON")
	managerPrepCmd.Flags().String("jq", "", "Filter JSON output with jq")
	managerCmd.AddCommand(managerPrepCmd)
}
