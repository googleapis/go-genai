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
			want:         true,
		},
		{
			name:         "empty",
			finishReason: "",
			want:         true,
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
			PromptTokenCount:        10,
			CachedContentTokenCount: 4,
			CandidatesTokenCount:    15,
			TotalTokenCount:         25,
			PromptTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 10},
			},
			CacheTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 4},
			},
			CandidatesTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 15},
			},
			TrafficType: TrafficType("DEFAULT"),
		},
	}

	hop2 := &GenerateContentResponse{
		UsageMetadata: &GenerateContentResponseUsageMetadata{
			PromptTokenCount:        10,
			CachedContentTokenCount: 4,
			CandidatesTokenCount:    20,
			TotalTokenCount:         30,
			PromptTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 10},
			},
			CacheTokensDetails: []*ModalityTokenCount{
				{Modality: MediaModalityText, TokenCount: 4},
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
	if um.CachedContentTokenCount != 8 {
		t.Errorf("CachedContentTokenCount = %d, want 8", um.CachedContentTokenCount)
	}
	if um.CandidatesTokenCount != 35 {
		t.Errorf("CandidatesTokenCount = %d, want 35", um.CandidatesTokenCount)
	}
	if um.TotalTokenCount != 55 {
		t.Errorf("TotalTokenCount = %d, want 55", um.TotalTokenCount)
	}
	if merged.InitialPromptTokenCount() != 10 {
		t.Errorf("merged.InitialPromptTokenCount() = %d, want 10", merged.InitialPromptTokenCount())
	}
	if um.InitialPromptTokenCount() != 10 {
		t.Errorf("um.InitialPromptTokenCount() = %d, want 10", um.InitialPromptTokenCount())
	}

	// Check PromptTokensDetails and CacheTokensDetails modality summing
	if len(um.PromptTokensDetails) != 1 || um.PromptTokensDetails[0].Modality != MediaModalityText || um.PromptTokensDetails[0].TokenCount != 20 {
		t.Errorf("PromptTokensDetails = %+v, want [Text: 20]", um.PromptTokensDetails)
	}
	if len(um.CacheTokensDetails) != 1 || um.CacheTokensDetails[0].Modality != MediaModalityText || um.CacheTokensDetails[0].TokenCount != 8 {
		t.Errorf("CacheTokensDetails = %+v, want [Text: 8]", um.CacheTokensDetails)
	}

	// Check modality breakdown grouping and summing for CandidatesTokensDetails
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

func TestMergeContinuationResponses_EmptyCandidatesHopClearsFinishReasonAndPreservesPromptFeedback(t *testing.T) {
	hop1 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				Content: &Content{
					Role:  RoleModel,
					Parts: []*Part{{Text: "Partial output before block."}},
				},
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("tok-hop-1"),
				FinishMessage:     "Continuing",
			},
		},
		UsageMetadata: &GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 100,
			TotalTokenCount:      110,
		},
	}

	for _, emptyCandidates := range [][]*Candidate{nil, {}} {
		hop2 := &GenerateContentResponse{
			Candidates: emptyCandidates,
			PromptFeedback: &GenerateContentResponsePromptFeedback{
				BlockReason:        BlockedReasonOther,
				BlockReasonMessage: "Blocked on continuation hop",
			},
			UsageMetadata: &GenerateContentResponseUsageMetadata{
				PromptTokenCount: 110,
				TotalTokenCount:  110,
			},
		}

		merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2})
		if err != nil {
			t.Fatalf("unexpected merge error: %v", err)
		}
		if merged.Text() != "Partial output before block." {
			t.Errorf("merged.Text() = %q, want %q", merged.Text(), "Partial output before block.")
		}
		if len(merged.Candidates) != 1 {
			t.Fatalf("len(merged.Candidates) = %d, want 1", len(merged.Candidates))
		}
		cand := merged.Candidates[0]
		if cand.FinishReason != "" {
			t.Errorf("FinishReason = %q, want empty", cand.FinishReason)
		}
		if len(cand.ContinuationToken) != 0 {
			t.Errorf("ContinuationToken = %v, want nil/empty", cand.ContinuationToken)
		}
		if cand.FinishMessage != "" {
			t.Errorf("FinishMessage = %q, want empty", cand.FinishMessage)
		}
		if merged.PromptFeedback == nil || merged.PromptFeedback.BlockReason != BlockedReasonOther {
			t.Errorf("PromptFeedback = %+v, want BlockReasonOther", merged.PromptFeedback)
		}
		if merged.PromptFeedback.BlockReasonMessage != "Blocked on continuation hop" {
			t.Errorf("BlockReasonMessage = %q, want %q", merged.PromptFeedback.BlockReasonMessage, "Blocked on continuation hop")
		}
		if merged.UsageMetadata == nil || merged.UsageMetadata.PromptTokenCount != 120 {
			t.Errorf("UsageMetadata = %+v, want PromptTokenCount 120", merged.UsageMetadata)
		}
		if merged.UsageMetadata.InitialPromptTokenCount() != 10 {
			t.Errorf("InitialPromptTokenCount() = %d, want 10", merged.UsageMetadata.InitialPromptTokenCount())
		}
	}
}

