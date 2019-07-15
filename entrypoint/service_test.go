package entrypoint

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MarcGrol/forwardhttp/uniqueid"

	"github.com/MarcGrol/forwardhttp/forwarder"
	"github.com/MarcGrol/forwardhttp/httpclient"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name                    string
		uidGenerator            uniqueid.Generator
		forwarder               forwarder.Forwarder
		request                 *http.Request
		expectedResponseStatus  int
		expectedResponsePayload string
	}{
		{
			name:                   "Get website with instructions",
			uidGenerator:           nil,
			forwarder:              nil,
			request:                httpRequest(t, "GET", "/", "request body"),
			expectedResponseStatus: 200,
		}, {
			name:                    "Missing mandatory param",
			uidGenerator:            nil,
			forwarder:               nil,
			request:                 httpRequest(t, "POST", "/doit?TryFirst=true", "request body"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "Error parsing request: Missing parameter: Missing mandatory parameter 'HostToForwardTo'",
		},
		{
			name:                    "Invalid optional param",
			uidGenerator:            nil,
			forwarder:               nil,
			request:                 httpRequest(t, "POST", "/doit?TryFirst=true", "request body"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "Error parsing request: Missing parameter: Missing mandatory parameter 'HostToForwardTo'",
		},
		{
			name:                    "Synchronous: success",
			uidGenerator:            generateUID(ctrl, "abc"),
			forwarder:               syncForwarder(ctrl, "abc", "POST", "/doit?a=b", 200, "response body", nil),
			request:                 httpRequest(t, "POST", "/doit?a=b&HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  200,
			expectedResponsePayload: "response body for request abc POST /doit?a=b",
		},
		{
			name:                    "Synchronous: Networking error",
			uidGenerator:            generateUID(ctrl, "abc"),
			forwarder:               syncForwarder(ctrl, "abc", "POST", "/doit", 0, "", fmt.Errorf("Networking error")),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  500,
			expectedResponsePayload: "Networking error",
		},
		{
			name:                    "Synchronous: Permanent http error",
			uidGenerator:            generateUID(ctrl, "abc"),
			forwarder:               syncForwarder(ctrl, "abc", "POST", "/doit", 400, "error reponse body", nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "error reponse body for request abc POST /doit",
		},
		{
			name:                    "Synchronous: Temporary http error",
			uidGenerator:            generateUID(ctrl, "abc"),
			forwarder:               completeForwarder(ctrl, 500, "error reponse body", nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  202,
			expectedResponsePayload: "",
		},
		{
			name:                    "Asynchronous: success",
			uidGenerator:            generateUID(ctrl, "abc"),
			forwarder:               asyncForwarder(ctrl, nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl", "request body"),
			expectedResponseStatus:  202,
			expectedResponsePayload: "",
		},
		{
			name:                    "Asynchronous: error",
			uidGenerator:            generateUID(ctrl, "abc"),
			forwarder:               asyncForwarder(ctrl, fmt.Errorf("queueing error")),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl", "request body"),
			expectedResponseStatus:  500,
			expectedResponsePayload: "Error enqueuing task: queueing error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			webservice := NewWebService(tc.uidGenerator, tc.forwarder)

			// when
			httpResp := httptest.NewRecorder()
			webservice.RegisterEndpoint(mux.NewRouter()).ServeHTTP(httpResp, tc.request)

			// then
			assert.Equal(t, tc.expectedResponseStatus, httpResp.Code)
			if tc.expectedResponsePayload != "" {
				assert.Equal(t, tc.expectedResponsePayload, httpResp.Body.String())
			}
		})
	}
}

func httpRequest(t *testing.T, method, url, body string) *http.Request {
	httpReq, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("Error creating http-request: %s", err)
	}
	httpReq.RequestURI = url
	httpReq.Header.Set("Content-type", "text/plain")
	httpReq.Header.Set("Accept", "text/plain")

	return httpReq
}

func generateUID(ctrlr *gomock.Controller, uid string) uniqueid.Generator {
	generatorMock := uniqueid.NewMockGenerator(ctrlr)
	generatorMock.
		EXPECT().
		Generate().
		Return(uid)

	return generatorMock
}

func syncForwarder(ctrlr *gomock.Controller, expectedUID, expectedMethod, expectedURL string, status int, respPayload string, err error) forwarder.Forwarder {
	forwarderMock := forwarder.NewMockForwarder(ctrlr)

	var resp *httpclient.Response = nil
	if err == nil {
		resp = &httpclient.Response{
			Status:  status,
			Headers: http.Header{},
			Body:    []byte(fmt.Sprintf("%s for request %s %s %s", respPayload, expectedUID, expectedMethod, expectedURL)),
		}
	}
	forwarderMock.
		EXPECT().
		Forward(gomock.Any(), gomock.Any()).
		Return(resp, err)

	return forwarderMock
}

func completeForwarder(ctrlr *gomock.Controller, status int, respPayload string, err error) forwarder.Forwarder {
	forwarderMock := forwarder.NewMockForwarder(ctrlr)

	var resp *httpclient.Response = nil
	if err == nil {
		resp = &httpclient.Response{
			Status:  status,
			Headers: http.Header{},
			Body:    []byte(respPayload),
		}
	}
	forwarderMock.
		EXPECT().
		Forward(gomock.Any(), gomock.Any()).
		Return(resp, err)

	forwarderMock.
		EXPECT().
		ForwardAsync(gomock.Any(), gomock.Any()).
		Return(err)

	return forwarderMock
}

func asyncForwarder(ctrlr *gomock.Controller, err error) forwarder.Forwarder {
	forwarderMock := forwarder.NewMockForwarder(ctrlr)

	forwarderMock.
		EXPECT().
		ForwardAsync(gomock.Any(), gomock.Any()).
		Return(err)

	return forwarderMock
}
