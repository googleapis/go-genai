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

package genai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/auth"
	"github.com/google/go-cmp/cmp"
)

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content *Content
		want    bool
	}{
		{"NilContent", nil, false},
		{"EmptyParts", &Content{Parts: []*Part{}}, false},
		{"NilPart", &Content{Parts: []*Part{nil}}, false},
		{"EmptyTextPart", &Content{Parts: []*Part{&Part{Text: ""}}}, false},
		{"ValidTextPart", &Content{Parts: []*Part{&Part{Text: "hello"}}}, true},
		{"ValidFunctionCall", &Content{Parts: []*Part{&Part{FunctionCall: &FunctionCall{Name: "test"}}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateContent(tt.content); got != tt.want {
				t.Errorf("validateContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *GenerateContentResponse
		want     bool
	}{
		{"NilResponse", nil, false},
		{"EmptyCandidates", &GenerateContentResponse{Candidates: []*Candidate{}}, false},
		{"NilContentInCandidate", &GenerateContentResponse{Candidates: []*Candidate{{Content: nil}}}, false},
		{"InvalidContent", &GenerateContentResponse{Candidates: []*Candidate{{Content: &Content{Parts: []*Part{{Text: ""}}}}}}, false},
		{"ValidContent", &GenerateContentResponse{Candidates: []*Candidate{{Content: &Content{Parts: []*Part{{Text: "hello"}}}}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateResponse(tt.response); got != tt.want {
				t.Errorf("validateResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCuratedHistory(t *testing.T) {
	validUser1 := &Content{Role: RoleUser, Parts: []*Part{{Text: "User 1"}}}
	validModel1 := &Content{Role: RoleModel, Parts: []*Part{{Text: "Model 1"}}}
	validUser2 := &Content{Role: RoleUser, Parts: []*Part{{Text: "User 2"}}}
	invalidModel := &Content{Role: RoleModel, Parts: []*Part{{Text: ""}}}
	validModel2 := &Content{Role: RoleModel, Parts: []*Part{{Text: "Model 2"}}}

	tests := []struct {
		name    string
		input   []*Content
		want    []*Content
		wantErr bool
	}{
		{"EmptyHistory", []*Content{}, []*Content{}, false},
		{"AllValid", []*Content{validUser1, validModel1, validUser2, validModel2}, []*Content{validUser1, validModel1, validUser2, validModel2}, false},
		{"InvalidModelResponse", []*Content{validUser1, invalidModel}, []*Content{}, false},
		{"InvalidTrappedBetweenValids", []*Content{validUser1, validModel1, validUser2, invalidModel}, []*Content{validUser1, validModel1}, false},
		{"ValidAfterInvalid", []*Content{validUser1, invalidModel, validUser2, validModel2}, []*Content{validUser2, validModel2}, false},
		{"StartsWithInvalidModel", []*Content{invalidModel, validUser1, validModel1}, []*Content{validUser1, validModel1}, false},
		{"ConsecutiveUser", []*Content{validUser1, validUser2, validModel1}, []*Content{validUser1, validUser2, validModel1}, false},
		{"ConsecutiveModel", []*Content{validUser1, validModel1, validModel2}, []*Content{validUser1, validModel1, validModel2}, false},
		{"EndsWithUser", []*Content{validUser1, validModel1, validUser2}, []*Content{validUser1, validModel1, validUser2}, false},
		{"InvalidRole", []*Content{{Role: "invalid", Parts: []*Part{{Text: "test"}}}}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractCuratedHistory(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractCuratedHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("extractCuratedHistory() mismatch (-want +got):\n %s", diff)
			}
		})
	}
}

func TestChatsUnitTest(t *testing.T) {
	ctx := context.Background()
	t.Run("TestServer", func(t *testing.T) {
		t.Parallel()
		if isDisabledTest(t) {
			t.Skip("Skip: disabled test")
		}
		// Create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [
								{
									"text": "1 + 2 = 3"
								}
							]
						},
						"finishReason": "STOP",
						"avgLogprobs": -0.6608115907699342
					}
				]
			}
			`)
		}))
		defer ts.Close()

		t.Logf("Using test server: %s", ts.URL)
		cc := &ClientConfig{
			HTTPOptions: HTTPOptions{
				BaseURL: ts.URL,
			},
			HTTPClient:  ts.Client(),
			Credentials: &auth.Credentials{},
		}
		ac := &apiClient{clientConfig: cc}
		client := &Client{
			clientConfig: *cc,
			Chats:        &Chats{apiClient: ac},
		}

		// Create a new Chat.
		var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
		chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
		if err != nil {
			log.Fatal(err)
		}

		part := Part{Text: "What is 1 + 2?"}

		result, err := chat.SendMessage(ctx, part)
		if err != nil {
			log.Fatal(err)
		}
		if result.Text() == "" {
			t.Errorf("Response text should not be empty")
		}

		// Test iterator break logic.
		for range chat.SendMessageStream(ctx, part) {
			break
		}
	})

}

func TestChatsText(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			// Create a new Chat.
			var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
			chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
			if err != nil {
				log.Fatal(err)
			}

			part := Part{Text: "What is 1 + 2?"}

			result, err := chat.SendMessage(ctx, part)
			if err != nil {
				log.Fatal(err)
			}
			if result.Text() == "" {
				t.Errorf("Response text should not be empty")
			}
		})
	}
}

func TestChatsParts(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			// Create a new Chat.
			var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
			chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
			if err != nil {
				log.Fatal(err)
			}

			parts := make([]Part, 2)
			parts[0] = Part{Text: "What is "}
			parts[1] = Part{Text: "1 + 2?"}

			// Send chat message.
			result, err := chat.SendMessage(ctx, parts...)
			if err != nil {
				log.Fatal(err)
			}
			if result.Text() == "" {
				t.Errorf("Response text should not be empty")
			}
		})
	}
}

func TestChats2Messages(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			// Create a new Chat.
			var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
			chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
			if err != nil {
				log.Fatal(err)
			}

			// Send first chat message.
			part := Part{Text: "What is 1 + 2?"}

			result, err := chat.SendMessage(ctx, part)
			if err != nil {
				log.Fatal(err)
			}
			if result.Text() == "" {
				t.Errorf("Response text should not be empty")
			}

			// Send second chat message.
			part = Part{Text: "Add 1 to the previous result."}
			result, err = chat.SendMessage(ctx, part)
			if err != nil {
				log.Fatal(err)
			}
			if result.Text() == "" {
				t.Errorf("Response text should not be empty")
			}
		})
	}
}

func TestChatsHistory(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			// Create a new Chat with handwritten history.
			var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
			history := []*Content{
				&Content{
					Role: "user",
					Parts: []*Part{
						&Part{Text: "What is 1 + 2?"},
					},
				},
				&Content{
					Role: "model",
					Parts: []*Part{
						&Part{Text: "3"},
					},
				},
			}
			chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, history)
			if err != nil {
				log.Fatal(err)
			}

			// Send chat message.
			part := Part{Text: "Add 1 to the previous result."}
			result, err := chat.SendMessage(ctx, part)
			if err != nil {
				log.Fatal(err)
			}
			if result.Text() == "" {
				t.Errorf("Response text should not be empty")
			}

			// Check comprehensive history.
			compHistory := chat.History(false)
			if len(compHistory) != 4 {
				t.Errorf("Expected 4 comprehensive history entries, got %d", len(compHistory))
			}
			if len(compHistory[3].Parts) != 1 || compHistory[3].Parts[0].Text == "" {
				t.Errorf("Expected single text part in latest model response in comprehensive history")
			}

			// Check curated history.
			curatedHistory := chat.History(true)
			if len(curatedHistory) != 4 {
				t.Errorf("Expected 4 curated history entries, got %d", len(curatedHistory))
			}
			if diff := cmp.Diff(compHistory, curatedHistory); diff != "" {
				t.Errorf("Curated history mismatch from comprehensive (-want +got): \n%s", diff)
			}
		})
	}
}

func TestChatsHistoryWithInvalidTurns(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	client, err := NewClient(ctx, &ClientConfig{Backend: backends[0].Backend})
	if err != nil {
		t.Fatal(err)
	}

	validInput := &Content{Role: RoleUser, Parts: []*Part{{Text: "Hello"}}}
	validOutput := &Content{Role: RoleModel, Parts: []*Part{{Text: "Hi there!"}}}
	invalidInput := &Content{Role: RoleUser, Parts: []*Part{{Text: "This will be invalid"}}}
	invalidOutput := &Content{Role: RoleModel, Parts: []*Part{}} // Invalid due to empty parts

	initialHistory := []*Content{validInput, validOutput, invalidInput, invalidOutput}
	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, initialHistory)
	if err != nil {
		t.Fatal(err)
	}

	compHistory := chat.History(false)
	if len(compHistory) != 4 {
		t.Errorf("Expected 4 comprehensive history entries, got %d", len(compHistory))
	}

	curatedHistory := chat.History(true)
	expectedCurated := []*Content{validInput, validOutput}
	if diff := cmp.Diff(expectedCurated, curatedHistory); diff != "" {
		t.Errorf("Curated history mismatch (-want +got): \n%s", diff)
	}
}

func TestChatsSendInvalidResponse(t *testing.T) {
	ctx := context.Background()
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{
			"candidates": [
				{
					"content": {
						"role": "model",
						"parts": []
					},
					"finishReason": "STOP"
				}
			]
		}`)
	}))
	defer ts.Close()

	cc := &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		HTTPClient:  ts.Client(),
		Credentials: &auth.Credentials{},
	}
	ac := &apiClient{clientConfig: cc}
	client := &Client{clientConfig: *cc, Chats: &Chats{apiClient: ac}}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = chat.SendMessage(ctx, Part{Text: "Test"})
	if err != nil {
		t.Fatal(err)
	}

	compHistory := chat.History(false)
	if len(compHistory) != 2 {
		t.Errorf("Expected 2 comprehensive history entries, got %d", len(compHistory))
	}

	curatedHistory := chat.History(true)
	if len(curatedHistory) != 0 {
		t.Errorf("Expected 0 curated history entries, got %d", len(curatedHistory))
	}
}

func TestChatsStreamInvalidResponse(t *testing.T) {
	ctx := context.Background()
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `data:{
			"candidates": [
				{
					"content": { "role": "model", "parts": [{"text": ""}] },
					"finishReason": "STOP"
				}
			]
		}`)
	}))
	defer ts.Close()

	cc := &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL},
		HTTPClient:  ts.Client(),
		Credentials: &auth.Credentials{},
	}
	ac := &apiClient{clientConfig: cc}
	client := &Client{clientConfig: *cc, Chats: &Chats{apiClient: ac}}

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for range chat.SendMessageStream(ctx, Part{Text: "Test"}) {
	}

	compHistory := chat.History(false)
	if len(compHistory) != 2 {
		t.Errorf("Expected 2 comprehensive history entries, got %d, %v", len(compHistory), compHistory)
	}

	curatedHistory := chat.History(true)
	if len(curatedHistory) != 0 {
		t.Errorf("Expected 0 curated history entries, got %d", len(curatedHistory))
	}
}

