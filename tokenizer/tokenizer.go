// Copyright 2025 Google LLC
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

// Package tokenizer provides local token counting for Gemini models. This
// tokenizer downloads its model from the web, but otherwise doesn't require
// an API call for every [CountTokens] invocation.
package tokenizer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	sentencepiece "github.com/eliben/go-sentencepiece"
	"google.golang.org/genai"
)

// geminiModelsToLocalTokenizerNames maps model names to their tokenizer types
var geminiModelsToLocalTokenizerNames = map[string]string{
	"gemini-1.0-pro":        "gemma2",
	"gemini-1.5-pro":        "gemma2",
	"gemini-1.5-flash":      "gemma2",
	"gemini-2.5-pro":        "gemma3",
	"gemini-2.5-flash":      "gemma3",
	"gemini-2.5-flash-lite": "gemma3",
	"gemini-2.0-flash":      "gemma3",
	"gemini-2.0-flash-lite": "gemma3",
}

// geminiStableModelsToLocalTokenizerNames maps stable model names to their tokenizer types
var geminiStableModelsToLocalTokenizerNames = map[string]string{
	"gemini-1.0-pro-001":                  "gemma2",
	"gemini-1.0-pro-002":                  "gemma2",
	"gemini-1.5-pro-001":                  "gemma2",
	"gemini-1.5-flash-001":                "gemma2",
	"gemini-1.5-flash-002":                "gemma2",
	"gemini-1.5-pro-002":                  "gemma2",
	"gemini-2.5-pro-preview-06-05":        "gemma3",
	"gemini-2.5-pro-preview-05-06":        "gemma3",
	"gemini-2.5-pro-exp-03-25":            "gemma3",
	"gemini-live-2.5-flash":               "gemma3",
	"gemini-2.5-flash-preview-05-20":      "gemma3",
	"gemini-2.5-flash-preview-04-17":      "gemma3",
	"gemini-2.5-flash-lite-preview-06-17": "gemma3",
	"gemini-2.0-flash-001":                "gemma3",
	"gemini-2.0-flash-lite-001":           "gemma3",
	"gemini-3-pro-preview":                "gemma3",
}

// tokenizerConfig holds the configuration for a tokenizer
type tokenizerConfig struct {
	modelURL  string
	modelHash string
}

// tokenizers maps tokenizer names to their configurations
var tokenizers = map[string]tokenizerConfig{
	"gemma2": {
		modelURL:  "https://raw.githubusercontent.com/google/gemma_pytorch/33b652c465537c6158f9a472ea5700e5e770ad3f/tokenizer/tokenizer.model",
		modelHash: "61a7b147390c64585d6c3543dd6fc636906c9af3865a5548f27f31aee1d4c8e2",
	},
	"gemma3": {
		modelURL:  "https://raw.githubusercontent.com/google/gemma_pytorch/014acb7ac4563a5f77c76d7ff98f31b568c16508/tokenizer/gemma3_cleaned_262144_v2.spiece.model",
		modelHash: "1299c11d7cf632ef3b4e11937501358ada021bbdf7c47638d13c0ee982f2e79c",
	},
}

// getLocalTokenizerName returns the tokenizer name for the given model name
func getLocalTokenizerName(modelName string) (string, error) {
	if tokenizerName, ok := geminiModelsToLocalTokenizerNames[modelName]; ok {
		return tokenizerName, nil
	}
	if tokenizerName, ok := geminiStableModelsToLocalTokenizerNames[modelName]; ok {
		return tokenizerName, nil
	}

	// Build list of supported models for error message
	var supportedModels []string
	for model := range geminiModelsToLocalTokenizerNames {
		supportedModels = append(supportedModels, model)
	}
	for model := range geminiStableModelsToLocalTokenizerNames {
		supportedModels = append(supportedModels, model)
	}

	return "", fmt.Errorf("model %s is not supported. Supported models: %v", modelName, supportedModels)
}

// LocalTokenizer is a local tokenizer for text.
type LocalTokenizer struct {
	processor *sentencepiece.Processor
}

var experimentalWarningLocalTokenizer sync.Once

