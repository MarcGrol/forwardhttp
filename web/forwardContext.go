package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type httpForwardContext struct {
	UID         string
	Method      string
	URL         string
	Headers     http.Header
	RequestBody []byte
}

func NewHttpForwardContext(method, url string, headers http.Header, body []byte) *httpForwardContext {
	fc := &httpForwardContext{
		UID:         "",
		Method:      method,
		URL:         url,
		Headers:     headers,
		RequestBody: body,
	}
	fc.UID = fc.composeUID()
	return fc
}

func (t httpForwardContext) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.URL)
}

func (t httpForwardContext) composeUID() string {
	// TODO should we use a hash? So we can detact duplicate input?
	id, _ := uuid.NewUUID()
	return strings.Replace(id.String(), "-", "", -1)
}
