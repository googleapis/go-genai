// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/genai/interactions"
)

func TestInteractionsWorkflow(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/interactions" {
			t.Errorf("Path unexpected: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Method unexpected: %s", r.Method)
		}

		// Return a mock response
		resp := map[string]any{
			"id":      "mock_interaction_id",
			"created": "2026-03-30T22:20:00Z",
			"updated": "2026-03-30T22:20:00Z",
			"status":  "completed",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer ts.Close()

	// Create client pointing to the mock server
	client, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendGeminiAPI,
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
		APIKey: "dummy_key",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Interactions == nil {
		t.Fatalf("client.Interactions is nil")
	}

	// Call NewModel
	res, err := client.Interactions.NewModel(context.Background(), interactions.NewModelParams{
		Model: "gemini-2.5-flash",
		Input: interactions.Input{
			ContentList: []interactions.Content{{
				Text: &interactions.TextContent{
					Text: "Hello",
				},
			}},
		},
	})
	if err != nil {
		t.Fatalf("Failed to call NewModel: %v", err)
	}

	if res.ID != "mock_interaction_id" {
		t.Errorf("Expected ID 'mock_interaction_id', got '%s'", res.ID)
	}
}
