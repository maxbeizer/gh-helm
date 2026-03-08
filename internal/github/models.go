package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	var plan Plan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return Plan{}, fmt.Errorf("parse plan: %w", err)
	}
	return plan, nil
}

func authToken(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
