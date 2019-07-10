package web

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
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
	fc.UID = fc.hash()
	return fc
}

func (t httpForwardContext) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.URL)
}

func (t httpForwardContext) hash() string {
	// Characterize payload based on all relevant fields
	h := sha1.New()
	io.WriteString(h, t.Method)
	io.WriteString(h, t.URL)
	h.Write(t.RequestBody)

	return fmt.Sprintf("%x", h.Sum(nil))
}
