package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maxbeizer/gh-helm/internal/github"
)

type CodespaceOpts struct {
	Repo        string
	Branch      string
	Machine     string
	IdleTimeout string
}

type codespaceInfo struct {
	Name   string `json:"name"`
	WebURL string `json:"webUrl"`
	State  string `json:"state"`
}

func CreateCodespace(ctx context.Context, opts CodespaceOpts) (string, string, error) {
	if opts.Repo == "" {
		return "", "", fmt.Errorf("codespace repo is required")
	}
	args := []string{"codespace", "create", "--repo", opts.Repo}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	if opts.Machine != "" {
		args = append(args, "--machine", opts.Machine)
	}
	if opts.IdleTimeout != "" {
		args = append(args, "--idle-timeout", opts.IdleTimeout)
	}
	args = append(args, "--json", "name,webUrl")

	out, err := github.RunWith(ctx, args...)
	if err != nil {
		return "", "", fmt.Errorf("create codespace for %s: %w", opts.Repo, err)
	}
	var payload codespaceInfo
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", "", fmt.Errorf("parse codespace response: %w", err)
	}
	return payload.Name, payload.WebURL, nil
}

func WaitForReady(ctx context.Context, name string, timeout time.Duration) error {
	if name == "" {
		return fmt.Errorf("codespace name is required")
	}
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("codespace %s not ready before timeout", name)
		}
		state, err := codespaceState(ctx, name)
		if err != nil {
			return fmt.Errorf("check codespace %s state: %w", name, err)
		}
		if state == "Available" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}

func DeleteCodespace(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("codespace name is required")
	}
	_, err := github.RunWith(ctx, "codespace", "delete", "--codespace", name, "--force")
	if err != nil {
		return fmt.Errorf("delete codespace %s: %w", name, err)
	}
	return nil
}

func codespaceState(ctx context.Context, name string) (string, error) {
	out, err := github.RunWith(ctx, "codespace", "list", "--json", "name,state")
	if err != nil {
		return "", fmt.Errorf("list codespaces: %w", err)
	}
	var payload []codespaceInfo
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("parse codespace list: %w", err)
	}
	for _, item := range payload {
		if item.Name == name {
			return item.State, nil
		}
	}
	return "", fmt.Errorf("codespace %s not found", name)
}
