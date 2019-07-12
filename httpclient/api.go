package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

//go:generate mockgen -source=api.go -destination=gen_HttpClientMock.go -package=httpclient github.com/MarcGrol/forwardhttp/httpclient HTTPSender

type Request struct {
	UID     string
	Method  string
	URL     string
	Headers http.Header `datastore:"-"`
	Body    []byte      `datastore:",noindex"`
}

func (r *Request) SetUID() {
	// TODO should we use a hash? So we can detact duplicate input?
	id, _ := uuid.NewUUID()
	r.UID = strings.Replace(id.String(), "-", "", -1)
}

func (r Request) String() string {
	return fmt.Sprintf("HTTP %s request %s: %s", r.Method, r.URL, r.UID)
}

type Response struct {
	Status  int
	Headers http.Header `datastore:"-"`
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

type HTTPSender interface {
	Send(c context.Context, req Request) (*Response, error)
}
