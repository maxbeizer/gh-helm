package cmd

import (
	"fmt"

	"github.com/maxbeizer/gh-helm/internal/config"
	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/output"
	"github.com/maxbeizer/gh-helm/internal/profile"
	"github.com/spf13/cobra"
)

var projectSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest issues based on developer profile",
	Long:  "Rank available issues by match with the developer's skills, growth areas, and interests from their profile.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("helm.toml")
		if err != nil {
			return err
		}

		profileRepo, _ := cmd.Flags().GetString("profile-repo")
		if profileRepo == "" {
			return fmt.Errorf("--profile-repo is required (e.g., owner/dev-1-1)")
		}

		devProfile, err := profile.Load(cmd.Context(), profileRepo)
		if err != nil {
			return fmt.Errorf("load profile: %w", err)
		}

		// Search for open issues with configured label
		label := "agent-ready"
		if len(cfg.Filters.Labels) > 0 {
			label = cfg.Filters.Labels[0]
		}
		query := fmt.Sprintf("is:issue is:open label:%s", label)
		if cfg.Project.Owner != "" {
			query += " org:" + cfg.Project.Owner
		}

		searchItems, err := github.SearchIssues(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("search issues: %w", err)
		}

		issues := make([]profile.IssueSummary, 0, len(searchItems))
		for _, item := range searchItems {
			labels := make([]string, 0, len(item.Labels))
			for _, l := range item.Labels {
				labels = append(labels, l.Name)
			}
			issues = append(issues, profile.IssueSummary{
				Number: item.Number,
				Title:  item.Title,
				Labels: labels,
				Body:   item.Body,
			})
		}

		suggestions := profile.SuggestWork(devProfile, issues)

		out := output.New(cmd)
		return out.Print(suggestions)
	},
}

func init() {
	projectSuggestCmd.Flags().String("profile-repo", "", "1-1 repo containing developer-profile.toml")
	projectCmd.AddCommand(projectSuggestCmd)
}