func TestChatsStream(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			// Create a new Chat.
			var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
			chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
			if err != nil {
				log.Fatal(err)
			}

			// Send first chat message.
			part := Part{Text: "What is 1 + 2?"}

			for _, err := range chat.SendMessageStream(ctx, part) {
				if err != nil {
					log.Fatal(err)
				}
			}
			history := chat.History(false)
			if len(history[0].Parts) != 1 || history[0].Parts[0].Text == "" {
				t.Errorf("Expected single text part in history")
			}

			// Send second chat message.
			part = Part{Text: "Add 1 to the previous result."}
			for _, err := range chat.SendMessageStream(ctx, part) {
				if err != nil {
					log.Fatal(err)
				}
			}

			history = chat.History(false)
			if len(history[0].Parts) != 1 || history[0].Parts[0].Text == "" {
				t.Errorf("Expected single text part in history")
			}
		})
	}
}

func TestChatsStreamUnitTest(t *testing.T) {
	ctx := context.Background()
	t.Run("TestServer", func(t *testing.T) {
		t.Parallel()
		if isDisabledTest(t) {
			t.Skip("Skip: disabled test")
		}
		// Create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `data:{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [
								{
									"text": "1 + "
								}
							]
						},
						"avgLogprobs": -0.6608115907699342
					}
				]
			}

