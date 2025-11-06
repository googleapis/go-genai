// Copyright 2024 Google LLC
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
	"reflect"
	"strings"
	"testing"
)

func createGenerateContentResponse(candidates []*Candidate) *GenerateContentResponse {
	return &GenerateContentResponse{
		Candidates: candidates,
	}
}

func TestText(t *testing.T) {
	tests := []struct {
		name          string
		response      *GenerateContentResponse
		expectedText  string
		expectedError error
	}{
		{
			name:         "Empty Candidates",
			response:     createGenerateContentResponse([]*Candidate{}),
			expectedText: "",
		},
		{
			name: "Multiple Candidates",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{Text: "text1", Thought: false}}}},
				{Content: &Content{Parts: []*Part{{Text: "text2", Thought: false}}}},
			}),
			expectedText: "text1",
		},
		{
			name: "Empty Parts",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{}}},
			}),
			expectedText: "",
		},
		{
			name: "Part With Text",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{Text: "text", Thought: false}}}},
			}),
			expectedText: "text",
		},
		{
			name: "Multiple Parts With Text",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{
					{Text: "text1", Thought: false},
					{Text: "text2", Thought: false},
				}}},
			}),
			expectedText: "text1text2",
		},
		{
			name: "Multiple Parts With Thought",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{
					{Text: "text1", Thought: false},
					{Text: "text2", Thought: true},
				}}},
			}),
			expectedText: "text1",
		},
		{
			name: "Part With InlineData",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{
					{Text: "text1", Thought: false},
					{InlineData: &Blob{}},
				}}},
			}),
			expectedText: "text1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.Text()

			if result != tt.expectedText {
				t.Fatalf("expected text %v, got %v", tt.expectedText, result)
			}
		})
	}
}

func TestFunctionCalls(t *testing.T) {
	tests := []struct {
		name                  string
		response              *GenerateContentResponse
		expectedFunctionCalls []*FunctionCall
	}{
		{
			name:                  "Empty Candidates",
			response:              createGenerateContentResponse([]*Candidate{}),
			expectedFunctionCalls: nil,
		},
		{
			name: "Multiple Candidates",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{FunctionCall: &FunctionCall{Name: "funcCall1", Args: map[string]any{"key1": "val1"}}}}}},
				{Content: &Content{Parts: []*Part{{FunctionCall: &FunctionCall{Name: "funcCall2", Args: map[string]any{"key2": "val2"}}}}}},
			}),
			expectedFunctionCalls: []*FunctionCall{
				{Name: "funcCall1", Args: map[string]any{"key1": "val1"}},
			},
		},
		{
			name: "Empty Parts",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{}}},
			}),
			expectedFunctionCalls: nil,
		},
		{
			name: "Part With FunctionCall",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{FunctionCall: &FunctionCall{Name: "funcCall1", Args: map[string]any{"key1": "val1"}}}}}},
			}),
			expectedFunctionCalls: []*FunctionCall{
				{Name: "funcCall1", Args: map[string]any{"key1": "val1"}},
			},
		},
		{
			name: "Multiple Parts With FunctionCall",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{
					{FunctionCall: &FunctionCall{Name: "funcCall1", Args: map[string]any{"key1": "val1"}}},
					{FunctionCall: &FunctionCall{Name: "funcCall2", Args: map[string]any{"key2": "val2"}}},
				}}},
			}),
			expectedFunctionCalls: []*FunctionCall{
				{Name: "funcCall1", Args: map[string]any{"key1": "val1"}},
				{Name: "funcCall2", Args: map[string]any{"key2": "val2"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.FunctionCalls()

			if !reflect.DeepEqual(result, tt.expectedFunctionCalls) {
				t.Fatalf("expected function calls %v, got %v", tt.expectedFunctionCalls, result)
			}
		})
	}
}

