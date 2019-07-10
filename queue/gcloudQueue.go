package queue

import (
	"context"
	"fmt"
	"log"
	"os"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2beta3"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2beta3"
)

type GcloudTaskQueue struct {
	client *cloudtasks.Client
}

func NewQueue(c context.Context) (TaskQueue, func(), error) {
	cloudTaskClient, err := cloudtasks.NewClient(c)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating cloudtask-client: %s", err)
	}
	return &GcloudTaskQueue{
			client: cloudTaskClient,
		}, func() {
			cloudTaskClient.Close()
		}, nil
}

func (q *GcloudTaskQueue) Enqueue(c context.Context, task Task) error {
	_, err := q.client.CreateTask(c, &taskspb.CreateTaskRequest{
		Parent: composeQueueName(),
		Task: &taskspb.Task{
			Name: composeTaskName(task.UID),
			PayloadType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        composeFullyQualifiedWebhookURL(task.WebhookURLPath),
					Body:       task.Payload,
				},
			},
			View: taskspb.Task_FULL,
		},
	})
	if err != nil {
		return fmt.Errorf("Error submitting task to queue: %s", err)
	}
	return nil
}

func composeQueueName() string {
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	locationId := os.Getenv("LOCATION_ID")
	queueName := os.Getenv("QUEUE_NAME")
	if queueName == "" {
		queueName = "default"
	}
	return fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectId, locationId, queueName)
}

func composeTaskName(taskUID string) string {
	return fmt.Sprintf("%s/tasks/%s", composeQueueName(), taskUID)
}

func composeFullyQualifiedWebhookURL(webhookUID string) string {
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	//serviceId := os.Getenv("GAE_SERVICE")
	//pushURL := fmt.Sprintf("https://%s-dot-%s.appspot.com/%s", serviceId, projectId, task.WebhookURLPath)

	// we are using the default service
	return fmt.Sprintf("https://%s.appspot.com/%s", projectId, webhookUID)
}

func (q *GcloudTaskQueue) IsLastAttempt(c context.Context, taskUID string) bool {
	var maxRetries int32 = -1

	// find characteristics of the queue
	queue, err := q.client.GetQueue(c, &taskspb.GetQueueRequest{
		Name: composeQueueName(),
	})
	if err != nil {
		log.Printf("Error creating submitting task to queue: %s", err)
		return false
	}

	if queue.RetryConfig != nil {
		log.Printf("queue: RetryConfig.MaxAttempts: %+v", queue.RetryConfig.MaxAttempts)
		maxRetries = queue.RetryConfig.MaxAttempts
	}

	// find characteristics of the task
	task, err := q.client.GetTask(c, &taskspb.GetTaskRequest{
		Name: composeTaskName(taskUID),
	})
	if err != nil {
		log.Printf("Error creating submitting task to queue: %s", err)
		return false
	}
	log.Printf("task: DispatchCount: %+v", task.DispatchCount)

	// Determine if this is the last attempt
	return maxRetries == task.DispatchCount
}
