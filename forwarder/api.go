package forwarder

import (
	"context"

	"github.com/MarcGrol/forwardhttp/httpclient"
)

//go:generate mockgen -source=api.go -destination=gen_ForwarderClientMock.go -package=forwarder github.com/MarcGrol/forwardhttp/forwarder Forwarder

type Forwarder interface {
	Forward(c context.Context, req httpclient.Request) (*httpclient.Response, error)
	ForwardAsync(c context.Context, req httpclient.Request) error
}
