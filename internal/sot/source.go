package sot

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Propose(path, decision, session, pr string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if !strings.Contains(content, "## Proposed Updates") {
		content = strings.TrimSpace(content) + "\n\n## Proposed Updates\n"
	}

	stamp := time.Now().Format("2006-01-02")
	line := fmt.Sprintf("> **%s (agent session %s):** %s", stamp, session, decision)
	if pr != "" {
		line = fmt.Sprintf("%s\n> Based on work in PR %s. Pending human review.", line, pr)
	} else {
		line = fmt.Sprintf("%s\n> Pending human review.", line)
	}

	content = strings.TrimSpace(content) + "\n\n" + line + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
