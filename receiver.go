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

func (rs *ReceiverService) HTTPHandlerWithRouter(router *mux.Router) *mux.Router {
	router.PathPrefix("/").Handler(rs)

	return router
}

func (ps *ReceiverService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.Method == "PUT" {
		ps.enqueueToForward(w, r)
		return
	}

	ps.explain(w, r)
}

func (rs *ReceiverService) enqueueToForward(w http.ResponseWriter, r *http.Request) {
	c := r.Context()

	task, err := parseRequestIntoTask(r)
	if err != nil {
		reportError(w, http.StatusBadRequest, fmt.Errorf("Error parsing request: %s", err))
		return
	}

	err = enqueue(c, task)
	if err != nil {
		reportError(w, http.StatusInternalServerError, fmt.Errorf("Error enqueuing request: %s", err))
		return
	}

	log.Printf("Successfully enqueued task %s", task.String())
}

func reportError(w http.ResponseWriter, httpResponseStatus int, err error) {
	log.Printf(err.Error())
	w.WriteHeader(httpResponseStatus)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, err.Error())
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
	queryParams.Del("HostToForwardTo") // not interesting to remote host
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
		value = r.Header.Get(fmt.Sprintf("X-%s", fieldName))
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
		return fmt.Errorf("Error marshalling task: %s", err)
	}

	cloudTaskClient, err := cloudtasks.NewClient(c)
	if err != nil {
		return fmt.Errorf("Error creating cloudtask-service: %s", err)
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
		return fmt.Errorf("Error creating submitting task to queue: %s", err)
	}

	return nil
}

func (ps *ReceiverService) explain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, serviceDescription)
}

const serviceDescription = `<html>
<head>
	<title>Retryer</title>
	<meta charset="utf-8"/>
	<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" 
		integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" 
		crossorigin="anonymous">
</head>
<body>
	<main role="main" class="container">
		<h1>Retrying HTTP forwarder</h1>
		<p>
			This HTTP-service will as a persistent and retrying queue.<br/>
			Upon receipt of a HTTP POST-request, the service will asynchronously forward the received HTTP request to a remote host.<br/>
			When the remote host does not return a success, the request will be retried untill success or 
            untill the retry scheme is exhausted.<br/>
			The remote host is indicated by:
			<ul>
				<li>the HTTP query parameeter "HostToForwardTo" or </li>
				<li>the HTTP-request-header "X-HostToForwardTo"</li>
			</ul>
		</p>
		
		<p>
		Example request that demonstrates a POST being forwarded to postman-echo.com<br/><br/>

<pre>
curl -vvv \
	--data "This is expected to be sent back as part of response body." \
	--X POST \
	"https://retryer-dot-retryer.appspot.com/post?a=b&c=d&HostToForwardTo=https://postman-echo.com"
</pre>
		</p>

	</main>
</body>
</html>
`
