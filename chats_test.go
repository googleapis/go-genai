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

	"golang.org/x/oauth2/google"
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
			Credentials: &google.Credentials{},
		}
		ac := &apiClient{clientConfig: cc}
		client := &Client{
			clientConfig: *cc,
			Chats:        &Chats{apiClient: ac},
		}

		// Create a new Chat.
		var config *GenerateContentConfig = &GenerateContentConfig{Temperature: Ptr[float32](0.5)}
		chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", config, nil)

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

			// Send chat message.
			part := Part{Text: "Add 1 to the previous result."}
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
