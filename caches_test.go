package genai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCachesAll(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name                   string
		serverResponses        []map[string]any
		expectedCachedContents []*CachedContent
	}{
		{
			name: "Pagination_SinglePage",
			serverResponses: []map[string]any{
				{
					"cachedContents": []*CachedContent{
						{Name: "cachedContent1", DisplayName: "Cache 1"},
						{Name: "cachedContent2", DisplayName: "Cache 2"},
					},
					"nextPageToken": "",
				},
			},
			expectedCachedContents: []*CachedContent{
				{Name: "cachedContent1", DisplayName: "Cache 1"},
				{Name: "cachedContent2", DisplayName: "Cache 2"},
			},
		},
		{
			name: "Pagination_MultiplePages",
			serverResponses: []map[string]any{
				{
					"cachedContents": []*CachedContent{
						{Name: "cachedContent1", DisplayName: "Cache 1"},
					},
					"nextPageToken": "next_page_token",
				},
				{
					"cachedContents": []*CachedContent{
						{Name: "cachedContent2", DisplayName: "Cache 2"},
						{Name: "cachedContent3", DisplayName: "Cache 3"},
					},
					"nextPageToken": "",
				},
			},
			expectedCachedContents: []*CachedContent{
				{Name: "cachedContent1", DisplayName: "Cache 1"},
				{Name: "cachedContent2", DisplayName: "Cache 2"},
				{Name: "cachedContent3", DisplayName: "Cache 3"},
			},
		},
		{
			name:                   "Empty_Response",
			serverResponses:        []map[string]any{{"cachedContents": []*CachedContent{}, "nextPageToken": ""}},
			expectedCachedContents: []*CachedContent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseIndex := 0
			// Create a test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if responseIndex > 0 && r.URL.Query().Get("pageToken") == "" {
					t.Errorf("Caches.All() failed to pass pageToken in the request")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				response, err := json.Marshal(tt.serverResponses[responseIndex])
				if err != nil {
					t.Errorf("Failed to marshal response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err = w.Write(response)
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				responseIndex++
			}))
			defer ts.Close()

			// Create a client with the test server
			client, err := NewClient(context.Background(), &ClientConfig{
				HTTPOptions: HTTPOptions{BaseURL: ts.URL},
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "test-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Convert iterator to slice
			gotCachedContents := []*CachedContent{}
			for cachedContent, err := range client.Caches.All(ctx) {
				if err != nil {
					t.Errorf("Caches.All() iteration error = %v", err)
					return
				}
				gotCachedContents = append(gotCachedContents, cachedContent)
			}

			// Compare results
			if diff := cmp.Diff(tt.expectedCachedContents, gotCachedContents); diff != "" {
				t.Errorf("Caches.All() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