// NewLocalTokenizer creates a new [LocalTokenizer] from a model name; the model name is the same
// as you would pass to a [genai.Client.GenerativeModel].
func NewLocalTokenizer(modelName string) (*LocalTokenizer, error) {
	experimentalWarningLocalTokenizer.Do(func() {
		fmt.Println("Warning: The SDK's local tokenizer implementation is experimental and may change in the future. It only supports text based tokenization.")
	})

	tokenizerName, err := getLocalTokenizerName(modelName)
	if err != nil {
		return nil, fmt.Errorf("model %s is not supported", modelName)
	}

	config, ok := tokenizers[tokenizerName]
	if !ok {
		return nil, fmt.Errorf("model %s is not supported", modelName)
	}

	data, err := loadModelData(config.modelURL, config.modelHash)
	if err != nil {
		return nil, fmt.Errorf("loading model: %w", err)
	}

	processor, err := sentencepiece.NewProcessor(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating processor: %w", err)
	}

	return &LocalTokenizer{processor: processor}, nil
}

// CountTokens counts tokens in the given contents with optional configuration,
// similar to the Python LocalLocalTokenizer.count_tokens method.
func (tok *LocalTokenizer) CountTokens(contents []*genai.Content, config *genai.CountTokensConfig) (*genai.CountTokensResult, error) {
	textAccumulator := newTextsAccumulator()

	// Add main contents
	textAccumulator.addContents(contents)

	// Process config if provided
	if config != nil {
		if config.Tools != nil {
			textAccumulator.addTools(config.Tools)
		}

		if config.SystemInstruction != nil {
			textAccumulator.addContents([]*genai.Content{config.SystemInstruction})
		}

		if config.GenerationConfig != nil && config.GenerationConfig.ResponseSchema != nil {
			textAccumulator.addSchema(config.GenerationConfig.ResponseSchema)
		}
	}

	// Encode all accumulated texts and sum token counts
	texts := textAccumulator.getTexts()
	totalTokens := 0

	for _, text := range texts {
		if text != "" {
			tokens := tok.processor.Encode(text)
			totalTokens += len(tokens)
		}
	}

	return &genai.CountTokensResult{TotalTokens: int32(totalTokens)}, nil
}

// ComputeTokens computes detailed token information for the given contents,
// similar to the Python LocalLocalTokenizer.compute_tokens method.
func (tok *LocalTokenizer) ComputeTokens(contents []*genai.Content) (*genai.ComputeTokensResult, error) {
	var tokensInfo []*genai.TokensInfo

	for _, content := range contents {
		if content == nil || content.Parts == nil {
			continue
		}

		for _, part := range content.Parts {
			if part.Text != "" {
				tokens := tok.processor.Encode(part.Text)

				tokenIDs := make([]int64, len(tokens))
				tokenBytes := make([][]byte, len(tokens))

				for i, token := range tokens {
					tokenIDs[i] = int64(token.ID)
					tokenBytes[i] = []byte(token.Text)
				}

				role := "user" // Default role
				if content.Role != "" {
					role = content.Role
				}

				tokensInfo = append(tokensInfo, &genai.TokensInfo{
					TokenIDs: tokenIDs,
					Tokens:   tokenBytes,
					Role:     role,
				})
			}
		}
	}

	return &genai.ComputeTokensResult{TokensInfo: tokensInfo}, nil
}

// textsAccumulator accumulates text from Content objects for tokenization.
type textsAccumulator struct {
	texts []string
}

// newTextsAccumulator creates a new textsAccumulator.
func newTextsAccumulator() *textsAccumulator {
	return &textsAccumulator{
		texts: make([]string, 0),
	}
}

// getTexts returns the accumulated texts.
func (ta *textsAccumulator) getTexts() []string {
	return ta.texts
}

// addContents processes multiple Content objects and extracts text.
func (ta *textsAccumulator) addContents(contents []*genai.Content) {
	for _, content := range contents {
		ta.addContent(content)
	}
}

// addContent processes a single Content object and extracts text.
func (ta *textsAccumulator) addContent(content *genai.Content) {
	if content == nil || content.Parts == nil {
		return
	}

	for _, part := range content.Parts {
		if part.Text != "" {
			ta.texts = append(ta.texts, part.Text)
		}
		if part.FunctionCall != nil {
			ta.addFunctionCall(part.FunctionCall)
		}
		if part.FunctionResponse != nil {
			ta.addFunctionResponse(part.FunctionResponse)
		}
	}
}

