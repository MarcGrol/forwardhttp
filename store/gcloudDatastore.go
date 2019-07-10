package store

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/datastore"
)

type GcloudDataStore struct {
	client *datastore.Client
}

func NewStore(c context.Context) (DataStore, func(), error) {
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	client, err := datastore.NewClient(c, projectId)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating datastore-client: %s", err)
	}
	return &GcloudDataStore{
			client: client,
		}, func() {
			client.Close()
		}, nil
}

func (s *GcloudDataStore) Put(c context.Context, kind, uid string, objectToStore interface{}) error {
	_, err := s.client.Put(c, datastore.NameKey(kind, uid, nil), objectToStore)
	if err != nil {
		return fmt.Errorf("Error creating entity %s-%s: %s", kind, uid, err)
	}
	return nil
}
