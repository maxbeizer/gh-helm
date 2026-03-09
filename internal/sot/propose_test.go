package sot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPropose_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SOT.md")
	if err := os.WriteFile(path, []byte("# SOT\n\nSome content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Propose(path, "Added new feature X", "sess-123", "")
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "## Proposed Updates") {
		t.Error("expected Proposed Updates section")
	}
	if !strings.Contains(content, "Added new feature X") {
		t.Error("expected decision text")
	}
	if !strings.Contains(content, "sess-123") {
		t.Error("expected session id")
	}
	if !strings.Contains(content, "Pending human review") {
		t.Error("expected pending review marker")
	}
}

func TestPropose_WithPR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SOT.md")
	if err := os.WriteFile(path, []byte("# SOT\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Propose(path, "Fixed bug Y", "sess-456", "#42")
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "Based on work in PR #42") {
		t.Error("expected PR reference")
	}
}

func TestPropose_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SOT.md")
	initial := "# SOT\n\n## Proposed Updates\n\n> Existing proposal\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Propose(path, "Second proposal", "sess-789", "")
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "Existing proposal") {
		t.Error("expected existing proposal to remain")
	}
	if !strings.Contains(content, "Second proposal") {
		t.Error("expected new proposal")
	}
}

func TestRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SOT.md")
	expected := "# Test SOT\n\nContent here.\n"
	if err := os.WriteFile(path, []byte(expected), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != expected {
		t.Errorf("Read = %q, want %q", got, expected)
	}
}

func TestRead_NotFound(t *testing.T) {
	_, err := Read("/nonexistent/path/SOT.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSummarizeDiff(t *testing.T) {
	diff := `diff --git a/cmd/project_sot.go b/cmd/project_sot.go
index abc..def 100644
--- a/cmd/project_sot.go
+++ b/cmd/project_sot.go
@@ -1,5 +1,10 @@
 some content
diff --git a/internal/sot/sync.go b/internal/sot/sync.go
index abc..def 100644
--- a/internal/sot/sync.go
+++ b/internal/sot/sync.go
@@ -1,3 +1,8 @@
 some content
diff --git a/docs/WORKFLOW.md b/docs/WORKFLOW.md
new file mode 100644
--- /dev/null
+++ b/docs/WORKFLOW.md
@@ -0,0 +1,5 @@
 some content`

	changes := summarizeDiff(diff)
	if len(changes) != 3 {
		t.Fatalf("expected 3 categories, got %d: %v", len(changes), changes)
	}

	expected := map[string]bool{
		"CLI commands":  true,
		"SOT logic":     true,
		"documentation": true,
	}
	for _, c := range changes {
		if !expected[c] {
			t.Errorf("unexpected category: %s", c)
		}
	}
}

func TestCategorizeFile(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"cmd/root.go", "CLI commands"},
		{"internal/sot/sync.go", "SOT logic"},
		{"internal/github/pr.go", "GitHub API layer"},
		{"internal/agent/start.go", "project agent"},
		{"internal/manager/observe.go", "manager agent"},
		{"internal/config/project.go", "configuration"},
		{"docs/WORKFLOW.md", "documentation"},
		{"internal/foo/bar_test.go", "tests"},
		{"internal/pillars/map.go", "pillars package"},
		{"go.mod", "go.mod"},
	}
	for _, tt := range tests {
		got := categorizeFile(tt.path)
		if got != tt.want {
			t.Errorf("categorizeFile(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
