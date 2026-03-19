package profile

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/maxbeizer/gh-helm/internal/github"
)

type DeveloperProfile struct {
	Skills      SkillSet    `toml:"skills"`
	GrowthAreas []string   `toml:"growth-areas"`
	Preferences Preferences `toml:"preferences"`
}

type SkillSet struct {
	Strong     []string `toml:"strong"`
	Growing    []string `toml:"growing"`
	Interested []string `toml:"interested"`
}

type Preferences struct {
	WorkStyle      string `toml:"work-style"`
	ChallengeLevel string `toml:"challenge-level"`
}

type WorkSuggestion struct {
	IssueNumber int      `json:"issue_number"`
	IssueTitle  string   `json:"issue_title"`
	Score       int      `json:"score"`
	Reasons     []string `json:"reasons"`
}

type IssueSummary struct {
	Number int
	Title  string
	Labels []string
	Body   string
}

// Load fetches a developer profile from their 1-1 repo.
// Looks for developer-profile.toml in the repo root.
func Load(ctx context.Context, repo string) (DeveloperProfile, error) {
	if repo == "" {
		return DeveloperProfile{}, fmt.Errorf("profile repo is required")
	}
	out, err := github.RunWith(ctx, "api", fmt.Sprintf("repos/%s/contents/developer-profile.toml", repo),
		"--jq", ".content", "-H", "Accept: application/vnd.github.raw+json")
	if err != nil {
		return DeveloperProfile{}, fmt.Errorf("fetch profile from %s: %w", repo, err)
	}
	var profile DeveloperProfile
	if _, err := toml.Decode(string(out), &profile); err != nil {
		return DeveloperProfile{}, fmt.Errorf("parse profile: %w", err)
	}
	return profile, nil
}

// SuggestWork scores and ranks issues based on the developer's profile.
// Returns suggestions sorted by score (highest first).
func SuggestWork(profile DeveloperProfile, issues []IssueSummary) []WorkSuggestion {
	suggestions := make([]WorkSuggestion, 0, len(issues))
	for _, issue := range issues {
		suggestion := scoreIssue(profile, issue)
		if suggestion.Score > 0 {
			suggestions = append(suggestions, suggestion)
		}
	}
	// Sort by score descending
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].Score > suggestions[i].Score {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
	return suggestions
}

func scoreIssue(profile DeveloperProfile, issue IssueSummary) WorkSuggestion {
	suggestion := WorkSuggestion{
		IssueNumber: issue.Number,
		IssueTitle:  issue.Title,
	}

	text := strings.ToLower(issue.Title + " " + issue.Body)
	labelSet := map[string]bool{}
	for _, l := range issue.Labels {
		labelSet[strings.ToLower(l)] = true
	}

	// Strong skills: good for fast delivery (score: 2 per match)
	for _, skill := range profile.Skills.Strong {
		if containsSkill(text, labelSet, skill) {
			suggestion.Score += 2
			suggestion.Reasons = append(suggestion.Reasons, fmt.Sprintf("Matches strong skill: %s", skill))
		}
	}

	// Growing skills: stretch opportunity (score: 3 per match — higher because growth)
	for _, skill := range profile.Skills.Growing {
		if containsSkill(text, labelSet, skill) {
			suggestion.Score += 3
			suggestion.Reasons = append(suggestion.Reasons, fmt.Sprintf("Growth opportunity: %s (growing skill)", skill))
		}
	}

	// Interested skills: engagement boost (score: 1 per match)
	for _, skill := range profile.Skills.Interested {
		if containsSkill(text, labelSet, skill) {
			suggestion.Score += 1
			suggestion.Reasons = append(suggestion.Reasons, fmt.Sprintf("Aligns with interest: %s", skill))
		}
	}

	return suggestion
}

func containsSkill(text string, labels map[string]bool, skill string) bool {
	lower := strings.ToLower(skill)
	if labels[lower] {
		return true
	}
	return strings.Contains(text, lower)
}
