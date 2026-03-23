package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd(jsonFlag bool, jqExpr string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", jsonFlag, "")
	cmd.Flags().String("jq", jqExpr, "")
	return cmd
}

// captureStdout redirects os.Stdout and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}

func TestPrint_JSON(t *testing.T) {
	cmd := newTestCmd(true, "")
	o := New(cmd)

	data := map[string]any{"version": 1, "name": "test"}
	out := captureStdout(t, func() {
		if err := o.Print(data); err != nil {
			t.Fatalf("Print: %v", err)
		}
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if parsed["name"] != "test" {
		t.Errorf("name = %v, want %q", parsed["name"], "test")
	}
}

func TestPrint_JQ(t *testing.T) {
	cmd := newTestCmd(false, ".version")
	o := New(cmd)

	data := map[string]any{"version": 42, "name": "test"}
	out := captureStdout(t, func() {
		if err := o.Print(data); err != nil {
			t.Fatalf("Print: %v", err)
		}
	})

	trimmed := strings.TrimSpace(out)
	if trimmed != "42" {
		t.Errorf("jq output = %q, want %q", trimmed, "42")
	}
}

func TestPrint_Plain(t *testing.T) {
	cmd := newTestCmd(false, "")
	o := New(cmd)

	data := map[string]string{"hello": "world"}
	out := captureStdout(t, func() {
		if err := o.Print(data); err != nil {
			t.Fatalf("Print: %v", err)
		}
	})

	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Errorf("plain output = %q, expected to contain key/value", out)
	}
}

func TestPrint_InvalidJQ(t *testing.T) {
	cmd := newTestCmd(false, ".[invalid")
	o := New(cmd)

	data := map[string]any{"key": "val"}
	// Capture stdout so any partial output doesn't leak.
	_ = captureStdout(t, func() {
		err := o.Print(data)
		if err == nil {
			t.Fatal("expected error for invalid jq expression, got nil")
		}
	})
}

func TestWantsJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonFlag bool
		jqExpr   string
		want     bool
	}{
		{"no flags", false, "", false},
		{"json flag only", true, "", true},
		{"jq flag only", false, ".field", true},
		{"both flags", true, ".field", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newTestCmd(tc.jsonFlag, tc.jqExpr)
			o := New(cmd)
			if got := o.WantsJSON(); got != tc.want {
				t.Errorf("WantsJSON() = %v, want %v", got, tc.want)
			}
		})
	}
}
