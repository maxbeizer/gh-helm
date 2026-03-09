package cmd

import (
	"errors"

	"github.com/maxbeizer/gh-helm/internal/config"
	gh "github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/maxbeizer/gh-helm/internal/sot"
	"github.com/spf13/cobra"
)

var projectSotCmd = &cobra.Command{
	Use:   "sot",
	Short: "Source of truth document",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("helm.toml")
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
		cfg, err := config.Load("helm.toml")
		if err != nil {
			return err
		}

		session, _ := cmd.Flags().GetString("session")
		prNumber, _ := cmd.Flags().GetInt("pr")
		decision, _ := cmd.Flags().GetString("decision")

		// PR-based proposal: auto-generate from PR context
		if prNumber > 0 {
			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				repo, err = gh.CurrentRepo(cmd.Context())
				if err != nil {
					return err
				}
			}
			proposed, err := sot.ProposeFromPR(cmd.Context(), cfg.SourceOfTruth, repo, prNumber, session)
			if err != nil {
				return err
			}
			out := output.New(cmd)
			return out.Print(map[string]string{"proposed": proposed})
		}

		// Manual proposal: --decision is required
		if decision == "" {
			return errors.New("--decision or --pr is required")
		}
		if err := sot.Propose(cfg.SourceOfTruth, decision, session, ""); err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(map[string]string{"proposed": decision})
	},
}

var projectSotSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Reconcile SOT with current issue state",
	Long:  "Cross-references items in the Next Up section with GitHub issue state and removes entries for closed issues.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("helm.toml")
		if err != nil {
			return err
		}
		repo, _ := cmd.Flags().GetString("repo")
		if repo == "" {
			repo, err = gh.CurrentRepo(cmd.Context())
			if err != nil {
				return err
			}
		}
		apply, _ := cmd.Flags().GetBool("apply")
		result, err := sot.Sync(cmd.Context(), cfg.SourceOfTruth, repo, apply)
		if err != nil {
			return err
		}
		out := output.New(cmd)
		return out.Print(result)
	},
}

func init() {
	projectSotProposeCmd.Flags().String("decision", "", "Decision to propose")
	projectSotProposeCmd.Flags().String("session", "", "Agent session id")
	projectSotProposeCmd.Flags().Int("pr", 0, "PR number to generate proposal from")
	projectSotProposeCmd.Flags().String("repo", "", "Repository owner/name")

	projectSotSyncCmd.Flags().String("repo", "", "Repository owner/name")
	projectSotSyncCmd.Flags().Bool("apply", false, "Apply changes (default is dry-run)")

	projectSotCmd.AddCommand(projectSotProposeCmd)
	projectSotCmd.AddCommand(projectSotSyncCmd)
	projectCmd.AddCommand(projectSotCmd)
}
