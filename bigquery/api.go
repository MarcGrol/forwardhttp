package bigquery

import "context"

type BigQueryStorer interface {
	Put(c context.Context, uid string, value interface{}) error
}
