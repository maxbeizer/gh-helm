package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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

	out, err := runCmdOutput(ctx, "gh", args...)
	if err != nil {
		return "", "", err
	}
	var payload codespaceInfo
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", "", err
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
			return err
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
	_, err := runCmdOutput(ctx, "gh", "codespace", "delete", "--codespace", name, "--force")
	return err
}

func codespaceState(ctx context.Context, name string) (string, error) {
	out, err := runCmdOutput(ctx, "gh", "codespace", "list", "--json", "name,state")
	if err != nil {
		return "", err
	}
	var payload []codespaceInfo
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", err
	}
	for _, item := range payload {
		if item.Name == name {
			return item.State, nil
		}
	}
	return "", fmt.Errorf("codespace %s not found", name)
}
