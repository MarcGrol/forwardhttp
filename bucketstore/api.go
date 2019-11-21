package bucketstore

import (
	"context"
	"time"
)

type Object struct {
	Name              string
	CreationTimestamp time.Time
	Data 				[]byte
}


type BucketStorer interface {
	Put(c context.Context, bucketName string, object Object) error
	ListMetaInfo(c context.Context, bucketName, objectName string) ([]Object, error)
	Delete(c context.Context, bucketName, objectName string) error
}