data:{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [
								{
									"text": "2"
								}
							]
						},
						"finishReason": "STOP",
						"avgLogprobs": -0.6608115907699342
					}
				]
			}

data:{
				"candidates": [
					{
						"content": {
							"role": "model",
							"parts": [
								{
									"text": " = 3"
								}
							]
						},
						"finishReason": "STOP",
						"avgLogprobs": -0.6608115907699342
					}
				]
			}
			`)
		}))
		defer ts.Close()

		t.Logf("Using test server: %s", ts.URL)
		cc := &ClientConfig{
			HTTPOptions: HTTPOptions{
				BaseURL: ts.URL,
			},
			HTTPClient:  ts.Client(),
			Credentials: &auth.Credentials{},
		}
		ac := &apiClient{clientConfig: cc}
		client := &Client{
			clientConfig: *cc,
			Chats:        &Chats{apiClient: ac},
		}

		// Create a new Chat.
		var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
		chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
		if err != nil {
			log.Fatal(err)
		}

		part := Part{Text: "What is 1 + 2?"}

		for result, err := range chat.SendMessageStream(ctx, part) {
			if err != nil {
				log.Fatal(err)
			}
			if result.Text() == "" {
				t.Errorf("Response text should not be empty")
			}
		}

		expectedResponses := []string{"1 + ", "2", " = 3"}
		history := chat.History(false)
		expectedUserMessage := "What is 1 + 2?"
		if history[0].Parts[0].Text != expectedUserMessage {
			t.Errorf("Expected history to start with %s, got %s", expectedUserMessage, history[0].Parts[0].Text)
		}
		for i, expectedResponse := range expectedResponses {
			gotResponse := history[i+1].Parts[0].Text
			if gotResponse != expectedResponse {
				t.Errorf("Expected model response to be %s, got %s", expectedResponse, gotResponse)
			}
		}
	})
}

func TestChatsStreamJoinResponsesUnitTest(t *testing.T) {
	ctx := context.Background()
	t.Run("TestServer", func(t *testing.T) {
		t.Parallel()
		if isDisabledTest(t) {
			t.Skip("Skip: disabled test")
		}
		// Create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `data:{
				"candidates": [
					{"content": {"role": "model", "parts": [{"text": "text1_candidate1"}]}},
					{"content": {"role": "model", "parts": [{"text": "text1_candidate2"}]}}
					]
			}

