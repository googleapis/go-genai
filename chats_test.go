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
)

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
		chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
			chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
			chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
			chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
			chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, history)
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
			history = chat.History(false)
			if len(history) != 4 {
				t.Errorf("Expected 4 history entries, got %d", len(history))
			}
			if len(history[3].Parts) != 1 || history[3].Parts[0].Text == "" {
				t.Errorf("Expected single text part in latest model response")
			}

			// Curated history is not supported
			history = chat.History(true)
			if history != nil {
				log.Fatal("Curated history should return nil since it is not supported yet")
			}
		})
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
			chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
		chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
		chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)
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
