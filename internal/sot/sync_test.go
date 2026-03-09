package sot

import (
	"testing"
)

func TestReconcile_NoIssueRefs(t *testing.T) {
	content := `# SOT

## Next Up

- AI-powered pillar inference
- Manager agent learning
`
	closed := map[int]bool{1: true, 2: true}
	_, result := reconcile(content, closed)

	if len(result.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(result.Removed))
	}
	if len(result.Kept) != 2 {
		t.Errorf("expected 2 kept, got %d", len(result.Kept))
	}
}

func TestReconcile_RemovesClosedIssues(t *testing.T) {
	content := `# SOT

## Outcomes

- [x] Existing outcome

## Next Up

- Fix auth bug (#42)
- AI-powered pillar inference
- Add dashboard (#99)
`
	closed := map[int]bool{42: true}
	newContent, result := reconcile(content, closed)

	if len(result.Removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(result.Removed))
	}
	if len(result.Kept) != 2 {
		t.Errorf("expected 2 kept, got %d: %v", len(result.Kept), result.Kept)
	}

	// Check that the closed item was moved to outcomes
	if !contains(newContent, "- [x] Fix auth bug (#42)") {
		t.Error("expected closed item to appear in outcomes")
	}
	// Check that the item was removed from Next Up section
	lines := splitLines(newContent)
	inNextUp := false
	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "## Next Up" {
			inNextUp = true
			continue
		}
		if len(trimmed) > 0 && trimmed[:1] == "#" {
			inNextUp = false
		}
		if inNextUp && contains(line, "#42") {
			t.Error("issue #42 should not remain in Next Up")
		}
	}
}

func TestReconcile_AllClosed(t *testing.T) {
	content := `# SOT

## Next Up

- Item one (#1)
- Item two (#2)
`
	closed := map[int]bool{1: true, 2: true}
	_, result := reconcile(content, closed)

	if len(result.Removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(result.Removed))
	}
	if len(result.Kept) != 0 {
		t.Errorf("expected 0 kept, got %d", len(result.Kept))
	}
}

func TestReconcile_NoClosed(t *testing.T) {
	content := `# SOT

## Next Up

- Open item (#5)
- Another open (#6)
`
	closed := map[int]bool{}
	_, result := reconcile(content, closed)

	if len(result.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(result.Removed))
	}
	if result.Summary != "SOT is up to date — no completed items found in Next Up" {
		t.Errorf("unexpected summary: %s", result.Summary)
	}
}

func TestReconcile_MultipleIssueRefsOneLine(t *testing.T) {
	content := `# SOT

## Next Up

- Combine #3 and #4 into one
`
	closed := map[int]bool{3: true}
	_, result := reconcile(content, closed)

	if len(result.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(result.Removed))
	}
}

func TestReconcile_AsteriskBullets(t *testing.T) {
	content := `# SOT

## Next Up

* Done item (#10)
* Open item
`
	closed := map[int]bool{10: true}
	_, result := reconcile(content, closed)

	if len(result.Removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(result.Removed))
	}
	if len(result.Kept) != 1 {
		t.Errorf("expected 1 kept, got %d", len(result.Kept))
	}
}

// helpers
func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
