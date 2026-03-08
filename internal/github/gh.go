package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

func runGh(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh %v: %w (%s)", args, err, stderr.String())
	}
	return out, nil
}

func CurrentUser(ctx context.Context) (string, error) {
	out, err := runGh(ctx, "api", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}
