package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type client struct {
}

func NewClient() HTTPSender {
	return &client{}
}

func (_ client) Send(c context.Context, req Request) (*Response, error) {
	req = composeRequestUID(req)

	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("Error creating http request for %s: %s", req.String(), err)
	}
	copyHeaders(httpReq.Header, req.Headers)

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Error sending %s: %s", req.String(), err)
	}
	defer httpResp.Body.Close()

	respPayload, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response %s: %s", req.String(), err)
	}

	return &Response{
		Status:  httpResp.StatusCode,
		Headers: httpResp.Header,
		Body:    respPayload,
	}, nil

}

func composeRequestUID(req Request) Request {
	if req.UID == "" {
		// TODO should we use a hash? So we can detact duplicate input?
		id, _ := uuid.NewUUID()
		req.UID = strings.Replace(id.String(), "-", "", -1)
	}
	return req
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
