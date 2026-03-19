package cmd

import (
	"context"
	"log/slog"
	"time"

	"github.com/maxbeizer/gh-helm/internal/output"
)

// outputHandler is an slog.Handler that writes structured JSON log entries
// through the output.Output writer, used by daemon commands when --json is set.
type outputHandler struct {
	out *output.Output
}

func newOutputHandler(out *output.Output) *outputHandler {
	return &outputHandler{out: out}
}

func (h *outputHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *outputHandler) Handle(_ context.Context, r slog.Record) error {
	payload := map[string]any{
		"time":    r.Time.Format(time.RFC3339),
		"message": r.Message,
	}
	r.Attrs(func(a slog.Attr) bool {
		payload[a.Key] = a.Value.Any()
		return true
	})
	return h.out.Print(payload)
}

func (h *outputHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *outputHandler) WithGroup(_ string) slog.Handler       { return h }
