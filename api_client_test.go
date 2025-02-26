package genai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

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
			wantErr:      &ClientError{apiError: apiError{Code: http.StatusBadRequest, Message: ""}},
		},
		{
			desc:         "500 error response",
			path:         "bar",
			method:       http.MethodGet,
			responseCode: http.StatusInternalServerError,
			responseBody: `{"error": {"code": 500, "message": "internal server error", "status": "INTERNAL_SERVER_ERROR", "details": [{"field": "value"}]}}`,
			wantErr:      &ServerError{apiError: apiError{Code: http.StatusInternalServerError, Message: ""}},
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
				if tt.responseCode >= 400 && tt.responseCode < 500 {
					_, ok := err.(ClientError)
					if !ok {
						t.Errorf("want ClientError, got %T(%s)", err, err.Error())
					}

				} else if tt.responseCode >= 500 {
					_, ok := err.(ServerError)
					if !ok {
						t.Errorf("want ServerError, got %T", err)
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
				PublicationDate: Ptr(civil.DateOf(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))),
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
