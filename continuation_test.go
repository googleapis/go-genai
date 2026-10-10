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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestShouldEnableAutomaticContinuation(t *testing.T) {
	tests := []struct {
		name   string
		config *GenerateContentConfig
		want   bool
	}{
		{
			name:   "nil config defaults to true",
			config: nil,
			want:   true,
		},
		{
			name:   "unset AutomaticContinuation defaults to true",
			config: &GenerateContentConfig{},
			want:   true,
		},
		{
			name:   "explicit true",
			config: &GenerateContentConfig{AutomaticContinuation: Ptr(true)},
			want:   true,
		},
		{
			name:   "explicit false",
			config: &GenerateContentConfig{AutomaticContinuation: Ptr(false)},
			want:   false,
		},
		{
			name: "max output tokens set does not override default true",
			config: &GenerateContentConfig{
				MaxOutputTokens: 100,
			},
			want: true,
		},
		{
			name: "max output tokens set does not override explicit true",
			config: &GenerateContentConfig{
				MaxOutputTokens:       100,
				AutomaticContinuation: Ptr(true),
			},
			want: true,
		},
		{
			name: "max output tokens set does not override explicit false",
			config: &GenerateContentConfig{
				MaxOutputTokens:       100,
				AutomaticContinuation: Ptr(false),
			},
			want: false,
		},
		{
			name: "max output tokens zero with explicit true",
			config: &GenerateContentConfig{
				MaxOutputTokens:       0,
				AutomaticContinuation: Ptr(true),
			},
			want: true,
		},
		{
			name: "max output tokens zero with explicit false",
			config: &GenerateContentConfig{
				MaxOutputTokens:       0,
				AutomaticContinuation: Ptr(false),
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldEnableAutomaticContinuation(tc.config)
			if got != tc.want {
				t.Errorf("shouldEnableAutomaticContinuation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsResumableFinishReason(t *testing.T) {
	tests := []struct {
		name         string
		finishReason FinishReason
		want         bool
	}{
		{
			name:         "CONTINUATION",
			finishReason: FinishReasonContinuation,
			want:         true,
		},
		{
			name:         "MAX_TOKENS",
			finishReason: FinishReasonMaxTokens,
			want:         false,
		},
		{
			name:         "STOP",
			finishReason: FinishReasonStop,
			want:         false,
		},
		{
			name:         "SAFETY",
			finishReason: FinishReasonSafety,
			want:         false,
		},
		{
			name:         "UNSPECIFIED",
			finishReason: FinishReasonUnspecified,
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isResumableFinishReason(tc.finishReason)
			if got != tc.want {
				t.Errorf("isResumableFinishReason(%v) = %v, want %v", tc.finishReason, got, tc.want)
			}
		})
	}
}

func TestShouldContinueGeneration(t *testing.T) {
	tests := []struct {
		name      string
		response  *GenerateContentResponse
		wantToken []byte
	}{
		{
			name:      "nil response",
			response:  nil,
			wantToken: nil,
		},
		{
			name:      "empty candidates",
			response:  &GenerateContentResponse{Candidates: []*Candidate{}},
			wantToken: nil,
		},
		{
			name: "continuation token with CONTINUATION reason",
			response: &GenerateContentResponse{
				Candidates: []*Candidate{
					{
						ContinuationToken: []byte("token123"),
						FinishReason:      FinishReasonContinuation,
					},
				},
			},
			wantToken: []byte("token123"),
		},
		{
			name: "continuation token with MAX_TOKENS reason does not continue",
			response: &GenerateContentResponse{
				Candidates: []*Candidate{
					{
						ContinuationToken: []byte("token456"),
						FinishReason:      FinishReasonMaxTokens,
					},
				},
			},
			wantToken: nil,
		},
		{
			name: "continuation token with STOP reason",
			response: &GenerateContentResponse{
				Candidates: []*Candidate{
					{
						ContinuationToken: []byte("token123"),
						FinishReason:      FinishReasonStop,
					},
				},
			},
			wantToken: nil,
		},
		{
			name: "empty token with CONTINUATION reason",
			response: &GenerateContentResponse{
				Candidates: []*Candidate{
					{
						ContinuationToken: nil,
						FinishReason:      FinishReasonContinuation,
					},
				},
			},
			wantToken: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldContinueGeneration(tc.response)
			if !bytes.Equal(got, tc.wantToken) {
				t.Errorf("shouldContinueGeneration() = %v, want %v", got, tc.wantToken)
			}
		})
	}
}

func TestPrepareContinuationConfig(t *testing.T) {
	t.Run("nil base config", func(t *testing.T) {
		token := []byte("token1")
		cfg := prepareContinuationConfig(nil, token)
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
		if !bytes.Equal(cfg.ContinuationToken, token) {
			t.Errorf("ContinuationToken = %v, want %v", cfg.ContinuationToken, token)
		}
	})

	t.Run("non-nil base config clones and leaves original untouched", func(t *testing.T) {
		orig := &GenerateContentConfig{
			Temperature:           Ptr(float32(0.7)),
			AutomaticContinuation: Ptr(true),
		}
		token := []byte("token2")
		cloned := prepareContinuationConfig(orig, token)

		if cloned == orig {
			t.Fatal("expected a distinct cloned pointer")
		}
		if orig.ContinuationToken != nil {
			t.Errorf("original ContinuationToken mutated: %v", orig.ContinuationToken)
		}
		if !bytes.Equal(cloned.ContinuationToken, token) {
			t.Errorf("cloned ContinuationToken = %v, want %v", cloned.ContinuationToken, token)
		}
		if *cloned.Temperature != *orig.Temperature {
			t.Errorf("cloned Temperature = %v, want %v", *cloned.Temperature, *orig.Temperature)
		}
	})
}

func TestMergeContinuationResponses_EmptyAndSingle(t *testing.T) {
	t.Run("empty responses", func(t *testing.T) {
		merged, err := mergeContinuationResponses(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if merged == nil {
			t.Fatal("expected non-nil response")
		}
	})

	t.Run("single response returned as-is", func(t *testing.T) {
		single := &GenerateContentResponse{
			ResponseID: "single-1",
			Candidates: []*Candidate{
				{
					Content:      &Content{Parts: []*Part{{Text: "hello"}}},
					FinishReason: FinishReasonStop,
				},
			},
		}
		merged, err := mergeContinuationResponses([]*GenerateContentResponse{single})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if merged != single {
			t.Errorf("expected original instance returned, got %v", merged)
		}
	})
}

func TestMergeContinuationResponses_CandidatePartsAndTerminalFields(t *testing.T) {
	hop1 := &GenerateContentResponse{
		ResponseID: "hop-1",
		Candidates: []*Candidate{
			{
				Content: &Content{
					Role:  RoleModel,
					Parts: []*Part{{Text: "Hello "}},
				},
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("token-hop-1"),
				FinishMessage:     "continued",
			},
		},
		SDKHTTPResponse: &HTTPResponse{Headers: http.Header{"X-Hop": []string{"1"}}},
	}

	hop2 := &GenerateContentResponse{
		ResponseID: "hop-2",
		Candidates: []*Candidate{
			{
				Content: &Content{
					Role:  RoleModel,
					Parts: []*Part{{Text: "world! "}, {Text: ""}},
				},
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("token-hop-2"),
				FinishMessage:     "continued again",
			},
		},
		SDKHTTPResponse: &HTTPResponse{Headers: http.Header{"X-Hop": []string{"2"}}},
	}

	hop3 := &GenerateContentResponse{
		ResponseID: "hop-3",
		Candidates: []*Candidate{
			{
				Content: &Content{
					Role:  RoleModel,
					Parts: []*Part{{Text: "Finished."}},
				},
				FinishReason:      FinishReasonStop,
				ContinuationToken: nil,
				FinishMessage:     "",
			},
		},
		SDKHTTPResponse: &HTTPResponse{Headers: http.Header{"X-Hop": []string{"3"}}},
	}

	merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2, hop3})
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}

	if len(merged.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(merged.Candidates))
	}

	cand := merged.Candidates[0]
	// Verify parts concatenation in order (including empty part)
	if len(cand.Content.Parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(cand.Content.Parts))
	}
	expectedParts := []string{"Hello ", "world! ", "", "Finished."}
	for i, exp := range expectedParts {
		if cand.Content.Parts[i].Text != exp {
			t.Errorf("part %d text = %q, want %q", i, cand.Content.Parts[i].Text, exp)
		}
	}

	// Verify terminal hop fields: finishReason, continuationToken, finishMessage
	if cand.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %v, want %v", cand.FinishReason, FinishReasonStop)
	}
	if len(cand.ContinuationToken) != 0 {
		t.Errorf("ContinuationToken = %v, want nil/empty", cand.ContinuationToken)
	}
	if cand.FinishMessage != "" {
		t.Errorf("FinishMessage = %q, want empty", cand.FinishMessage)
	}

	// Verify SDKHTTPResponse retained from latest hop
	if merged.SDKHTTPResponse == nil || merged.SDKHTTPResponse.Headers.Get("X-Hop") != "3" {
		t.Errorf("SDKHTTPResponse not retained from latest hop: %v", merged.SDKHTTPResponse)
	}

	// Verify Text() helper
	if merged.Text() != "Hello world! Finished." {
		t.Errorf("merged.Text() = %q, want %q", merged.Text(), "Hello world! Finished.")
	}
}

