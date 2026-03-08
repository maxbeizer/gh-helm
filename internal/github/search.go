package github

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type SearchItem struct {
	Title         string  `json:"title"`
	Number        int     `json:"number"`
	HTMLURL       string  `json:"html_url"`
	RepositoryURL string  `json:"repository_url"`
	Labels        []Label `json:"labels"`
	State         string  `json:"state"`
	CreatedAt     string  `json:"created_at"`
	ClosedAt      string  `json:"closed_at"`
	UpdatedAt     string  `json:"updated_at"`
	Body          string  `json:"body"`
}

type searchResponse struct {
	Items []SearchItem `json:"items"`
}

func SearchIssues(ctx context.Context, query string) ([]SearchItem, error) {
	sleepRateLimit()
	out, err := runGh(ctx, "api", "search/issues", "-f", "q="+query)
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func PullFiles(ctx context.Context, repo string, number int) ([]string, error) {
	sleepRateLimit()
	endpoint := "repos/" + repo + "/pulls/" + itoa(number) + "/files"
	out, err := runGh(ctx, "api", endpoint, "-f", "per_page=100")
	if err != nil {
		return nil, err
	}
	var payload []struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, err
	}
	files := make([]string, 0, len(payload))
	for _, file := range payload {
		if file.Filename != "" {
			files = append(files, file.Filename)
		}
	}
	return files, nil
}

func RepoFromURL(repoURL string) string {
	return strings.TrimPrefix(repoURL, "https://api.github.com/repos/")
}

func sleepRateLimit() {
	time.Sleep(3 * time.Second)
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
