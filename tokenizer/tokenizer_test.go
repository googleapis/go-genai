// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tokenizer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/genai"
)

func TestDownload(t *testing.T) {
	config := tokenizers["gemma2"]
	b, err := downloadModelFile(config.modelURL)
	if err != nil {
		t.Fatal(err)
	}

	if hashString(b) != config.modelHash {
		t.Errorf("gemma model hash doesn't match")
	}
}

func TestLoadModelData(t *testing.T) {
	config := tokenizers["gemma2"]
	// Tests that loadModelData manages to load the model properly, and download
	// a new one as needed.
	checkDataAndErr := func(data []byte, err error) {
		t.Helper()
		if err != nil {
			t.Error(err)
		}
		gotHash := hashString(data)
		if gotHash != config.modelHash {
			t.Errorf("got hash=%v, want=%v", gotHash, config.modelHash)
		}
	}

	data, err := loadModelData(config.modelURL, config.modelHash)
	checkDataAndErr(data, err)

	// The cache should exist now and have the right data, try again.
	data, err = loadModelData(config.modelURL, config.modelHash)
	checkDataAndErr(data, err)

	// Overwrite cache file with wrong data, and try again.
	cacheDir := filepath.Join(os.TempDir(), "vertexai_tokenizer_model")
	cachePath := filepath.Join(cacheDir, hashString([]byte(config.modelURL)))
	_ = os.MkdirAll(cacheDir, 0770)
	_ = os.WriteFile(cachePath, []byte{0, 1, 2, 3}, 0660)
	data, err = loadModelData(config.modelURL, config.modelHash)
	checkDataAndErr(data, err)
}

func TestCreateLocalTokenizer(t *testing.T) {
	// Create a tokenizer successfully with gemma2 model
	_, err := NewLocalTokenizer("gemini-1.5-flash")
	if err != nil {
		t.Error(err)
	}

	// Create a tokenizer successfully with gemma3 model
	_, err = NewLocalTokenizer("gemini-2.5-pro")
	if err != nil {
		t.Error(err)
	}

	// Create a tokenizer with stable version
	_, err = NewLocalTokenizer("gemini-2.0-flash-001")
	if err != nil {
		t.Error(err)
	}

	// Create a tokenizer with gemini-3-pro-preview
	_, err = NewLocalTokenizer("gemini-3-pro-preview")
	if err != nil {
		t.Error(err)
	}

	// Create a tokenizer with an unsupported model
	_, err = NewLocalTokenizer("gemini-0.92")
	if err == nil {
		t.Errorf("got no error, want error")
	}
}

func TestCountTokens(t *testing.T) {
	var tests = []struct {
		contents  []*genai.Content
		wantCount int32
	}{
		{[]*genai.Content{genai.NewContentFromText("hello world", "user")}, 2},
		{[]*genai.Content{genai.NewContentFromText("<table><th></th></table>", "user")}, 4},
		{[]*genai.Content{
			genai.NewContentFromText("hello world", "user"),
			genai.NewContentFromText("<table><th></th></table>", "user"),
		}, 6},
	}

	tok, err := NewLocalTokenizer("gemini-1.5-flash")
	if err != nil {
		t.Error(err)
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := tok.CountTokens(tt.contents, nil)
			if err != nil {
				t.Error(err)
			}
			if got.TotalTokens != tt.wantCount {
				t.Errorf("got %v, want %v", got.TotalTokens, tt.wantCount)
			}
		})
	}
}

func TestCountTokensNonText(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-1.5-flash")
	if err != nil {
		t.Error(err)
	}

	// Create content with non-text parts
	content := &genai.Content{
		Parts: []*genai.Part{
			genai.NewPartFromText("foo"),
			genai.NewPartFromBytes([]byte{0, 1}, "image/png"),
		},
		Role: "user",
	}

	// The new CountTokens method should handle this gracefully by only processing text parts
	got, err := tok.CountTokens([]*genai.Content{content}, nil)
	if err != nil {
		t.Error("unexpected error:", err)
	}
	// Should only count tokens for the text part "foo"
	if got.TotalTokens == 0 {
		t.Error("expected tokens for text part, got 0")
	}
}

func TestCountTokensWithSystemInstruction(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		t.Error(err)
	}

	content := genai.NewContentFromText("hello world", "user")
	systemInstruction := genai.NewContentFromText("you are a helpful assistant", "system")

	config := &genai.CountTokensConfig{
		SystemInstruction: systemInstruction,
	}

	got, err := tok.CountTokens([]*genai.Content{content}, config)
	if err != nil {
		t.Error(err)
	}

	// Should count tokens for both content and system instruction
	if got.TotalTokens <= 2 {
		t.Errorf("expected more than 2 tokens (content + system instruction), got %v", got.TotalTokens)
	}

	// Compare with content only
	gotContentOnly, err := tok.CountTokens([]*genai.Content{content}, nil)
	if err != nil {
		t.Error(err)
	}

	if got.TotalTokens <= gotContentOnly.TotalTokens {
		t.Errorf("expected system instruction to add tokens, got %v for content+system vs %v for content only", got.TotalTokens, gotContentOnly.TotalTokens)
	}
}

