package entrypoint

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MarcGrol/forwardhttp/uniqueid"

	"github.com/MarcGrol/forwardhttp/forwarder"
	"github.com/MarcGrol/forwardhttp/httpclient"
	"github.com/gorilla/mux"
)

func NewWebService(uidGenerator uniqueid.Generator, forwarder forwarder.Forwarder) *webService {
	s := &webService{
		uidGenerator: uidGenerator,
		forwarder:    forwarder,
	}
	return s
}

func (s *webService) RegisterEndpoint(router *mux.Router) *mux.Router {
	router.PathPrefix("/").Handler(s)
	return router
}

func (s *webService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.Method == "PUT" {
		s.forward(w, r)
		return
	}

	s.explain(w, r)
}

func (s *webService) forward(w http.ResponseWriter, r *http.Request) {
	c := r.Context()

	tryFirst, httpRequest, err := s.parseRequest(r)
	if err != nil {
		reportError(w, http.StatusBadRequest, fmt.Errorf("Error parsing request: %s", err))
		return
	}

	if tryFirst {
		httpResponse, err := s.forwarder.Forward(c, httpRequest)
		if err != nil {
			writeResponse(w, &httpclient.Response{Status: 500, Body: []byte(err.Error())})
			return
		}
		if httpResponse.IsPermanentError() {
			writeResponse(w, httpResponse)
			return
		}
		// continue async
	}

	err = s.forwarder.ForwardAsync(c, httpRequest)
	if err != nil {
		reportError(w, http.StatusInternalServerError, fmt.Errorf("Error enqueuing task: %s", err))
		return
	}

	// Indicate we have successfully received but not yet processed
	w.WriteHeader(http.StatusAccepted)
}

func writeResponse(w http.ResponseWriter, resp *httpclient.Response) {
	for k, v := range resp.Headers {
		for _, hv := range v {
			w.Header().Add(k, hv)
		}
	}
	w.WriteHeader(resp.Status)
	w.Write(resp.Body)
}

func reportError(w http.ResponseWriter, httpResponseStatus int, err error) {
	log.Printf(err.Error())
	w.WriteHeader(httpResponseStatus)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, err.Error())
}

func (s *webService) parseRequest(r *http.Request) (bool, httpclient.Request, error) {
	hostToForwardTo, err := extractMandatoryStringParameter(r, "HostToForwardTo")
	if err != nil {
		return false, httpclient.Request{}, fmt.Errorf("Missing parameter: %s", err)
	}

	taskUID := extractStringParameter(r, "TaskUid")
	if taskUID == "" {
		taskUID = s.uidGenerator.Generate()
	}
	tryFirst := extractBool(r, "TryFirst")

	req := httpclient.Request{
		Method:  r.Method,
		Headers: r.Header,
		TaskUID: taskUID,
	}

	req.URL, err = composeTargetURL(r.RequestURI, hostToForwardTo)
	if err != nil {
		return tryFirst, req, fmt.Errorf("Error composing target url: %s", err)
	}

	req.Body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return tryFirst, req, fmt.Errorf("Error reading request body: %s", err)
	}

	return tryFirst, req, nil
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
	queryParams.Del("TaskUid")         // not interesting to remote host
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

func extractMandatoryStringParameter(r *http.Request, fieldName string) (string, error) {
	value := extractStringParameter(r, fieldName)
	if value == "" {
		return "", fmt.Errorf("Missing mandatory parameter '%s'", fieldName)
	}

	return value, nil
}

func extractStringParameter(r *http.Request, fieldName string) string {
	value := r.URL.Query().Get(fieldName)
	if value == "" {
		value = r.FormValue(fieldName)
	}

	if value == "" {
		value = r.Header.Get(fmt.Sprintf("X-%s", fieldName))
	}

	return value
}

func extractBool(r *http.Request, fieldName string) bool {
	valueAsString := r.URL.Query().Get(fieldName)
	if valueAsString == "" {
		valueAsString = r.FormValue(fieldName)
	}
	if valueAsString == "" {
		valueAsString = r.Header.Get(fmt.Sprintf("X-%s", fieldName))
	}
	if valueAsString == "" {
		return false
	}

	value, err := strconv.ParseBool(valueAsString)
	if err != nil {
		return false
	}

	return value
}

func (s *webService) explain(w http.ResponseWriter, r *http.Request) {
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
			This web-service will act as a persistent and retrying queue.<br/>
			Upon receipt of a POST or PUT-request, the service will asynchronously forward the received HTTP request to a remote host.<br/>
			When the remote host does not return a success, the request will be retried untill success or 
            untill the retry-scheme is exhausted.<br/>
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
        --request POST \
        --data "This is expected to be sent back as part of response body." \
        "https://forwardhttp.appspot.com/post?HostToForwardTo=postman-echo.com&TryFirst=false"    
</pre>
		</p>

	</main>
</body>
</html>
`