func TestMergeContinuationResponses_UsageMetadata(t *testing.T) {
	hop1 := &GenerateContentResponse{
		UsageMetadata: &GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 15,
			TotalTokenCount:      25,
			PromptTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 10},
			},
			CandidatesTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 15},
			},
			TrafficType: TrafficType("DEFAULT"),
		},
	}

	hop2 := &GenerateContentResponse{
		UsageMetadata: &GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
			PromptTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 10},
			},
			CandidatesTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 20},
				{Modality: MediaModalityImage, TokenCount: 5},
			},
			TrafficType: TrafficType("DEFAULT"),
		},
	}

	merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2})
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}

	um := merged.UsageMetadata
	if um == nil {
		t.Fatal("expected non-nil UsageMetadata")
	}

	if um.PromptTokenCount != 20 {
		t.Errorf("PromptTokenCount = %d, want 20", um.PromptTokenCount)
	}
	if um.CandidatesTokenCount != 35 {
		t.Errorf("CandidatesTokenCount = %d, want 35", um.CandidatesTokenCount)
	}
	if um.TotalTokenCount != 55 {
		t.Errorf("TotalTokenCount = %d, want 55", um.TotalTokenCount)
	}

	// Check modality breakdown grouping and summing
	if len(um.CandidatesTokensDetails) != 2 {
		t.Fatalf("expected 2 candidates tokens details, got %d", len(um.CandidatesTokensDetails))
	}
	if um.CandidatesTokensDetails[0].Modality != MediaModalityText || um.CandidatesTokensDetails[0].TokenCount != 35 {
		t.Errorf("CandidatesTokensDetails[0] = %+v, want Text: 35", um.CandidatesTokensDetails[0])
	}
	if um.CandidatesTokensDetails[1].Modality != MediaModalityImage || um.CandidatesTokensDetails[1].TokenCount != 5 {
		t.Errorf("CandidatesTokensDetails[1] = %+v, want Image: 5", um.CandidatesTokensDetails[1])
	}
}

