package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func runGh(ctx context.Context, args ...string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		cmd := exec.CommandContext(ctx, "gh", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err == nil {
			return out, nil
		}
		lastErr = fmt.Errorf("gh %v: %w (%s)", args, err, stderr.String())
		// Don't retry on non-transient errors (auth, not found, etc.)
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "404") || strings.Contains(stderrStr, "401") || strings.Contains(stderrStr, "403") {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func RunWith(ctx context.Context, args ...string) ([]byte, error) {
	return runGh(ctx, args...)
}

func CurrentUser(ctx context.Context) (string, error) {
	out, err := runGh(ctx, "api", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}
