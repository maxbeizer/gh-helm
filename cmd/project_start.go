package cmd

import (
	"errors"

	"github.com/maxbeizer/max-ops/internal/agent"
	"github.com/maxbeizer/max-ops/internal/output"
	"github.com/spf13/cobra"
)

var projectStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start working an issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		issueNumber, _ := cmd.Flags().GetInt("issue")
		if issueNumber == 0 {
			return errors.New("--issue is required")
		}
		repo, _ := cmd.Flags().GetString("repo")
		model, _ := cmd.Flags().GetString("model")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		agentRunner := agent.NewProjectAgent()
		result, err := agentRunner.Start(cmd.Context(), agent.StartOptions{
			IssueNumber: issueNumber,
			Repo:        repo,
			Model:       model,
			DryRun:      dryRun,
		})
		if err != nil {
			return err
		}

		out := output.New(cmd)
		return out.Print(result)
	},
}

func init() {
	projectStartCmd.Flags().Int("issue", 0, "Issue number")
	projectStartCmd.Flags().String("repo", "", "Repository owner/name")
	projectStartCmd.Flags().String("model", "", "AI model")
	projectStartCmd.Flags().Bool("dry-run", false, "Show plan without executing")
	projectCmd.AddCommand(projectStartCmd)
}
