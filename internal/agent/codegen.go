package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/github"
)

func GeneratePlan(ctx context.Context, model string, issue github.Issue, sotContent string) (github.Plan, error) {
	structure, _ := repoStructure(ctx)
	messages := []map[string]string{
		{
			"role": "system",
			"content": "You are a senior software engineer. Given this issue and project context, produce a plan of what files to create/modify and the code changes needed. Output as JSON: {plan: string, files: [{path, action: \"create\"|\"modify\", content: string, description: string}]}.",
		},
		{
			"role": "user",
			"content": fmt.Sprintf("Issue: %s\n\nBody:\n%s\n\nSource of Truth:\n%s\n\nRepo structure (sample):\n%s", issue.Title, issue.Body, sotContent, structure),
		},
	}
	return github.GeneratePlan(ctx, model, messages)
}

func repoStructure(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", `find . -type f \( -name '*.go' -o -name '*.ts' -o -name '*.py' \) | head -50`)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
