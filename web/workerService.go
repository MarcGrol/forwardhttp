package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/MarcGrol/forwardhttp/queue"
	"github.com/MarcGrol/forwardhttp/store"
	"github.com/gorilla/mux"
)

type WorkerService struct {
	Queue queue.TaskQueue
	Store store.DataStore
}

func NewWorkerService(queue queue.TaskQueue, store store.DataStore) *WorkerService {
	return &WorkerService{
		Queue: queue,
		Store: store,
	}
}
func (ws *WorkerService) HTTPHandlerWithRouter(router *mux.Router) {
	subRouter := router.PathPrefix("/_ah/tasks").Subrouter()
	subRouter.HandleFunc("/forward", ws.onForwardTaskReceived()).Methods("POST")
}

func (ws *WorkerService) onForwardTaskReceived() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context()

		var req httpForwardContext
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Printf("Error parsing json task payload:%s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		isLastAttempt := ws.Queue.IsLastAttempt(c, req.UID)
		httpRespCode, _, respPayload, err := sendOverHttp(req.Method, req.URL, req.Headers, req.RequestBody)
		if err != nil {
			log.Printf("Error forwarding %s: resp-status: %d: %s", req.String(), httpRespCode, err)
			if isLastAttempt {
				log.Printf("***** This was the last attempt ********")
				// should we call a last resort?
			} else {
				log.Printf("Will try later again")
			}

			// keep track
			storeResult(c, ws.Store, req, httpRespCode, respPayload, false, isLastAttempt)

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// keep track
		storeResult(c, ws.Store, req, httpRespCode, respPayload, true, isLastAttempt)
	}
}

func sendOverHttp(method, url string, headers http.Header, requestBody []byte) (int, http.Header, []byte, error) {
	httpReq, err := http.NewRequest(method, url, bytes.NewReader(requestBody))
	if err != nil {
		return 1, http.Header{}, []byte{}, fmt.Errorf("Error creating http request for '%s %s': %s", method, url, err)
	}
	copyHeaders(httpReq.Header, headers)

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return 2, http.Header{}, []byte{}, fmt.Errorf("Error sending %s %s: %s", method, url, err)
	}
	defer httpResp.Body.Close()

	respPayload, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return 3, httpResp.Header, []byte{}, fmt.Errorf("Error reading response: %s", err)
	}

	log.Printf("%s %s returned %s with payload '%s'", method, url, httpResp.Status, string(respPayload))

	if !isSuccess(httpResp.StatusCode) {
		return httpResp.StatusCode, httpResp.Header, []byte{}, fmt.Errorf("Error forwarding %s %s: http-resp-status=%d", method, url, httpResp.StatusCode)
	}

	return httpResp.StatusCode, httpResp.Header, respPayload, nil
}

func isSuccess(httpRespStatus int) bool {
	return httpRespStatus >= http.StatusOK && httpRespStatus < http.StatusMultipleChoices
}

func storeResult(c context.Context, store store.DataStore, req httpForwardContext, respStatusCode int,
	respPayload []byte, success, lastAttempt bool) error {
	err := store.Put(c, "TaskStatus", req.UID, &TaskStatus{
		UID:            req.UID,
		Timestamp:      time.Now(),
		Method:         req.Method,
		RequestURL:     req.URL,
		RequestBody:    string(req.RequestBody),
		ResponseStatus: respStatusCode,
		ResponseBody:   string(respPayload),
		Success:        success,
		Done:           lastAttempt,
	})
	if err != nil {
		log.Printf("Error storing task-status: %s", err)
		return fmt.Errorf("Error storing task-status: %s", err)
	}

	log.Printf("Successfully stored task-status")
	return nil
}
