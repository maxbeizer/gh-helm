package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	slog.Debug("fetching project info", "owner", owner, "project", number)
	sleepRateLimit()

	type projectNode struct {
		ID    string `json:"id"`
		Items struct {
			TotalCount int `json:"totalCount"`
		} `json:"items"`
	}

	// Try user first, then organization. We can't query both in the same
	// GraphQL request because gh treats a "Could not resolve" as a fatal error.
	for _, ownerType := range []string{"user", "organization"} {
		query := fmt.Sprintf(`
query($owner: String!, $number: Int!) {
  %s(login: $owner) {
    projectV2(number: $number) {
      id
      items(first: 1) {
        totalCount
      }
    }
  }
}`, ownerType)
		out, err := runGh(ctx, "api", "graphql", "-f", "query="+query, "-F", "owner="+owner, "-F", fmt.Sprintf("number=%d", number))
		if err != nil {
			slog.Debug("project info: owner type not matched", "ownerType", ownerType, "error", err)
			continue
		}

		// gh wraps responses in {"data": {...}}. Parse accordingly.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(out, &raw); err != nil {
			slog.Debug("project info: unmarshal failed", "ownerType", ownerType, "error", err)
			continue
		}
		// The top-level key might be "data" (gh api graphql) or the owner type directly.
		payload := raw[ownerType]
		if dataRaw, ok := raw["data"]; ok {
			var data map[string]json.RawMessage
			if err := json.Unmarshal(dataRaw, &data); err == nil {
				if p, ok := data[ownerType]; ok {
					payload = p
				}
			}
		}
		if payload == nil {
			slog.Debug("project info: no payload for owner type", "ownerType", ownerType)
			continue
		}

		var ownerNode struct {
			Project *projectNode `json:"projectV2"`
		}
		if err := json.Unmarshal(payload, &ownerNode); err != nil || ownerNode.Project == nil {
			slog.Debug("project info: no project in response", "ownerType", ownerType)
			continue
		}

		p := ownerNode.Project
		slog.Debug("found project", "ownerType", ownerType, "projectID", p.ID, "itemCount", p.Items.TotalCount)
		return ProjectInfo{ID: p.ID, ItemCount: p.Items.TotalCount, OwnerScope: ownerType}, nil
	}
	return ProjectInfo{}, fmt.Errorf("project %d not found for owner %q (tried user and organization)", number, owner)
}
