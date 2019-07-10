package main

import (
	"context"
	"fmt"
	"os"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2beta3"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2beta3"
)

type Task struct {
	WebhookURL string
	Payload    []byte
}

type TaskQueue interface {
	Enqueue(c context.Context, task Task) error
}

type GcloudTaskQueue struct {
	client *cloudtasks.Client
}

func NewQueue(c context.Context) (TaskQueue, error) {
	cloudTaskClient, err := cloudtasks.NewClient(c)
	if err != nil {
		return nil, fmt.Errorf("Error creating cloudtask-service: %s", err)
	}
	return &GcloudTaskQueue{
		client: cloudTaskClient,
	}, nil
}

func (q *GcloudTaskQueue) Enqueue(c context.Context, task Task) error {
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	locationId := os.Getenv("LOCATION_ID")
	queueName := os.Getenv("QUEUE_NAME")
	if queueName == "" {
		queueName = "default"
	}

	parentId := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectId, locationId, queueName)

	//serviceId := os.Getenv("GAE_SERVICE")
	//pushURL := fmt.Sprintf("https://%s-dot-%s.appspot.com/%s", serviceId, projectId, task.WebhookURL)
	pushURL := fmt.Sprintf("https://%s.appspot.com/%s", projectId, task.WebhookURL)

	_, err := q.client.CreateTask(c, &taskspb.CreateTaskRequest{
		Parent: parentId,
		Task: &taskspb.Task{
			PayloadType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        pushURL,
					Body:       task.Payload,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error creating submitting task to queue: %s", err)
	}
	return nil
}
