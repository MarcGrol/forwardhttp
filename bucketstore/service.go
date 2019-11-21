package bucketstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type bucketStore struct {
	client  *storage.Client
}

func NewBucketStore(c context.Context) (BucketStorer,error) {
	storageClient, err := storage.NewClient(c)
	if err != nil {
			return nil, fmt.Errorf("Error creating storage-client: %s", err)
	}
	return &bucketStore{
		client:storageClient,
	}, nil
}

func (b bucketStore)Put(c context.Context, bucketName string, object Object) error{
	writer := b.client.Bucket(bucketName).Object(object.Name).NewWriter(c)
	reader := bytes.NewReader(object.Data)
	_, err := io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("Error uploading bucket object %s/%s: %s", bucketName, object.Name, err)
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("Error closing bucket object %s/%s: %s", bucketName, object.Name, err)
	}
	return nil
}

func (b bucketStore)ListMetaInfo(c context.Context, bucketName, objectName string) ([]Object, error){
	objects := []Object{}
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	it := b.client.Buckets(c, projectId)
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error fetching bucket objects: %s", err)
		}
		objects  = append(objects, Object{
			Name:bucketAttrs.Name,
			CreationTimestamp:bucketAttrs.Created,
		})
	}
	return objects, nil
}

func (b bucketStore)Delete(c context.Context, bucketName, objectName string) error{
	err := b.client.Bucket(bucketName).Object(objectName).Delete(c)
	if err != nil {
		return fmt.Errorf("Error deleting bucket object %s/%s: %s", bucketName, objectName, err)
	}
	return nil
}


