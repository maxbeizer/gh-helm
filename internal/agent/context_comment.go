package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// BuildContextComment generates a structured markdown comment with project
// context to help a coding agent (e.g., Copilot) understand the codebase.
func BuildContextComment(ctx context.Context, sotContent string) string {
	pc := DetectProjectContext(ctx)

	var sb strings.Builder
	sb.WriteString("> 🤖 **gh-helm agent context** — This information was added automatically to help\n")
	sb.WriteString("> the coding agent understand the project. It supplements the issue description above.\n\n")

	if pc.Language != "" && pc.Language != "Unknown" {
		sb.WriteString(fmt.Sprintf("**Language:** %s\n", pc.Language))
	}

	if cmds := detectBuildCommands(); cmds != "" {
		sb.WriteString(cmds)
	}

	if len(pc.Manifests) > 0 {
		sb.WriteString(fmt.Sprintf("**Manifests:** %s\n", strings.Join(pc.Manifests, ", ")))
	}

	sb.WriteString("\n")

	if sotContent != "" {
		sb.WriteString("<details><summary>Source of Truth</summary>\n\n")
		sb.WriteString(sotContent)
		sb.WriteString("\n\n</details>\n\n")
	}

	if pc.ManifestSummary != "" {
		sb.WriteString("<details><summary>Manifest Contents</summary>\n\n")
		sb.WriteString("```\n")
		sb.WriteString(pc.ManifestSummary)
		sb.WriteString("\n```\n\n</details>\n\n")
	}

	if pc.Tree != "" {
		sb.WriteString("<details><summary>File Tree</summary>\n\n")
		sb.WriteString("```\n")
		sb.WriteString(pc.Tree)
		sb.WriteString("\n```\n\n</details>\n\n")
	}

	if pc.GitIgnore != "" {
		sb.WriteString("<details><summary>.gitignore</summary>\n\n")
		sb.WriteString("```\n")
		sb.WriteString(pc.GitIgnore)
		sb.WriteString("\n```\n\n</details>\n")
	}

	return sb.String()
}

// detectBuildCommands looks for common build/test commands based on files
// in the current directory.
func detectBuildCommands() string {
	var lines []string

	if _, err := os.Stat("Makefile"); err == nil {
		lines = append(lines, "**Build:** `make`")
	}
	if _, err := os.Stat("go.mod"); err == nil {
		lines = append(lines, "**Build:** `go build ./...`")
		lines = append(lines, "**Test:** `go test ./...`")
	}
	if _, err := os.Stat("package.json"); err == nil {
		lines = append(lines, "**Build:** `npm run build`")
		lines = append(lines, "**Test:** `npm test`")
	}
	if _, err := os.Stat("Cargo.toml"); err == nil {
		lines = append(lines, "**Build:** `cargo build`")
		lines = append(lines, "**Test:** `cargo test`")
	}
	if _, err := os.Stat("pyproject.toml"); err == nil {
		lines = append(lines, "**Test:** `pytest`")
	}
	if _, err := os.Stat("Gemfile"); err == nil {
		lines = append(lines, "**Test:** `bundle exec rspec`")
	}
	if _, err := os.Stat("pom.xml"); err == nil {
		lines = append(lines, "**Build:** `mvn package`")
		lines = append(lines, "**Test:** `mvn test`")
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
