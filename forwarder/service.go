package forwarder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/MarcGrol/forwardhttp/lastdelivery"

	"github.com/MarcGrol/forwardhttp/httpclient"
	"github.com/MarcGrol/forwardhttp/queue"
	"github.com/MarcGrol/forwardhttp/warehouse"
	"github.com/gorilla/mux"
)

const (
	taskEndpointBaseURL     = "/_ah/tasks"
	taskEndpointFORWARDPath = "/forward"

	taskEndpointURL = taskEndpointBaseURL + taskEndpointFORWARDPath
)

type forwarderService struct {
	queue        queue.TaskQueuer
	httpClient   httpclient.HTTPSender
	warehouse    warehouse.Warehouser
	lastDelivery lastdelivery.LastDeliverer
}

func NewService(queue queue.TaskQueuer, httpClient httpclient.HTTPSender, warehouse warehouse.Warehouser, lastDelivery lastdelivery.LastDeliverer) *forwarderService {
	s := &forwarderService{
		queue:        queue,
		httpClient:   httpClient,
		warehouse:    warehouse,
		lastDelivery: lastDelivery,
	}
	return s
}

func (s *forwarderService) RegisterEndPoint(router *mux.Router) *mux.Router {
	subRouter := router.PathPrefix(taskEndpointBaseURL).Subrouter()
	subRouter.HandleFunc(taskEndpointFORWARDPath, s.dequeue()).Methods("POST")
	return router
}

func (s *forwarderService) Forward(c context.Context, httpReq httpclient.Request) (*httpclient.Response, error) {
	httpResp, err := s.httpClient.Send(c, httpReq)
	defer s.warehouse.Put(c, httpReq, httpResp, err, warehouse.Stats{RetryCount: 0, MaxRetryCount: 0})
	if err == nil {
		return nil, err
	}
	if httpResp.IsPermanentError() {
		return nil, fmt.Errorf("Permanant http-error %d returned", httpResp.Status)
	}
	return httpResp, nil
}

func (s *forwarderService) ForwardAsync(c context.Context, req httpclient.Request) error {
	return s.enqueue(c, req)
}

func (s *forwarderService) enqueue(c context.Context, httpRequest httpclient.Request) error {

	taskPayload, err := json.Marshal(httpRequest)
	if err != nil {
		return fmt.Errorf("Error marshalling forwardContext: %s", err)
	}

	err = s.queue.Enqueue(c, queue.Task{
		UID:            httpRequest.UID,
		WebhookURLPath: taskEndpointURL,
		Payload:        taskPayload,
	})
	if err != nil {
		return fmt.Errorf("Error submitting forwardContext to queue: %s", err)
	}

	log.Printf("Successfully enqueued %s", httpRequest.String())

	return nil
}

func (s *forwarderService) dequeue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context()

		var httpReq httpclient.Request
		err := json.NewDecoder(r.Body).Decode(&httpReq)
		if err != nil {
			log.Printf("Error parsing json task payload:%s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// collect statistics
		numAttempts, maxAttempts, isLastAttempt := s.queue.IsLastAttempt(c, httpReq.UID)

		// do forward over http
		httpResp, err := s.httpClient.Send(c, httpReq)
		defer s.warehouse.Put(c, httpReq, httpResp, err, warehouse.Stats{RetryCount: numAttempts, MaxRetryCount: maxAttempts})
		if err != nil {
			log.Printf("Error forwarding %s: %s", httpReq.String(), err)
			if isLastAttempt {
				s.lastDelivery.OnLastDelivery(c, httpReq, nil, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if httpResp.IsError() {
			log.Printf("Error forwarding %s: resp-status: %d", httpReq.String(), httpResp.Status)
			if isLastAttempt {
				s.lastDelivery.OnLastDelivery(c, httpReq, httpResp, nil)
			}
			w.WriteHeader(httpResp.Status)
			return
		}

		log.Printf("Successfully forwarded %s:%s", httpReq.String(), httpResp.String())
		s.lastDelivery.OnLastDelivery(c, httpReq, httpResp, nil)
	}
}