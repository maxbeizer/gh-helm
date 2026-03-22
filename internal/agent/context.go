package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectContext holds detected information about the target project.
type ProjectContext struct {
	Language   string   // Primary language (e.g., "Go", "TypeScript")
	Manifests  []string // Detected manifest files
	ManifestSummary string // Key contents from manifest files
	GitIgnore  string   // Contents of .gitignore if present
	Tree       string   // Directory tree snapshot
}

// manifest maps file names to their associated language.
var manifestLanguages = map[string]string{
	"go.mod":         "Go",
	"go.sum":         "Go",
	"package.json":   "JavaScript/TypeScript",
	"tsconfig.json":  "TypeScript",
	"Cargo.toml":     "Rust",
	"pyproject.toml": "Python",
	"setup.py":       "Python",
	"requirements.txt": "Python",
	"Pipfile":        "Python",
	"pom.xml":        "Java",
	"build.gradle":   "Java",
	"build.gradle.kts": "Kotlin",
	"Gemfile":        "Ruby",
	"mix.exs":        "Elixir",
	"composer.json":  "PHP",
	"Makefile":       "Make",
}

// extensionLanguages maps file extensions to languages for fallback detection.
var extensionLanguages = map[string]string{
	".go":   "Go",
	".ts":   "TypeScript",
	".tsx":  "TypeScript",
	".js":   "JavaScript",
	".jsx":  "JavaScript",
	".py":   "Python",
	".rs":   "Rust",
	".java": "Java",
	".kt":   "Kotlin",
	".rb":   "Ruby",
	".ex":   "Elixir",
	".exs":  "Elixir",
	".php":  "PHP",
	".cs":   "C#",
	".cpp":  "C++",
	".c":    "C",
	".swift": "Swift",
}

// DetectProjectContext gathers language, manifest, gitignore, and tree
// information from the current working directory.
func DetectProjectContext(ctx context.Context) ProjectContext {
	pc := ProjectContext{}

	pc.Language, pc.Manifests, pc.ManifestSummary = detectLanguage(ctx)
	pc.GitIgnore = readGitIgnore()
	pc.Tree = buildTree(ctx)

	return pc
}

// detectLanguage checks for manifest files first, then falls back to
// counting file extensions.
func detectLanguage(ctx context.Context) (string, []string, string) {
	var found []string
	var summaryParts []string
	langVotes := map[string]int{}

	// Check manifests in the current directory.
	for file, lang := range manifestLanguages {
		if _, err := os.Stat(file); err == nil {
			found = append(found, file)
			langVotes[lang]++

			// Read key manifest contents (first 2KB).
			if data, err := os.ReadFile(file); err == nil {
				content := string(data)
				if len(content) > 2048 {
					content = content[:2048] + "\n... (truncated)"
				}
				summaryParts = append(summaryParts, fmt.Sprintf("--- %s ---\n%s", file, content))
			}
		}
	}

	sort.Strings(found)
	sort.Strings(summaryParts)

	// If we found manifests, pick the language with the most votes.
	if len(langVotes) > 0 {
		return topLanguage(langVotes), found, strings.Join(summaryParts, "\n\n")
	}

	// Fallback: count file extensions.
	lang := detectByExtensions(ctx)
	return lang, found, ""
}

// detectByExtensions scans the repo for source files and picks the most
// common language by extension.
func detectByExtensions(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	langCounts := map[string]int{}
	for _, line := range strings.Split(string(out), "\n") {
		ext := filepath.Ext(strings.TrimSpace(line))
		if lang, ok := extensionLanguages[ext]; ok {
			langCounts[lang]++
		}
	}

	if len(langCounts) == 0 {
		return "Unknown"
	}
	return topLanguage(langCounts)
}

func topLanguage(votes map[string]int) string {
	best := ""
	bestCount := 0
	for lang, count := range votes {
		if count > bestCount || (count == bestCount && lang < best) {
			best = lang
			bestCount = count
		}
	}
	return best
}

func readGitIgnore() string {
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		return ""
	}
	content := string(data)
	if len(content) > 2048 {
		content = content[:2048] + "\n... (truncated)"
	}
	return content
}

// buildTree produces an indented directory tree of tracked files, excluding
// common noise directories. Capped at 80 entries.
func buildTree(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "--name-only", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		// Fallback for repos with no commits yet.
		return fallbackTree(ctx)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if shouldSkipPath(line) {
			continue
		}
		filtered = append(filtered, line)
		if len(filtered) >= 80 {
			filtered = append(filtered, "... (truncated)")
			break
		}
	}
	return strings.Join(filtered, "\n")
}

func fallbackTree(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "find", ".", "-type", "f",
		"-not", "-path", "./.git/*",
		"-not", "-path", "./node_modules/*",
		"-not", "-path", "./vendor/*",
		"-not", "-path", "./.helm/*",
	)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 80 {
		lines = append(lines[:80], "... (truncated)")
	}
	return strings.Join(lines, "\n")
}

func shouldSkipPath(path string) bool {
	skipPrefixes := []string{
		"vendor/", "node_modules/", ".git/", ".helm/",
		"dist/", "build/", "__pycache__/", ".next/",
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