func TestExecutableCode(t *testing.T) {
	tests := []struct {
		name                   string
		response               *GenerateContentResponse
		expectedExecutableCode string
	}{
		{
			name:                   "Empty Candidates",
			response:               createGenerateContentResponse([]*Candidate{}),
			expectedExecutableCode: "",
		},
		{
			name: "Multiple Candidates",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{ExecutableCode: &ExecutableCode{Code: "code1", Language: LanguagePython}}}}},
				{Content: &Content{Parts: []*Part{{ExecutableCode: &ExecutableCode{Code: "code2", Language: LanguagePython}}}}},
			}),
			expectedExecutableCode: "code1",
		},
		{
			name: "Empty Parts",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{}}},
			}),
			expectedExecutableCode: "",
		},
		{
			name: "Part With ExecutableCode",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{ExecutableCode: &ExecutableCode{Code: "code1", Language: LanguagePython}}}}},
			}),
			expectedExecutableCode: "code1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.ExecutableCode()

			if !reflect.DeepEqual(result, tt.expectedExecutableCode) {
				t.Fatalf("expected executable code %v, got %v", tt.expectedExecutableCode, result)
			}
		})
	}
}

func TestCodeExecutionResult(t *testing.T) {
	tests := []struct {
		name                        string
		response                    *GenerateContentResponse
		expectedCodeExecutionResult string
	}{
		{
			name:                        "Empty Candidates",
			response:                    createGenerateContentResponse([]*Candidate{}),
			expectedCodeExecutionResult: "",
		},
		{
			name: "Multiple Candidates",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{CodeExecutionResult: &CodeExecutionResult{Outcome: OutcomeOK, Output: "output1"}}}}},
				{Content: &Content{Parts: []*Part{{CodeExecutionResult: &CodeExecutionResult{Outcome: OutcomeOK, Output: "output2"}}}}},
			}),
			expectedCodeExecutionResult: "output1",
		},
		{
			name: "Empty Parts",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{}}},
			}),
			expectedCodeExecutionResult: "",
		},
		{
			name: "Part With CodeExecutionResult",
			response: createGenerateContentResponse([]*Candidate{
				{Content: &Content{Parts: []*Part{{CodeExecutionResult: &CodeExecutionResult{Outcome: OutcomeOK, Output: "output1"}}}}},
			}),
			expectedCodeExecutionResult: "output1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.CodeExecutionResult()

			if !reflect.DeepEqual(result, tt.expectedCodeExecutionResult) {
				t.Fatalf("expected code execution result %v, got %v", tt.expectedCodeExecutionResult, result)
			}
		})
	}
}

