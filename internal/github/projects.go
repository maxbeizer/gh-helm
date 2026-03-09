package github

import (
	"context"
	"encoding/json"
	"fmt"
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

type projectQueryResponse struct {
	Organization *struct {
		Project *projectV2 `json:"projectV2"`
	} `json:"organization"`
	User *struct {
		Project *projectV2 `json:"projectV2"`
	} `json:"user"`
}

func MoveIssueToStatus(ctx context.Context, owner string, projectNumber int, issueNodeID string, status string) error {
	// Fetch project fields and items in a single query. We paginate items
	// with a generous first:100 to cover most boards; if the issue is not
	// found in the first page we follow cursors.
	query := `
query($owner: String!, $number: Int!, $after: String) {
  organization(login: $owner) {
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
  user(login: $owner) {
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
	projectID := ""
	fieldID := ""
	optionID := ""
	itemID := ""
	cursor := ""

	for {
		args := []string{"api", "graphql", "-f", "query=" + query, "-F", "owner=" + owner, "-F", fmt.Sprintf("number=%d", projectNumber)}
		if cursor != "" {
			args = append(args, "-F", "after="+cursor)
		} else {
			args = append(args, "-F", "after=")
		}

		out, err := runGh(ctx, args...)
		if err != nil {
			return err
		}

		var resp projectQueryResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return err
		}

		holder := resp.Organization
		if holder == nil || holder.Project == nil {
			holder = resp.User
		}
		if holder == nil || holder.Project == nil {
			return fmt.Errorf("project not found")
		}

		proj := holder.Project
		if projectID == "" {
			projectID = proj.ID
			for _, field := range proj.Fields.Nodes {
				if field.Name == "Status" {
					fieldID = field.ID
					for _, opt := range field.Options {
						if opt.Name == status {
							optionID = opt.ID
						}
					}
				}
			}
		}

		for _, item := range proj.Items.Nodes {
			if item.Content.ID == issueNodeID {
				itemID = item.ID
				break
			}
		}

		if itemID != "" || !proj.Items.PageInfo.HasNextPage {
			break
		}
		cursor = proj.Items.PageInfo.EndCursor
	}

	if fieldID == "" || optionID == "" {
		return fmt.Errorf("status field or option %q not found", status)
	}
	if itemID == "" {
		return fmt.Errorf("issue not on project board")
	}

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
	_, err := runGh(ctx, "api", "graphql", "-f", "query="+mutation, "-F", "projectId="+projectID, "-F", "itemId="+itemID, "-F", "fieldId="+fieldID, "-F", "optionId="+optionID)
	return err
}
