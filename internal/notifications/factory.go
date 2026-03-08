package notifications

import "github.com/maxbeizer/gh-helm/internal/config"

func New(cfg config.Config, repo string, issueNumber int) Notifier {
	switch cfg.Notifications.Channel {
	case "slack":
		if cfg.Notifications.WebhookURL == "" {
			return nil
		}
		return &SlackNotifier{WebhookURL: cfg.Notifications.WebhookURL}
	case "github":
		return &GitHubNotifier{Repo: repo, IssueNumber: issueNumber}
	default:
		return nil
	}
}
