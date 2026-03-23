package cmd

import (
	"fmt"
	"os"

	"github.com/maxbeizer/gh-helm/internal/agent"
	"github.com/maxbeizer/gh-helm/internal/output"
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
		if out.WantsJSON() {
			return out.Print(status)
		}
		printStatusHuman(status)
		return nil
	},
}

func printStatusHuman(s agent.State) {
	fmt.Fprintln(os.Stdout, "🤖 gh-helm project agent status")
	fmt.Fprintln(os.Stdout)

	if s.Session == "none" || s.Session == "" {
		fmt.Fprintln(os.Stdout, "  No active session — run 'gh helm project start --issue <N>' to begin")
		return
	}

	fmt.Fprintf(os.Stdout, "  Session:       %s\n", s.Session)
	if !s.LastActivity.IsZero() {
		fmt.Fprintf(os.Stdout, "  Last Activity: %s\n", s.LastActivity.Format("2006-01-02 15:04:05"))
	}

	if len(s.IssuesWorked) > 0 {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "  Issues Worked:")
		for _, issue := range s.IssuesWorked {
			fmt.Fprintf(os.Stdout, "    #%-6d %s\n", issue.Number, issue.Title)
		}
	}

	if len(s.PullsCreated) > 0 {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "  Pull Requests:")
		for _, pr := range s.PullsCreated {
			fmt.Fprintf(os.Stdout, "    #%-6d %s\n", pr.Number, pr.URL)
		}
	}
}

func init() {
	projectCmd.AddCommand(projectStatusCmd)
}