func TestNewPartFromURI(t *testing.T) {
	fileURI := "http://example.com/video.mp4"
	mimeType := "video/mp4"
	expected := &Part{
		FileData: &FileData{
			FileURI:  fileURI,
			MIMEType: mimeType,
		},
	}

	result := NewPartFromURI(fileURI, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromFile(t *testing.T) {
	fileURI := "http://example.com/video.mp4"
	mimeType := "video/mp4"
	file := File{
		URI:      fileURI,
		MIMEType: mimeType,
	}
	expected := &Part{
		FileData: &FileData{
			FileURI:  fileURI,
			MIMEType: mimeType,
		},
	}

	result := NewPartFromFile(file)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromText(t *testing.T) {
	text := "Hello, world!"
	expected := &Part{
		Text: text,
	}

	result := NewPartFromText(text)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	mimeType := "application/octet-stream"
	expected := &Part{
		InlineData: &Blob{
			Data:     data,
			MIMEType: mimeType,
		},
	}

	result := NewPartFromBytes(data, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromFunctionCall(t *testing.T) {
	funcName := "myFunction"
	args := map[string]any{"arg1": "value1"}
	expected := &Part{
		FunctionCall: &FunctionCall{
			Name: "myFunction",
			Args: map[string]any{"arg1": "value1"},
		},
	}

	result := NewPartFromFunctionCall(funcName, args)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromFunctionResponse(t *testing.T) {
	funcName := "myFunction"
	response := map[string]any{"result": "success"}
	expected := &Part{
		FunctionResponse: &FunctionResponse{
			Name:     "myFunction",
			Response: map[string]any{"result": "success"},
		},
	}

	result := NewPartFromFunctionResponse(funcName, response)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromFunctionResponseWithParts(t *testing.T) {
	funcName := "myFunction"
	response := map[string]any{"result": "success"}
	parts := []*FunctionResponsePart{
		{
			InlineData: &FunctionResponseBlob{
				Data:     []byte{0x01, 0x02, 0x03},
				MIMEType: "image/png",
			},
		},
	}
	expected := &Part{
		FunctionResponse: &FunctionResponse{
			Name:     "myFunction",
			Response: map[string]any{"result": "success"},
			Parts:    parts,
		},
	}

	result := NewPartFromFunctionResponseWithParts(funcName, response, parts)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromExecutableCode(t *testing.T) {
	code := "print('Hello, world!')"
	language := LanguagePython
	expected := &Part{
		ExecutableCode: &ExecutableCode{
			Code:     code,
			Language: language,
		},
	}

	result := NewPartFromExecutableCode(code, language)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewPartFromCodeExecutionResult(t *testing.T) {
	outcome := OutcomeOK
	output := "Execution output"
	expected := &Part{
		CodeExecutionResult: &CodeExecutionResult{
			Outcome: outcome,
			Output:  output,
		},
	}

	result := NewPartFromCodeExecutionResult(outcome, output)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewFunctionResponsePartFromBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	mimeType := "application/octet-stream"
	expected := &FunctionResponsePart{
		InlineData: &FunctionResponseBlob{
			Data:     data,
			MIMEType: mimeType,
		},
	}

	result := NewFunctionResponsePartFromBytes(data, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewFunctionResponsePartFromURI(t *testing.T) {
	fileURI := "http://example.com/video.mp4"
	mimeType := "video/mp4"
	expected := &FunctionResponsePart{
		FileData: &FunctionResponseFileData{
			FileURI:  fileURI,
			MIMEType: mimeType,
		},
	}

	result := NewFunctionResponsePartFromURI(fileURI, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewContentFromParts(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	parts := []*Part{
		{Text: "Hello, world!"},
		{Text: "This is a test."},
	}

	for _, tt := range tests {
		expected := &Content{
			Parts: parts,
			Role:  tt.wantRole,
		}
		result := NewContentFromParts(parts, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestNewContentFromText(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	text := "Hello, world!"

	for _, tt := range tests {
		expected := &Content{
			Parts: []*Part{
				{Text: "Hello, world!"},
			},
			Role: tt.wantRole,
		}
		result := NewContentFromText(text, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestNewContentFromBytes(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	data := []byte{0x01, 0x02, 0x03}
	mimeType := "application/octet-stream"

	for _, tt := range tests {
		expected := &Content{
			Parts: []*Part{
				{InlineData: &Blob{Data: data, MIMEType: mimeType}},
			},
			Role: tt.wantRole,
		}
		result := NewContentFromBytes(data, mimeType, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestNewContentFromURI(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	fileURI := "http://example.com/video.mp4"
	mimeType := "video/mp4"

	for _, tt := range tests {
		expected := &Content{
			Parts: []*Part{
				{FileData: &FileData{FileURI: fileURI, MIMEType: mimeType}},
			},
			Role: tt.wantRole,
		}
		result := NewContentFromURI(fileURI, mimeType, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestNewContentFromFunctionResponse(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	funcName := "myFunction"
	response := map[string]any{"result": "success"}

	for _, tt := range tests {
		expected := &Content{
			Parts: []*Part{
				{FunctionResponse: &FunctionResponse{Name: funcName, Response: response}},
			},
			Role: tt.wantRole,
		}
		result := NewContentFromFunctionResponse(funcName, response, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestNewContentFromExecutableCode(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	code := "print('Hello, world!')"
	language := LanguagePython

	for _, tt := range tests {
		expected := &Content{
			Parts: []*Part{
				{ExecutableCode: &ExecutableCode{Code: code, Language: language}},
			},
			Role: tt.wantRole,
		}

		result := NewContentFromExecutableCode(code, language, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestNewContentFromCodeExecutionResult(t *testing.T) {
	tests := []struct {
		role     Role
		wantRole string
	}{
		{role: RoleUser, wantRole: RoleUser},
		{role: RoleModel, wantRole: RoleModel},
		{role: "", wantRole: RoleUser},
	}
	outcome := OutcomeOK
	output := "Execution output"

	for _, tt := range tests {
		expected := &Content{
			Parts: []*Part{
				{CodeExecutionResult: &CodeExecutionResult{Outcome: outcome, Output: output}},
			},
			Role: tt.wantRole,
		}

		result := NewContentFromCodeExecutionResult(outcome, output, tt.role)
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestToGenerationConfig(t *testing.T) {
	// Helper function to create pointer values
	float32Ptr := func(f float32) *float32 { return &f }
	int32Ptr := func(i int32) *int32 { return &i }

	tests := []struct {
		name        string
		config      GenerateContentConfig
		backend     Backend
		wantErr     bool
		errContains string
		validate    func(*testing.T, *GenerationConfig)
	}{
		{
			name: "Basic config with GeminiAPI backend",
			config: GenerateContentConfig{
				Temperature:     float32Ptr(0.7),
				TopP:            float32Ptr(0.9),
				TopK:            float32Ptr(40),
				MaxOutputTokens: 1024,
				CandidateCount:  1,
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.Temperature == nil || *gc.Temperature != 0.7 {
					t.Errorf("expected Temperature 0.7, got %v", gc.Temperature)
				}
				if gc.TopP == nil || *gc.TopP != 0.9 {
					t.Errorf("expected TopP 0.9, got %v", gc.TopP)
				}
				if gc.TopK == nil || *gc.TopK != 40 {
					t.Errorf("expected TopK 40, got %v", gc.TopK)
				}
				if gc.MaxOutputTokens != 1024 {
					t.Errorf("expected MaxOutputTokens 1024, got %d", gc.MaxOutputTokens)
				}
				if gc.CandidateCount != 1 {
					t.Errorf("expected CandidateCount 1, got %d", gc.CandidateCount)
				}
			},
		},
		{
			name: "Basic config with VertexAI backend",
			config: GenerateContentConfig{
				Temperature:     float32Ptr(0.5),
				TopP:            float32Ptr(0.8),
				MaxOutputTokens: 512,
			},
			backend: BackendVertexAI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.Temperature == nil || *gc.Temperature != 0.5 {
					t.Errorf("expected Temperature 0.5, got %v", gc.Temperature)
				}
				if gc.TopP == nil || *gc.TopP != 0.8 {
					t.Errorf("expected TopP 0.8, got %v", gc.TopP)
				}
				if gc.MaxOutputTokens != 512 {
					t.Errorf("expected MaxOutputTokens 512, got %d", gc.MaxOutputTokens)
				}
			},
		},
		{
			name: "Config with response MIME type and schema",
			config: GenerateContentConfig{
				ResponseMIMEType: "application/json",
				ResponseSchema: &Schema{
					Type: TypeObject,
					Properties: map[string]*Schema{
						"name": {Type: TypeString},
						"age":  {Type: TypeInteger},
					},
					Required: []string{"name"},
				},
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.ResponseMIMEType != "application/json" {
					t.Errorf("expected ResponseMIMEType 'application/json', got %s", gc.ResponseMIMEType)
				}
				if gc.ResponseSchema == nil {
					t.Fatal("expected ResponseSchema to be set")
				}
				if gc.ResponseSchema.Type != TypeObject {
					t.Errorf("expected schema type OBJECT, got %v", gc.ResponseSchema.Type)
				}
			},
		},
		{
			name: "Config with stop sequences",
			config: GenerateContentConfig{
				StopSequences: []string{"END", "STOP"},
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if !reflect.DeepEqual(gc.StopSequences, []string{"END", "STOP"}) {
					t.Errorf("expected StopSequences ['END', 'STOP'], got %v", gc.StopSequences)
				}
			},
		},
		{
			name: "Config with presence and frequency penalties",
			config: GenerateContentConfig{
				PresencePenalty:  float32Ptr(0.5),
				FrequencyPenalty: float32Ptr(0.3),
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.PresencePenalty == nil || *gc.PresencePenalty != 0.5 {
					t.Errorf("expected PresencePenalty 0.5, got %v", gc.PresencePenalty)
				}
				if gc.FrequencyPenalty == nil || *gc.FrequencyPenalty != 0.3 {
					t.Errorf("expected FrequencyPenalty 0.3, got %v", gc.FrequencyPenalty)
				}
			},
		},
		{
			name: "Config with seed",
			config: GenerateContentConfig{
				Seed: int32Ptr(42),
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.Seed == nil || *gc.Seed != 42 {
					t.Errorf("expected Seed 42, got %v", gc.Seed)
				}
			},
		},
		{
			name: "Config with response logprobs",
			config: GenerateContentConfig{
				ResponseLogprobs: true,
				Logprobs:         int32Ptr(5),
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if !gc.ResponseLogprobs {
					t.Error("expected ResponseLogprobs to be true")
				}
				if gc.Logprobs == nil || *gc.Logprobs != 5 {
					t.Errorf("expected Logprobs 5, got %v", gc.Logprobs)
				}
			},
		},
		{
			name: "Config with response modalities",
			config: GenerateContentConfig{
				ResponseModalities: []string{"TEXT", "IMAGE"},
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if len(gc.ResponseModalities) != 2 {
					t.Errorf("expected 2 response modalities, got %d", len(gc.ResponseModalities))
				}
			},
		},
		{
			name: "Config with media resolution",
			config: GenerateContentConfig{
				MediaResolution: MediaResolutionHigh,
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.MediaResolution != MediaResolutionHigh {
					t.Errorf("expected MediaResolution HIGH, got %v", gc.MediaResolution)
				}
			},
		},
		{
			name: "Config with thinking config",
			config: GenerateContentConfig{
				ThinkingConfig: &ThinkingConfig{
					IncludeThoughts: true,
					ThinkingBudget:  int32Ptr(1000),
				},
			},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc.ThinkingConfig == nil {
					t.Fatal("expected ThinkingConfig to be set")
				}
				if !gc.ThinkingConfig.IncludeThoughts {
					t.Error("expected IncludeThoughts to be true")
				}
				if gc.ThinkingConfig.ThinkingBudget == nil || *gc.ThinkingConfig.ThinkingBudget != 1000 {
					t.Errorf("expected ThinkingBudget 1000, got %v", gc.ThinkingConfig.ThinkingBudget)
				}
			},
		},
		{
			name:    "Empty config",
			config:  GenerateContentConfig{},
			backend: BackendGeminiAPI,
			wantErr: false,
			validate: func(t *testing.T, gc *GenerationConfig) {
				if gc == nil {
					t.Fatal("expected non-nil GenerationConfig")
				}
			},
		},
		{
			name:        "Unsupported backend",
			config:      GenerateContentConfig{},
			backend:     Backend(999),
			wantErr:     true,
			errContains: "Unsupported backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.ToGenerationConfig(tt.backend)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" {
					errStr := err.Error()
					if !strings.Contains(errStr, tt.errContains) {
						t.Errorf("expected error containing '%s', got '%s'", tt.errContains, errStr)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestToGenerationConfig_WithSafetySettings(t *testing.T) {
	config := GenerateContentConfig{
		SafetySettings: []*SafetySetting{
			{
				Category:  HarmCategoryHarassment,
				Threshold: HarmBlockThresholdBlockMediumAndAbove,
			},
			{
				Category:  HarmCategoryHateSpeech,
				Threshold: HarmBlockThresholdBlockOnlyHigh,
			},
		},
	}

	_, err := config.ToGenerationConfig(BackendGeminiAPI)
	if err == nil {
		t.Fatal("expected error for unsupported SafetySettings field, got none")
	}

	// The error should indicate unsupported conversion
	errStr := err.Error()
	if len(errStr) == 0 {
		t.Error("expected non-empty error message")
	}
}

func TestToGenerationConfig_WithTools(t *testing.T) {
	config := GenerateContentConfig{
		Tools: []*Tool{
			{
				FunctionDeclarations: []*FunctionDeclaration{
					{
						Name:        "getCurrentWeather",
						Description: "Get the current weather in a given location",
						Parameters: &Schema{
							Type: TypeObject,
							Properties: map[string]*Schema{
								"location": {Type: TypeString},
							},
							Required: []string{"location"},
						},
					},
				},
			},
		},
	}

	_, err := config.ToGenerationConfig(BackendGeminiAPI)
	if err == nil {
		t.Fatal("expected error for unsupported Tools field, got none")
	}

	// The error should indicate unsupported conversion
	errStr := err.Error()
	if len(errStr) == 0 {
		t.Error("expected non-empty error message")
	}
}
