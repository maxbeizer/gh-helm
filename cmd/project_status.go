package cmd

import (
	"github.com/maxbeizer/max-ops/internal/agent"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var projectStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current agent status",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := agent.ReadStatus()
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(status)
	},
}

func init() {
	projectCmd.AddCommand(projectStatusCmd)
}
