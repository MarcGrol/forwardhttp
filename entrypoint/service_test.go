package entrypoint

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		forwarder               forwarder.Forwarder
		request                 *http.Request
		expectedResponseStatus  int
		expectedResponsePayload string
	}{
		{
			name:                    "Missing mandatory param",
			forwarder:               nil,
			request:                 httpRequest(t, "POST", "/doit?TryFirst=true", "request body"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "Error parsing request: Missing parameter: Missing parameter 'HostToForwardTo'",
		},
		{
			name:                    "Synchronous: success",
			forwarder:               syncForwarder(ctrl, 200, "response body", nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  200,
			expectedResponsePayload: "response body",
		},
		{
			name:                    "Synchronous: Networking error",
			forwarder:               syncForwarder(ctrl, 0, "", fmt.Errorf("Networking error")),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  500,
			expectedResponsePayload: "Networking error",
		},
		{
			name:                    "Synchronous: Permanent http error",
			forwarder:               syncForwarder(ctrl, 400, "error reponse body", nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "error reponse body",
		},
		{
			name:                    "Synchronous: Temporary http error",
			forwarder:               completeForwarder(ctrl, 500, "error reponse body", nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl&TryFirst=true", "request body"),
			expectedResponseStatus:  202,
			expectedResponsePayload: "",
		},
		{
			name:                    "Asynchronous: success",
			forwarder:               asyncForwarder(ctrl, nil),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl", "request body"),
			expectedResponseStatus:  202,
			expectedResponsePayload: "",
		},
		{
			name:                    "Asynchronous: error",
			forwarder:               asyncForwarder(ctrl, fmt.Errorf("queueing error")),
			request:                 httpRequest(t, "POST", "/doit?HostToForwardTo=home.nl", "request body"),
			expectedResponseStatus:  500,
			expectedResponsePayload: "Error enqueuing task: queueing error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			webservice := NewWebService(tc.forwarder)

			// when
			httpResp := httptest.NewRecorder()
			webservice.RegisterEndpoint(mux.NewRouter()).ServeHTTP(httpResp, tc.request)

			// then
			assert.Equal(t, tc.expectedResponseStatus, httpResp.Code)
			assert.Contains(t, tc.expectedResponsePayload, httpResp.Body.String())
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

func syncForwarder(ctrlr *gomock.Controller, status int, respPayload string, err error) forwarder.Forwarder {
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
