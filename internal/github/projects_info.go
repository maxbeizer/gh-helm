package github

import (
	"context"
	"encoding/json"
	"fmt"
)

type ProjectInfo struct {
	ID         string
	ItemCount  int
	OwnerScope string
}

type projectInfoResponse struct {
	Organization *struct {
		Project *struct {
			ID    string `json:"id"`
			Items struct {
				TotalCount int `json:"totalCount"`
			} `json:"items"`
		} `json:"projectV2"`
	} `json:"organization"`
	User *struct {
		Project *struct {
			ID    string `json:"id"`
			Items struct {
				TotalCount int `json:"totalCount"`
			} `json:"items"`
		} `json:"projectV2"`
	} `json:"user"`
}

func FetchProjectInfo(ctx context.Context, owner string, number int) (ProjectInfo, error) {
	sleepRateLimit()
	query := `
query($owner: String!, $number: Int!) {
  organization(login: $owner) {
    projectV2(number: $number) {
      id
      items(first: 1) {
        totalCount
      }
    }
  }
  user(login: $owner) {
    projectV2(number: $number) {
      id
      items(first: 1) {
        totalCount
      }
    }
  }
}`
	out, err := runGh(ctx, "api", "graphql", "-f", "query="+query, "-F", "owner="+owner, "-F", fmt.Sprintf("number=%d", number))
	if err != nil {
		return ProjectInfo{}, err
	}
	var resp projectInfoResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return ProjectInfo{}, err
	}
	if resp.Organization != nil && resp.Organization.Project != nil {
		return ProjectInfo{ID: resp.Organization.Project.ID, ItemCount: resp.Organization.Project.Items.TotalCount, OwnerScope: "organization"}, nil
	}
	if resp.User != nil && resp.User.Project != nil {
		return ProjectInfo{ID: resp.User.Project.ID, ItemCount: resp.User.Project.Items.TotalCount, OwnerScope: "user"}, nil
	}
	return ProjectInfo{}, fmt.Errorf("project not found")
}