func TestMergeContinuationResponses_SafetyRatings(t *testing.T) {
	hop1 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				SafetyRatings: []*SafetyRating{
					{Category: HarmCategoryHateSpeech, Probability: HarmProbabilityLow},
					{Category: HarmCategoryHarassment, Probability: HarmProbabilityLow},
				},
			},
		},
	}

	hop2 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				SafetyRatings: []*SafetyRating{
					{Category: HarmCategoryHateSpeech, Probability: HarmProbabilityMedium},
					{Category: HarmCategoryDangerousContent, Probability: HarmProbabilityNegligible},
				},
			},
		},
	}

	merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2})
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}

	cand := merged.Candidates[0]
	if len(cand.SafetyRatings) != 3 {
		t.Fatalf("expected 3 safety ratings, got %d", len(cand.SafetyRatings))
	}

	ratingsMap := make(map[HarmCategory]HarmProbability)
	for _, r := range cand.SafetyRatings {
		ratingsMap[r.Category] = r.Probability
	}

	if ratingsMap[HarmCategoryHateSpeech] != HarmProbabilityMedium {
		t.Errorf("HateSpeech probability = %v, want %v", ratingsMap[HarmCategoryHateSpeech], HarmProbabilityMedium)
	}
	if ratingsMap[HarmCategoryHarassment] != HarmProbabilityLow {
		t.Errorf("Harassment probability = %v, want %v", ratingsMap[HarmCategoryHarassment], HarmProbabilityLow)
	}
	if ratingsMap[HarmCategoryDangerousContent] != HarmProbabilityNegligible {
		t.Errorf("DangerousContent probability = %v, want %v", ratingsMap[HarmCategoryDangerousContent], HarmProbabilityNegligible)
	}
}

