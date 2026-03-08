package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type SlackNotifier struct {
	WebhookURL string
}

func (s *SlackNotifier) Notify(ctx context.Context, msg Message) error {
	payload := map[string]string{
		"text": msg.Title + "\n" + msg.Body,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
