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
	fc.UID = fc.hash()
	return fc
}

func (t httpForwardContext) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.URL)
}

func (t httpForwardContext) hash() string {
	// TODO Does not seem to work
	//// Characterize payload based on all relevant fields
	//h := sha256.New()
	//io.WriteString(h, t.Method)
	//io.WriteString(h, t.URL)
	//h.Write(t.RequestBody)
	//
	//return fmt.Sprintf("%x", h.Sum(nil))

	id, _ := uuid.NewUUID()
	return strings.Replace(id.String(), "-", "", -1)
}
