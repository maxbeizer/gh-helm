package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// statusColors maps status names to GitHub Projects v2 colors.
var statusColors = map[string]string{
	"Ready":       "GREEN",
	"Todo":        "GRAY",
	"In Progress": "YELLOW",
	"In Review":   "BLUE",
	"Done":        "PURPLE",
}

// FetchBoardStatuses returns the current status option names, project ID, and
// status field ID for the given project board.
func FetchBoardStatuses(ctx context.Context, owner string, projectNumber int) (statuses []string, projectID string, fieldID string, err error) {
	info, err := FetchProjectInfo(ctx, owner, projectNumber)
	if err != nil {
		return nil, "", "", err
	}

	query := fmt.Sprintf(`
query($owner: String!, $number: Int!) {
  %s(login: $owner) {
    projectV2(number: $number) {
      id
      fields(first: 50) {
        nodes {
          ... on ProjectV2SingleSelectField {
            id
            name
            options { id name }
          }
        }
      }
    }
  }
}`, info.OwnerScope)

	out, err := runGh(ctx, "api", "graphql", "-f", "query="+query,
		"-F", "owner="+owner, "-F", fmt.Sprintf("number=%d", projectNumber))
	if err != nil {
		return nil, "", "", fmt.Errorf("fetch board fields: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, "", "", err
	}
	payload := raw[info.OwnerScope]
	if dataRaw, ok := raw["data"]; ok {
		var data map[string]json.RawMessage
		if err := json.Unmarshal(dataRaw, &data); err == nil {
			if p, ok := data[info.OwnerScope]; ok {
				payload = p
			}
		}
	}

	var ownerNode struct {
		Project *struct {
			ID     string `json:"id"`
			Fields struct {
				Nodes []projectField `json:"nodes"`
			} `json:"fields"`
		} `json:"projectV2"`
	}
	if err := json.Unmarshal(payload, &ownerNode); err != nil || ownerNode.Project == nil {
		return nil, "", "", fmt.Errorf("project %d not found", projectNumber)
	}

	projectID = ownerNode.Project.ID
	for _, field := range ownerNode.Project.Fields.Nodes {
		if field.Name == "Status" {
			fieldID = field.ID
			for _, opt := range field.Options {
				statuses = append(statuses, opt.Name)
			}
			break
		}
	}
	if fieldID == "" {
		return nil, projectID, "", fmt.Errorf("no Status field found on project %d", projectNumber)
	}
	return statuses, projectID, fieldID, nil
}

// AddBoardStatuses adds missing status options to a project board's Status field.
// It rebuilds the full option list (existing + new) because the API requires all
// options to be specified at once.
func AddBoardStatuses(ctx context.Context, projectID, fieldID string, existing, missing []string) error {
	// Build the full options list: existing first, then new ones appended.
	var optionsList string
	for _, name := range existing {
		color := statusColors[name]
		if color == "" {
			color = "GRAY"
		}
		optionsList += fmt.Sprintf(`{name: %q, color: %s, description: %q} `, name, color, "")
	}
	for _, name := range missing {
		color := statusColors[name]
		if color == "" {
			color = "GRAY"
		}
		optionsList += fmt.Sprintf(`{name: %q, color: %s, description: %q} `, name, color, "gh-helm required status")
	}

	mutation := fmt.Sprintf(`
mutation {
  updateProjectV2Field(input: {
    fieldId: %q
    singleSelectOptions: [%s]
  }) {
    clientMutationId
  }
}`, fieldID, optionsList)

	slog.Debug("adding board statuses", "missing", missing, "fieldID", fieldID)
	_, err := runGh(ctx, "api", "graphql", "-f", "query="+mutation)
	if err != nil {
		return fmt.Errorf("update board statuses: %w", err)
	}
	slog.Debug("board statuses updated successfully")
	return nil
}
