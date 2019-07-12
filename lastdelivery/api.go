package lastdelivery

import (
	"context"

	"github.com/MarcGrol/forwardhttp/httpclient"
)

//go:generate mockgen -source=api.go -destination=gen_LastDeliveryMock.go -package=lastdelivery github.com/MarcGrol/forwardhttp/lastdelivery LastDeliverer

type LastDeliverer interface {
	OnLastDelivery(c context.Context, req httpclient.Request, resp *httpclient.Response, err error)
}
