package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/maxbeizer/gh-helm/internal/github"
)

func GeneratePlan(ctx context.Context, model string, issue github.Issue, sotContent string) (github.Plan, error) {
	pc := DetectProjectContext(ctx)

	systemPrompt := buildSystemPrompt(pc)
	userPrompt := buildUserPrompt(issue, sotContent, pc)

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": userPrompt},
	}
	return github.GeneratePlan(ctx, model, messages)
}

func buildSystemPrompt(pc ProjectContext) string {
	var sb strings.Builder
	sb.WriteString("You are a senior software engineer.")

	if pc.Language != "" && pc.Language != "Unknown" {
		sb.WriteString(fmt.Sprintf(" This is a %s project.", pc.Language))
		sb.WriteString(fmt.Sprintf(" All generated code MUST be written in %s following the project's existing patterns and conventions.", pc.Language))
	}

	sb.WriteString(" Given the issue and project context below, produce a plan of what files to create/modify and the code changes needed.")
	sb.WriteString(` Output as JSON: {plan: string, files: [{path, action: "create"|"modify", content: string, description: string}]}.`)
	sb.WriteString(" Only create files that belong in version control — never generate compiled binaries, build artifacts, or files that would be excluded by .gitignore.")

	return sb.String()
}

func buildUserPrompt(issue github.Issue, sotContent string, pc ProjectContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Issue\n**%s**\n\n%s\n", issue.Title, issue.Body))

	if sotContent != "" {
		sb.WriteString(fmt.Sprintf("\n## Project Architecture (Source of Truth)\n%s\n", sotContent))
	}

	if pc.Language != "" && pc.Language != "Unknown" {
		sb.WriteString(fmt.Sprintf("\n## Detected Language\n%s\n", pc.Language))
	}

	if len(pc.Manifests) > 0 {
		sb.WriteString(fmt.Sprintf("\n## Manifest Files Found\n%s\n", strings.Join(pc.Manifests, ", ")))
	}

	if pc.ManifestSummary != "" {
		sb.WriteString(fmt.Sprintf("\n## Manifest Contents\n%s\n", pc.ManifestSummary))
	}

	if pc.GitIgnore != "" {
		sb.WriteString(fmt.Sprintf("\n## .gitignore\n```\n%s\n```\n", pc.GitIgnore))
	}

	if pc.Tree != "" {
		sb.WriteString(fmt.Sprintf("\n## Repository File Tree\n```\n%s\n```\n", pc.Tree))
	}

	return sb.String()
}
