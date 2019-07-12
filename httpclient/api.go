package httpclient

import (
	"context"
)

//go:generate mockgen -source=api.go -destination=gen_HttpClientMock.go -package=httpclient github.com/MarcGrol/forwardhttp/httpclient HTTPSender

type HTTPSender interface {
	Send(c context.Context, req Request) (*Response, error)
}
