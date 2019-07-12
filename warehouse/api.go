package warehouse

import (
	"context"

	"github.com/MarcGrol/forwardhttp/httpclient"
)

type Stats struct {
	RetryCount    int32
	MaxRetryCount int32
}

//go:generate mockgen -source=api.go -destination=gen_WarehouseClientMock.go -package=warehouse github.com/MarcGrol/forwardhttp/warehouse Warehouser

type Warehouser interface {
	Put(c context.Context, req httpclient.Request, resp *httpclient.Response, err error, stats Stats) error
}
