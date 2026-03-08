package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/maxbeizer/max-ops/internal/github"
)

type State struct {
	Session      string      `json:"session"`
	LastActivity time.Time   `json:"last_activity"`
	IssuesWorked []IssueInfo `json:"issues_worked"`
	PullsCreated []PullInfo  `json:"pulls_created"`
}

type IssueInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

type PullInfo struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

func statusPath() string {
	return filepath.Join(".max-ops", "state.json")
}

func writeStatus(session string, issue github.Issue, pr PullRequest) error {
	state := State{
		Session:      session,
		LastActivity: time.Now(),
		IssuesWorked: []IssueInfo{{Number: issue.Number, Title: issue.Title}},
		PullsCreated: []PullInfo{{Number: pr.Number, URL: pr.URL}},
	}
	if err := os.MkdirAll(filepath.Dir(statusPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statusPath(), data, 0o644)
}

func ReadStatus() (State, error) {
	data, err := os.ReadFile(statusPath())
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}
