package cmd

import (
	"github.com/maxbeizer/gh-helm/internal/manager"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/spf13/cobra"
)

var managerStatsCmd = &cobra.Command{
	Use:   "stats [handle]",
	Short: "Team or individual statistics",
	Long:  "Show aggregate statistics: PRs merged, cycle time, review turnaround, pillar coverage, and bus factor analysis.",
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")

		mgr, err := manager.Load("helm-manager.toml")
		if err != nil {
			return err
		}

		handle := ""
		if len(args) > 0 {
			handle = args[0]
		}

		stats, err := mgr.Stats(cmd.Context(), manager.StatsOptions{
			Since:  since,
			Handle: handle,
		})
		if err != nil {
			return err
		}

		out := output.New(cmd)
		return out.Print(stats)
	},
}

func init() {
	managerStatsCmd.Flags().String("since", "30d", "Look back period (e.g. 30d, 720h)")
	managerCmd.AddCommand(managerStatsCmd)
}
