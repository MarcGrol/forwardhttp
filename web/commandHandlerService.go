package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/MarcGrol/forwardhttp/queue"
	"github.com/gorilla/mux"
)

type CommandHandlerService struct {
	Queue queue.TaskQueue
}

func NewCommandHandlerService(queue queue.TaskQueue) *CommandHandlerService {
	return &CommandHandlerService{
		Queue: queue,
	}
}

func (cs *CommandHandlerService) HTTPHandlerWithRouter(router *mux.Router) {
	router.PathPrefix("/").Handler(cs)
}

func (cs *CommandHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.Method == "PUT" {
		cs.enqueueToForward(w, r)
		return
	}

	cs.explain(w, r)
}

func (cs *CommandHandlerService) enqueueToForward(w http.ResponseWriter, r *http.Request) {
	c := r.Context()

	forwardContext, err := parseRequestIntoForwardContext(r)
	if err != nil {
		reportError(w, http.StatusBadRequest, fmt.Errorf("Error parsing request: %s", err))
		return
	}

	err = enqueue(c, cs.Queue, forwardContext)
	if err != nil {
		reportError(w, http.StatusInternalServerError, fmt.Errorf("Error enqueuing task: %s", err))
		return
	}

	log.Printf("Successfully enqueued task %s", forwardContext.String())

	w.WriteHeader(http.StatusAccepted)
}

func reportError(w http.ResponseWriter, httpResponseStatus int, err error) {
	log.Printf(err.Error())
	w.WriteHeader(httpResponseStatus)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, err.Error())
}

func parseRequestIntoForwardContext(r *http.Request) (*httpForwardContext, error) {
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

	return NewHttpForwardContext(r.Method, forwardURL, r.Header, body), nil
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

func enqueue(c context.Context, q queue.TaskQueue, forwardContext *httpForwardContext) error {
	log.Printf("forwardContext: %+v", forwardContext)

	taskPayload, err := json.Marshal(forwardContext)
	if err != nil {
		return fmt.Errorf("Error marshalling forwardContext: %s", err)
	}
	err = q.Enqueue(c, queue.Task{
		UID:            forwardContext.UID,
		WebhookURLPath: "/_ah/tasks/forward",
		Payload:        taskPayload,
	})
	if err != nil {
		return fmt.Errorf("Error submitting task to Queue: %s", err)
	}

	return nil
}

func (cs *CommandHandlerService) explain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, serviceDescription)
}

const serviceDescription = `<html>
<head>
	<title>Forwardhttp</title>
	<meta charset="utf-8"/>
	<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" 
		integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" 
		crossorigin="anonymous">
</head>
<body>
	<main role="main" class="container">
		<h1>Retrying HTTP forwarder</h1>
		<p>
			This HTTP-service will as a persistent and retrying Queue.<br/>
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
	--data "$(date): This is expected to be sent back as part of response body." \
	--X POST \
    "https://forwardhttp.appspot.com/post?HostToForwardTo=https://postman-echo.com"   
</pre>
		</p>

	</main>
</body>
</html>
`
