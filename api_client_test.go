package genai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/civil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSendRequest(t *testing.T) {
	ctx := context.Background()
	// Setup test cases
	tests := []struct {
		desc           string
		path           string
		method         string
		requestBody    map[string]any
		clientTimeout  *time.Duration
		requestTimeout *time.Duration
		serverLatency  time.Duration
		responseCode   int
		responseBody   string
		want           map[string]any
		wantErr        error
	}{
		{
			desc:         "successful post request",
			path:         "foo",
			method:       http.MethodPost,
			requestBody:  map[string]any{"key": "value"},
			responseCode: http.StatusOK,
			responseBody: `{"response": "ok"}`,
			want:         map[string]any{"response": "ok"},
			wantErr:      nil,
		},
		{
			desc:         "successful get request",
			path:         "foo",
			method:       http.MethodGet,
			requestBody:  map[string]any{},
			responseCode: http.StatusOK,
			responseBody: `{"response": "ok"}`,
			want:         map[string]any{"response": "ok"},
			wantErr:      nil,
		},
		{
			desc:         "successful patch request",
			path:         "foo",
			method:       http.MethodPatch,
			requestBody:  map[string]any{"key": "value"},
			responseCode: http.StatusOK,
			responseBody: `{"response": "ok"}`,
			want:         map[string]any{"response": "ok"},
			wantErr:      nil,
		},
		{
			desc:         "successful delete request",
			path:         "foo",
			method:       http.MethodDelete,
			requestBody:  map[string]any{"key": "value"},
			responseCode: http.StatusOK,
			responseBody: `{"response": "ok"}`,
			want:         map[string]any{"response": "ok"},
			wantErr:      nil,
		},
		{
			desc:         "400 error response",
			path:         "bar",
			method:       http.MethodGet,
			responseCode: http.StatusBadRequest,
			responseBody: `{"error": {"code": 400, "message": "bad request", "status": "INVALID_ARGUMENTS", "details": [{"field": "value"}]}}`,
			wantErr:      &APIError{Code: http.StatusBadRequest, Message: "bad request", Details: []map[string]any{{"field": "value"}}},
		},
		{
			desc:         "500 error response",
			path:         "bar",
			method:       http.MethodGet,
			responseCode: http.StatusInternalServerError,
			responseBody: `{"error": {"code": 500, "message": "internal server error", "status": "INTERNAL_SERVER_ERROR", "details": [{"field": "value"}]}}`,
			wantErr:      &APIError{Code: http.StatusInternalServerError, Message: "internal server error", Details: []map[string]any{{"field": "value"}}},
		},
		{
			desc:         "invalid response body",
			path:         "baz",
			method:       http.MethodPut,
			responseCode: http.StatusOK,
			responseBody: `invalid json`,
			wantErr:      fmt.Errorf("deserializeUnaryResponse: error unmarshalling response: invalid character"),
		},
		{
			desc:          "client timeout",
			path:          "foo",
			method:        http.MethodPost,
			clientTimeout: Ptr(600 * time.Millisecond),
			requestBody:   map[string]any{"key": "value"},
			responseCode:  http.StatusOK,
			responseBody:  `{"response": "ok"}`,
			serverLatency: 700 * time.Millisecond,
			wantErr:       fmt.Errorf("context deadline exceeded"),
		},
		{
			desc:           "request timeout",
			path:           "foo",
			method:         http.MethodPost,
			requestTimeout: Ptr(500 * time.Millisecond),
			requestBody:    map[string]any{"key": "value"},
			responseCode:   http.StatusOK,
			responseBody:   `{"response": "ok"}`,
			serverLatency:  700 * time.Millisecond,
			wantErr:        fmt.Errorf("context deadline exceeded"),
		},
		{
			desc:           "client timeout with request timeout",
			path:           "foo",
			method:         http.MethodPost,
			clientTimeout:  Ptr(600 * time.Millisecond),
			requestTimeout: Ptr(500 * time.Millisecond),
			requestBody:    map[string]any{"key": "value"},
			responseCode:   http.StatusOK,
			responseBody:   `{"response": "ok"}`,
			serverLatency:  550 * time.Millisecond,
			wantErr:        fmt.Errorf("context deadline exceeded"),
		},
		{
			desc:           "With 0 client timeout and request timeout",
			method:         "POST",
			path:           "test",
			requestBody:    map[string]any{"key": "value"},
			responseBody:   `{"response": "ok"}`,
			responseCode:   http.StatusOK,
			clientTimeout:  Ptr(0 * time.Millisecond),
			requestTimeout: Ptr(100 * time.Millisecond),
			serverLatency:  150 * time.Millisecond,
			wantErr:        fmt.Errorf("context deadline exceeded"),
		},
		{
			desc:           "With client timeout and 0 request timeout",
			method:         "POST",
			path:           "test",
			requestBody:    map[string]any{"key": "value"},
			responseBody:   `{"response": "ok"}`,
			responseCode:   http.StatusOK,
			clientTimeout:  Ptr(200 * time.Millisecond),
			requestTimeout: Ptr(0 * time.Millisecond),
			serverLatency:  250 * time.Millisecond,
		},
		{
			desc:           "With 0 client timeout and 0 request timeout",
			method:         "POST",
			path:           "test",
			requestBody:    map[string]any{"key": "value"},
			responseBody:   `{"response": "ok"}`,
			responseCode:   http.StatusOK,
			clientTimeout:  Ptr(0 * time.Millisecond),
			requestTimeout: Ptr(0 * time.Millisecond),
			serverLatency:  150 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Create a test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverLatency > 0 {
					time.Sleep(tt.serverLatency)
				}
				w.WriteHeader(tt.responseCode)
				fmt.Fprintln(w, tt.responseBody)
			}))
			defer ts.Close()

			// Create a test client with the test server's URL
			ac := &apiClient{
				clientConfig: &ClientConfig{
					HTTPOptions: HTTPOptions{
						BaseURL: ts.URL,
						Timeout: tt.clientTimeout,
					},
					HTTPClient: ts.Client(),
				},
			}

			got, err := sendRequest(ctx, ac, tt.path, tt.method, tt.requestBody, &HTTPOptions{BaseURL: ts.URL, Timeout: tt.requestTimeout})

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("sendRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil && err != nil {
				// For error cases, check for want error types
				if tt.responseCode >= 400 {
					_, ok := err.(APIError)
					if !ok {
						t.Errorf("want Error, got %T(%s)", err, err.Error())
					}
				} else { // build request error
					if !strings.Contains(err.Error(), tt.wantErr.Error()) {
						t.Errorf("unexpected error, want: %v, got: %v", tt.wantErr, err)
					}
				}
			}

			if tt.wantErr != nil && !cmp.Equal(got, tt.want) {
				t.Errorf("sendRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendStreamRequest(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		body             map[string]any
		httpOptions      *HTTPOptions
		mockResponse     string
		mockStatusCode   int
		converterErr     error
		maxIteration     *int
		clientTimeout    *time.Duration
		requestTimeout   *time.Duration
		serverLatency    time.Duration
		wantResponse     []map[string]any
		wantErr          bool
		wantErrorMessage string
	}{
		{
			name:           "Successful Stream",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			wantErr: false,
		},
		{
			name:           "Successful Stream with Empty Lines",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\n\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			wantErr: false,
		},
		{
			name:           "Successful Stream with Windows Newlines",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\r\n\r\ndata:{\"key2\":\"value2\"}\r\n\r\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			wantErr: false,
		},
		{
			name:           "Empty Stream",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "",
			mockStatusCode: http.StatusOK,
			wantResponse:   nil,
			wantErr:        false,
		},
		{
			name:           "Stream with Empty Data",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{},
			},
			wantErr: false,
		},
		{
			name:           "Stream with Invalid JSON",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:invalid\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
			},
			wantErr:          true,
			wantErrorMessage: "error unmarshalling data data:invalid. error: invalid character 'i' looking for beginning of value",
		},
		{
			name:             "Stream with Invalid Seperator",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     "data:{\"key1\":\"value1\"}\t\tdata:{\"key2\":\"value2\"}",
			mockStatusCode:   http.StatusOK,
			wantResponse:     nil,
			wantErr:          true,
			wantErrorMessage: "iterateResponseStream: error unmarshalling data data:{\"key1\":\"value1\"}\t\tdata:{\"key2\":\"value2\"}. error: invalid character 'd' after top-level value",
		},
		{
			name:             "Stream with Coverter Error",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}",
			mockStatusCode:   http.StatusOK,
			converterErr:     fmt.Errorf("converter error"),
			wantResponse:     nil,
			wantErr:          true,
			wantErrorMessage: "converter error",
		},
		{
			name:           "Stream with Max Iteration",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}",
			mockStatusCode: http.StatusOK,
			maxIteration:   Ptr(1),
			wantResponse: []map[string]any{
				{"key1": "value1"},
			},
		},
		{
			name:           "Stream with Non-Data Prefix",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\nerror:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
			},
			wantErr:          true,
			wantErrorMessage: "iterateResponseStream: invalid stream chunk: error:{\"key2\":\"value2\"}",
		},
		{
			name:             "Error Response",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     `{"error":{"code":400,"message":"test error message","status":"INVALID_ARGUMENT"}}`,
			mockStatusCode:   http.StatusBadRequest,
			wantErr:          true,
			wantErrorMessage: "Error 400, Message: test error message, Status: INVALID_ARGUMENT, Details: []",
		},
		{
			name:             "Error Response with empty body",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     ``,
			mockStatusCode:   http.StatusBadRequest,
			wantErr:          true,
			wantErrorMessage: "Error 400, Message: , Status: 400 Bad Request, Details: []",
		},
		{
			name:             "Error Response with invalid json",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     `invalid json`,
			mockStatusCode:   http.StatusBadRequest,
			wantErr:          true,
			wantErrorMessage: "Error 400, Message: invalid json, Status: 400 Bad Request, Details: []",
		},
		{
			name:             "Error Response with server error",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     `{"error":{"code":500,"message":"test error message","status":"INTERNAL"}}`,
			mockStatusCode:   http.StatusInternalServerError,
			wantErr:          true,
			wantErrorMessage: "Error 500, Message: test error message, Status: INTERNAL, Details: []",
		},
		{
			name:             "Error Response with server error and empty body",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     ``,
			mockStatusCode:   http.StatusInternalServerError,
			wantErr:          true,
			wantErrorMessage: "Error 500, Message: , Status: 500 Internal Server Error, Details: []",
		},
		{
			name:             "Error Response with server error and invalid json",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     `invalid json`,
			mockStatusCode:   http.StatusInternalServerError,
			wantErr:          true,
			wantErrorMessage: "Error 500, Message: invalid json, Status: 500 Internal Server Error, Details: []",
		},
		{
			name:             "Error Response with status ok but error stream chunk",
			method:           "POST",
			path:             "test",
			body:             map[string]any{"key": "value"},
			mockResponse:     `{"error": {"code": 500, "message": "internal server error", "status": "INTERNAL_SERVER_ERROR"}}`,
			mockStatusCode:   http.StatusOK,
			wantErr:          true,
			wantErrorMessage: "Error 500, Message: internal server error, Status: INTERNAL_SERVER_ERROR, Details: []",
		},
		{
			name:           "Request Error",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "",
			mockStatusCode: http.StatusOK,
			httpOptions: &HTTPOptions{
				BaseURL: "invalid-url",
			},
			wantErr:          true,
			wantErrorMessage: "doRequest: error sending request: Post \"invalid-url/test\": unsupported protocol scheme",
		},
		{
			name:           "With client timeout",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			clientTimeout:    Ptr(100 * time.Millisecond),
			serverLatency:    150 * time.Millisecond,
			wantErr:          true,
			wantErrorMessage: "context deadline exceeded",
		},
		{
			name:           "With request timeout",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			requestTimeout:   Ptr(100 * time.Millisecond),
			serverLatency:    150 * time.Millisecond,
			wantErr:          true,
			wantErrorMessage: "context deadline exceeded",
		},
		{
			name:           "With client timeout and request timeout",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			clientTimeout:    Ptr(200 * time.Millisecond),
			requestTimeout:   Ptr(100 * time.Millisecond),
			serverLatency:    150 * time.Millisecond,
			wantErr:          true,
			wantErrorMessage: "context deadline exceeded",
		},
		{
			name:           "With 0 client timeout and request timeout",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			clientTimeout:    Ptr(0 * time.Millisecond),
			requestTimeout:   Ptr(100 * time.Millisecond),
			serverLatency:    150 * time.Millisecond,
			wantErr:          true,
			wantErrorMessage: "context deadline exceeded",
		},
		{
			name:           "With client timeout and 0 request timeout",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			clientTimeout:  Ptr(200 * time.Millisecond),
			requestTimeout: Ptr(0 * time.Millisecond),
			serverLatency:  250 * time.Millisecond,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			wantErr: false,
		},
		{
			name:           "With 0 client timeout and 0 request timeout",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\n\ndata:{\"key2\":\"value2\"}\n\n",
			mockStatusCode: http.StatusOK,
			clientTimeout:  Ptr(0 * time.Millisecond),
			requestTimeout: Ptr(0 * time.Millisecond),
			serverLatency:  150 * time.Millisecond,
			wantResponse: []map[string]any{
				{"key1": "value1"},
				{"key2": "value2"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverLatency > 0 {
					time.Sleep(tt.serverLatency)
				}
				if r.Method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, tt.path) {
					t.Errorf("Expected path to end with %s, got %s", tt.path, r.URL.Path)
				}

				if tt.body != nil {
					var gotBody map[string]any
					err := json.NewDecoder(r.Body).Decode(&gotBody)
					if err != nil {
						t.Fatalf("Failed to decode request body: %v", err)
					}
					if diff := cmp.Diff(tt.body, gotBody); diff != "" {
						t.Errorf("Request body mismatch (-want +got):\n%s", diff)
					}
				}

				if !slices.Contains(r.Header[http.CanonicalHeaderKey("User-Agent")], "test-user-agent") {
					t.Errorf("Expected User-Agent header to contain 'test-user-agent', got %v", r.Header)
				}
				if !slices.Contains(r.Header["X-Goog-Api-Key"], "test-api-key") {
					t.Errorf("Expected X-Goog-Api-Key header to contain 'test-api-key', got %v", r.Header)
				}

				w.WriteHeader(tt.mockStatusCode)
				_, _ = fmt.Fprint(w, tt.mockResponse)
			}))
			defer ts.Close()

			clientConfig := &ClientConfig{
				Backend: BackendGeminiAPI,
				HTTPOptions: HTTPOptions{
					BaseURL:    ts.URL,
					APIVersion: "v0",
					Headers: http.Header{
						"User-Agent":     []string{"test-user-agent"},
						"X-Goog-Api-Key": []string{"test-api-key"},
					},
					Timeout: tt.clientTimeout,
				},
				HTTPClient: ts.Client(),
			}

			if tt.httpOptions != nil {
				clientConfig.HTTPOptions = *tt.httpOptions
			}

			ac := &apiClient{clientConfig: clientConfig}
			var output responseStream[map[string]any]
			err := sendStreamRequest(context.Background(), ac, tt.path, tt.method, tt.body, &HTTPOptions{Timeout: tt.requestTimeout, BaseURL: clientConfig.HTTPOptions.BaseURL}, &output)

			if err != nil && tt.wantErr {
				if tt.wantErrorMessage != "" && !strings.Contains(err.Error(), tt.wantErrorMessage) {
					t.Errorf("sendStreamRequest() error message = %v, wantErrorMessage %v", err.Error(), tt.wantErrorMessage)
				}
				return
			}

			var gotResponse []map[string]any
			iterCount := 0
			for resp, iterErr := range iterateResponseStream(&output, func(responseMap map[string]any) (*map[string]any, error) {
				return &responseMap, tt.converterErr
			}) {
				err = iterErr
				if iterErr != nil {
					break
				}
				iterCount++
				if tt.maxIteration != nil && iterCount > *tt.maxIteration {
					break
				}
				gotResponse = append(gotResponse, *resp)
			}
			if err != nil != tt.wantErr {
				t.Errorf("iterateResponseStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr {
				if tt.wantErrorMessage != "" && !strings.Contains(err.Error(), tt.wantErrorMessage) {
					t.Errorf("sendStreamRequest() error message = %v, wantErrorMessage %v", err.Error(), tt.wantErrorMessage)
				}
				return
			}

			if diff := cmp.Diff(tt.wantResponse, gotResponse); diff != "" {
				t.Errorf("sendStreamRequest() response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMapToStruct(t *testing.T) {
	testCases := []struct {
		name      string
		inputMap  map[string]any
		wantValue any
	}{
		{
			name: "TokensInfo",
			inputMap: map[string]any{
				"role":     "test-role",
				"TokenIDs": []string{"123", "456"},
				"Tokens":   [][]byte{[]byte("token1"), []byte("token2")}},
			wantValue: TokensInfo{
				Role:     "test-role",
				TokenIDs: []int64{123, 456},
				Tokens:   [][]byte{[]byte("token1"), []byte("token2")}},
		},
		{
			name: "Citation",
			inputMap: map[string]any{
				"startIndex":      float64(0),
				"endIndex":        float64(20),
				"title":           "Citation Title",
				"uri":             "https://example.com",
				"publicationDate": map[string]int{"year": 2000, "month": 1, "day": 1},
			},
			wantValue: Citation{
				StartIndex:      0,
				EndIndex:        20,
				Title:           "Citation Title",
				URI:             "https://example.com",
				PublicationDate: civil.Date{Year: 2000, Month: 1, Day: 1},
			},
		},
		{
			name: "Citation year only",
			inputMap: map[string]any{
				"startIndex":      float64(0),
				"endIndex":        float64(20),
				"title":           "Citation Title",
				"uri":             "https://example.com",
				"publicationDate": map[string]int{"year": 2000},
			},
			wantValue: Citation{
				StartIndex:      0,
				EndIndex:        20,
				Title:           "Citation Title",
				URI:             "https://example.com",
				PublicationDate: civil.Date{Year: 2000},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputValue := reflect.New(reflect.TypeOf(tc.wantValue)).Interface()

			err := mapToStruct(tc.inputMap, &outputValue)

			if err != nil {
				t.Fatalf("mapToStruct failed: %v", err)
			}

			want := reflect.ValueOf(tc.wantValue).Interface()
			got := reflect.ValueOf(outputValue).Elem().Interface()

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("mapToStruct mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	timeout := 10 * time.Second
	tests := []struct {
		name            string
		clientConfig    *ClientConfig
		path            string
		body            map[string]any
		method          string
		httpOptions     *HTTPOptions
		contextTimeout  time.Duration
		want            *http.Request
		wantErr         bool
		expectedError   string
		expectedTimeout *time.Duration
	}{
		{
			name: "MLDev API with API Key",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"key": "value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
				Headers: http.Header{
					"X-Test-Header": []string{"test-value"},
				},
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "generativelanguage.googleapis.com",
					Path:   "/v1beta/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Goog-Api-Key":    []string{"test-api-key"},
					"X-Test-Header":     []string{"test-value"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader("{\"key\":\"value\"}\n")),
			},
			wantErr: false,
		},
		{
			name: "Vertex AI API",
			clientConfig: &ClientConfig{
				Project:     "test-project",
				Location:    "test-location",
				Backend:     BackendVertexAI,
				HTTPClient:  &http.Client{},
				Credentials: &auth.Credentials{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"key": "value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://test-location-aiplatform.googleapis.com",
				APIVersion: "v1beta1",
				Headers: http.Header{
					"X-Test-Header": []string{"test-value"},
				},
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "test-location-aiplatform.googleapis.com",
					Path:   "/v1beta1/projects/test-project/locations/test-location/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Test-Header":     []string{"test-value"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader("{\"key\":\"value\"}\n")),
			},
			wantErr: false,
		},
		{
			name: "Vertex AI API with full path",
			clientConfig: &ClientConfig{
				Project:     "test-project",
				Location:    "test-location",
				Backend:     BackendVertexAI,
				HTTPClient:  &http.Client{},
				Credentials: &auth.Credentials{},
			},
			path:   "projects/test-project/locations/test-location/models/test-model:generateContent",
			body:   map[string]any{"key": "value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://test-location-aiplatform.googleapis.com",
				APIVersion: "v1beta1",
				Headers: http.Header{
					"X-Test-Header": []string{"test-value"},
				},
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "test-location-aiplatform.googleapis.com",
					Path:   "/v1beta1/projects/test-project/locations/test-location/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Test-Header":     []string{"test-value"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader("{\"key\":\"value\"}\n")),
			},
			wantErr: false,
		},
		{
			name: "Vertex AI query base model",
			clientConfig: &ClientConfig{
				Project:     "test-project",
				Location:    "test-location",
				Backend:     BackendVertexAI,
				HTTPClient:  &http.Client{},
				Credentials: &auth.Credentials{},
			},
			path:   "publishers/google/models/model-name",
			body:   map[string]any{},
			method: "GET",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://test-location-aiplatform.googleapis.com",
				APIVersion: "v1beta1",
			},
			want: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "test-location-aiplatform.googleapis.com",
					Path:   "/v1beta1/publishers/google/models/model-name",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader(``)),
			},
			wantErr: false,
		},
		{
			name: "Vertex AI with API key",
			clientConfig: &ClientConfig{
				APIKey:      "test-api-key",
				Backend:     BackendVertexAI,
				HTTPClient:  &http.Client{},
				Credentials: &auth.Credentials{},
			},
			path:   "publishers/google/models/model-name",
			body:   map[string]any{},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://aiplatform.googleapis.com",
				APIVersion: "v1beta1",
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "aiplatform.googleapis.com",
					Path:   "/v1beta1/publishers/google/models/model-name",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Key":    []string{"test-api-key"},
				},
				Body: io.NopCloser(strings.NewReader(``)),
			},
			wantErr: false,
		},
		{
			name: "MLDev with empty body",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "generativelanguage.googleapis.com",
					Path:   "/v1beta/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Goog-Api-Key":    []string{"test-api-key"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader(``)),
			},
			wantErr: false,
		},
		{
			name: "Vertex AI with empty body",
			clientConfig: &ClientConfig{
				Project:     "test-project",
				Location:    "test-location",
				Backend:     BackendVertexAI,
				HTTPClient:  &http.Client{},
				Credentials: &auth.Credentials{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://test-location-aiplatform.googleapis.com",
				APIVersion: "v1beta1",
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "test-location-aiplatform.googleapis.com",
					Path:   "/v1beta1/projects/test-project/locations/test-location/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader(``)),
			},
			wantErr: false,
		},
		{
			name: "Invalid URL",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				HTTPClient: &http.Client{},
				Backend:    BackendGeminiAPI,
			},
			path:   ":invalid",
			body:   map[string]any{},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    ":invalid",
				APIVersion: "v1beta",
			},
			wantErr:       true,
			expectedError: "createAPIURL: error parsing base URL: parse \":invalid\": missing protocol scheme",
		},
		{
			name: "Invalid json",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"key": make(chan int)},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
			},
			wantErr:       true,
			expectedError: "buildRequest: error encoding body",
		},
		{
			name: "With ExtrasRequestProvider",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"original_key": "original_value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
				ExtrasRequestProvider: func(body map[string]any) map[string]any {
					body["extra_key"] = "extra_value"
					return body
				},
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "generativelanguage.googleapis.com",
					Path:   "/v1beta/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Goog-Api-Key":    []string{"test-api-key"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader("{\"extra_key\":\"extra_value\",\"original_key\":\"original_value\"}\n")),
			},
			wantErr: false,
		},
		{
			name: "With timeout",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"key": "value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
				Headers: http.Header{
					"X-Test-Header": []string{"test-value"},
				},
				Timeout: &timeout,
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "generativelanguage.googleapis.com",
					Path:   "/v1beta/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Goog-Api-Key":    []string{"test-api-key"},
					"X-Test-Header":     []string{"test-value"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Server-Timeout":  []string{"10"},
				},
				Body: io.NopCloser(strings.NewReader("{\"key\":\"value\"}\n")),
			},
			wantErr: false,
		},
		{
			name: "Header merging",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
				HTTPOptions: HTTPOptions{
					Headers: http.Header{
						"X-Client-Header": []string{"client-value"},
						"X-Common-Header": []string{"client-common"},
					},
				},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"key": "value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
				Headers: http.Header{
					"X-Request-Header": []string{"request-value"},
					"X-Common-Header":  []string{"request-common"},
				},
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "generativelanguage.googleapis.com",
					Path:   "/v1beta/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Goog-Api-Key":    []string{"test-api-key"},
					"X-Client-Header":   []string{"client-value"},
					"X-Common-Header":   []string{"request-common"},
					"X-Request-Header":  []string{"request-value"},
					"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader("{\"key\":\"value\"}\n")),
			},
			wantErr: false,
		},
		{
			name: "Custom User-Agent",
			clientConfig: &ClientConfig{
				APIKey:     "test-api-key",
				Backend:    BackendGeminiAPI,
				HTTPClient: &http.Client{},
			},
			path:   "models/test-model:generateContent",
			body:   map[string]any{"key": "value"},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIVersion: "v1beta",
				Headers: http.Header{
					"User-Agent": []string{"my-custom-agent/1.0"},
				},
			},
			want: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "https",
					Host:   "generativelanguage.googleapis.com",
					Path:   "/v1beta/models/test-model:generateContent",
				},
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Goog-Api-Key":    []string{"test-api-key"},
					"User-Agent":        []string{"my-custom-agent/1.0", fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
					"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				},
				Body: io.NopCloser(strings.NewReader("{\"key\":\"value\"}\n")),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &apiClient{clientConfig: tt.clientConfig}
			ctx := context.Background()
			var cancel context.CancelFunc
			if tt.contextTimeout != 0 {
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
			}

			req, _, err := buildRequest(ctx, ac, tt.path, tt.body, tt.method, tt.httpOptions)

			if tt.wantErr {
				if err == nil {
					t.Errorf("buildRequest() expected an error but got nil")
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("buildRequest() expected error to contain: %v , but got %v", tt.expectedError, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("buildRequest() returned an unexpected error: %v", err)
			}

			if tt.want.Method != req.Method {
				t.Errorf("buildRequest() got Method = %v, want %v", req.Method, tt.want.Method)
			}

			if diff := cmp.Diff(tt.want.URL, req.URL); diff != "" {
				t.Errorf("buildRequest() URL mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.want.Header, req.Header); diff != "" {
				t.Errorf("buildRequest() Header mismatch (-want +got):\n%s", diff)
			}

			gotBodyBytes, _ := io.ReadAll(req.Body)
			wantBodyBytes, _ := io.ReadAll(tt.want.Body)

			if diff := cmp.Diff(string(wantBodyBytes), string(gotBodyBytes)); diff != "" {
				t.Errorf("buildRequest() body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPatchHTTPOptions(t *testing.T) {
	timeout1 := 10 * time.Second
	timeout2 := 20 * time.Second
	dummyExtrasProvider1 := func(body map[string]any) map[string]any { body["p1"] = 1; return body }
	dummyExtrasProvider2 := func(body map[string]any) map[string]any { body["p2"] = 2; return body }

	tests := []struct {
		name         string
		options      HTTPOptions
		patchOptions HTTPOptions
		want         *HTTPOptions
	}{
		{
			name: "patch empty",
			options: HTTPOptions{
				BaseURL:    "base",
				APIVersion: "v1",
				Headers:    http.Header{"H1": []string{"v1"}},
				Timeout:    &timeout1,
			},
			patchOptions: HTTPOptions{},
			want: &HTTPOptions{
				BaseURL:    "base",
				APIVersion: "v1",
				Headers:    http.Header{"H1": []string{"v1"}, "User-Agent": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}, "X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}},
				Timeout:    &timeout1,
			},
		},
		{
			name: "patch all fields",
			options: HTTPOptions{
				BaseURL:               "base",
				APIVersion:            "v1",
				Headers:               http.Header{"H1": []string{"v1"}, "H2": []string{"v2"}},
				Timeout:               &timeout1,
				ExtrasRequestProvider: dummyExtrasProvider1,
			},
			patchOptions: HTTPOptions{
				BaseURL:               "patched",
				APIVersion:            "v2",
				Headers:               http.Header{"H2": []string{"v2-patched"}, "H3": []string{"v3"}},
				Timeout:               &timeout2,
				ExtrasRequestProvider: dummyExtrasProvider2,
			},
			want: &HTTPOptions{
				BaseURL:               "patched",
				APIVersion:            "v2",
				Headers:               http.Header{"H1": []string{"v1"}, "H2": []string{"v2-patched"}, "H3": []string{"v3"}, "User-Agent": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}, "X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}},
				Timeout:               &timeout2,
				ExtrasRequestProvider: dummyExtrasProvider2,
			},
		},
		{
			name:    "empty options",
			options: HTTPOptions{},
			patchOptions: HTTPOptions{
				BaseURL:    "patched",
				APIVersion: "v2",
				Headers:    http.Header{"H1": []string{"v1"}},
				Timeout:    &timeout2,
			},
			want: &HTTPOptions{
				BaseURL:    "patched",
				APIVersion: "v2",
				Headers:    http.Header{"H1": []string{"v1"}, "User-Agent": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}, "X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}},
				Timeout:    &timeout2,
			},
		},
		{
			name: "patch with nil timeout",
			options: HTTPOptions{
				Timeout: &timeout1,
			},
			patchOptions: HTTPOptions{
				Timeout: nil,
			},
			want: &HTTPOptions{
				Timeout: &timeout1,
				Headers: http.Header{"User-Agent": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}, "X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}},
			},
		},
		{
			name: "patch with zero timeout",
			options: HTTPOptions{
				Timeout: &timeout1,
			},
			patchOptions: HTTPOptions{
				Timeout: Ptr(0 * time.Second),
			},
			want: &HTTPOptions{
				Timeout: Ptr(0 * time.Second),
				Headers: http.Header{"User-Agent": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}, "X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchHTTPOptions(tt.options, tt.patchOptions)
			if err != nil {
				t.Errorf("patchHTTPOptions() returned an unexpected error: %v", err)
			}

			// Can't compare functions directly.
			opt := cmpopts.IgnoreFields(HTTPOptions{}, "ExtrasRequestProvider")
			if diff := cmp.Diff(tt.want, got, opt); diff != "" {
				t.Errorf("patchHTTPOptions() mismatch (-want +got):\n%s", diff)
			}

			if tt.want.ExtrasRequestProvider != nil {
				if got.ExtrasRequestProvider == nil {
					t.Error("ExtrasRequestProvider: got nil, want non-nil")
				}
				// check if they are the same function
				gotResult := got.ExtrasRequestProvider(make(map[string]any))
				wantResult := tt.want.ExtrasRequestProvider(make(map[string]any))
				if diff := cmp.Diff(wantResult, gotResult); diff != "" {
					t.Errorf("ExtrasRequestProvider function produced different results (-want +got):\n%s", diff)
				}
			} else if got.ExtrasRequestProvider != nil {
				t.Error("ExtrasRequestProvider: got non-nil, want nil")
			}
		})
	}
}

// createTestFile creates a temporary file with the specified size containing dummy text data.
// It returns the file path and a cleanup function to remove the file.
func createTestFile(t *testing.T, size int64) (string, func()) {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "upload-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	buf := make([]byte, 1024*1024) // 1MB buffer
	pattern := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()")
	for i := 0; i < len(buf); i++ {
		buf[i] = pattern[i%len(pattern)]
	}

	var written int64
	for written < size {
		bytesToWrite := int64(len(buf))
		if written+bytesToWrite > size {
			bytesToWrite = size - written
		}
		n, err := tmpfile.Write(buf[:bytesToWrite])
		if err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		written += int64(n)
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to close temp file: %v", err)
	}

	cleanup := func() {
		os.Remove(tmpfile.Name())
	}
	return tmpfile.Name(), cleanup
}

// mockUploadServer simulates the resumable upload endpoint.
func mockUploadServer(t *testing.T, expectedSize int64, headers []http.Header) (*httptest.Server, *sync.Map) {
	t.Helper()
	var totalReceived int64
	var mu sync.Mutex
	currentIndex := 0
	// Use sync.Map to store received data per upload URL (though in this test we only use one)
	receivedData := &sync.Map{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
			return
		}
		uploadCommand := r.Header.Get("X-Goog-Upload-Command")
		uploadOffsetStr := r.Header.Get("X-Goog-Upload-Offset")
		contentLengthStr := r.Header.Get("Content-Length")
		uploadOffset, err := strconv.ParseInt(uploadOffsetStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid X-Goog-Upload-Offset", http.StatusBadRequest)
			return
		}

		contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid Content-Length", http.StatusBadRequest)
			return
		}

		mu.Lock()
		if uploadOffset != totalReceived {
			mu.Unlock()
			t.Errorf("Offset mismatch: expected %d, got %d", totalReceived, uploadOffset)
			http.Error(w, "Offset mismatch", http.StatusBadRequest)
			return
		}
		mu.Unlock()

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		if int64(len(bodyBytes)) != contentLength {
			t.Errorf("Content-Length mismatch: header %d, body %d", contentLength, len(bodyBytes))
			http.Error(w, "Content-Length mismatch", http.StatusBadRequest)
			return
		}

		// Store received data chunk (optional, but useful for verification)
		mu.Lock()
		for key, value := range headers[currentIndex] {
			w.Header().Set(key, value[0])
		}
		currentTotal := totalReceived
		isEmptyUploadStatus := headers[currentIndex].Get("X-Goog-Upload-Status") == ""
		if !isEmptyUploadStatus {
			totalReceived += contentLength
			currentTotal = totalReceived
		}
		currentIndex++
		mu.Unlock()
		isFinal := strings.Contains(uploadCommand, "finalize")

		if isFinal && !isEmptyUploadStatus {
			if currentTotal != expectedSize {
				t.Errorf("Final size mismatch: expected %d, received %d", expectedSize, currentTotal)
				http.Error(w, "Final size mismatch", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			finalFile := map[string]any{
				"file": map[string]any{
					"name":      fmt.Sprintf("files/upload-%d", time.Now().UnixNano()),
					"sizeBytes": strconv.FormatInt(currentTotal, 10),
					"mimeType":  "text/plain", // Assuming text for simplicity
				},
			}
			if err := json.NewEncoder(w).Encode(finalFile); err != nil {
				t.Errorf("Failed to encode final file: %v", err)
				http.Error(w, "Failed to encode final file", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))

	return server, receivedData
}

func TestUploadFile(t *testing.T) {
	ctx := context.Background()

	testSizes := []struct {
		name    string
		size    int64 // Size in bytes
		headers []http.Header
	}{
		{"1MB", 1 * 1024 * 1024, []http.Header{
			{
				"Content-Type":         []string{"application/json"},
				"X-Goog-Upload-Status": []string{"final"},
			},
		}},
		{"8MB", 8 * 1024 * 1024, []http.Header{
			{
				"X-Goog-Upload-Status": []string{"active"},
			},
			{
				"Content-Type":         []string{"application/json"},
				"X-Goog-Upload-Status": []string{"final"},
			},
		}}, // Exactly maxChunkSize
		{"9MB", 9 * 1024 * 1024, []http.Header{
			{
				"X-Goog-Upload-Status": []string{"active"},
			},
			{
				"Content-Type":         []string{"application/json"},
				"X-Goog-Upload-Status": []string{"final"},
			},
		}}, // Requires multiple chunks
		{"9MB-missing-header", 9 * 1024 * 1024, []http.Header{
			{
				"X-Goog-Upload-Status": []string{"active"},
			},
			{
				"X-Goog-Upload-Status": []string{""},
			},
			{
				"Content-Type":         []string{"application/json"},
				"X-Goog-Upload-Status": []string{"final"},
			},
		}}, // Requires multiple chunks
	}

	for _, ts := range testSizes {
		t.Run(ts.name, func(t *testing.T) {
			filePath, cleanup := createTestFile(t, ts.size)
			defer cleanup()

			server, _ := mockUploadServer(t, ts.size, ts.headers)
			defer server.Close()

			ac := &apiClient{
				clientConfig: &ClientConfig{
					HTTPClient: server.Client(),
					APIKey:     "test-key-upload",
				},
			}

			httpOpts := &HTTPOptions{
				Headers: http.Header{},
			}

			fileReader, err := os.Open(filePath)
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", filePath, err)
			}
			defer fileReader.Close()

			uploadURL := server.URL + "/upload"

			uploadedFile, err := ac.uploadFile(ctx, fileReader, uploadURL, httpOpts)

			if err != nil {
				t.Fatalf("uploadFile failed: %v", err)
			}

			if uploadedFile == nil {
				t.Fatal("uploadFile returned nil File, expected a valid File object")
			}

			if uploadedFile.Name == "" {
				t.Error("uploadedFile.Name is empty")
			}
			// Convert SizeBytes to int64 if it's a pointer
			var gotSizeBytes int64
			if uploadedFile.SizeBytes != nil {
				gotSizeBytes = *uploadedFile.SizeBytes
			} else {
				t.Error("uploadedFile.SizeBytes is nil")
			}

			if gotSizeBytes != ts.size {
				t.Errorf("uploadedFile.SizeBytes mismatch: want %d, got %d", ts.size, gotSizeBytes)
			}
			if uploadedFile.MIMEType != "text/plain" { // Matches mock server response
				t.Errorf("uploadedFile.MIMEType mismatch: want 'text/plain', got '%s'", uploadedFile.MIMEType)
			}

		})
	}
}

func TestInferTimeout(t *testing.T) {
	tests := []struct {
		name              string
		requestTimeout    *time.Duration
		httpClientTimeout time.Duration
		contextTimeout    time.Duration // 0 means no deadline
		want              time.Duration
		tolerance         time.Duration // for context timeout
	}{
		{
			name:              "no timeouts",
			requestTimeout:    nil,
			httpClientTimeout: 0,
			contextTimeout:    0,
			want:              0,
		},
		{
			name:              "only request timeout",
			requestTimeout:    Ptr(10 * time.Second),
			httpClientTimeout: 0,
			contextTimeout:    0,
			want:              10 * time.Second,
		},
		{
			name:              "only http client timeout",
			requestTimeout:    nil,
			httpClientTimeout: 20 * time.Second,
			contextTimeout:    0,
			want:              20 * time.Second,
		},
		{
			name:              "only context timeout",
			requestTimeout:    nil,
			httpClientTimeout: 0,
			contextTimeout:    30 * time.Second,
			want:              30 * time.Second,
			tolerance:         100 * time.Millisecond,
		},
		{
			name:              "request timeout is smallest",
			requestTimeout:    Ptr(10 * time.Second),
			httpClientTimeout: 20 * time.Second,
			contextTimeout:    30 * time.Second,
			want:              10 * time.Second,
		},
		{
			name:              "http client timeout is smallest",
			requestTimeout:    Ptr(20 * time.Second),
			httpClientTimeout: 10 * time.Second,
			contextTimeout:    30 * time.Second,
			want:              10 * time.Second,
		},
		{
			name:              "context timeout is smallest",
			requestTimeout:    Ptr(30 * time.Second),
			httpClientTimeout: 20 * time.Second,
			contextTimeout:    10 * time.Second,
			want:              10 * time.Second,
			tolerance:         100 * time.Millisecond,
		},
		{
			name:              "request timeout is zero",
			requestTimeout:    Ptr(0 * time.Second),
			httpClientTimeout: 20 * time.Second,
			contextTimeout:    30 * time.Second,
			want:              20 * time.Second,
		},
		{
			name:              "request timeout is zero, no other timeouts",
			requestTimeout:    Ptr(0 * time.Second),
			httpClientTimeout: 0,
			contextTimeout:    0,
			want:              0,
		},
		{
			name:              "request timeout is zero, only context timeout",
			requestTimeout:    Ptr(0 * time.Second),
			httpClientTimeout: 0,
			contextTimeout:    30 * time.Second,
			want:              30 * time.Second,
			tolerance:         100 * time.Millisecond,
		},
		{
			name:              "request timeout is zero, only http client timeout",
			requestTimeout:    Ptr(0 * time.Second),
			httpClientTimeout: 20 * time.Second,
			contextTimeout:    0,
			want:              20 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &apiClient{
				clientConfig: &ClientConfig{
					HTTPClient: &http.Client{
						Timeout: tt.httpClientTimeout,
					},
				},
			}

			ctx := context.Background()
			if tt.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
			}

			got := inferTimeout(ctx, ac, tt.requestTimeout)

			if tt.tolerance > 0 {
				if got > tt.want || got < tt.want-tt.tolerance {
					t.Errorf("inferTimeout() got = %v, want around %v (with tolerance %v)", got, tt.want, tt.tolerance)
				}
			} else {
				if got != tt.want {
					t.Errorf("inferTimeout() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestRecursiveMapMerge(t *testing.T) {
	tests := []struct {
		name        string
		dest        map[string]any
		src         map[string]any
		want        map[string]any
		wantWarning string
	}{
		{
			name: "simple merge with new keys",
			dest: map[string]any{"a": 1},
			src:  map[string]any{"b": 2},
			want: map[string]any{"a": 1, "b": 2},
		},
		{
			name: "overwrite existing value",
			dest: map[string]any{"a": 1},
			src:  map[string]any{"a": 2},
			want: map[string]any{"a": 2},
		},
		{
			name: "recursive merge of nested maps",
			dest: map[string]any{"nested": map[string]any{"x": 10, "z": 30}},
			src:  map[string]any{"nested": map[string]any{"y": 20, "x": 100}},
			want: map[string]any{"nested": map[string]any{"x": 100, "y": 20, "z": 30}},
		},
		{
			name:        "type mismatch overwrite",
			dest:        map[string]any{"key": "string value"},
			src:         map[string]any{"key": 123},
			want:        map[string]any{"key": 123},
			wantWarning: "Warning: Type mismatch for key 'key'. Existing type: string, new type: int. Overwriting.",
		},
		{
			name:        "overwrite non-map with map",
			dest:        map[string]any{"key": "a string"},
			src:         map[string]any{"key": map[string]any{"nested": true}},
			want:        map[string]any{"key": map[string]any{"nested": true}},
			wantWarning: "Warning: Type mismatch for key 'key'. Existing type: string, new type: map[string]interface {}. Overwriting.",
		},
		{
			name:        "overwrite map with non-map",
			dest:        map[string]any{"key": map[string]any{"nested": true}},
			src:         map[string]any{"key": "a string"},
			want:        map[string]any{"key": "a string"},
			wantWarning: "Warning: Type mismatch for key 'key'. Existing type: map[string]interface {}, new type: string. Overwriting.",
		},
		{
			name: "dest is nil",
			dest: nil,
			src:  map[string]any{"a": 1},
			want: nil,
		},
		{
			name: "src is nil",
			dest: map[string]any{"a": 1},
			src:  nil,
			want: map[string]any{"a": 1},
		},
		{
			name: "overwrite with nil value",
			dest: map[string]any{"a": 1},
			src:  map[string]any{"a": nil},
			want: map[string]any{"a": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			if tt.wantWarning != "" {
				// Redirect log output
				originalOutput := log.Writer()
				log.SetOutput(&logBuf)
				defer log.SetOutput(originalOutput)
			}

			recursiveMapMerge(tt.dest, tt.src)
			if diff := cmp.Diff(tt.want, tt.dest); diff != "" {
				t.Errorf("recursiveMapMerge() mismatch (-want +got):\n%s", diff)
			}

			if tt.wantWarning != "" {
				if !strings.Contains(logBuf.String(), tt.wantWarning) {
					t.Errorf("recursiveMapMerge() log output = %q, want to contain %q", logBuf.String(), tt.wantWarning)
				}
			}
		})
	}
}
