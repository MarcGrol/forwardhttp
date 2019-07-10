package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2beta3"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2beta3"
	"github.com/gorilla/mux"
)

type ReceiverService struct{}

func (ps *ReceiverService) HTTPHandlerWithRouter(router *mux.Router) *mux.Router {
	router.PathPrefix("/").Handler(ps)

	return router
}

func (ps *ReceiverService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := r.Context()

	task, err := parseRequestIntoTask(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Error parsing request: %s", err)
		return
	}

	err = enqueue(c, task)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Error enqueuimg request: %s", err)
		return
	}

	log.Printf("Successfully enqueued task %s", task.String())
}

func parseRequestIntoTask(r *http.Request) (*httpForwardTask, error) {
	hostToForwardTo, err := extractString(r, "HostToForwardTo", true)
	if err != nil {
		return nil, fmt.Errorf("Missing url parameter: %s", err)
	}

	forwardURL, err := composeTargetURL(r.RequestURI, hostToForwardTo)
	if err != nil {
		return nil, fmt.Errorf("Error composing target url: %s", err)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading request body: %s", err)
	}

	return &httpForwardTask{
		Method:  r.Method,
		URL:     forwardURL,
		Headers: r.Header,
		Body:    body,
	}, nil
}

func composeTargetURL(requestURI, hostToForwardTo string) (string, error) {
	// strip scheme
	hostToForwardTo = strings.Replace(hostToForwardTo, "http://", "", 1)
	hostToForwardTo = strings.Replace(hostToForwardTo, "https://", "", 1)

	url, err := url.Parse(requestURI)
	if err != nil {
		return "", fmt.Errorf("Error parsing url path %s: %s", requestURI, err)
	}
	queryParams := url.Query()
	queryParams.Del("HostToForwardTo")
	url.RawQuery = queryParams.Encode()
	url.Host = hostToForwardTo
	url.Scheme = "https"
	return url.String(), nil
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func extractString(r *http.Request, fieldName string, mandatory bool) (string, error) {
	value := r.URL.Query().Get(fieldName)
	if value == "" {
		value = r.FormValue(fieldName)
	}
	if value == "" {
		pathParams := mux.Vars(r)
		value = pathParams[fieldName]
	}
	if value == "" {
		if mandatory {
			return "", fmt.Errorf("Missing parameter '%s'", fieldName)
		}
		return "", nil
	}

	return value, nil
}

func enqueue(c context.Context, task *httpForwardTask) error {
	jsonTask, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("Error creating task: %s", err)
	}

	cloudTaskClient, err := cloudtasks.NewClient(c)
	if err != nil {
		return fmt.Errorf("Error creating cloudtask-sevice: %s", err)
	}

	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	locationId := os.Getenv("LOCATION_ID")
	serviceId := os.Getenv("GAE_SERVICE")

	parentId := fmt.Sprintf("projects/%s/locations/%s/queues/default", projectId, locationId)
	pushURL := fmt.Sprintf("https://%s-dot-%s.appspot.com/_ah/tasks/forward", serviceId, projectId)

	_, err = cloudTaskClient.CreateTask(c, &taskspb.CreateTaskRequest{
		Parent: parentId,
		Task: &taskspb.Task{
			PayloadType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        pushURL,
					Body:       jsonTask,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("Error creating submitting task: %s", err)
	}
	return nil
}