func TestMergeContinuationResponses_GroundingSupportsShiftChunkAndPartIndices(t *testing.T) {
	hop1 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				Content: &Content{
					Role:  RoleModel,
					Parts: []*Part{{Text: "Thought...", Thought: true}, {Text: "Hello "}},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{Image: &GroundingChunkImage{ImageURI: "https://image-foo.example", Title: "zero"}},
						{Image: &GroundingChunkImage{ImageURI: "https://image-bar.example", Title: "three"}},
					},
					GroundingSupports: []*GroundingSupport{
						{
							GroundingChunkIndices: []int32{0, 1},
							Segment:               &Segment{PartIndex: 1, StartIndex: 0, EndIndex: 5, Text: "Hello"},
						},
						{
							GroundingChunkIndices: []int32{1},
						},
					},
					ImageSearchQueries: []string{"image-query1"},
				},
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("tok-1"),
			},
		},
	}

	hop2 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				Content: &Content{
					Role:  RoleModel,
					Parts: []*Part{{Text: "world"}},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{Web: &GroundingChunkWeb{URI: "https://foo.example", Title: "one"}},
						{Web: &GroundingChunkWeb{URI: "https://bar.example", Title: "two"}},
					},
					GroundingSupports: []*GroundingSupport{
						{
							GroundingChunkIndices: []int32{0, 1},
							Segment:               &Segment{PartIndex: 0, StartIndex: 0, EndIndex: 5, Text: "world"},
						},
						{
							GroundingChunkIndices: []int32{1},
						},
					},
					WebSearchQueries: []string{"web-query1"},
				},
				FinishReason: FinishReasonStop,
			},
		},
	}

	merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2})
	if err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}

	cand := merged.Candidates[0]
	gm := cand.GroundingMetadata
	if gm == nil {
		t.Fatal("expected non-nil GroundingMetadata")
	}
	if len(gm.GroundingChunks) != 4 {
		t.Fatalf("expected 4 GroundingChunks, got %d", len(gm.GroundingChunks))
	}
	if len(gm.GroundingSupports) != 4 {
		t.Fatalf("expected 4 GroundingSupports, got %d", len(gm.GroundingSupports))
	}

	// Hop 1 supports remain at [0, 1] and [1], PartIndex 1
	if got := gm.GroundingSupports[0].GroundingChunkIndices; len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("support[0].GroundingChunkIndices = %v, want [0 1]", got)
	}
	if got := gm.GroundingSupports[0].Segment.PartIndex; got != 1 {
		t.Errorf("support[0].Segment.PartIndex = %d, want 1", got)
	}
	if got := gm.GroundingSupports[1].GroundingChunkIndices; len(got) != 1 || got[0] != 1 {
		t.Errorf("support[1].GroundingChunkIndices = %v, want [1]", got)
	}

	// Hop 2 supports are shifted by +2 for chunks ([2, 3] and [3]) and +2 for parts (PartIndex 2)
	if got := gm.GroundingSupports[2].GroundingChunkIndices; len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("support[2].GroundingChunkIndices = %v, want [2 3]", got)
	}
	if got := gm.GroundingSupports[2].Segment.PartIndex; got != 2 {
		t.Errorf("support[2].Segment.PartIndex = %d, want 2", got)
	}
	if got := cand.Content.Parts[gm.GroundingSupports[2].Segment.PartIndex].Text; got != "world" {
		t.Errorf("support[2] part text = %q, want %q", got, "world")
	}
	if got := gm.GroundingSupports[3].GroundingChunkIndices; len(got) != 1 || got[0] != 3 {
		t.Errorf("support[3].GroundingChunkIndices = %v, want [3]", got)
	}
	if got := gm.GroundingChunks[gm.GroundingSupports[3].GroundingChunkIndices[0]].Web.URI; got != "https://bar.example" {
		t.Errorf("support[3] resolves to %q, want %q", got, "https://bar.example")
	}

	// Verify hop2's original struct was not mutated
	if got := hop2.Candidates[0].GroundingMetadata.GroundingSupports[0].GroundingChunkIndices[0]; got != 0 {
		t.Errorf("hop2 original GroundingChunkIndices[0] was mutated to %d, want 0", got)
	}
	if got := hop2.Candidates[0].GroundingMetadata.GroundingSupports[0].Segment.PartIndex; got != 0 {
		t.Errorf("hop2 original Segment.PartIndex was mutated to %d, want 0", got)
	}
}

