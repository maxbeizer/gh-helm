package github

import (
	"bytes"
	"context"
	"encoding/json"
)

type labelInfo struct {
	Name string `json:"name"`
}

func CurrentRepo(ctx context.Context) (string, error) {
	sleepRateLimit()
	out, err := runGh(ctx, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}

func ListLabels(ctx context.Context, repo string) ([]string, error) {
	sleepRateLimit()
	args := []string{"label", "list", "--json", "name"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	out, err := runGh(ctx, args...)
	if err != nil {
		return nil, err
	}
	var labels []labelInfo
	if err := json.Unmarshal(out, &labels); err != nil {
		return nil, err
	}
	results := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Name != "" {
			results = append(results, label.Name)
		}
	}
	return results, nil
}

func CreateLabel(ctx context.Context, repo string, name string, color string, description string) error {
	sleepRateLimit()
	args := []string{"label", "create", name}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	if color != "" {
		args = append(args, "--color", color)
	}
	if description != "" {
		args = append(args, "--description", description)
	}
	_, err := runGh(ctx, args...)
	return err
}
