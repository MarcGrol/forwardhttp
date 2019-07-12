package forwarder

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarcGrol/forwardhttp/lastdelivery"

	"github.com/MarcGrol/forwardhttp/queue"

	"github.com/MarcGrol/forwardhttp/httpclient"
	"github.com/MarcGrol/forwardhttp/warehouse"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestWebHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name                    string
		httpClient              httpclient.HTTPSender
		warehouse               warehouse.Warehouser
		queue                   queue.TaskQueuer
		request                 *http.Request
		lastDeliverer           lastdelivery.LastDeliverer
		expectedResponseStatus  int
		expectedResponsePayload string
	}{
		{
			name:                    "Task processing success",
			httpClient:              httpClient(ctrl, 200, "success response", nil),
			warehouse:               warehouseClient(ctrl, nil),
			queue:                   queueClient(ctrl, true),
			lastDeliverer:           lastDeliveryHandler(ctrl, httpResponse(200, "success response"), nil),
			request:                 httpRequest(t, "POST", "/_ah/tasks/forward", "request payload"),
			expectedResponseStatus:  200,
			expectedResponsePayload: "success response",
		},
		{
			name:          "Http bad request error: last attempt",
			httpClient:    httpClient(ctrl, 400, "error response", nil),
			warehouse:     warehouseClient(ctrl, nil),
			queue:         queueClient(ctrl, true),
			lastDeliverer: lastDeliveryHandler(ctrl, httpResponse(400, "error response"), nil),
			//lastDeliverer:           lastdelivery.NewLastDelivery(),
			request:                 httpRequest(t, "POST", "/_ah/tasks/forward", "request payload"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "error response",
		},
		{
			name:                    "Http bad request error",
			httpClient:              httpClient(ctrl, 400, "error response", nil),
			warehouse:               warehouseClient(ctrl, nil),
			queue:                   queueClient(ctrl, false),
			lastDeliverer:           nil,
			request:                 httpRequest(t, "POST", "/_ah/tasks/forward", "request payload"),
			expectedResponseStatus:  400,
			expectedResponsePayload: "error response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			service := NewService(tc.queue, tc.httpClient, tc.warehouse, tc.lastDeliverer)

			// when
			httpResp := httptest.NewRecorder()
			service.RegisterEndPoint(mux.NewRouter()).ServeHTTP(httpResp, tc.request)

			// then
			assert.Equal(t, tc.expectedResponseStatus, httpResp.Code)
			assert.Contains(t, tc.expectedResponsePayload, httpResp.Body.String())
		})
	}
}

func httpRequest(t *testing.T, method, url, body string) *http.Request {
	jsonPayload, err := json.Marshal(httpclient.Request{
		Method: method,
		URL:    "/myurl",
		Body:   []byte(body),
	})
	assert.NoError(t, err)
	httpReq, err := http.NewRequest(method, url, bytes.NewReader(jsonPayload))
	if err != nil {
		t.Fatalf("Error creating http-request: %s", err)
	}
	httpReq.RequestURI = url
	httpReq.Header.Set("Content-type", "application/json")

	return httpReq
}

func httpResponse(status int, payload string) *httpclient.Response {
	return &httpclient.Response{
		Status:  status,
		Headers: http.Header{},
		Body:    []byte(payload),
	}
}

func httpClient(ctrlr *gomock.Controller, status int, respPayload string, err error) httpclient.HTTPSender {
	httpSender := httpclient.NewMockHTTPSender(ctrlr)

	var resp *httpclient.Response = nil
	if err == nil {
		resp = &httpclient.Response{
			Status:  status,
			Headers: http.Header{},
			Body:    []byte(respPayload),
		}
	}

	httpSender.
		EXPECT().
		Send(gomock.Any(), gomock.Any()).
		Return(resp, err)

	return httpSender
}

func warehouseClient(ctrlr *gomock.Controller, err error) warehouse.Warehouser {
	warehouse := warehouse.NewMockWarehouser(ctrlr)

	warehouse.
		EXPECT().
		Put(gomock.Any(), gomock.Any()).
		Return(err)

	return warehouse
}

func queueClient(ctrlr *gomock.Controller, isLast bool) queue.TaskQueuer {
	queue := queue.NewMockTaskQueuer(ctrlr)

	if isLast {
		queue.
			EXPECT().
			IsLastAttempt(gomock.Any(), gomock.Any()).
			Return(int32(1), int32(1))
	} else {
		queue.
			EXPECT().
			IsLastAttempt(gomock.Any(), gomock.Any()).
			Return(int32(1), int32(10))
	}

	return queue
}

func lastDeliveryHandler(ctrlr *gomock.Controller, resp *httpclient.Response, err error) lastdelivery.LastDeliverer {
	lastdelivery := lastdelivery.NewMockLastDeliverer(ctrlr)

	lastdelivery.
		EXPECT().
		OnLastDelivery(gomock.Any(), gomock.Any(), gomock.Eq(resp), err).
		Return()

	return lastdelivery
}
