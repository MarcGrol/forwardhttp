package warehouse

import (
	"context"

	"github.com/MarcGrol/forwardhttp/httpclient"
)

type Stats struct {
	RetryCount    int32
	MaxRetryCount int32
}

func (s Stats) IsLastAttempt() bool {
	return s.RetryCount == s.MaxRetryCount
}

//go:generate mockgen -source=api.go -destination=gen_WarehouseClientMock.go -package=warehouse github.com/MarcGrol/forwardhttp/warehouse Warehouser

type ForwardSummary struct {
	HttpRequest  httpclient.Request
	HttpResponse *httpclient.Response
	Error        error
	Stats        Stats
}
type Warehouser interface {
	Put(c context.Context, summary ForwardSummary) error
}
