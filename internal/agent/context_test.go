package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxbeizer/gh-helm/internal/github"
)

func TestDetectProjectContext_GoProject(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	os.WriteFile("go.mod", []byte("module example.com/myapp\n\ngo 1.22\n"), 0o644)
	os.WriteFile(".gitignore", []byte("bin/\n*.exe\n"), 0o644)

	pc := DetectProjectContext(context.Background())

	if pc.Language != "Go" {
		t.Errorf("Language = %q, want %q", pc.Language, "Go")
	}
	if len(pc.Manifests) == 0 || pc.Manifests[0] != "go.mod" {
		t.Errorf("Manifests = %v, want [go.mod]", pc.Manifests)
	}
	if !strings.Contains(pc.ManifestSummary, "module example.com/myapp") {
		t.Errorf("ManifestSummary missing go.mod content: %q", pc.ManifestSummary)
	}
	if !strings.Contains(pc.GitIgnore, "bin/") {
		t.Errorf("GitIgnore missing expected content: %q", pc.GitIgnore)
	}
}

func TestDetectProjectContext_NodeProject(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	os.WriteFile("package.json", []byte(`{"name": "my-app", "version": "1.0.0"}`), 0o644)
	os.WriteFile("tsconfig.json", []byte(`{"compilerOptions": {}}`), 0o644)

	pc := DetectProjectContext(context.Background())

	// package.json → "JavaScript/TypeScript", tsconfig.json → "TypeScript"
	// Both are valid; the important thing is it's not "Python" or "Go"
	if !strings.Contains(pc.Language, "TypeScript") {
		t.Errorf("Language = %q, want a TypeScript-related language", pc.Language)
	}
	if len(pc.Manifests) < 2 {
		t.Errorf("Manifests = %v, want at least 2 entries", pc.Manifests)
	}
}

func TestDetectProjectContext_PythonProject(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	os.WriteFile("requirements.txt", []byte("flask==3.0\nrequests\n"), 0o644)

	pc := DetectProjectContext(context.Background())

	if pc.Language != "Python" {
		t.Errorf("Language = %q, want %q", pc.Language, "Python")
	}
}

func TestDetectProjectContext_NoManifest(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	// No manifest files, no git repo — should return "Unknown"
	pc := DetectProjectContext(context.Background())

	// Without a git repo or manifest, language detection falls back.
	if pc.Language == "" {
		t.Error("Language should not be empty string, expected 'Unknown' or a detected language")
	}
}

func TestDetectProjectContext_NoGitIgnore(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	pc := DetectProjectContext(context.Background())

	if pc.GitIgnore != "" {
		t.Errorf("GitIgnore = %q, want empty string when no .gitignore exists", pc.GitIgnore)
	}
}

func TestReadGitIgnore(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	t.Run("file exists", func(t *testing.T) {
		os.WriteFile(".gitignore", []byte("bin/\n*.o\n"), 0o644)
		got := readGitIgnore()
		if got != "bin/\n*.o\n" {
			t.Errorf("readGitIgnore() = %q, want %q", got, "bin/\n*.o\n")
		}
	})

	t.Run("file missing", func(t *testing.T) {
		os.Remove(".gitignore")
		got := readGitIgnore()
		if got != "" {
			t.Errorf("readGitIgnore() = %q, want empty string", got)
		}
	})
}

func TestTopLanguage(t *testing.T) {
	tests := []struct {
		name  string
		votes map[string]int
		want  string
	}{
		{"single language", map[string]int{"Go": 3}, "Go"},
		{"clear winner", map[string]int{"Go": 5, "Python": 2}, "Go"},
		{"tie breaks alphabetically", map[string]int{"Python": 3, "Go": 3}, "Go"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := topLanguage(tc.votes)
			if got != tc.want {
				t.Errorf("topLanguage(%v) = %q, want %q", tc.votes, got, tc.want)
			}
		})
	}
}

func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"vendor/pkg/mod.go", true},
		{"node_modules/express/index.js", true},
		{".git/config", true},
		{"src/main.go", false},
		{"cmd/root.go", false},
		{"dist/bundle.js", true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := shouldSkipPath(tc.path)
			if got != tc.want {
				t.Errorf("shouldSkipPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	t.Run("includes language", func(t *testing.T) {
		pc := ProjectContext{Language: "Go"}
		prompt := buildSystemPrompt(pc)
		if !strings.Contains(prompt, "Go project") {
			t.Errorf("prompt missing language mention: %q", prompt)
		}
		if !strings.Contains(prompt, "MUST be written in Go") {
			t.Errorf("prompt missing language requirement: %q", prompt)
		}
	})

	t.Run("unknown language omits specifics", func(t *testing.T) {
		pc := ProjectContext{Language: "Unknown"}
		prompt := buildSystemPrompt(pc)
		if strings.Contains(prompt, "Unknown project") {
			t.Errorf("prompt should not mention Unknown as language: %q", prompt)
		}
	})

	t.Run("mentions gitignore avoidance", func(t *testing.T) {
		pc := ProjectContext{}
		prompt := buildSystemPrompt(pc)
		if !strings.Contains(prompt, ".gitignore") {
			t.Errorf("prompt should mention .gitignore: %q", prompt)
		}
	})
}

