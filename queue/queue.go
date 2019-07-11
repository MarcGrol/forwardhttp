package queue

import (
	"context"
)

type Task struct {
	UID            string
	WebhookURLPath string
	Payload        []byte
	IsLastAttempt  bool
}

type TaskQueue interface {
	Enqueue(c context.Context, task Task) error
	IsLastAttempt(c context.Context, taskUID string) (int32, int32, bool)
}
