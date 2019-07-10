package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type ForwarderService struct{}

func (ps *ForwarderService) HTTPHandlerWithRouter(router *mux.Router) *mux.Router {
	router.PathPrefix("/_ah/tasks/forward").Handler(ps)

	return router
}

func (ps *ForwarderService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//c := r.Context()
	//
	//traceUID := r.Header.Get("X-Cloud-Trace-Context")
	//// Creates a client.
	//client, err := logging.NewClient(c, os.Getenv("GOOGLE_CLOUD_PROJECT"))
	//if err != nil {
	//	log.Printf("Error creating logging client: %v", err)
	//	w.WriteHeader(http.StatusInternalServerError)
	//	return
	//}
	//defer client.Close()
	//
	//logger := client.Logger(os.Getenv("GAE_SERVICE"))
	//defer logger.Flush()
	//logger.Log(logging.Entry{
	//	Severity: logging.Info,
	//	Payload:  "Example info",
	//	Trace:    traceUID,
	//})

	var req httpForwardTask
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error parsing task request payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// decode request body
	respStatus, err := sendOverHttp(req.Method, req.URL, req.Headers, req.Body)
	if err != nil {
		log.Printf("Error forwarding %s: %s", req.String(), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !isSuccess(respStatus) {
		log.Printf("Error forwarding %s: http-resp-status=%d", req.String(), respStatus)
		w.WriteHeader(respStatus)
		return
	}

	log.Printf("Successfully forwarded http %s to %s", req.Method, req.URL)
}

func sendOverHttp(method, url string, headers http.Header, requestBody []byte) (int, error) {
	httpReq, err := http.NewRequest(method, url, bytes.NewReader(requestBody))
	if err != nil {
		return 0, fmt.Errorf("Error creating http request for '%s %s': %s", method, url, err)
	}
	copyHeaders(httpReq.Header, headers)

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("Error sending %s %s: %s", method, url, err)
	}
	defer httpResp.Body.Close()

	respPayload, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return 0, fmt.Errorf("Error reading response: %s", err)
	}

	log.Printf("%s %s returned %s with payload '%s'", method, url, httpResp.Status, string(respPayload))

	return httpResp.StatusCode, nil
}

func isSuccess(httpRespStatus int) bool {
	return httpRespStatus >= http.StatusOK && httpRespStatus < http.StatusMultipleChoices
}