// addFunctionCall extracts text from a function call.
func (ta *textsAccumulator) addFunctionCall(fc *genai.FunctionCall) {
	if fc == nil {
		return
	}
	if fc.Name != "" {
		ta.texts = append(ta.texts, fc.Name)
	}
	if fc.Args != nil {
		ta.traverseMap(fc.Args)
	}
}

// addFunctionResponse extracts text from a function response.
func (ta *textsAccumulator) addFunctionResponse(fr *genai.FunctionResponse) {
	if fr == nil {
		return
	}
	if fr.Name != "" {
		ta.texts = append(ta.texts, fr.Name)
	}
	if fr.Response != nil {
		ta.traverseMap(fr.Response)
	}
}

// addTools processes tools and extracts text.
func (ta *textsAccumulator) addTools(tools []*genai.Tool) {
	for _, tool := range tools {
		ta.addTool(tool)
	}
}

// addTool processes a single tool and extracts text.
func (ta *textsAccumulator) addTool(tool *genai.Tool) {
	if tool == nil || tool.FunctionDeclarations == nil {
		return
	}

	for _, fd := range tool.FunctionDeclarations {
		if fd.Name != "" {
			ta.texts = append(ta.texts, fd.Name)
		}
		if fd.Description != "" {
			ta.texts = append(ta.texts, fd.Description)
		}
		if fd.Parameters != nil {
			ta.addSchema(fd.Parameters)
		}
	}
}

// addSchema processes a schema and extracts text.
func (ta *textsAccumulator) addSchema(schema *genai.Schema) {
	if schema == nil {
		return
	}

	if schema.Description != "" {
		ta.texts = append(ta.texts, schema.Description)
	}
	if schema.Enum != nil {
		ta.texts = append(ta.texts, schema.Enum...)
	}
	if schema.Required != nil {
		ta.texts = append(ta.texts, schema.Required...)
	}
	if schema.Properties != nil {
		for key, prop := range schema.Properties {
			ta.texts = append(ta.texts, key)
			ta.addSchema(prop)
		}
	}
	if schema.Items != nil {
		ta.addSchema(schema.Items)
	}
}

// traverseMap recursively extracts strings from a map.
func (ta *textsAccumulator) traverseMap(m map[string]any) {
	for key, value := range m {
		ta.texts = append(ta.texts, key)
		ta.traverseAny(value)
	}
}

// traverseAny recursively extracts strings from any value.
func (ta *textsAccumulator) traverseAny(value any) {
	switch v := value.(type) {
	case string:
		ta.texts = append(ta.texts, v)
	case map[string]any:
		ta.traverseMap(v)
	case []any:
		for _, item := range v {
			ta.traverseAny(item)
		}
	}
}

// downloadModelFile downloads a file from the given URL.
func downloadModelFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// hashString computes a hex string of the SHA256 hash of data.
func hashString(data []byte) string {
	hash256 := sha256.Sum256(data)
	return hex.EncodeToString(hash256[:])
}

// loadModelData loads model data from the given URL, using a local file-system
// cache. wantHash is the hash (as returned by [hashString] expected on the
// loaded data.
//
// Caching logic:
//
// Assuming $TEMP_DIR is the temporary directory used by the OS, this function
// uses the file $TEMP_DIR/vertexai_tokenizer_model/$urlhash as a cache, where
// $urlhash is hashString(url).
//
// If this cache file doesn't exist, or the data it contains doesn't match
// wantHash, downloads data from the URL and writes it into the cache. If the
// URL's data doesn't match the hash, an error is returned.
func loadModelData(url string, wantHash string) ([]byte, error) {
	urlhash := hashString([]byte(url))
	cacheDir := filepath.Join(os.TempDir(), "vertexai_tokenizer_model")
	cachePath := filepath.Join(cacheDir, urlhash)

	cacheData, err := os.ReadFile(cachePath)
	if err != nil || hashString(cacheData) != wantHash {
		cacheData, err = downloadModelFile(url)
		if err != nil {
			return nil, fmt.Errorf("loading cache and downloading model: %w", err)
		}

		if hashString(cacheData) != wantHash {
			return nil, fmt.Errorf("downloaded model hash mismatch")
		}

		err = os.MkdirAll(cacheDir, 0770)
		if err != nil {
			return nil, fmt.Errorf("creating cache dir: %w", err)
		}
		err = os.WriteFile(cachePath, cacheData, 0660)
		if err != nil {
			return nil, fmt.Errorf("writing cache file: %w", err)
		}
	}

	return cacheData, nil
}
