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
			result, err := tt.response.Text()

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Fatalf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if result != tt.expectedText {
					t.Fatalf("expected text %v, got %v", tt.expectedText, result)
				}
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

func TestNewPartFromVideoMetadata(t *testing.T) {
	endOffset := "00:01:00"
	startOffset := "00:00:00"
	expected := &Part{
		VideoMetadata: &VideoMetadata{
			EndOffset:   endOffset,
			StartOffset: startOffset,
		},
	}

	result := NewPartFromVideoMetadata(endOffset, startOffset)
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

func TestNewUserContentFromParts(t *testing.T) {
	parts := []*Part{
		{Text: "Hello, world!"},
		{Text: "This is a test."},
	}
	expected := &Content{
		Parts: parts,
		Role:  "user",
	}

	result := NewUserContentFromParts(parts)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewUserContentFromText(t *testing.T) {
	text := "Hello, world!"
	expected := &Content{
		Parts: []*Part{
			{Text: "Hello, world!"},
		},
		Role: "user",
	}

	result := NewUserContentFromText(text)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewUserContentFromBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	mimeType := "application/octet-stream"
	expected := &Content{
		Parts: []*Part{
			{InlineData: &Blob{Data: data, MIMEType: mimeType}},
		},
		Role: "user",
	}

	result := NewUserContentFromBytes(data, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewUserContentFromURI(t *testing.T) {
	fileURI := "http://example.com/video.mp4"
	mimeType := "video/mp4"
	expected := &Content{
		Parts: []*Part{
			{FileData: &FileData{FileURI: fileURI, MIMEType: mimeType}},
		},
		Role: "user",
	}

	result := NewUserContentFromURI(fileURI, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewUserContentFromFunctionResponse(t *testing.T) {
	funcName := "myFunction"
	response := map[string]any{"result": "success"}
	expected := &Content{
		Parts: []*Part{
			{FunctionResponse: &FunctionResponse{Name: funcName, Response: response}},
		},
		Role: "user",
	}

	result := NewUserContentFromFunctionResponse(funcName, response)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewUserContentFromExecutableCode(t *testing.T) {
	code := "print('Hello, world!')"
	language := LanguagePython
	expected := &Content{
		Parts: []*Part{
			{ExecutableCode: &ExecutableCode{Code: code, Language: language}},
		},
		Role: "user",
	}

	result := NewUserContentFromExecutableCode(code, language)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewUserContentFromCodeExecutionResult(t *testing.T) {
	outcome := OutcomeOK
	output := "Execution output"
	expected := &Content{
		Parts: []*Part{
			{CodeExecutionResult: &CodeExecutionResult{Outcome: outcome, Output: output}},
		},
		Role: "user",
	}

	result := NewUserContentFromCodeExecutionResult(outcome, output)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromParts(t *testing.T) {
	parts := []*Part{
		{Text: "Hello, world!"},
		{Text: "This is a test."},
	}
	expected := &Content{
		Parts: parts,
		Role:  "model",
	}

	result := NewModelContentFromParts(parts)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromText(t *testing.T) {
	text := "Hello, world!"
	expected := &Content{
		Parts: []*Part{
			{Text: "Hello, world!"},
		},
		Role: "model",
	}

	result := NewModelContentFromText(text)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	mimeType := "application/octet-stream"
	expected := &Content{
		Parts: []*Part{
			{InlineData: &Blob{Data: data, MIMEType: mimeType}},
		},
		Role: "model",
	}

	result := NewModelContentFromBytes(data, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromURI(t *testing.T) {
	fileURI := "http://example.com/video.mp4"
	mimeType := "video/mp4"
	expected := &Content{
		Parts: []*Part{
			{FileData: &FileData{FileURI: fileURI, MIMEType: mimeType}},
		},
		Role: "model",
	}

	result := NewModelContentFromURI(fileURI, mimeType)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromFunctionCall(t *testing.T) {
	funcName := "myFunction"
	args := map[string]any{"arg1": "value1"}
	expected := &Content{
		Parts: []*Part{
			{FunctionCall: &FunctionCall{Name: funcName, Args: args}},
		},
		Role: "model",
	}

	result := NewModelContentFromFunctionCall(funcName, args)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromExecutableCode(t *testing.T) {
	code := "print('Hello, world!')"
	language := LanguagePython
	expected := &Content{
		Parts: []*Part{
			{ExecutableCode: &ExecutableCode{Code: code, Language: language}},
		},
		Role: "model",
	}

	result := NewModelContentFromExecutableCode(code, language)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestNewModelContentFromCodeExecutionResult(t *testing.T) {
	outcome := OutcomeOK
	output := "Execution output"
	expected := &Content{
		Parts: []*Part{
			{CodeExecutionResult: &CodeExecutionResult{Outcome: outcome, Output: output}},
		},
		Role: "model",
	}

	result := NewModelContentFromCodeExecutionResult(outcome, output)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}