func TestMergeContinuationResponses_GroundingSupportsCornerCases(t *testing.T) {
	// Corner cases:
	// 1. 3-hop cumulative shifting
	// 2. Hop 1 has Content.Parts (2 parts) and nil GroundingMetadata; Hop 2 has 1 part, 2 GroundingChunks, and no GroundingSupports; Hop 3 has 1 part, 1 GroundingChunk, and supports with:
	//    - nil Segment (only GroundingChunkIndices shifted by 0+2 = 2)
	//    - empty GroundingChunkIndices (only Segment.PartIndex shifted by 2+1 = 3)
	//    - both GroundingChunkIndices and Segment.PartIndex shifted
	// 3. Multiple candidates (Candidates[0] and Candidates[1]) shifting independently by their own part and chunk counts.
	hop1 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				Index: 0,
				Content: &Content{
					Role:  "model",
					Parts: []*Part{{Text: "c0-h1-p0 "}, {Text: "c0-h1-p1 "}},
				},
				GroundingMetadata: nil,
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("t1"),
			},
			{
				Index: 1,
				Content: &Content{
					Role:  "model",
					Parts: []*Part{{Text: "c1-h1-p0 "}},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{Web: &GroundingChunkWeb{URI: "https://c1-s0.example"}},
					},
				},
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("t1-c1"),
			},
		},
	}

	hop2 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				Index: 0,
				Content: &Content{
					Role:  "model",
					Parts: []*Part{{Text: "c0-h2-p0 "}},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{Web: &GroundingChunkWeb{URI: "https://c0-s0.example"}},
						{Web: &GroundingChunkWeb{URI: "https://c0-s1.example"}},
					},
					GroundingSupports: nil,
				},
				FinishReason:      FinishReasonContinuation,
				ContinuationToken: []byte("t2"),
			},
			{
				Index: 1,
				Content: &Content{
					Role:  "model",
					Parts: []*Part{{Text: "c1-h2-p0 "}},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{Web: &GroundingChunkWeb{URI: "https://c1-s1.example"}},
					},
					GroundingSupports: []*GroundingSupport{
						{
							GroundingChunkIndices: []int32{0},
							Segment:               &Segment{PartIndex: 0, Text: "c1-h2-p0 "},
						},
					},
				},
				FinishReason: FinishReasonStop,
			},
		},
	}

	hop3 := &GenerateContentResponse{
		Candidates: []*Candidate{
			{
				Index: 0,
				Content: &Content{
					Role:  "model",
					Parts: []*Part{{Text: "c0-h3-p0."}},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{Web: &GroundingChunkWeb{URI: "https://c0-s2.example"}},
					},
					GroundingSupports: []*GroundingSupport{
						{
							// nil Segment: only GroundingChunkIndices shifted by +2 (from hop2)
							GroundingChunkIndices: []int32{0},
							Segment:               nil,
						},
						{
							// empty GroundingChunkIndices: only Segment.PartIndex shifted by +3 (2 from hop1 + 1 from hop2)
							GroundingChunkIndices: nil,
							Segment:               &Segment{PartIndex: 0, Text: "c0-h3-p0."},
						},
						{
							// Both shifted: chunk index 0 -> 2, partIndex 0 -> 3
							GroundingChunkIndices: []int32{0},
							Segment:               &Segment{PartIndex: 0, Text: "c0-h3-p0."},
						},
					},
				},
				FinishReason: FinishReasonStop,
			},
		},
	}

	merged, err := mergeContinuationResponses([]*GenerateContentResponse{hop1, hop2, hop3})
	if err != nil {
		t.Fatalf("mergeContinuationResponses failed: %v", err)
	}

	// Candidate 0 checks (3 hops: 2 + 1 + 1 = 4 parts; 0 + 2 + 1 = 3 chunks)
	cand0 := merged.Candidates[0]
	if got := len(cand0.Content.Parts); got != 4 {
		t.Fatalf("len(cand0.Content.Parts) = %d, want 4", got)
	}
	gm0 := cand0.GroundingMetadata
	if got := len(gm0.GroundingChunks); got != 3 {
		t.Fatalf("len(gm0.GroundingChunks) = %d, want 3", got)
	}
	if got := len(gm0.GroundingSupports); got != 3 {
		t.Fatalf("len(gm0.GroundingSupports) = %d, want 3", got)
	}
	// Support 0: nil Segment, chunk index 0 -> 2
	if gm0.GroundingSupports[0].Segment != nil {
		t.Errorf("gm0.GroundingSupports[0].Segment = %v, want nil", gm0.GroundingSupports[0].Segment)
	}
	if got := gm0.GroundingSupports[0].GroundingChunkIndices; len(got) != 1 || got[0] != 2 {
		t.Errorf("gm0.GroundingSupports[0].GroundingChunkIndices = %v, want [2]", got)
	}
	// Support 1: empty GroundingChunkIndices, partIndex 0 -> 3
	if len(gm0.GroundingSupports[1].GroundingChunkIndices) != 0 {
		t.Errorf("gm0.GroundingSupports[1].GroundingChunkIndices = %v, want empty", gm0.GroundingSupports[1].GroundingChunkIndices)
	}
	if got := gm0.GroundingSupports[1].Segment.PartIndex; got != 3 {
		t.Errorf("gm0.GroundingSupports[1].Segment.PartIndex = %d, want 3", got)
	}
	// Support 2: chunk index 2 -> https://c0-s2.example, partIndex 3 -> "c0-h3-p0."
	s2 := gm0.GroundingSupports[2]
	if gotURI := gm0.GroundingChunks[s2.GroundingChunkIndices[0]].Web.URI; gotURI != "https://c0-s2.example" {
		t.Errorf("cand0 support 2 URI = %q, want https://c0-s2.example", gotURI)
	}
	if gotPart := cand0.Content.Parts[s2.Segment.PartIndex].Text; gotPart != "c0-h3-p0." {
		t.Errorf("cand0 support 2 part = %q, want c0-h3-p0.", gotPart)
	}

	// Candidate 1 checks (shifted independently: 1 + 1 = 2 parts; 1 + 1 = 2 chunks)
	cand1 := merged.Candidates[1]
	gm1 := cand1.GroundingMetadata
	if got := gm1.GroundingSupports[0].GroundingChunkIndices; len(got) != 1 || got[0] != 1 {
		t.Errorf("cand1 GroundingChunkIndices = %v, want [1]", got)
	}
	if got := gm1.GroundingSupports[0].Segment.PartIndex; got != 1 {
		t.Errorf("cand1 Segment.PartIndex = %d, want 1", got)
	}
	if gotURI := gm1.GroundingChunks[gm1.GroundingSupports[0].GroundingChunkIndices[0]].Web.URI; gotURI != "https://c1-s1.example" {
		t.Errorf("cand1 support URI = %q, want https://c1-s1.example", gotURI)
	}
	if gotPart := cand1.Content.Parts[gm1.GroundingSupports[0].Segment.PartIndex].Text; gotPart != "c1-h2-p0 " {
		t.Errorf("cand1 support part = %q, want c1-h2-p0 ", gotPart)
	}
}

