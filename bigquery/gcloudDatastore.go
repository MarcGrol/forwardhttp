package bigquery

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"os"
)

type bigqueryDataStore struct {
	client *bigquery.Client
}

func NewStore(c context.Context) (BigQueryStorer, func(), error) {
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	client, err := bigquery.NewClient(c, projectId)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating bigquery-client: %s", err)
	}
	return &bigqueryDataStore{
			client: client,
		}, func() {
			client.Close()
		}, nil
}

func (s *bigqueryDataStore) Put(c context.Context, uid string, objectToStore interface{}) error {
	err := s.client.Dataset("forwardhttp_dataset").Table("RequestResponseTable").Uploader().Put(c, objectToStore)
	if err != nil {
		return fmt.Errorf("Error creating record %s: %s", uid, err)
	}
	return nil
}