func TestCountTokensWithTools(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		t.Error(err)
	}

	content := genai.NewContentFromText("hello world", "user")

	// Create a tool with function declarations
	tool := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters: &genai.Schema{
					Type:        "object",
					Description: "Weather function parameters",
					Properties: map[string]*genai.Schema{
						"location": {
							Type:        "string",
							Description: "The location to get weather for",
						},
					},
					Required: []string{"location"},
				},
			},
		},
	}

	config := &genai.CountTokensConfig{
		Tools: []*genai.Tool{tool},
	}

	got, err := tok.CountTokens([]*genai.Content{content}, config)
	if err != nil {
		t.Error(err)
	}

	// Should count tokens for both content and tools
	if got.TotalTokens <= 2 {
		t.Errorf("expected more than 2 tokens (content + tools), got %v", got.TotalTokens)
	}

	// Compare with content only
	gotContentOnly, err := tok.CountTokens([]*genai.Content{content}, nil)
	if err != nil {
		t.Error(err)
	}

	if got.TotalTokens <= gotContentOnly.TotalTokens {
		t.Errorf("expected tools to add tokens, got %v for content+tools vs %v for content only", got.TotalTokens, gotContentOnly.TotalTokens)
	}
}

func TestCountTokensWithResponseSchema(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		t.Error(err)
	}

	content := genai.NewContentFromText("hello world", "user")

	responseSchema := &genai.Schema{
		Type:        "object",
		Description: "Response schema for structured output",
		Properties: map[string]*genai.Schema{
			"answer": {
				Type:        "string",
				Description: "The answer to the question",
			},
			"confidence": {
				Type:        "number",
				Description: "Confidence score",
			},
		},
		Required: []string{"answer"},
	}

	config := &genai.CountTokensConfig{
		GenerationConfig: &genai.GenerationConfig{
			ResponseSchema: responseSchema,
		},
	}

	got, err := tok.CountTokens([]*genai.Content{content}, config)
	if err != nil {
		t.Error(err)
	}

	// Should count tokens for both content and response schema
	if got.TotalTokens <= 2 {
		t.Errorf("expected more than 2 tokens (content + schema), got %v", got.TotalTokens)
	}

	// Compare with content only
	gotContentOnly, err := tok.CountTokens([]*genai.Content{content}, nil)
	if err != nil {
		t.Error(err)
	}

	if got.TotalTokens <= gotContentOnly.TotalTokens {
		t.Errorf("expected response schema to add tokens, got %v for content+schema vs %v for content only", got.TotalTokens, gotContentOnly.TotalTokens)
	}
}

func TestComputeTokens(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name     string
		contents []*genai.Content
	}{
		{
			name: "single_content_single_part",
			contents: []*genai.Content{
				genai.NewContentFromText("hello world", "user"),
			},
		},
		{
			name: "single_content_multiple_parts",
			contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						genai.NewPartFromText("hello"),
						genai.NewPartFromText("world"),
					},
					Role: "user",
				},
			},
		},
		{
			name: "multiple_contents",
			contents: []*genai.Content{
				genai.NewContentFromText("hello", "user"),
				genai.NewContentFromText("world", "model"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tok.ComputeTokens(tt.contents)
			if err != nil {
				t.Error(err)
			}

			if got.TokensInfo == nil {
				t.Error("expected TokensInfo to be non-nil")
				return
			}

			// Verify that we have token information
			totalTokens := 0
			for _, info := range got.TokensInfo {
				if info.TokenIDs == nil || info.Tokens == nil {
					t.Error("expected TokenIDs and Tokens to be non-nil")
				}
				if len(info.TokenIDs) != len(info.Tokens) {
					t.Errorf("expected TokenIDs and Tokens to have same length, got %v vs %v", len(info.TokenIDs), len(info.Tokens))
				}
				if info.Role == "" {
					t.Error("expected Role to be non-empty")
				}
				totalTokens += len(info.TokenIDs)
			}

			if totalTokens == 0 {
				t.Error("expected some tokens to be computed")
			}
		})
	}
}

func TestComputeTokensWithNonText(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		t.Error(err)
	}

	// Create content with both text and non-text parts
	content := &genai.Content{
		Parts: []*genai.Part{
			genai.NewPartFromText("foo"),
			genai.NewPartFromBytes([]byte{0, 1}, "image/png"),
			genai.NewPartFromText("bar"),
		},
		Role: "user",
	}

	got, err := tok.ComputeTokens([]*genai.Content{content})
	if err != nil {
		t.Error("unexpected error:", err)
	}

	if got.TokensInfo == nil {
		t.Error("expected TokensInfo to be non-nil")
		return
	}

	// Should have tokens for both text parts "foo" and "bar"
	if len(got.TokensInfo) != 2 {
		t.Errorf("expected 2 TokensInfo entries (for 2 text parts), got %v", len(got.TokensInfo))
	}

	totalTokens := 0
	for _, info := range got.TokensInfo {
		totalTokens += len(info.TokenIDs)
	}

	if totalTokens == 0 {
		t.Error("expected some tokens for text parts")
	}
}

func TestComputeTokensEmptyContent(t *testing.T) {
	tok, err := NewLocalTokenizer("gemini-2.5-flash")
	if err != nil {
		t.Error(err)
	}

	// Test with empty content slice
	got, err := tok.ComputeTokens([]*genai.Content{})
	if err != nil {
		t.Error("unexpected error:", err)
	}

	if len(got.TokensInfo) != 0 {
		t.Errorf("expected empty TokensInfo for empty content, got %v entries", len(got.TokensInfo))
	}

	// Test with nil content
	got, err = tok.ComputeTokens([]*genai.Content{nil})
	if err != nil {
		t.Error("unexpected error:", err)
	}

	if len(got.TokensInfo) != 0 {
		t.Errorf("expected empty TokensInfo for nil content, got %v entries", len(got.TokensInfo))
	}
}