func TestModelsGenerateContent_MergedGroundingSupportsResolveToCorrectChunkAndPart(t *testing.T) {
	ctx := context.Background()
	var n atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		i := n.Add(1)
		cand := map[string]any{
			"content": map[string]any{
				"role":  "model",
				"parts": []any{map[string]any{"text": fmt.Sprintf("part %d ", i)}},
			},
			"groundingMetadata": map[string]any{
				"groundingChunks": []any{
					map[string]any{"web": map[string]any{"uri": fmt.Sprintf("https://source%d.example", i)}},
				},
				"groundingSupports": []any{
					map[string]any{
						"groundingChunkIndices": []any{0},
						"segment":               map[string]any{"partIndex": 0},
					},
				},
			},
		}
		if i == 1 {
			cand["finishReason"], cand["continuationToken"] = "CONTINUATION", "dDE="
		} else {
			cand["finishReason"] = "STOP"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{cand}})
	}))
	defer ts.Close()

	c, err := NewClient(ctx, &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		envVarProvider: func() map[string]string {
			return map[string]string{"GOOGLE_API_KEY": "test-key"}
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := c.Models.GenerateContent(ctx, "gemini-2.5-flash", Text("hi"), nil)
	if err != nil {
		t.Fatal(err)
	}
	cand := resp.Candidates[0]
	gm := cand.GroundingMetadata
	s := gm.GroundingSupports[len(gm.GroundingSupports)-1]
	gotURI := gm.GroundingChunks[s.GroundingChunkIndices[0]].Web.URI
	if gotURI != "https://source2.example" {
		t.Errorf("support from the second response cites %s, want https://source2.example", gotURI)
	}
	gotPart := cand.Content.Parts[s.Segment.PartIndex].Text
	if gotPart != "part 2 " {
		t.Errorf("support from the second response points to part %q (index %d), want %q", gotPart, s.Segment.PartIndex, "part 2 ")
	}
}

func TestModelsGenerateContentStream_GroundingMetadataAcrossHopsAndMerged(t *testing.T) {
	ctx := context.Background()
	var n atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		i := n.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Chunk 1 of hop i: text part without groundingMetadata
		chunk1 := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"role":  "model",
						"parts": []any{map[string]any{"text": fmt.Sprintf("hop %d intro, ", i)}},
					},
				},
			},
		}
		// Chunk 2 (final chunk of hop i): text part + hop's GroundingMetadata
		cand2 := map[string]any{
			"content": map[string]any{
				"role":  "model",
				"parts": []any{map[string]any{"text": fmt.Sprintf("hop %d grounded. ", i)}},
			},
			"groundingMetadata": map[string]any{
				"groundingChunks": []any{
					map[string]any{"web": map[string]any{"uri": fmt.Sprintf("https://source%d.example", i)}},
				},
				"groundingSupports": []any{
					map[string]any{
						"groundingChunkIndices": []any{0},
						"segment":               map[string]any{"partIndex": 0, "text": fmt.Sprintf("hop %d grounded. ", i)},
					},
				},
			},
		}
		if i == 1 {
			cand2["finishReason"] = "CONTINUATION"
			cand2["continuationToken"] = "dDE="
		} else {
			cand2["finishReason"] = "STOP"
		}
		chunk2 := map[string]any{"candidates": []any{cand2}}

		b1, _ := json.Marshal(chunk1)
		b2, _ := json.Marshal(chunk2)
		fmt.Fprintf(w, "data: %s\n\n", b1)
		fmt.Fprintf(w, "data: %s\n\n", b2)
	}))
	defer ts.Close()

	c, err := NewClient(ctx, &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		envVarProvider: func() map[string]string {
			return map[string]string{"GOOGLE_API_KEY": "test-key"}
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var chunks []*GenerateContentResponse
	for chunk, err := range c.Models.GenerateContentStream(ctx, "gemini-2.5-flash", Text("hi"), nil) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 4 {
		t.Fatalf("len(chunks) = %d, want 4", len(chunks))
	}
	// Verify each yielded terminal hop chunk (chunks[1] and chunks[3]) preserves its per-chunk GroundingMetadata
	for idx, hopNum := range []int{1, 2} {
		chunk := chunks[idx*2+1]
		gm := chunk.Candidates[0].GroundingMetadata
		if gm == nil || len(gm.GroundingSupports) != 1 {
			t.Fatalf("chunk %d missing GroundingSupports: %+v", idx*2+1, gm)
		}
		s := gm.GroundingSupports[0]
		wantURI := fmt.Sprintf("https://source%d.example", hopNum)
		if gotURI := gm.GroundingChunks[s.GroundingChunkIndices[0]].Web.URI; gotURI != wantURI {
			t.Errorf("stream chunk %d support cites %q, want %q", idx*2+1, gotURI, wantURI)
		}
		wantPart := fmt.Sprintf("hop %d grounded. ", hopNum)
		if gotPart := chunk.Candidates[0].Content.Parts[s.Segment.PartIndex].Text; gotPart != wantPart {
			t.Errorf("stream chunk %d support points to part %q, want %q", idx*2+1, gotPart, wantPart)
		}
	}
	// Intermediate CONTINUATION finishReason is cleared on chunks[1], and STOP is preserved on chunks[3]
	if chunks[1].Candidates[0].FinishReason != "" {
		t.Errorf("chunks[1] FinishReason = %q, want empty", chunks[1].Candidates[0].FinishReason)
	}
	if chunks[3].Candidates[0].FinishReason != FinishReasonStop {
		t.Errorf("chunks[3] FinishReason = %q, want STOP", chunks[3].Candidates[0].FinishReason)
	}

	// Verify merging the streamed chunks shifts GroundingSupports across all 4 chunks (parts at indices 1 and 3, chunks at indices 0 and 1)
	merged, err := mergeContinuationResponses(chunks)
	if err != nil {
		t.Fatalf("mergeContinuationResponses failed: %v", err)
	}
	cand := merged.Candidates[0]
	gm := cand.GroundingMetadata
	if len(gm.GroundingSupports) != 2 {
		t.Fatalf("len(merged GroundingSupports) = %d, want 2", len(gm.GroundingSupports))
	}
	// First grounded chunk was chunks[1] (preceded by chunks[0] which had 1 part and 0 groundingChunks)
	s0 := gm.GroundingSupports[0]
	if gotURI := gm.GroundingChunks[s0.GroundingChunkIndices[0]].Web.URI; gotURI != "https://source1.example" {
		t.Errorf("merged support 0 cites %q, want https://source1.example", gotURI)
	}
	if gotPart := cand.Content.Parts[s0.Segment.PartIndex].Text; gotPart != "hop 1 grounded. " {
		t.Errorf("merged support 0 points to part %q (index %d), want %q", gotPart, s0.Segment.PartIndex, "hop 1 grounded. ")
	}
	// Second grounded chunk was chunks[3] (preceded by 3 parts and 1 groundingChunk)
	s1 := gm.GroundingSupports[1]
	if gotURI := gm.GroundingChunks[s1.GroundingChunkIndices[0]].Web.URI; gotURI != "https://source2.example" {
		t.Errorf("merged support 1 cites %q, want https://source2.example", gotURI)
	}
	if gotPart := cand.Content.Parts[s1.Segment.PartIndex].Text; gotPart != "hop 2 grounded. " {
		t.Errorf("merged support 1 points to part %q (index %d), want %q", gotPart, s1.Segment.PartIndex, "hop 2 grounded. ")
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
	var streamChunks []*GenerateContentResponse
	for resp, err := range client.Models.GenerateContentStream(ctx, "gemini-2.5-flash", []*Content{
		NewContentFromText("Stream please", RoleUser),
	}, nil) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if resp != nil {
			accumulatedText += resp.Text()
			streamChunks = append(streamChunks, resp)
		}
	}

	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 stream requests across hops by default, got %d", atomic.LoadInt32(&requestCount))
	}
	if accumulatedText != "Stream part 1, stream part 2." {
		t.Errorf("accumulated stream text = %q, want %q", accumulatedText, "Stream part 1, stream part 2.")
	}
	if len(streamChunks) != 4 {
		t.Fatalf("expected 4 stream chunks, got %d", len(streamChunks))
	}
	for i := 0; i < 3; i++ {
		if streamChunks[i].Candidates[0].FinishReason != "" {
			t.Errorf("chunk %d FinishReason = %q, want empty", i, streamChunks[i].Candidates[0].FinishReason)
		}
	}
	if streamChunks[3].Candidates[0].FinishReason != FinishReasonStop {
		t.Errorf("final chunk FinishReason = %q, want %q", streamChunks[3].Candidates[0].FinishReason, FinishReasonStop)
	}

	// Verify explicit AutomaticContinuation=false does not continue and preserves FinishReasonContinuation
	atomic.StoreInt32(&requestCount, 0)
	var optOutText string
	var optOutChunks []*GenerateContentResponse
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
			optOutChunks = append(optOutChunks, resp)
		}
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 stream request when AutomaticContinuation is false, got %d", atomic.LoadInt32(&requestCount))
	}
	if optOutText != "Stream part 1, " {
		t.Errorf("optOutText = %q, want %q", optOutText, "Stream part 1, ")
	}
	if len(optOutChunks) != 2 {
		t.Fatalf("expected 2 chunks when AutomaticContinuation is false, got %d", len(optOutChunks))
	}
	if optOutChunks[1].Candidates[0].FinishReason != FinishReasonContinuation {
		t.Errorf("optOut final chunk FinishReason = %q, want %q", optOutChunks[1].Candidates[0].FinishReason, FinishReasonContinuation)
	}

	t.Run("resumes from checkpoint continuationToken on mid-stream cutoff or error and clears stale token on STOP", func(t *testing.T) {
		var hopCount int32
		ckptTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&hopCount, 1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			switch count {
			case 1:
				// Hop 1: intermediate checkpoint chunk (no finishReason), then stream cuts off cleanly
				fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Ckpt 1, "}]}, "continuationToken": "Y2twdC0x"}]}`)
			case 2:
				// Hop 2: intermediate checkpoint chunk, then malformed SSE JSON causing mid-stream error
				fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Ckpt 2, "}]}, "continuationToken": "Y2twdC0y"}]}`)
				fmt.Fprintf(w, "data: %s\n\n", `{malformed json`)
			case 3:
				// Hop 3: intermediate checkpoint chunk followed by STOP without token (must not resume again)
				fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Ckpt 3, "}]}, "continuationToken": "c3RhbGU="}]}`)
				fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Done."}]}, "finishReason": "STOP"}]}`)
			default:
				t.Fatalf("unexpected stream hop request count: %d", count)
			}
		}))
		defer ckptTs.Close()

		ckptClient, err := NewClient(ctx, &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ckptTs.URL},
			envVarProvider: func() map[string]string {
				return map[string]string{"GOOGLE_API_KEY": "test-key"}
			},
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		var gotText string
		for resp, err := range ckptClient.Models.GenerateContentStream(ctx, "gemini-2.5-flash", []*Content{
			NewContentFromText("Stream with cutoff and error", RoleUser),
		}, nil) {
			if err != nil {
				t.Fatalf("unexpected stream error: %v", err)
			}
			if resp != nil {
				gotText += resp.Text()
			}
		}

		if atomic.LoadInt32(&hopCount) != 3 {
			t.Errorf("expected 3 stream hop requests, got %d", atomic.LoadInt32(&hopCount))
		}
		if gotText != "Ckpt 1, Ckpt 2, Ckpt 3, Done." {
			t.Errorf("gotText = %q, want %q", gotText, "Ckpt 1, Ckpt 2, Ckpt 3, Done.")
		}
	})
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
		var streamChunks []*GenerateContentResponse
		for chunk, err := range chat.SendMessageStream(ctx, Part{Text: "Stream test"}) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if chunk != nil {
				text += chunk.Text()
				streamChunks = append(streamChunks, chunk)
			}
		}

		if atomic.LoadInt32(&requestCount) != 2 {
			t.Errorf("expected 2 requests, got %d", atomic.LoadInt32(&requestCount))
		}
		if text != "Chat stream 1, chat stream 2." {
			t.Errorf("stream text = %q, want %q", text, "Chat stream 1, chat stream 2.")
		}
		if len(streamChunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(streamChunks))
		}
		if streamChunks[0].Candidates[0].FinishReason != "" {
			t.Errorf("chunk 0 FinishReason = %q, want empty", streamChunks[0].Candidates[0].FinishReason)
		}
		if streamChunks[1].Candidates[0].FinishReason != FinishReasonStop {
			t.Errorf("chunk 1 FinishReason = %q, want %q", streamChunks[1].Candidates[0].FinishReason, FinishReasonStop)
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
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 1. "}]}, "finishReason": "CONTINUATION", "continuationToken": "%s"}], "usageMetadata": {"promptTokenCount": 100, "cachedContentTokenCount": 20, "candidatesTokenCount": 10, "thoughtsTokenCount": 5, "totalTokenCount": 115}}`, tokens[0]))
		case 2:
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 2. "}]}, "finishReason": "CONTINUATION", "continuationToken": "%s"}], "usageMetadata": {"promptTokenCount": 115, "cachedContentTokenCount": 20, "candidatesTokenCount": 15, "thoughtsTokenCount": 10, "totalTokenCount": 140}}`, tokens[1]))
		case 3:
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 3. "}]}, "finishReason": "CONTINUATION", "continuationToken": "%s"}], "usageMetadata": {"promptTokenCount": 140, "cachedContentTokenCount": 20, "candidatesTokenCount": 20, "thoughtsTokenCount": 15, "totalTokenCount": 175}}`, tokens[2]))
		case 4:
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Stream 4."}]}, "finishReason": "STOP"}], "usageMetadata": {"promptTokenCount": 175, "cachedContentTokenCount": 20, "candidatesTokenCount": 5, "thoughtsTokenCount": 2, "totalTokenCount": 182}}`)
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
	var chunks []*GenerateContentResponse
	for chunk, err := range chat.SendMessageStream(ctx, Part{Text: "Stream across 4 hops"}) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if chunk != nil {
			text += chunk.Text()
			chunks = append(chunks, chunk)
		}
	}

	if atomic.LoadInt32(&requestCount) != 4 {
		t.Errorf("expected 4 stream requests across hops, got %d", atomic.LoadInt32(&requestCount))
	}
	if text != "Stream 1. Stream 2. Stream 3. Stream 4." {
		t.Errorf("accumulated stream text = %q, want %q", text, "Stream 1. Stream 2. Stream 3. Stream 4.")
	}

	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}
	for i := 0; i < 3; i++ {
		if chunks[i].Candidates[0].FinishReason != "" {
			t.Errorf("chunk %d FinishReason = %q, want empty", i, chunks[i].Candidates[0].FinishReason)
		}
	}
	if chunks[3].Candidates[0].FinishReason != FinishReasonStop {
		t.Errorf("final chunk FinishReason = %q, want %q", chunks[3].Candidates[0].FinishReason, FinishReasonStop)
	}
	lastChunk := chunks[len(chunks)-1]
	lastUM := lastChunk.UsageMetadata
	if lastUM == nil {
		t.Fatal("expected non-nil UsageMetadata on final chunk")
	}
	if lastUM.PromptTokenCount != 530 {
		t.Errorf("final chunk PromptTokenCount = %d, want 530", lastUM.PromptTokenCount)
	}
	if lastUM.CachedContentTokenCount != 80 {
		t.Errorf("final chunk CachedContentTokenCount = %d, want 80", lastUM.CachedContentTokenCount)
	}
	if lastUM.CandidatesTokenCount != 50 {
		t.Errorf("final chunk CandidatesTokenCount = %d, want 50", lastUM.CandidatesTokenCount)
	}
	if lastUM.ThoughtsTokenCount != 32 {
		t.Errorf("final chunk ThoughtsTokenCount = %d, want 32", lastUM.ThoughtsTokenCount)
	}
	if lastUM.TotalTokenCount != 612 {
		t.Errorf("final chunk TotalTokenCount = %d, want 612", lastUM.TotalTokenCount)
	}
	if lastChunk.InitialPromptTokenCount() != 100 {
		t.Errorf("final chunk InitialPromptTokenCount() = %d, want 100", lastChunk.InitialPromptTokenCount())
	}
	if lastUM.InitialPromptTokenCount() != 100 {
		t.Errorf("final chunk UsageMetadata.InitialPromptTokenCount() = %d, want 100", lastUM.InitialPromptTokenCount())
	}

	// Verify history has user turn + 4 model chunks
	hist := chat.History(true)
	if len(hist) != 5 {
		t.Fatalf("expected 5 history items (user + 4 chunks), got %d", len(hist))
	}
}

