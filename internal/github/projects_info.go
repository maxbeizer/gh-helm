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
		// Normalize the response key to parse uniformly.
		type infoResp struct {
			Owner *struct {
				Project *struct {
					ID    string `json:"id"`
					Items struct {
						TotalCount int `json:"totalCount"`
					} `json:"items"`
				} `json:"projectV2"`
			} `json:"owner"`
		}
		normalized := fmt.Sprintf(`{"owner":%s}`,
			json.RawMessage(out[len(fmt.Sprintf(`{%q:`, ownerType)):len(out)-1]))
		var resp infoResp
		if err := json.Unmarshal([]byte(normalized), &resp); err != nil {
			// Fallback: try the original response structure
			slog.Debug("project info: normalize failed, trying raw parse", "ownerType", ownerType)
			var rawResp projectInfoResponse
			if err2 := json.Unmarshal(out, &rawResp); err2 != nil {
				continue
			}
			if ownerType == "organization" && rawResp.Organization != nil && rawResp.Organization.Project != nil {
				p := rawResp.Organization.Project
				return ProjectInfo{ID: p.ID, ItemCount: p.Items.TotalCount, OwnerScope: "organization"}, nil
			}
			if ownerType == "user" && rawResp.User != nil && rawResp.User.Project != nil {
				p := rawResp.User.Project
				return ProjectInfo{ID: p.ID, ItemCount: p.Items.TotalCount, OwnerScope: "user"}, nil
			}
			continue
		}
		if resp.Owner != nil && resp.Owner.Project != nil {
			p := resp.Owner.Project
			slog.Debug("found project", "ownerType", ownerType, "projectID", p.ID, "itemCount", p.Items.TotalCount)
			return ProjectInfo{ID: p.ID, ItemCount: p.Items.TotalCount, OwnerScope: ownerType}, nil
		}
	}
	return ProjectInfo{}, fmt.Errorf("project %d not found for owner %q (tried user and organization)", number, owner)
}
