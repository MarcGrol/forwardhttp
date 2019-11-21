package datastore

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/datastore"
)

type gcloudDataStore struct {
	client *datastore.Client
}

func NewStore(c context.Context) (DataStorer, func(), error) {
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	client, err := datastore.NewClient(c, projectId)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating datastore-client: %s", err)
	}
	return &gcloudDataStore{
			client: client,
		}, func() {
			client.Close()
		}, nil
}

func (s *gcloudDataStore) Put(c context.Context, kind, uid string, objectToStore interface{}) error {
	_, err := s.client.Put(c, datastore.NameKey(kind, uid, nil), objectToStore)
	if err != nil {
		return fmt.Errorf("Error creating entity %s-%s: %s", kind, uid, err)
	}
	return nil
}