func TestGenerateContentStream_UsageMetadataAcrossHops(t *testing.T) {
	ctx := context.Background()

	var requestCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		switch count {
		case 1:
			// Hop 1 emits 2 chunks with within-hop cumulative usage
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Hop1-A "}]}}], "usageMetadata": {"promptTokenCount": 50, "cachedContentTokenCount": 10, "candidatesTokenCount": 5, "thoughtsTokenCount": 2, "totalTokenCount": 57}}`)
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Hop1-B "}]}, "finishReason": "CONTINUATION", "continuationToken": "aG9wLTE="}], "usageMetadata": {"promptTokenCount": 50, "cachedContentTokenCount": 10, "candidatesTokenCount": 12, "thoughtsTokenCount": 4, "totalTokenCount": 66}}`)
		case 2:
			// Hop 2 emits 2 chunks with Hop 2 usage; totals are summed across hops while InitialPromptTokenCount preserves Hop 1's 50
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Hop2-A "}]}}], "usageMetadata": {"promptTokenCount": 66, "cachedContentTokenCount": 10, "candidatesTokenCount": 8, "thoughtsTokenCount": 3, "totalTokenCount": 77}}`)
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Hop2-B."}]}, "finishReason": "STOP"}], "usageMetadata": {"promptTokenCount": 66, "cachedContentTokenCount": 10, "candidatesTokenCount": 18, "thoughtsTokenCount": 6, "totalTokenCount": 90}}`)
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

	var chunks []*GenerateContentResponse
	for chunk, err := range client.Models.GenerateContentStream(ctx, "gemini-2.5-pro", []*Content{
		NewContentFromText("Test usage across hops", RoleUser),
	}, nil) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}

	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks across 2 hops, got %d", len(chunks))
	}
	for i := 0; i < 3; i++ {
		if chunks[i].Candidates[0].FinishReason != "" {
			t.Errorf("chunk %d FinishReason = %q, want empty", i, chunks[i].Candidates[0].FinishReason)
		}
	}
	if chunks[3].Candidates[0].FinishReason != FinishReasonStop {
		t.Errorf("final chunk FinishReason = %q, want %q", chunks[3].Candidates[0].FinishReason, FinishReasonStop)
	}

	// Chunk 0 (Hop 1, chunk 1): prompt=50, cached=10, candidates=5, thoughts=2, total=57, initialPrompt=50
	if um := chunks[0].UsageMetadata; um.PromptTokenCount != 50 || um.CachedContentTokenCount != 10 || um.CandidatesTokenCount != 5 || um.ThoughtsTokenCount != 2 || um.TotalTokenCount != 57 || chunks[0].InitialPromptTokenCount() != 50 {
		t.Errorf("chunk 0 UsageMetadata = %+v, initialPrompt = %d", um, chunks[0].InitialPromptTokenCount())
	}
	// Chunk 1 (Hop 1, chunk 2): prompt=50, cached=10, candidates=12, thoughts=4, total=66, initialPrompt=50
	if um := chunks[1].UsageMetadata; um.PromptTokenCount != 50 || um.CachedContentTokenCount != 10 || um.CandidatesTokenCount != 12 || um.ThoughtsTokenCount != 4 || um.TotalTokenCount != 66 || chunks[1].InitialPromptTokenCount() != 50 {
		t.Errorf("chunk 1 UsageMetadata = %+v, initialPrompt = %d", um, chunks[1].InitialPromptTokenCount())
	}
	// Chunk 2 (Hop 2, chunk 1): prompt=50+66=116, cached=10+10=20, candidates=12+8=20, thoughts=4+3=7, total=66+77=143, initialPrompt=50
	if um := chunks[2].UsageMetadata; um.PromptTokenCount != 116 || um.CachedContentTokenCount != 20 || um.CandidatesTokenCount != 20 || um.ThoughtsTokenCount != 7 || um.TotalTokenCount != 143 || chunks[2].InitialPromptTokenCount() != 50 || um.InitialPromptTokenCount() != 50 {
		t.Errorf("chunk 2 UsageMetadata = %+v, initialPrompt = %d, want prompt=116 cached=20 candidates=20 thoughts=7 total=143 initialPrompt=50", um, chunks[2].InitialPromptTokenCount())
	}
	// Chunk 3 (Hop 2, chunk 2): prompt=50+66=116, cached=10+10=20, candidates=12+18=30, thoughts=4+6=10, total=66+90=156, initialPrompt=50
	if um := chunks[3].UsageMetadata; um.PromptTokenCount != 116 || um.CachedContentTokenCount != 20 || um.CandidatesTokenCount != 30 || um.ThoughtsTokenCount != 10 || um.TotalTokenCount != 156 || chunks[3].InitialPromptTokenCount() != 50 || um.InitialPromptTokenCount() != 50 {
		t.Errorf("chunk 3 UsageMetadata = %+v, initialPrompt = %d, want prompt=116 cached=20 candidates=30 thoughts=10 total=156 initialPrompt=50", um, chunks[3].InitialPromptTokenCount())
	}
}

