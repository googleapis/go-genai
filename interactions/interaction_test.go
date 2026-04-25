// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package interactions_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"google.golang.org/genai/interactions"
	"google.golang.org/genai/interactions/internal/testutil"
	"google.golang.org/genai/interactions/option"
)

func TestInteractionDelete(t *testing.T) {
	t.Skip("Mock server tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := interactions.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Interactions.Delete(
		context.TODO(),
		"id",
		interactions.DeleteParams{
			APIVersion: "api_version",
		},
	)
	if err != nil {
		var apierr *interactions.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestInteractionCancel(t *testing.T) {
	t.Skip("Mock server tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := interactions.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Interactions.Cancel(
		context.TODO(),
		"id",
		interactions.CancelParams{
			APIVersion: "api_version",
		},
	)
	if err != nil {
		var apierr *interactions.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestInteractionNewAgentWithOptionalParams(t *testing.T) {
	t.Skip("Mock server tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := interactions.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Interactions.NewAgent(context.TODO(), interactions.NewAgentParams{
		APIVersion: "api_version",
		Agent:      "deep-research-pro-preview-12-2025",
		Input: interactions.Input{
			TextContent: &interactions.TextContent{
				Text: "text",
				Annotations: []interactions.Annotation{{
					URLCitation: &interactions.URLCitation{
						EndIndex:   new(0),
						StartIndex: new(0),
						Title:      "title",
						URL:        "url",
					},
				}},
			},
		},
		AgentConfig: &interactions.NewAgentParamsAgentConfig{
			Dynamic: &interactions.DynamicAgentConfig{},
		},
		Background:            new(true),
		PreviousInteractionID: "previous_interaction_id",
		ResponseFormat:        map[string]any{},
		ResponseMimeType:      "response_mime_type",
		ResponseModalities:    []string{"text"},
		ServiceTier:           "flex",
		Store:                 new(true),
		SystemInstruction:     "system_instruction",
		Tools: []interactions.Tool{{
			Function: &interactions.Function{
				Description: "description",
				Name:        "name",
				Parameters:  map[string]any{},
			},
		}},
		WebhookConfig: interactions.WebhookConfig{
			Uris: []string{"string"},
			UserMetadata: map[string]any{
				"foo": "bar",
			},
		},
	})
	if err != nil {
		var apierr *interactions.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestInteractionNewModelWithOptionalParams(t *testing.T) {
	t.Skip("Mock server tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := interactions.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Interactions.NewModel(context.TODO(), interactions.NewModelParams{
		APIVersion: "api_version",
		Input: interactions.Input{
			TextContent: &interactions.TextContent{
				Text: "text",
				Annotations: []interactions.Annotation{{
					URLCitation: &interactions.URLCitation{
						EndIndex:   new(0),
						StartIndex: new(0),
						Title:      "title",
						URL:        "url",
					},
				}},
			},
		},
		Model:      "gemini-2.5-computer-use-preview-10-2025",
		Background: new(true),
		GenerationConfig: interactions.GenerationConfig{
			ImageConfig: interactions.ImageConfig{
				AspectRatio: "1:1",
				ImageSize:   "1K",
			},
			MaxOutputTokens: new(0),
			Seed:            new(0),
			SpeechConfig: []interactions.SpeechConfig{{
				Language: "language",
				Speaker:  "speaker",
				Voice:    "voice",
			}},
			StopSequences:     []string{"string"},
			Temperature:       new(float64(0)),
			ThinkingLevel:     interactions.ThinkingLevelMinimal,
			ThinkingSummaries: "auto",
			ToolChoice: &interactions.GenerationConfigToolChoice{
				ToolChoiceType: interactions.ToolChoiceTypeAuto,
			},
			TopP: new(float64(0)),
		},
		PreviousInteractionID: "previous_interaction_id",
		ResponseFormat:        map[string]any{},
		ResponseMimeType:      "response_mime_type",
		ResponseModalities:    []string{"text"},
		ServiceTier:           "flex",
		Store:                 new(true),
		SystemInstruction:     "system_instruction",
		Tools: []interactions.Tool{{
			Function: &interactions.Function{
				Description: "description",
				Name:        "name",
				Parameters:  map[string]any{},
			},
		}},
		WebhookConfig: interactions.WebhookConfig{
			Uris: []string{"string"},
			UserMetadata: map[string]any{
				"foo": "bar",
			},
		},
	})
	if err != nil {
		var apierr *interactions.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestInteractionGetWithOptionalParams(t *testing.T) {
	t.Skip("Mock server tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := interactions.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Interactions.Get(
		context.TODO(),
		"id",
		interactions.GetParams{
			APIVersion:   "api_version",
			IncludeInput: true,
			LastEventID:  "last_event_id",
		},
	)
	if err != nil {
		var apierr *interactions.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}
