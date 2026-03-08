package cmd

import (
	"github.com/maxbeizer/gh-helm/internal/manager"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

var managerPillarsCmd = &cobra.Command{
	Use:   "pillars",
	Short: "Show configured pillar definitions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := manager.Load("helm-manager.toml")
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