func TestMergeContinuationResponses_GroundingMetadataPreserved(t *testing.T) {
	hop1 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				GroundingMetadata: &GroundingMetadata{
					WebSearchQueries: []string{"test query"},
				},
			},
		},
		PromptFeedback: &GenerateContentResponsePromptFeedback{
			BlockReason: BlockedReasonSafety,
		},
	}

	hop2 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				GroundingMetadata: nil,
			},
		},
		PromptFeedback: nil,
	}

	merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2})
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}

	if merged.Candidates[0].GroundingMetadata == nil || len(merged.Candidates[0].GroundingMetadata.WebSearchQueries) != 1 {
		t.Errorf("GroundingMetadata was not preserved from hop 1: %+v", merged.Candidates[0].GroundingMetadata)
	}
	if merged.PromptFeedback == nil || merged.PromptFeedback.BlockReason != BlockedReasonSafety {
		t.Errorf("PromptFeedback was not preserved from hop 1: %+v", merged.PromptFeedback)
	}
}

func TestModelsGenerateContent_AutomaticContinuationLoop(t *testing.T) {
	ctx := context.Background()

	t.Run("continues across hops when automatic continuation is enabled", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			body, _ := io.ReadAll(r.Body)

			var reqMap map[string]any
			_ = json.Unmarshal(body, &reqMap)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if count == 1 {
				// Verify first request does not have continuation_token
				if reqMap["continuation_token"] != nil && reqMap["continuationToken"] != nil {
					t.Errorf("hop 1 request unexpectedly contained continuationToken: %v", reqMap)
				}
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Part one of message. "}]
							},
							"finishReason": "CONTINUATION",
							"continuationToken": "dG9rZW4tZm9yLWhvcC0y"
						}
					],
					"usageMetadata": {
						"candidatesTokenCount": 10,
						"totalTokenCount": 10
					}
				}`)
			} else if count == 2 {
				// Verify second request contains continuation_token
				tok := reqMap["continuation_token"]
				if tok == nil {
					tok = reqMap["continuationToken"]
				}
				if tok == nil {
					t.Errorf("hop 2 request missing continuationToken: %v", reqMap)
				}
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Part two of message."}]
							},
							"finishReason": "STOP"
						}
					],
					"usageMetadata": {
						"candidatesTokenCount": 8,
						"totalTokenCount": 8
					}
				}`)
			} else {
				t.Fatalf("unexpected request count: %d", count)
			}
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*Content{
			NewContentFromText("Tell me a story", RoleUser),
		}, &GenerateContentConfig{
			AutomaticContinuation: Ptr(true),
		})
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 2 {
			t.Errorf("expected 2 requests, got %d", atomic.LoadInt32(&requestCount))
		}

		if resp.Text() != "Part one of message. Part two of message." {
			t.Errorf("merged response text = %q, want %q", resp.Text(), "Part one of message. Part two of message.")
		}
		if resp.Candidates[0].FinishReason != FinishReasonStop {
			t.Errorf("finish reason = %v, want %v", resp.Candidates[0].FinishReason, FinishReasonStop)
		}
		if resp.UsageMetadata.CandidatesTokenCount != 18 {
			t.Errorf("CandidatesTokenCount = %d, want 18", resp.UsageMetadata.CandidatesTokenCount)
		}
	})

	t.Run("continues by default when config is nil on Models", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if count == 1 {
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Hop 1. "}]
							},
							"finishReason": "CONTINUATION",
							"continuationToken": "dG9rZW4tMQ=="
						}
					]
				}`)
			} else {
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Hop 2."}]
							},
							"finishReason": "STOP"
						}
					]
				}`)
			}
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*Content{
			NewContentFromText("Hello", RoleUser),
		}, nil)
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 2 {
			t.Errorf("expected 2 requests, got %d", atomic.LoadInt32(&requestCount))
		}
		if resp.Text() != "Hop 1. Hop 2." {
			t.Errorf("resp.Text() = %q, want %q", resp.Text(), "Hop 1. Hop 2.")
		}
	})

	t.Run("does not continue when automatic continuation is explicitly false on Models", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [{"text": "Incomplete response"}]
						},
						"finishReason": "CONTINUATION",
						"continuationToken": "dG9rZW4tMQ=="
					}
				]
			}`)
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*Content{
			NewContentFromText("Hello", RoleUser),
		}, &GenerateContentConfig{
			AutomaticContinuation: Ptr(false),
		})
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 1 {
			t.Errorf("expected 1 request, got %d", atomic.LoadInt32(&requestCount))
		}
		if resp.Candidates[0].FinishReason != FinishReasonContinuation {
			t.Errorf("finish reason = %v, want CONTINUATION", resp.Candidates[0].FinishReason)
		}
	})

	t.Run("does not continue when finish reason is MAX_TOKENS even with continuation token", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [{"text": "Partial"}]
						},
						"finishReason": "MAX_TOKENS",
						"continuationToken": "dG9rZW4tMQ=="
					}
				]
			}`)
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*Content{
			NewContentFromText("Hello", RoleUser),
		}, &GenerateContentConfig{
			AutomaticContinuation: Ptr(true),
		})
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 1 {
			t.Errorf("expected 1 request, got %d", atomic.LoadInt32(&requestCount))
		}
		if resp.Candidates[0].FinishReason != FinishReasonMaxTokens {
			t.Errorf("finish reason = %v, want MAX_TOKENS", resp.Candidates[0].FinishReason)
		}
	})

	t.Run("continues when max output tokens is set and backend returns CONTINUATION", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if count == 1 {
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Part 1. "}]
							},
							"finishReason": "CONTINUATION",
							"continuationToken": "dG9rZW4tMg=="
						}
					]
				}`)
			} else {
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Part 2."}]
							},
							"finishReason": "STOP"
						}
					]
				}`)
			}
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*Content{
			NewContentFromText("Hello", RoleUser),
		}, &GenerateContentConfig{
			AutomaticContinuation: Ptr(true),
			MaxOutputTokens:       10,
		})
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 2 {
			t.Errorf("expected 2 requests, got %d", atomic.LoadInt32(&requestCount))
		}
		if resp.Text() != "Part 1. Part 2." {
			t.Errorf("resp.Text() = %q, want %q", resp.Text(), "Part 1. Part 2.")
		}
	})
}

