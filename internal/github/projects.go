package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

type projectField struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Options []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"options"`
}

type projectV2 struct {
	ID     string `json:"id"`
	Fields struct {
		Nodes []projectField `json:"nodes"`
	} `json:"fields"`
	Items struct {
		Nodes []struct {
			ID      string `json:"id"`
			Content struct {
				TypeName string `json:"__typename"`
				ID       string `json:"id"`
			} `json:"content"`
		} `json:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
	} `json:"items"`
}

func MoveIssueToStatus(ctx context.Context, owner string, projectNumber int, issueNodeID string, status string) error {
	slog.Debug("move issue to status",
		"owner", owner, "project", projectNumber,
		"issueNodeID", issueNodeID, "status", status)

	// Detect whether owner is a user or organization.
	info, err := FetchProjectInfo(ctx, owner, projectNumber)
	if err != nil {
		return fmt.Errorf("cannot find project %d for owner %q: %w", projectNumber, owner, err)
	}
	ownerType := info.OwnerScope
	slog.Debug("resolved project owner", "ownerType", ownerType, "projectID", info.ID, "itemCount", info.ItemCount)

	queryTmpl := `
query($owner: String!, $number: Int!, $after: String) {
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
      items(first: 100, after: $after) {
        nodes {
          id
          content { __typename ... on Issue { id } ... on PullRequest { id } }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}
`

	fetchPage := func(cursor string) (*projectV2, error) {
		query := fmt.Sprintf(queryTmpl, ownerType)
		args := []string{"api", "graphql", "-f", "query=" + query,
			"-F", "owner=" + owner,
			"-F", fmt.Sprintf("number=%d", projectNumber)}
		if cursor != "" {
			args = append(args, "-F", "after="+cursor)
		} else {
			args = append(args, "-F", "after=")
		}
		out, err := runGh(ctx, args...)
		if err != nil {
			return nil, err
		}
		// Normalize the JSON key to "owner" for uniform unmarshalling.
		normalized := strings.Replace(string(out), fmt.Sprintf("%q:", ownerType), `"owner":`, 1)
		var resp struct {
			Owner *struct {
				Project *projectV2 `json:"projectV2"`
			} `json:"owner"`
		}
		if err := json.Unmarshal([]byte(normalized), &resp); err != nil {
			return nil, fmt.Errorf("parse project response: %w", err)
		}
		if resp.Owner == nil || resp.Owner.Project == nil {
			return nil, fmt.Errorf("project %d not found for %s %q", projectNumber, ownerType, owner)
		}
		return resp.Owner.Project, nil
	}

	projectID := ""
	fieldID := ""
	optionID := ""
	itemID := ""
	cursor := ""
	pagesScanned := 0

	for {
		pagesScanned++
		proj, err := fetchPage(cursor)
		if err != nil {
			return err
		}

		if projectID == "" {
			projectID = proj.ID
			var statusNames []string
			for _, field := range proj.Fields.Nodes {
				if field.Name == "Status" {
					fieldID = field.ID
					for _, opt := range field.Options {
						statusNames = append(statusNames, opt.Name)
						if opt.Name == status {
							optionID = opt.ID
						}
					}
				}
			}
			slog.Debug("found project fields", "projectID", projectID, "statusFieldID", fieldID, "availableStatuses", statusNames)
			if fieldID == "" {
				return fmt.Errorf("project %d has no 'Status' field — is this a GitHub Projects v2 board?", projectNumber)
			}
			if optionID == "" {
				return fmt.Errorf("status %q not found on project %d (available: %s)", status, projectNumber, strings.Join(statusNames, ", "))
			}
		}

		for _, item := range proj.Items.Nodes {
			if item.Content.ID == issueNodeID {
				itemID = item.ID
				slog.Debug("found issue on board", "itemID", itemID, "page", pagesScanned)
				break
			}
		}

		if itemID != "" || !proj.Items.PageInfo.HasNextPage {
			break
		}
		cursor = proj.Items.PageInfo.EndCursor
		slog.Debug("scanning next page of items", "cursor", cursor, "page", pagesScanned)
	}

	if itemID == "" {
		return fmt.Errorf("issue (node %s) is not on project board %d — add it to the board first", issueNodeID, projectNumber)
	}

	slog.Debug("updating project item status",
		"projectID", projectID, "itemID", itemID,
		"fieldID", fieldID, "optionID", optionID)

	mutation := `
mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
  updateProjectV2ItemFieldValue(input: {
    projectId: $projectId,
    itemId: $itemId,
    fieldId: $fieldId,
    value: { singleSelectOptionId: $optionId }
  }) {
    projectV2Item { id }
  }
}
`
	_, err = runGh(ctx, "api", "graphql", "-f", "query="+mutation,
		"-F", "projectId="+projectID, "-F", "itemId="+itemID,
		"-F", "fieldId="+fieldID, "-F", "optionId="+optionID)
	if err != nil {
		return fmt.Errorf("update status to %q: %w", status, err)
	}
	slog.Debug("status updated successfully", "status", status)
	return nil
}