func TestBuildUserPrompt(t *testing.T) {
	issue := github.Issue{Title: "Add auth", Body: "We need JWT auth"}
	pc := ProjectContext{
		Language:        "Go",
		Manifests:       []string{"go.mod"},
		ManifestSummary: "--- go.mod ---\nmodule example.com/app",
		GitIgnore:       "bin/\n",
		Tree:            "cmd/main.go\ninternal/auth.go",
	}

	prompt := buildUserPrompt(issue, "# Architecture doc", pc)

	checks := []string{
		"Add auth",
		"JWT auth",
		"Architecture doc",
		"go.mod",
		"module example.com/app",
		"bin/",
		"cmd/main.go",
	}
	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("user prompt missing %q", check)
		}
	}
}

func TestStageFiles(t *testing.T) {
	var captured []string
	orig := RunGitFunc
	RunGitFunc = func(_ context.Context, args ...string) error {
		captured = args
		return nil
	}
	t.Cleanup(func() { RunGitFunc = orig })

	files := []github.FileChange{
		{Path: "cmd/main.go", Content: "package main"},
		{Path: "", Content: "skip me"},
		{Path: "internal/auth.go", Content: "package internal"},
	}

	err := stageFiles(context.Background(), files)
	if err != nil {
		t.Fatalf("stageFiles() error: %v", err)
	}

	// Should be: ["add", "--", "cmd/main.go", "internal/auth.go"]
	if len(captured) != 4 {
		t.Fatalf("git args = %v, want 4 elements", captured)
	}
	if captured[0] != "add" || captured[1] != "--" {
		t.Errorf("git args prefix = %v, want [add --]", captured[:2])
	}

	paths := captured[2:]
	if paths[0] != "cmd/main.go" || paths[1] != "internal/auth.go" {
		t.Errorf("staged paths = %v, want [cmd/main.go internal/auth.go]", paths)
	}
}

func TestStageFiles_Empty(t *testing.T) {
	orig := RunGitFunc
	called := false
	RunGitFunc = func(_ context.Context, args ...string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { RunGitFunc = orig })

	err := stageFiles(context.Background(), nil)
	if err != nil {
		t.Fatalf("stageFiles() error: %v", err)
	}
	if called {
		t.Error("stageFiles() should not call git when no files to stage")
	}
}

func TestManifestLanguages_Coverage(t *testing.T) {
	// Ensure we have entries for common manifest files.
	expected := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "pom.xml", "Gemfile"}
	for _, name := range expected {
		if _, ok := manifestLanguages[name]; !ok {
			t.Errorf("manifestLanguages missing entry for %q", name)
		}
	}
}

func TestDetectLanguage_MultipleManifests(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	// Go project with a Makefile — should detect Go, not Make.
	os.WriteFile("go.mod", []byte("module example.com/app\n\ngo 1.22\n"), 0o644)
	os.WriteFile("go.sum", []byte(""), 0o644)
	os.WriteFile("Makefile", []byte("build:\n\tgo build ./...\n"), 0o644)

	lang, manifests, _ := detectLanguage(context.Background())
	if lang != "Go" {
		t.Errorf("Language = %q, want %q", lang, "Go")
	}

	// All three manifests should be found.
	foundGoMod := false
	for _, m := range manifests {
		if m == "go.mod" {
			foundGoMod = true
		}
	}
	if !foundGoMod {
		t.Errorf("manifests = %v, expected go.mod", manifests)
	}
}

func TestDetectLanguage_ManifestContents(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(tmp)

	goMod := "module example.com/app\n\ngo 1.22\n\nrequire github.com/spf13/cobra v1.8.0\n"
	os.WriteFile("go.mod", []byte(goMod), 0o644)

	_, _, summary := detectLanguage(context.Background())
	if !strings.Contains(summary, "github.com/spf13/cobra") {
		t.Errorf("manifest summary should include dependency info: %q", summary)
	}
}

// Test that stageFiles uses filepath properly.
func TestStageFiles_NestedPaths(t *testing.T) {
	var captured []string
	orig := RunGitFunc
	RunGitFunc = func(_ context.Context, args ...string) error {
		captured = args
		return nil
	}
	t.Cleanup(func() { RunGitFunc = orig })

	files := []github.FileChange{
		{Path: filepath.Join("internal", "deep", "file.go"), Content: "package deep"},
	}

	err := stageFiles(context.Background(), files)
	if err != nil {
		t.Fatalf("stageFiles() error: %v", err)
	}

	expectedPath := filepath.Join("internal", "deep", "file.go")
	if len(captured) != 3 || captured[2] != expectedPath {
		t.Errorf("git args = %v, want [add -- %s]", captured, expectedPath)
	}
}
