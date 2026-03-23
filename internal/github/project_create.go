package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// CreateProjectResult holds the result of creating a new project board.
type CreateProjectResult struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

// CreateProject creates a new GitHub Projects v2 board and configures it
// with the standard gh-helm statuses (Ready, In Progress, In Review, Done).
func CreateProject(ctx context.Context, owner, title string) (CreateProjectResult, error) {
	slog.Debug("creating project board", "owner", owner, "title", title)

	out, err := runGh(ctx, "project", "create", "--owner", owner, "--title", title, "--format", "json")
	if err != nil {
		return CreateProjectResult{}, fmt.Errorf("create project: %w", err)
	}

	var resp struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return CreateProjectResult{}, fmt.Errorf("parse project response: %w", err)
	}

	result := CreateProjectResult{
		Number: resp.Number,
		URL:    resp.URL,
		ID:     resp.ID,
	}
	slog.Debug("project created", "number", result.Number, "url", result.URL)

	// Configure statuses on the new board.
	if err := setupBoardStatuses(ctx, owner, result.Number); err != nil {
		slog.Warn("could not configure board statuses", "error", err)
		// Non-fatal — board is created, statuses can be added later via upgrade.
	}

	return result, nil
}

func setupBoardStatuses(ctx context.Context, owner string, projectNumber int) error {
	existing, _, fieldID, err := FetchBoardStatuses(ctx, owner, projectNumber)
	if err != nil {
		return err
	}

	required := []string{"Ready", "In Progress", "In Review", "Done"}
	existingSet := map[string]bool{}
	for _, s := range existing {
		existingSet[s] = true
	}

	var missing []string
	for _, s := range required {
		if !existingSet[s] {
			missing = append(missing, s)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return AddBoardStatuses(ctx, "", fieldID, existing, missing)
}
