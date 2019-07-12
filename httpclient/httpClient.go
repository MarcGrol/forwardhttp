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

type Request struct {
	UID     string
	Method  string
	URL     string
	Headers http.Header `datastore:",noindex"`
	Body    []byte      `datastore:",noindex"`
}

func (r Request) String() string {
	return fmt.Sprintf("HTTP request %s %s", r.Method, r.URL)
}

type Response struct {
	Status  int
	Headers http.Header `datastore:",noindex"`
	Body    []byte      `datastore:",noindex"`
}

func (r Response) String() string {
	return fmt.Sprintf("HTTP response %d", r.Status)
}

func (r Response) IsError() bool {
	return r.Status >= http.StatusBadRequest
}

func (r Response) IsPermanentError() bool {
	return r.Status >= http.StatusOK && r.Status < http.StatusInternalServerError
}

func (r Response) isTemporaryError() bool {
	return !r.IsPermanentError()
}

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
