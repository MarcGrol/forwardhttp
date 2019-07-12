package lastdelivery

import (
	"context"
	"log"

	"github.com/MarcGrol/forwardhttp/httpclient"
)

type LastDelivery struct {
}

func NewLastDelivery() LastDeliverer {
	return &LastDelivery{}
}
func (l LastDelivery) OnLastDelivery(c context.Context, req httpclient.Request, resp *httpclient.Response, err error) {
	log.Printf("Last delivery: req: %+v, resp: %+v, err: %s", req, resp, err)
}
