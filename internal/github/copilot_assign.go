package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// AssignCopilot assigns the Copilot coding agent to an issue using the
// GraphQL API with the required feature flag header.
func AssignCopilot(ctx context.Context, issueNodeID string) error {
	slog.Debug("assigning Copilot to issue", "issueNodeID", issueNodeID)

	// First, find the Copilot agent's node ID via suggestedActors.
	agentID, err := findCopilotAgentID(ctx, issueNodeID)
	if err != nil {
		return fmt.Errorf("find Copilot agent: %w", err)
	}

	// Assign using the GraphQL mutation with the feature flag header.
	mutation := `
mutation($issueId: ID!, $agentId: ID!) {
  addAssigneesToAssignable(input: {
    assignableId: $issueId,
    assigneeIds: [$agentId]
  }) {
    assignable {
      ... on Issue {
        id
      }
    }
  }
}`
	_, err = runGh(ctx, "api", "graphql",
		"-H", "GraphQL-Features: issues_copilot_assignment_api_support",
		"-f", "query="+mutation,
		"-F", "issueId="+issueNodeID,
		"-F", "agentId="+agentID,
	)
	if err != nil {
		return fmt.Errorf("assign Copilot via GraphQL: %w", err)
	}

	slog.Debug("Copilot assigned successfully")
	return nil
}

// findCopilotAgentID queries the suggestedActors API to find the Copilot
// coding agent's node ID for a given issue.
func findCopilotAgentID(ctx context.Context, issueNodeID string) (string, error) {
	query := `
query($issueId: ID!) {
  node(id: $issueId) {
    ... on Issue {
      suggestedActors(first: 20) {
        nodes {
          ... on User { login id }
          ... on Bot { login id }
          ... on Mannequin { login id }
        }
      }
    }
  }
}`
	out, err := runGh(ctx, "api", "graphql",
		"-H", "GraphQL-Features: issues_copilot_assignment_api_support",
		"-f", "query="+query,
		"-F", "issueId="+issueNodeID,
	)
	if err != nil {
		return "", fmt.Errorf("query suggested actors: %w", err)
	}

	var resp struct {
		Node struct {
			SuggestedActors struct {
				Nodes []struct {
					Login string `json:"login"`
					ID    string `json:"id"`
				} `json:"nodes"`
			} `json:"suggestedActors"`
		} `json:"node"`
	}

	// Handle gh's {"data": ...} wrapper.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	payload := out
	if dataRaw, ok := raw["data"]; ok {
		payload = dataRaw
	}

	if err := json.Unmarshal(payload, &resp); err != nil {
		return "", fmt.Errorf("parse suggested actors: %w", err)
	}

	// Look for copilot-swe-agent or any copilot-related actor.
	for _, actor := range resp.Node.SuggestedActors.Nodes {
		if actor.Login == "copilot-swe-agent" || actor.Login == "copilot" {
			slog.Debug("found Copilot agent", "login", actor.Login, "id", actor.ID)
			return actor.ID, nil
		}
	}

	// Log available actors for debugging.
	var logins []string
	for _, actor := range resp.Node.SuggestedActors.Nodes {
		logins = append(logins, actor.Login)
	}
	return "", fmt.Errorf("Copilot coding agent not available on this repo (available assignees: %v) — ensure Copilot coding agent is enabled in repo settings", logins)
}
