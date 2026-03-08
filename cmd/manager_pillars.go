package cmd

import (
	"github.com/maxbeizer/max-ops/internal/manager"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var managerPillarsCmd = &cobra.Command{
	Use:   "pillars",
	Short: "Show configured pillar definitions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := manager.Load("manager-ops.yaml")
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(mgr.Config.Pillars)
	},
}

func init() {
	managerCmd.AddCommand(managerPillarsCmd)
}
