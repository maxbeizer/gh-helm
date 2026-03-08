package notifications

import "context"

type Notifier interface {
	Notify(ctx context.Context, msg Message) error
}

type Message struct {
	Title   string
	Body    string
	Channel string
	URL     string
}
