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

	gaos_interactions "google.golang.org/genai/gaos/models/interactions"
	"google.golang.org/genai/gaos/models/operations"
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

	// Call Create
	modelInput := gaos_interactions.CreateInteractionsInputArrayOfContent([]gaos_interactions.Content{{
		TextContent: &gaos_interactions.TextContent{
			Text: "Hello",
		},
	}})
	body := operations.CreateCreateInteractionRequestBodyCreateModelInteraction(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model("gemini-2.5-flash"),
		Input: modelInput,
	})
	res, err := client.Interactions.Create(context.Background(), body, nil)
	if err != nil {
		t.Fatalf("Failed to call Create: %v", err)
	}

	if res.Interaction.ID == nil || *res.Interaction.ID != "mock_interaction_id" {
		t.Errorf("Expected ID 'mock_interaction_id', got '%v'", res.Interaction.ID)
	}
}