data:{
				"candidates": [
					{"content": {"role": "model", "parts": [{"text": " "}]}},
					{"content": {"role": "model", "parts": [{"text": " "}]}}
					]
			}

data:{
				"candidates": [
					{"content": {"role": "model", "parts": [{"text": "text3_candidate1"}, {"text": " additional text3_candidate1 "}]}},
					{"content": {"role": "model", "parts": [{"text": "text3_candidate2"}, {"text": " additional text3_candidate2 "}]}}
					]
			}

data:{
				"candidates": [
					{"content": {"role": "model", "parts": [{"text": "text4_candidate1"}, {"text": " additional text4_candidate1"}]}},
					{"content": {"role": "model", "parts": [{"text": "text4_candidate2"}, {"text": " additional text4_candidate2"}]}}
					]
			}
			`)
		}))
		defer ts.Close()

		t.Logf("Using test server: %s", ts.URL)
		cc := &ClientConfig{
			HTTPOptions: HTTPOptions{
				BaseURL: ts.URL,
			},
			HTTPClient:  ts.Client(),
			Credentials: &auth.Credentials{},
		}
		ac := &apiClient{clientConfig: cc}
		client := &Client{
			clientConfig: *cc,
			Chats:        &Chats{apiClient: ac},
		}

		// Create a new Chat.
		var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
		chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
		if err != nil {
			log.Fatal(err)
		}

		part := Part{Text: "What is 1 + 2?"}

		for _, err := range chat.SendMessageStream(ctx, part) {
			if err != nil {
				log.Fatal(err)
			}
		}

		var expectedResponses []*Content
		expectedResponses = append(expectedResponses, &Content{Role: "model", Parts: []*Part{&Part{Text: "text1_candidate1"}}})
		expectedResponses = append(expectedResponses, &Content{Role: "model", Parts: []*Part{&Part{Text: " "}}})
		expectedResponses = append(expectedResponses, &Content{Role: "model", Parts: []*Part{&Part{Text: "text3_candidate1"}, &Part{Text: " additional text3_candidate1 "}}})
		expectedResponses = append(expectedResponses, &Content{Role: "model", Parts: []*Part{&Part{Text: "text4_candidate1"}, &Part{Text: " additional text4_candidate1"}}})

		history := chat.History(false)
		expectedUserMessage := "What is 1 + 2?"
		if history[0].Parts[0].Text != expectedUserMessage {
			t.Errorf("Expected history to start with %s, got %s", expectedUserMessage, history[0].Parts[0].Text)
		}
		for i, expectedResponse := range expectedResponses {
			for j, expectedPart := range history[i+1].Parts {
				if expectedPart.Text != expectedResponse.Parts[j].Text {
					t.Errorf("Expected model response to be %s, got %s", expectedResponse.Parts[j].Text, part.Text)
				}
			}
		}

	})
}
