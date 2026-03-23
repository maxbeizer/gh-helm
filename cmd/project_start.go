package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/maxbeizer/gh-helm/internal/agent"
	"github.com/maxbeizer/gh-helm/internal/output"
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
		codespace, _ := cmd.Flags().GetBool("codespace")
		delegate, _ := cmd.Flags().GetBool("delegate")

		agentRunner := agent.NewProjectAgent()
		result, err := agentRunner.Start(cmd.Context(), agent.StartOptions{
			IssueNumber: issueNumber,
			Repo:        repo,
			Model:       model,
			DryRun:      dryRun,
			Codespace:   codespace,
			Delegate:    delegate,
		})
		if err != nil {
			return err
		}

		out := output.New(cmd)
		if out.WantsJSON() {
			return out.Print(result)
		}
		printStartResult(result)
		return nil
	},
}

func printStartResult(r agent.StartResult) {
	if r.Delegated {
		fmt.Fprintln(os.Stdout, "🤖 gh-helm agent — delegated to Copilot")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "  Issue:    #%d %s\n", r.Issue.Number, r.Issue.Title)
		fmt.Fprintf(os.Stdout, "  Context:  Added project context comment\n")
		fmt.Fprintf(os.Stdout, "  Assigned: @copilot\n")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "  Next: Copilot will open a PR when ready\n")
		return
	}

	if r.DryRun {
		fmt.Fprintln(os.Stdout, "🤖 gh-helm agent — dry run")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "  Issue:   #%d %s\n", r.Issue.Number, r.Issue.Title)
		fmt.Fprintf(os.Stdout, "  Plan:    %s\n", r.Plan.Plan)
		fmt.Fprintf(os.Stdout, "  Files:   %d\n", len(r.Plan.Files))
		for _, f := range r.Plan.Files {
			fmt.Fprintf(os.Stdout, "           %s %s\n", f.Action, f.Path)
		}
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "  Next: run without --dry-run to create the PR\n")
		return
	}

	fmt.Fprintln(os.Stdout, "🤖 gh-helm agent — done!")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "  Issue:    #%d %s\n", r.Issue.Number, r.Issue.Title)
	fmt.Fprintf(os.Stdout, "  Branch:   %s\n", r.Branch)
	fmt.Fprintf(os.Stdout, "  PR:       #%d %s\n", r.Pull.Number, r.Pull.URL)
	fmt.Fprintf(os.Stdout, "  Files:    %d\n", len(r.Plan.Files))
	for _, f := range r.Plan.Files {
		fmt.Fprintf(os.Stdout, "            %s %s\n", f.Action, f.Path)
	}
	if r.CodespaceURL != "" {
		fmt.Fprintf(os.Stdout, "  Codespace: %s\n", r.CodespaceURL)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "  Next: review the PR, then merge with 'gh pr merge %d --squash'\n", r.Pull.Number)
}

func init() {
	projectStartCmd.Flags().Int("issue", 0, "Issue number")
	projectStartCmd.Flags().String("repo", "", "Repository owner/name")
	projectStartCmd.Flags().String("model", "", "AI model")
	projectStartCmd.Flags().Bool("dry-run", false, "Show plan without executing")
	projectStartCmd.Flags().Bool("codespace", false, "Create a Codespace on the PR branch")
	projectStartCmd.Flags().Bool("delegate", false, "Delegate to Copilot coding agent instead of generating code directly")
	projectCmd.AddCommand(projectStartCmd)
}
