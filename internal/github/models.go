package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Plan struct {
	Plan  string       `json:"plan"`
	Files []FileChange `json:"files"`
}

type FileChange struct {
	Path        string `json:"path"`
	Action      string `json:"action"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

type modelsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

const (
	modelsTimeout    = 120 * time.Second
	modelsMaxRetries = 3
	modelsRetryDelay = 2 * time.Second
	modelsInitialMaxTokens = 16384
)

type outputTruncatedError struct {
	ContentLength int
}

func (e *outputTruncatedError) Error() string {
	return fmt.Sprintf("model output truncated at %d characters — response exceeded max_tokens", e.ContentLength)
}

func GeneratePlan(ctx context.Context, model string, messages []map[string]string) (Plan, error) {
	token, err := authToken(ctx)
	if err != nil {
		return Plan{}, err
	}

	maxTokens := modelsInitialMaxTokens

	var lastErr error
	for attempt := 0; attempt < modelsMaxRetries; attempt++ {
		if attempt > 0 {
			delay := modelsRetryDelay * time.Duration(1<<uint(attempt-1))
			slog.Debug("models api: retrying", "attempt", attempt+1, "delay", delay, "max_tokens", maxTokens)
			select {
			case <-ctx.Done():
				return Plan{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		payload := map[string]any{
			"model":      model,
			"messages":   messages,
			"max_tokens": maxTokens,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return Plan{}, err
		}

		plan, err := doModelsRequest(ctx, token, body)
		if err == nil {
			return plan, nil
		}
		lastErr = err

		// On truncation, double max_tokens and retry.
		var truncErr *outputTruncatedError
		if errors.As(err, &truncErr) {
			maxTokens *= 2
			slog.Warn("model output was truncated, retrying with more tokens",
				"previous_length", truncErr.ContentLength, "new_max_tokens", maxTokens)
			continue
		}

		// Don't retry on non-transient errors.
		errMsg := err.Error()
		if strings.Contains(errMsg, "400") || strings.Contains(errMsg, "401") ||
			strings.Contains(errMsg, "403") || strings.Contains(errMsg, "404") ||
			strings.Contains(errMsg, "no choices") {
			return Plan{}, err
		}
		// Parse errors and malformed JSON are transient — the model may
		// produce valid output on retry.
		slog.Debug("models api: transient error", "attempt", attempt+1, "error", err)
	}
	return Plan{}, fmt.Errorf("models api failed after %d attempts: %w", modelsMaxRetries, lastErr)
}

func doModelsRequest(ctx context.Context, token string, body []byte) (Plan, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://models.github.ai/inference/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Plan{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: modelsTimeout}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Plan{}, fmt.Errorf("models api request cancelled: %w", ctx.Err())
		}
		return Plan{}, fmt.Errorf("models api request failed (possible timeout): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return Plan{}, fmt.Errorf("models api rate limited (429) — try again later or use a different model")
	}
	if resp.StatusCode == 408 || resp.StatusCode == 504 {
		return Plan{}, fmt.Errorf("models api timeout (%d) — request may be too complex", resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		return Plan{}, fmt.Errorf("models api server error (%d) — retrying", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Plan{}, fmt.Errorf("models api error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var apiResp modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return Plan{}, err
	}
	if len(apiResp.Choices) == 0 {
		return Plan{}, fmt.Errorf("models api returned no choices")
	}

	if apiResp.Choices[0].FinishReason == "length" {
		return Plan{}, &outputTruncatedError{
			ContentLength: len(apiResp.Choices[0].Message.Content),
		}
	}

	content := strings.TrimSpace(apiResp.Choices[0].Message.Content)
	slog.Debug("models api response", "contentLength", len(content), "preview", truncate(content, 200))

	// The model may return raw JSON, markdown-fenced JSON, or prose with
	// embedded JSON. Try to extract the JSON object.
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return Plan{}, fmt.Errorf("parse plan: model response does not contain a JSON object\n\nResponse preview:\n%s", truncate(content, 500))
	}
	slog.Debug("extracted JSON from response", "length", len(jsonStr))

	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return Plan{}, fmt.Errorf("parse plan: %w\n\nJSON preview:\n%s", err, truncate(jsonStr, 500))
	}
	return plan, nil
}

// extractJSON finds the first JSON object in content, handling:
// - raw JSON: {"plan": ...}
// - markdown fenced: ```json\n{...}\n```
// - prose with embedded JSON
func extractJSON(content string) string {
	// Try direct parse first.
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") {
		return content
	}

	// Try markdown code fence.
	if idx := strings.Index(content, "```json"); idx != -1 {
		start := idx + len("```json")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + len("```")
		// Skip optional language tag on same line.
		if nl := strings.Index(content[start:], "\n"); nl != -1 {
			start += nl + 1
		}
		if end := strings.Index(content[start:], "```"); end != -1 {
			candidate := strings.TrimSpace(content[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}

	// Last resort: find first { and last }.
	first := strings.Index(content, "{")
	last := strings.LastIndex(content, "}")
	if first != -1 && last > first {
		return content[first : last+1]
	}

	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func authToken(ctx context.Context) (string, error) {
	out, err := runGh(ctx, "auth", "token")
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	return string(bytes.TrimSpace(out)), nil
}
