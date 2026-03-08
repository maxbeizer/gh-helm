package github

import (
	"context"
	"encoding/json"
	"fmt"
)

type projectV2 struct {
	ID     string `json:"id"`
	Fields struct {
		Nodes []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Options []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"options"`
		} `json:"nodes"`
	} `json:"fields"`
	ItemByContent *struct {
		ID string `json:"id"`
	} `json:"projectV2ItemByContent"`
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
	query := `
query($owner: String!, $number: Int!, $issueId: ID!) {
  organization(login: $owner) {
    projectV2(number: $number) {
      id
      fields(first: 50) {
        nodes {
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
        }
      }
      projectV2ItemByContent(contentId: $issueId) {
        id
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
            options {
              id
              name
            }
          }
        }
      }
      projectV2ItemByContent(contentId: $issueId) {
        id
      }
    }
  }
}
`
	out, err := runGh(ctx, "api", "graphql", "-f", "query="+query, "-F", "owner="+owner, "-F", fmt.Sprintf("number=%d", projectNumber), "-F", "issueId="+issueNodeID)
	if err != nil {
		return err
	}

	var resp projectQueryResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return err
	}

	project := resp.Organization
	if project == nil || project.Project == nil {
		project = resp.User
	}
	if project == nil || project.Project == nil {
		return fmt.Errorf("project not found")
	}

	fieldID := ""
	optionID := ""
	for _, field := range project.Project.Fields.Nodes {
		if field.Name != "Status" {
			continue
		}
		fieldID = field.ID
		for _, option := range field.Options {
			if option.Name == status {
				optionID = option.ID
				break
			}
		}
	}
	if fieldID == "" || optionID == "" {
		return fmt.Errorf("status field or option not found")
	}
	if project.Project.ItemByContent == nil {
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
	_, err = runGh(ctx, "api", "graphql", "-f", "query="+mutation, "-F", "projectId="+project.Project.ID, "-F", "itemId="+project.Project.ItemByContent.ID, "-F", "fieldId="+fieldID, "-F", "optionId="+optionID)
	return err
}
