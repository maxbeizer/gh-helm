package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
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
	} `json:"choices"`
}

func GeneratePlan(ctx context.Context, model string, messages []map[string]string) (Plan, error) {
	token, err := authToken(ctx)
	if err != nil {
		return Plan{}, err
	}

	payload := map[string]any{
		"model":    model,
		"messages": messages,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Plan{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://models.github.ai/inference/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Plan{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Plan{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Plan{}, fmt.Errorf("models api status %d", resp.StatusCode)
	}

	var apiResp modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return Plan{}, err
	}
	if len(apiResp.Choices) == 0 {
		return Plan{}, fmt.Errorf("models api returned no choices")
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
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
