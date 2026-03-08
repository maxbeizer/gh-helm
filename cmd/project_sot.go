package cmd

import (
	"errors"

	"github.com/maxbeizer/max-ops/internal/config"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/maxbeizer/max-ops/internal/sot"
	"github.com/spf13/cobra"
)

var projectSotCmd = &cobra.Command{
	Use:   "sot",
	Short: "Source of truth document",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("max-ops.yaml")
		if err != nil {
			return err
		}
		content, err := sot.Read(cfg.SourceOfTruth)
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(map[string]string{"source-of-truth": content})
	},
}

var projectSotProposeCmd = &cobra.Command{
	Use:   "propose",
	Short: "Propose an update to the source of truth",
	RunE: func(cmd *cobra.Command, args []string) error {
		decision, _ := cmd.Flags().GetString("decision")
		if decision == "" {
			return errors.New("--decision is required")
		}
		cfg, err := config.Load("max-ops.yaml")
		if err != nil {
			return err
		}
		session, _ := cmd.Flags().GetString("session")
		if err := sot.Propose(cfg.SourceOfTruth, decision, session, ""); err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(map[string]string{"proposed": decision})
	},
}

func init() {
	projectSotProposeCmd.Flags().String("decision", "", "Decision to propose")
	projectSotProposeCmd.Flags().String("session", "", "Agent session id")
	projectSotCmd.AddCommand(projectSotProposeCmd)
	projectCmd.AddCommand(projectSotCmd)
}
