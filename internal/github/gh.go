package github

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// RunGhFunc is the function used to execute gh CLI commands.
// Override in tests to provide mock responses.
var RunGhFunc = defaultRunGh

func runGh(ctx context.Context, args ...string) ([]byte, error) {
	return RunGhFunc(ctx, args...)
}

// ghCommandSummary returns a short description of the gh command for logging.
// It avoids dumping full GraphQL queries into debug output.
func ghCommandSummary(args []string) string {
	if len(args) == 0 {
		return "gh"
	}
	// For graphql calls, show "gh api graphql" plus the variable flags
	if len(args) >= 2 && args[0] == "api" && args[1] == "graphql" {
		var vars []string
		for _, a := range args[2:] {
			if strings.HasPrefix(a, "-F") || strings.HasPrefix(a, "-f") {
				continue
			}
			if strings.HasPrefix(a, "query=") {
				continue
			}
			vars = append(vars, a)
		}
		return fmt.Sprintf("gh api graphql [%s]", strings.Join(vars, " "))
	}
	return "gh " + strings.Join(args, " ")
}

func defaultRunGh(ctx context.Context, args ...string) ([]byte, error) {
	summary := ghCommandSummary(args)
	slog.Debug("gh: executing", "cmd", summary)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s
			slog.Debug("gh: retrying", "cmd", summary, "attempt", attempt+1, "delay", delay)
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
			slog.Debug("gh: success", "cmd", summary, "bytes", len(out))
			return out, nil
		}
		stderrStr := strings.TrimSpace(stderr.String())
		lastErr = fmt.Errorf("gh %v: %w (%s)", args, err, stderrStr)
		slog.Debug("gh: error", "cmd", summary, "attempt", attempt+1, "stderr", stderrStr)

		// Don't retry on non-transient errors (auth, not found, etc.)
		if strings.Contains(stderrStr, "404") || strings.Contains(stderrStr, "401") || strings.Contains(stderrStr, "403") ||
			strings.Contains(stderrStr, "Could not resolve") || strings.Contains(stderrStr, "doesn't exist on type") {
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
