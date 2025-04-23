package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
)

// TODO(b/384580303): Add streaming request tests.
func TestSendRequest(t *testing.T) {
	ctx := context.Background()
	// Setup test cases
	tests := []struct {
		desc         string
		path         string
		method       string
		requestBody  map[string]any
		responseCode int
		responseBody string
		want         map[string]any
		wantErr      error
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
			wantErr:      fmt.Errorf("newAPIError: unmarshal response to error failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Create a test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				fmt.Fprintln(w, tt.responseBody)
			}))
			defer ts.Close()

			// Create a test client with the test server's URL
			ac := &apiClient{
				clientConfig: &ClientConfig{
					HTTPOptions: HTTPOptions{
						BaseURL: ts.URL,
					},
					HTTPClient: ts.Client(),
				},
			}

			got, err := sendRequest(ctx, ac, tt.path, tt.method, tt.requestBody, &HTTPOptions{BaseURL: ts.URL})

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
				} else if tt.path == "" { // build request error
					if !strings.Contains(err.Error(), tt.wantErr.Error()) {
						t.Errorf("unexpected error, want error that contains 'createAPIURL: error parsing', got: %v", err)
					}

				} else { // deserialize error
					if !strings.Contains(err.Error(), "deserializeUnaryResponse: error unmarshalling response") {
						t.Errorf("unexpected error, want error that contains 'deserializeUnaryResponse: error unmarshalling response', got: %v", err)
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
			name:           "Stream with Invalid Seperator",
			method:         "POST",
			path:           "test",
			body:           map[string]any{"key": "value"},
			mockResponse:   "data:{\"key1\":\"value1\"}\t\tdata:{\"key2\":\"value2\"}",
			mockStatusCode: http.StatusOK,
			// converterErr:     fmt.Errorf("converter error"),
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
			wantErrorMessage: "doRequest: error sending request: Post \"invalid-url//test\": unsupported protocol scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				},
				HTTPClient: ts.Client(),
			}
			if tt.httpOptions != nil {
				clientConfig.HTTPOptions = *tt.httpOptions
			}

			ac := &apiClient{clientConfig: clientConfig}
			var output responseStream[map[string]any]
			err := sendStreamRequest(context.Background(), ac, tt.path, tt.method, tt.body, &clientConfig.HTTPOptions, &output)

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
	tests := []struct {
		name          string
		clientConfig  *ClientConfig
		path          string
		body          map[string]any
		method        string
		httpOptions   *HTTPOptions
		want          *http.Request
		wantErr       bool
		expectedError string
	}{
		{
			name: "MLDev API with API Key",
			clientConfig: &ClientConfig{
				APIKey:  "test-api-key",
				Backend: BackendGeminiAPI,
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
			name: "MLDev with empty body",
			clientConfig: &ClientConfig{
				APIKey:  "test-api-key",
				Backend: BackendGeminiAPI,
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
				APIKey:  "test-api-key",
				Backend: BackendGeminiAPI,
			},
			path:   ":invalid",
			body:   map[string]any{},
			method: "POST",
			httpOptions: &HTTPOptions{
				BaseURL:    ":invalid",
				APIVersion: "v1beta",
			},
			wantErr:       true,
			expectedError: "createAPIURL: error parsing ML Dev URL",
		},
		{
			name: "Invalid json",
			clientConfig: &ClientConfig{
				APIKey:  "test-api-key",
				Backend: BackendGeminiAPI,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &apiClient{clientConfig: tt.clientConfig}

			req, err := buildRequest(context.Background(), ac, tt.path, tt.body, tt.method, tt.httpOptions)

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

			if !reflect.DeepEqual(req.Context(), tt.want.Context()) {
				t.Errorf("buildRequest() Context mismatch got %+v, want %+v", req.Context(), tt.want.Context())
			}
		})
	}
}

func Test_sdkHeader(t *testing.T) {
	type args struct {
		ac *apiClient
	}
	tests := []struct {
		name string
		args args
		want http.Header
	}{
		{
			name: "with_api_key",
			args: args{&apiClient{clientConfig: &ClientConfig{APIKey: "test_api_key"}}},
			want: http.Header{
				"Content-Type":      []string{"application/json"},
				"X-Goog-Api-Key":    []string{"test_api_key"},
				"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
			},
		},
		{
			name: "without_api_key",
			args: args{&apiClient{clientConfig: &ClientConfig{}}},
			want: http.Header{
				"Content-Type":      []string{"application/json"},
				"User-Agent":        []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
				"X-Goog-Api-Client": []string{fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(sdkHeader(tt.args.ac), tt.want); diff != "" {
				t.Errorf("sdkHeader() mismatch (-want +got):\n%s", diff)
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
func mockUploadServer(t *testing.T, expectedSize int64) (*httptest.Server, *sync.Map) {
	t.Helper()
	var totalReceived int64
	var mu sync.Mutex
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
		receivedData.Store(uploadOffset, bodyBytes)

		mu.Lock()
		totalReceived += contentLength
		currentTotal := totalReceived
		mu.Unlock()

		isFinal := strings.Contains(uploadCommand, "finalize")

		if isFinal {
			if currentTotal != expectedSize {
				t.Errorf("Final size mismatch: expected %d, received %d", expectedSize, currentTotal)
				http.Error(w, "Final size mismatch", http.StatusBadRequest)
				return
			}
			w.Header().Set("X-Goog-Upload-Status", "final")
			w.Header().Set("Content-Type", "application/json")
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
			w.Header().Set("X-Goog-Upload-Status", "active")
			w.WriteHeader(http.StatusOK)
		}
	}))

	return server, receivedData
}

func TestUploadFile(t *testing.T) {
	ctx := context.Background()

	testSizes := []struct {
		name string
		size int64 // Size in bytes
	}{
		{"1MB", 1 * 1024 * 1024},
		{"8MB", 8 * 1024 * 1024}, // Exactly maxChunkSize
		{"9MB", 9 * 1024 * 1024}, // Requires multiple chunks
	}

	for _, ts := range testSizes {
		t.Run(ts.name, func(t *testing.T) {
			filePath, cleanup := createTestFile(t, ts.size)
			defer cleanup()

			server, _ := mockUploadServer(t, ts.size)
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
