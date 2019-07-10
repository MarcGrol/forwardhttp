package main

import (
	"fmt"
	"net/http"
)

type httpForwardTask struct {
	Method  string
	URL     string
	Headers http.Header
	Body    []byte
}

func (t httpForwardTask) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.URL)
}
