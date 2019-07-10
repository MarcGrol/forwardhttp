package store

import "context"

type DataStore interface {
	Put(c context.Context, kind, uid string, value interface{}) error
}
