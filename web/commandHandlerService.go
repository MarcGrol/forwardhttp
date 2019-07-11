package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MarcGrol/forwardhttp/store"

	"github.com/MarcGrol/forwardhttp/queue"
	"github.com/gorilla/mux"
)

type CommandHandlerService struct {
	Queue queue.TaskQueue
	Store store.DataStore
}

func NewCommandHandlerService(queue queue.TaskQueue, store store.DataStore) *CommandHandlerService {
	return &CommandHandlerService{
		Queue: queue,
		Store: store,
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

	tryFirst, forwardContext, err := parseRequestIntoForwardContext(r)
	if err != nil {
		reportError(w, http.StatusBadRequest, fmt.Errorf("Error parsing request: %s", err))
		return
	}

	if tryFirst {
		// Try synchronous forward first
		respStatus, respHeaders, respPayload, err := sendOverHttp(forwardContext.Method, forwardContext.URL, forwardContext.Headers, forwardContext.RequestBody)
		if err == nil || isPermanentError(respStatus) {
			// keep track
			storeResult(c, cs.Store, *forwardContext, respStatus, respPayload, err == nil, 1, 1, true)

			// return a response right away
			writeResponse(w, respStatus, respHeaders, respPayload)
			return
		}
	}

	err = enqueue(c, cs.Queue, forwardContext)
	if err != nil {
		reportError(w, http.StatusInternalServerError, fmt.Errorf("Error enqueuing task: %s", err))
		return
	}

	// Indicate we have successfully received but not yet processed
	w.WriteHeader(http.StatusAccepted)
}

func writeResponse(w http.ResponseWriter, httpResponseStatus int, headers http.Header, responsePayload []byte) {
	for k, v := range headers {
		for _, hv := range v {
			w.Header().Add(k, hv)
		}
	}
	w.WriteHeader(httpResponseStatus)
	w.Write(responsePayload)
}

func reportError(w http.ResponseWriter, httpResponseStatus int, err error) {
	log.Printf(err.Error())
	w.WriteHeader(httpResponseStatus)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, err.Error())
}

func parseRequestIntoForwardContext(r *http.Request) (bool, *httpForwardContext, error) {
	hostToForwardTo, err := extractString(r, "HostToForwardTo", true)
	if err != nil {
		return false, nil, fmt.Errorf("Missing parameter: %s", err)
	}
	tryFirst, _ := extractBool(r, "TryFirst")

	forwardURL, err := composeTargetURL(r.RequestURI, hostToForwardTo)
	if err != nil {
		return tryFirst, nil, fmt.Errorf("Error composing target url: %s", err)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return tryFirst, nil, fmt.Errorf("Error reading request body: %s", err)
	}

	return tryFirst, NewHttpForwardContext(r.Method, forwardURL, r.Header, body), nil
}

func composeTargetURL(requestURI, hostToForwardTo string) (string, error) {
	// strip scheme

	url, err := url.Parse(requestURI)
	if err != nil {
		return "", fmt.Errorf("Error parsing url path %s: %s", requestURI, err)
	}
	queryParams := url.Query()
	queryParams.Del("HostToForwardTo") // not interesting to remote host
	queryParams.Del("TryFirst")        // not interesting to remote host
	url.RawQuery = queryParams.Encode()
	scheme, host := determineSchemeHostname(hostToForwardTo)
	url.Host = host
	url.Scheme = scheme
	return url.String(), nil
}

func determineSchemeHostname(hostToForwardTo string) (string, string) {
	scheme := ""
	if strings.HasPrefix(hostToForwardTo, "http://") {
		scheme = "http"
		hostToForwardTo = strings.Replace(hostToForwardTo, "http://", "", 1)
	} else {
		scheme = "https"
		hostToForwardTo = strings.Replace(hostToForwardTo, "https://", "", 1)
	}

	return scheme, hostToForwardTo
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

func extractBool(r *http.Request, fieldName string) (bool, error) {
	valueAsString := r.URL.Query().Get(fieldName)
	if valueAsString == "" {
		valueAsString = r.FormValue(fieldName)
	}
	if valueAsString == "" {
		pathParams := mux.Vars(r)
		valueAsString = pathParams[fieldName]
	}
	if valueAsString == "" {
		valueAsString = r.Header.Get(fmt.Sprintf("X-%s", fieldName))
	}

	if valueAsString == "" {
		return false, nil
	}

	value, err := strconv.ParseBool(valueAsString)
	if err != nil {
		return false, fmt.Errorf("Invalid bool parameter '%s': %s", fieldName, valueAsString)
	}

	return value, nil
}

func enqueue(c context.Context, q queue.TaskQueue, forwardContext *httpForwardContext) error {

	taskPayload, err := json.Marshal(forwardContext)
	if err != nil {
		return fmt.Errorf("Error marshalling forwardContext: %s", err)
	}

	log.Printf("Start enqueuing %s with uid %s: %s", forwardContext.String(), forwardContext.UID, forwardContext.RequestBody)
	err = q.Enqueue(c, queue.Task{
		UID:            forwardContext.UID,
		WebhookURLPath: "/_ah/tasks/forward",
		Payload:        taskPayload,
	})
	if err != nil {
		return fmt.Errorf("Error submitting forwardContext to Queue: %s", err)
	}

	log.Printf("Successfully enqueued %s", forwardContext.String())

	return nil
}

func isPermanentError(httpRespStatus int) bool {
	return !isTemporaryError(httpRespStatus)
}

func isTemporaryError(httpRespStatus int) bool {
	return httpRespStatus < http.StatusContinue || httpRespStatus >= http.StatusInternalServerError
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
			Upon receipt of a HTTP POST or PUT-request, the service will asynchronously forward the received HTTP request to a remote host.<br/>
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
    "https://forwardhttp.appspot.com/post?HostToForwardTo=https://postman-echo.com&TryFirst=true"   
</pre>
		</p>

	</main>
</body>
</html>
`
