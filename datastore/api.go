package datastore

import "context"

type DataStorer interface {
	Put(c context.Context, kind, uid string, value interface{}) error
}
