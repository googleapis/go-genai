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
)

func TestBatchesGetInlinedEmbeddings(t *testing.T) {
	ctx := context.Background()

	// Mock the exact structure the Gemini Developer API returns:
	// "embedding" results are nested inside "inlinedResponses"
	mockResponse := map[string]any{
		"name": "batches/mock-batch-job",
		"metadata": map[string]any{
			"state": "BATCH_STATE_SUCCEEDED",
			"output": map[string]any{
				"inlinedResponses": map[string]any{
					"inlinedResponses": []any{
						map[string]any{
							"response": map[string]any{
								"embedding": map[string]any{
									"values": []float32{0.1, 0.2, 0.3},
								},
							},
						},
					},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, err := json.Marshal(mockResponse)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))
	defer ts.Close()

	client, err := NewClient(ctx, &ClientConfig{
		Backend:     BackendGeminiAPI,
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		APIKey:      "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	job, err := client.Batches.Get(ctx, "batches/mock-batch-job", nil)
	if err != nil {
		t.Fatalf("Batches.Get() failed: %v", err)
	}

	if job.Dest == nil {
		t.Fatalf("job.Dest is nil")
	}

	// Verify that the transformer successfully routed the data to InlinedEmbedContentResponses
	if len(job.Dest.InlinedEmbedContentResponses) == 0 {
		t.Errorf("job.Dest.InlinedEmbedContentResponses is empty, expected 1")
	} else {
		embedding := job.Dest.InlinedEmbedContentResponses[0].Response.Embedding
		if embedding == nil || len(embedding.Values) == 0 {
			t.Errorf("Expected embedding values, got none")
		} else if embedding.Values[0] != 0.1 {
			t.Errorf("Expected first embedding value to be 0.1, got %v", embedding.Values[0])
		}
	}
}

func TestBatchesGetInlinedText(t *testing.T) {
	ctx := context.Background()

	// Mock the exact structure the Gemini Developer API returns for text generation:
	// "candidates" results are nested inside "inlinedResponses"
	mockResponse := map[string]any{
		"name": "batches/mock-batch-job",
		"metadata": map[string]any{
			"state": "BATCH_STATE_SUCCEEDED",
			"output": map[string]any{
				"inlinedResponses": map[string]any{
					"inlinedResponses": []any{
						map[string]any{
							"response": map[string]any{
								"candidates": []any{
									map[string]any{
										"content": map[string]any{
											"parts": []any{
												map[string]any{"text": "mock text response"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, err := json.Marshal(mockResponse)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))
	defer ts.Close()

	client, err := NewClient(ctx, &ClientConfig{
		Backend:     BackendGeminiAPI,
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		APIKey:      "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	job, err := client.Batches.Get(ctx, "batches/mock-batch-job", nil)
	if err != nil {
		t.Fatalf("Batches.Get() failed: %v", err)
	}

	if job.Dest == nil {
		t.Fatalf("job.Dest is nil")
	}

	// Verify that the transformer left the text generation responses in InlinedResponses untouched
	if len(job.Dest.InlinedResponses) == 0 {
		t.Errorf("job.Dest.InlinedResponses is empty, expected 1")
	} else {
		candidates := job.Dest.InlinedResponses[0].Response.Candidates
		if len(candidates) == 0 {
			t.Errorf("Expected candidates, got none")
		} else if candidates[0].Content.Parts[0].Text != "mock text response" {
			t.Errorf("Expected text 'mock text response', got %v", candidates[0].Content.Parts[0].Text)
		}
	}
}