func TestGenerateContentStream_ContextCancellation_DoesNotResumeFromCheckpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var requestCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if count == 1 {
			// Emit checkpoint chunk
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Pre-cancel "}]}, "continuationToken": "Y2twMS10b2tlbg=="}]}`)
			// Cancel context and abruptly drop connection
			cancel()
			return
		}
		t.Fatalf("unexpected continuation request %d after context cancellation", count)
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

	var chunks []*GenerateContentResponse
	var gotErr error
	for chunk, err := range client.Models.GenerateContentStream(ctx, "gemini-2.5-pro", []*Content{
		NewContentFromText("Test cancellation", RoleUser),
	}, nil) {
		if err != nil {
			gotErr = err
			break
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}

	if gotErr == nil {
		t.Errorf("expected context cancellation error, got nil")
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("requestCount = %d, want 1 (must not attempt continuation on cancelled context)", requestCount)
	}
}

func TestGenerateContentStream_StaleCheckpointTokenDiscardedOnStop(t *testing.T) {
	ctx := context.Background()
	var requestCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if count == 1 {
			// Chunk 1 with checkpoint continuationToken
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Chunk 1 checkpoint "}]}, "continuationToken": "Y2twMS10b2tlbg=="}]}`)
			// Chunk 2 with finishReason STOP
			fmt.Fprintf(w, "data: %s\n\n", `{"candidates": [{"content": {"role": "model", "parts": [{"text": "Chunk 2 stop"}]}, "finishReason": "STOP"}]}`)
			return
		}
		t.Fatalf("unexpected continuation request %d after finishReason STOP", count)
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

	var chunks []*GenerateContentResponse
	for chunk, err := range client.Models.GenerateContentStream(ctx, "gemini-2.5-pro", []*Content{
		NewContentFromText("Test stale token discard", RoleUser),
	}, nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[1].Candidates[0].FinishReason != FinishReasonStop {
		t.Errorf("chunk 1 FinishReason = %q, want STOP", chunks[1].Candidates[0].FinishReason)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("requestCount = %d, want 1", requestCount)
	}
}