func TestModelsGenerateContentStream_AutomaticContinuationLoop(t *testing.T) {
	ctx := context.Background()

	var requestCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if count == 1 {
			// Hop 1 stream chunks
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream part 1, "}]}}]}`)
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"finishReason": "CONTINUATION", "continuationToken": "c3RyZWFtLXRva2VuLTI="}]}`)
		} else if count == 2 {
			// Hop 2 stream chunks
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "stream part 2."}]}}]}`)
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"finishReason": "STOP"}]}`)
		} else {
			t.Fatalf("unexpected stream request count: %d", count)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ctx, &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		envVarProvider: func() map[string]string {
			return map[string]string{"GOOGLE_API_KEY": "test-key"}
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var accumulatedText string
	for resp, err := range client.Models.GenerateContentStream(ctx, "gemini-2.5-flash", []*Content{
		NewContentFromText("Stream please", RoleUser),
	}, nil) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if resp != nil {
			accumulatedText += resp.Text()
		}
	}

	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 stream requests across hops by default, got %d", atomic.LoadInt32(&requestCount))
	}
	if accumulatedText != "Stream part 1, stream part 2." {
		t.Errorf("accumulated stream text = %q, want %q", accumulatedText, "Stream part 1, stream part 2.")
	}

	// Verify explicit AutomaticContinuation=false does not continue
	atomic.StoreInt32(&requestCount, 0)
	var optOutText string
	for resp, err := range client.Models.GenerateContentStream(ctx, "gemini-2.5-flash", []*Content{
		NewContentFromText("Stream please", RoleUser),
	}, &GenerateContentConfig{
		AutomaticContinuation: Ptr(false),
	}) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if resp != nil {
			optOutText += resp.Text()
		}
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 stream request when AutomaticContinuation is false, got %d", atomic.LoadInt32(&requestCount))
	}
	if optOutText != "Stream part 1, " {
		t.Errorf("optOutText = %q, want %q", optOutText, "Stream part 1, ")
	}
}

func TestChat_AutomaticContinuation(t *testing.T) {
	ctx := context.Background()

	t.Run("chat automatically continues unary response by default", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if count == 1 {
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Chat answer part 1. "}]
							},
							"finishReason": "CONTINUATION",
							"continuationToken": "Y2hhdC10b2tlbi0y"
						}
					]
				}`)
			} else if count == 2 {
				fmt.Fprintln(w, `{
					"candidates": [
						{
							"content": {
								"role": "model",
								"parts": [{"text": "Chat answer part 2."}]
							},
							"finishReason": "STOP"
						}
					]
				}`)
			}
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
		if err != nil {
			t.Fatalf("failed to create chat: %v", err)
		}

		resp, err := chat.SendMessage(ctx, Part{Text: "Explain continuation"})
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 2 {
			t.Errorf("expected 2 requests for chat continuation, got %d", atomic.LoadInt32(&requestCount))
		}
		if resp.Text() != "Chat answer part 1. Chat answer part 2." {
			t.Errorf("resp.Text() = %q, want %q", resp.Text(), "Chat answer part 1. Chat answer part 2.")
		}

		// Verify chat history records the merged content
		hist := chat.History(true)
		if len(hist) != 2 {
			t.Fatalf("expected 2 history entries (user + model), got %d", len(hist))
		}
		if hist[0].Role != RoleUser || hist[0].Parts[0].Text != "Explain continuation" {
			t.Errorf("unexpected user history: %+v", hist[0])
		}
		if hist[1].Role != RoleModel {
			t.Errorf("expected model role in history, got %v", hist[1].Role)
		}
		var historyModelText string
		for _, p := range hist[1].Parts {
			historyModelText += p.Text
		}
		if historyModelText != "Chat answer part 1. Chat answer part 2." {
			t.Errorf("history model text = %q, want %q", historyModelText, "Chat answer part 1. Chat answer part 2.")
		}
	})

	t.Run("chat honors explicit AutomaticContinuation=false", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [{"text": "Single hop only"}]
						},
						"finishReason": "CONTINUATION",
						"continuationToken": "dG9r"
					}
				]
			}`)
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", &GenerateContentConfig{
			AutomaticContinuation: Ptr(false),
		}, nil)
		if err != nil {
			t.Fatalf("failed to create chat: %v", err)
		}

		resp, err := chat.SendMessage(ctx, Part{Text: "Test"})
		if err != nil {
			t.Fatalf("SendMessage failed: %v", err)
		}

		if atomic.LoadInt32(&requestCount) != 1 {
			t.Errorf("expected 1 request, got %d", atomic.LoadInt32(&requestCount))
		}
		if resp.Text() != "Single hop only" {
			t.Errorf("resp.Text() = %q, want %q", resp.Text(), "Single hop only")
		}
	})

	t.Run("chat streaming automatically continues across hops", func(t *testing.T) {
		var requestCount int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			if count == 1 {
				fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Chat stream 1, "}]}, "finishReason": "CONTINUATION", "continuationToken": "c3RyZWFtLXRvaw=="}]}`)
			} else if count == 2 {
				fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "chat stream 2."}]}, "finishReason": "STOP"}]}`)
			}
		}))
		defer ts.Close()

		client, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
		if err != nil {
			t.Fatalf("failed to create chat: %v", err)
		}

		var text string
		for chunk, err := range chat.SendMessageStream(ctx, Part{Text: "Stream test"}) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if chunk != nil {
				text += chunk.Text()
			}
		}

		if atomic.LoadInt32(&requestCount) != 2 {
			t.Errorf("expected 2 requests, got %d", atomic.LoadInt32(&requestCount))
		}
		if text != "Chat stream 1, chat stream 2." {
			t.Errorf("stream text = %q, want %q", text, "Chat stream 1, chat stream 2.")
		}

		// Verify chat history has user message plus both model chunks recorded
		hist := chat.History(true)
		if len(hist) != 3 {
			t.Fatalf("expected 3 history entries (user + 2 stream chunks), got %d", len(hist))
		}
		if hist[0].Role != RoleUser || hist[0].Parts[0].Text != "Stream test" {
			t.Errorf("unexpected user history: %+v", hist[0])
		}
		if hist[1].Role != RoleModel || hist[1].Parts[0].Text != "Chat stream 1, " {
			t.Errorf("unexpected chunk 1 history: %+v", hist[1])
		}
		if hist[2].Role != RoleModel || hist[2].Parts[0].Text != "chat stream 2." {
			t.Errorf("unexpected chunk 2 history: %+v", hist[2])
		}
	})
}

func TestChat_AutomaticContinuation_FourHopsUnary(t *testing.T) {
	ctx := context.Background()

	var requestCount int32
	tokens := []string{
		"aG9wLXRva2VuLTE=", // "hop-token-1"
		"aG9wLXRva2VuLTI=", // "hop-token-2"
		"aG9wLXRva2VuLTM=", // "hop-token-3"
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		body, _ := io.ReadAll(r.Body)

		var reqMap map[string]any
		_ = json.Unmarshal(body, &reqMap)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		getContinuationToken := func() any {
			if tok := reqMap["continuationToken"]; tok != nil {
				return tok
			}
			return reqMap["continuation_token"]
		}

		switch count {
		case 1:
			if tok := getContinuationToken(); tok != nil {
				t.Errorf("hop 1 unexpectedly had continuation token: %v", tok)
			}
			fmt.Fprintf(w, `{
				"candidates": [
					{
						"content": {"role": "model", "parts": [{"text": "Hop 1. "}]},
						"finishReason": "CONTINUATION",
						"continuationToken": "%s"
					}
				],
				"usageMetadata": {"candidatesTokenCount": 10}
			}`, tokens[0])
		case 2:
			if tok := getContinuationToken(); tok != tokens[0] {
				t.Errorf("hop 2 expected token %s, got %v", tokens[0], tok)
			}
			fmt.Fprintf(w, `{
				"candidates": [
					{
						"content": {"role": "model", "parts": [{"text": "Hop 2. "}]},
						"finishReason": "CONTINUATION",
						"continuationToken": "%s"
					}
				],
				"usageMetadata": {"candidatesTokenCount": 15}
			}`, tokens[1])
		case 3:
			if tok := getContinuationToken(); tok != tokens[1] {
				t.Errorf("hop 3 expected token %s, got %v", tokens[1], tok)
			}
			fmt.Fprintf(w, `{
				"candidates": [
					{
						"content": {"role": "model", "parts": [{"text": "Hop 3. "}]},
						"finishReason": "CONTINUATION",
						"continuationToken": "%s"
					}
				],
				"usageMetadata": {"candidatesTokenCount": 20}
			}`, tokens[2])
		case 4:
			if tok := getContinuationToken(); tok != tokens[2] {
				t.Errorf("hop 4 expected token %s, got %v", tokens[2], tok)
			}
			fmt.Fprintln(w, `{
				"candidates": [
					{
						"content": {"role": "model", "parts": [{"text": "Hop 4."}]},
						"finishReason": "STOP"
					}
				],
				"usageMetadata": {"candidatesTokenCount": 5}
			}`)
		default:
			t.Fatalf("unexpected hop request %d", count)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ctx, &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		envVarProvider: func() map[string]string {
			return map[string]string{"GOOGLE_API_KEY": "test-key"}
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-pro", nil, nil)
	if err != nil {
		t.Fatalf("failed to create chat: %v", err)
	}

	resp, err := chat.SendMessage(ctx, Part{Text: "Write a long essay across 4 hops"})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if atomic.LoadInt32(&requestCount) != 4 {
		t.Errorf("expected 4 requests across hops, got %d", atomic.LoadInt32(&requestCount))
	}
	if resp.Text() != "Hop 1. Hop 2. Hop 3. Hop 4." {
		t.Errorf("merged response text = %q, want %q", resp.Text(), "Hop 1. Hop 2. Hop 3. Hop 4.")
	}
	if resp.Candidates[0].FinishReason != FinishReasonStop {
		t.Errorf("finish reason = %v, want STOP", resp.Candidates[0].FinishReason)
	}
	if len(resp.Candidates[0].ContinuationToken) != 0 {
		t.Errorf("continuation token = %v, want empty", resp.Candidates[0].ContinuationToken)
	}
	if resp.UsageMetadata.CandidatesTokenCount != 50 {
		t.Errorf("CandidatesTokenCount = %d, want 50", resp.UsageMetadata.CandidatesTokenCount)
	}

	// Verify history contains merged model turn
	hist := chat.History(true)
	if len(hist) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(hist))
	}
	var modelHistText string
	for _, p := range hist[1].Parts {
		modelHistText += p.Text
	}
	if modelHistText != "Hop 1. Hop 2. Hop 3. Hop 4." {
		t.Errorf("model history text = %q, want %q", modelHistText, "Hop 1. Hop 2. Hop 3. Hop 4.")
	}
}

func TestChat_AutomaticContinuation_FourHopsStream(t *testing.T) {
	ctx := context.Background()

	var requestCount int32
	tokens := []string{
		"c3RyZWFtLXRvay0x", // "stream-tok-1"
		"c3RyZWFtLXRvay0y", // "stream-tok-2"
		"c3RyZWFtLXRvay0z", // "stream-tok-3"
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		switch count {
		case 1:
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 1. "}]}, "finishReason": "CONTINUATION", "continuationToken": "%s"}]}`, tokens[0]))
		case 2:
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 2. "}]}, "finishReason": "CONTINUATION", "continuationToken": "%s"}]}`, tokens[1]))
		case 3:
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 3. "}]}, "finishReason": "CONTINUATION", "continuationToken": "%s"}]}`, tokens[2]))
		case 4:
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 4."}]}, "finishReason": "STOP"}]}`)
		default:
			t.Fatalf("unexpected stream hop request %d", count)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ctx, &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		envVarProvider: func() map[string]string {
			return map[string]string{"GOOGLE_API_KEY": "test-key"}
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-pro", nil, nil)
	if err != nil {
		t.Fatalf("failed to create chat: %v", err)
	}

	var text string
	for chunk, err := range chat.SendMessageStream(ctx, Part{Text: "Stream across 4 hops"}) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if chunk != nil {
			text += chunk.Text()
		}
	}

	if atomic.LoadInt32(&requestCount) != 4 {
		t.Errorf("expected 4 stream requests across hops, got %d", atomic.LoadInt32(&requestCount))
	}
	if text != "Stream 1. Stream 2. Stream 3. Stream 4." {
		t.Errorf("accumulated stream text = %q, want %q", text, "Stream 1. Stream 2. Stream 3. Stream 4.")
	}

	// Verify history has user turn + 4 model chunks
	hist := chat.History(true)
	if len(hist) != 5 {
		t.Fatalf("expected 5 history items (user + 4 chunks), got %d", len(hist))
	}
}

