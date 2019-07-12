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

//go:generate mockgen -source=api.go -destination=gen_TaskQueuerMock.go -package=queue github.com/MarcGrol/forwardhttp/queue TaskQueuer

type TaskQueuer interface {
	Enqueue(c context.Context, task Task) error
	IsLastAttempt(c context.Context, taskUID string) (int32, int32, bool)
}
