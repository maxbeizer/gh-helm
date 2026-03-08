package profile

import (
	"testing"
)

func TestSuggestWork(t *testing.T) {
	p := HubberProfile{
		Skills: SkillSet{
			Strong:     []string{"go", "api-design"},
			Growing:    []string{"security", "observability"},
			Interested: []string{"ml-ops"},
		},
		GrowthAreas: []string{"Wants security exposure"},
	}

	issues := []IssueSummary{
		{Number: 55, Title: "Refactor auth middleware", Labels: []string{"security", "go"}, Body: "Security improvements to auth"},
		{Number: 58, Title: "Add rate limiting", Labels: []string{"go", "api-design"}, Body: "Rate limit the API endpoints"},
		{Number: 61, Title: "Set up observability pipeline", Labels: []string{"observability"}, Body: "Add monitoring and tracing"},
		{Number: 62, Title: "Update README", Labels: []string{"documentation"}, Body: "Fix typos in docs"},
	}

	suggestions := SuggestWork(p, issues)

	if len(suggestions) == 0 {
		t.Fatal("expected suggestions, got none")
	}

	// Issue 55 should rank high: go (strong:2) + security (growing:3) = 5
	// Issue 58: go (strong:2) + api-design (strong:2) = 4
	// Issue 61: observability (growing:3) = 3
	// Issue 62: no matches = 0 (filtered out)

	if suggestions[0].IssueNumber != 55 {
		t.Errorf("top suggestion = #%d, want #55", suggestions[0].IssueNumber)
	}
	if len(suggestions) != 3 {
		t.Errorf("got %d suggestions, want 3 (doc issue should be filtered)", len(suggestions))
	}

	if len(suggestions[0].Reasons) == 0 {
		t.Error("top suggestion should have reasons")
	}
}

func TestSuggestWorkEmptyProfile(t *testing.T) {
	p := HubberProfile{}
	issues := []IssueSummary{
		{Number: 1, Title: "Something", Body: "Body"},
	}
	suggestions := SuggestWork(p, issues)
	if len(suggestions) != 0 {
		t.Errorf("empty profile should produce no suggestions, got %d", len(suggestions))
	}
}

func TestScoreIssueGrowingSkillsRankHigher(t *testing.T) {
	p := HubberProfile{
		Skills: SkillSet{
			Strong:  []string{"go"},
			Growing: []string{"go"},
		},
	}
	issue := IssueSummary{Number: 1, Title: "Go refactor", Labels: []string{"go"}, Body: "refactor go code"}
	suggestion := scoreIssue(p, issue)
	// strong: 2 + growing: 3 = 5
	if suggestion.Score != 5 {
		t.Errorf("score = %d, want 5", suggestion.Score)
	}
}
