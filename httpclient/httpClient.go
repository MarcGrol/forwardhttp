package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const httpClientTimeout = 20 * time.Second

type client struct {
}

func NewClient() HTTPSender {
	return &client{}
}

func (_ client) Send(c context.Context, req Request) (*Response, error) {
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("Error creating http request for %s: %s", req.String(), err)
	}
	copyHeaders(httpReq.Header, req.Headers)

	log.Printf("HTTP request: %s %s", req.Method, req.URL)
	httpClient := &http.Client{
		Timeout: httpClientTimeout,
	}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Error sending %s: %s", req.String(), err)
	}
	defer httpResp.Body.Close()

	respPayload, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response %s: %s", req.String(), err)
	}
	log.Printf("HTTP resp: %d", httpResp.StatusCode)

	return &Response{
		Status:  httpResp.StatusCode,
		Headers: httpResp.Header,
		Body:    respPayload,
	}, nil

}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
