package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maxbeizer/gh-helm/internal/github"
	"github.com/maxbeizer/gh-helm/internal/state"
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
	return filepath.Join(".helm", "state.json")
}

const maxStatusHistory = 50

func writeStatus(session string, issue github.Issue, pr PullRequest) error {
	s, err := ReadStatus()
	if err != nil {
		// Start fresh if we can't read existing state
		s = State{
			IssuesWorked: []IssueInfo{},
			PullsCreated: []PullInfo{},
		}
	}

	s.Session = session
	s.LastActivity = time.Now()
	s.IssuesWorked = append(s.IssuesWorked, IssueInfo{Number: issue.Number, Title: issue.Title})
	s.PullsCreated = append(s.PullsCreated, PullInfo{Number: pr.Number, URL: pr.URL})

	// Cap history to prevent unbounded growth
	if len(s.IssuesWorked) > maxStatusHistory {
		s.IssuesWorked = s.IssuesWorked[len(s.IssuesWorked)-maxStatusHistory:]
	}
	if len(s.PullsCreated) > maxStatusHistory {
		s.PullsCreated = s.PullsCreated[len(s.PullsCreated)-maxStatusHistory:]
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	return state.WriteAtomic(statusPath(), data, 0o644)
}

func ReadStatus() (State, error) {
	data, err := os.ReadFile(statusPath())
	if err != nil {
		if os.IsNotExist(err) {
			return State{Session: "none", IssuesWorked: []IssueInfo{}, PullsCreated: []PullInfo{}}, nil
		}
		return State{}, fmt.Errorf("read status file: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse status file: %w", err)
	}
	return state, nil
}
