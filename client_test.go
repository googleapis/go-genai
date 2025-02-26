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
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2/google"
)

func unsetEnvVars(t *testing.T) {
	t.Helper()
	os.Unsetenv("GOOGLE_CLOUD_LOCATION")
	os.Unsetenv("GOOGLE_CLOUD_REGION")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI")
	os.Unsetenv("GOOGLE_API_KEY")
}

// TestNewClient only runs in replay mode.
func TestNewClient(t *testing.T) {
	if *mode != replayMode {
		t.Skip("Skipping env vars tests in env mode")
	}

	ctx := context.Background()
	t.Run("VertexAI", func(t *testing.T) {
		// Needed for account default credential.
		// Usually this file is in ~/.config/gcloud/application_default_credentials.json
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "testdata/credentials.json")
		t.Cleanup(func() { os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS") })

		t.Run("Project Location from config", func(t *testing.T) {
			projectID := "test-project"
			location := "test-location"
			client, err := NewClient(ctx, &ClientConfig{Project: projectID, Location: location, Backend: BackendVertexAI})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Project != projectID {
				t.Errorf("Expected project %q, got %q", projectID, client.clientConfig.Project)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location %q, got %q", location, client.clientConfig.Location)
			}
		})

		t.Run("Missing project", func(t *testing.T) {
			// Unset environment variables
			unsetEnvVars(t)
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI})
			if err == nil {
				t.Errorf("Expected error, got empty")
			}
		})

		t.Run("Missing location", func(t *testing.T) {
			unsetEnvVars(t)
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "test-project"})
			if err == nil {
				t.Errorf("Expected error, got empty")
			}
		})

		t.Run("Credentials is read from passed config", func(t *testing.T) {
			creds := &google.Credentials{}
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Credentials: creds, Project: "test-project", Location: "test-location"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.Models.apiClient.clientConfig.Credentials != creds {
				t.Errorf("Credentials want %#v, got %#v", creds, client.Models.apiClient.clientConfig.Credentials)
			}
		})

		t.Run("API Key from environment ignored when set VertexAI", func(t *testing.T) {
			apiKey := "test-api-key-env"
			os.Setenv("GOOGLE_API_KEY", apiKey)
			t.Cleanup(func() { os.Unsetenv("GOOGLE_API_KEY") })
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "test-project", Location: "test-location"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API ignored, got %q", client.clientConfig.APIKey)
			}
		})

		t.Run("Project from environment", func(t *testing.T) {
			projectID := "test-project-env"
			os.Setenv("GOOGLE_CLOUD_PROJECT", projectID)
			t.Cleanup(func() { os.Unsetenv("GOOGLE_CLOUD_PROJECT") })
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Location: "test-location"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Project != projectID {
				t.Errorf("Expected project %q, got %q", projectID, client.clientConfig.Project)
			}
		})

		t.Run("Location from GOOGLE_CLOUD_REGION environment", func(t *testing.T) {
			location := "test-region-env"
			os.Setenv("GOOGLE_CLOUD_REGION", location)
			t.Cleanup(func() { os.Unsetenv("GOOGLE_CLOUD_REGION") })

			// Unset GOOGLE_CLOUD_LOCATION to ensure GOOGLE_CLOUD_REGION is used
			os.Unsetenv("GOOGLE_CLOUD_LOCATION")

			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Backend: BackendVertexAI})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location %q, got %q", location, client.clientConfig.Location)
			}
		})

		t.Run("Location from GOOGLE_CLOUD_LOCATION environment", func(t *testing.T) {
			location := "test-location-env"
			os.Setenv("GOOGLE_CLOUD_LOCATION", location)
			t.Cleanup(func() { os.Unsetenv("GOOGLE_CLOUD_LOCATION") })
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Backend: BackendVertexAI})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location %q, got %q", location, client.clientConfig.Location)
			}
		})

		t.Run("VertexAI set from environment", func(t *testing.T) {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "true")
			t.Cleanup(func() { os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI") })

			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Location: "test-location"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected location %s, got %s", BackendVertexAI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI false from environment", func(t *testing.T) {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "false")
			t.Cleanup(func() { os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI") })

			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendGeminiAPI {
				t.Errorf("Expected location %s, got %s", BackendGeminiAPI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI from config", func(t *testing.T) {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "false")
			t.Cleanup(func() { os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI") })

			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "test-project", Location: "test-location"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %s, got %s", BackendVertexAI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI is unset from config and environment is false", func(t *testing.T) {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "false")
			t.Cleanup(func() { os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI") })

			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendGeminiAPI {
				t.Errorf("Expected Backend %s, got %s", BackendGeminiAPI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI is unset from config but environment is true", func(t *testing.T) {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "true")
			t.Cleanup(func() { os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI") })

			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, APIKey: "test-api-key"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendGeminiAPI {
				t.Errorf("Expected Backend %s, got %s", BackendGeminiAPI, client.clientConfig.Backend)
			}
		})
	})

	t.Run("GoogleAI", func(t *testing.T) {
		t.Run("API Key from config", func(t *testing.T) {
			apiKey := "test-api-key"
			client, err := NewClient(ctx, &ClientConfig{APIKey: apiKey})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("No api key when using GoogleAI", func(t *testing.T) {
			unsetEnvVars(t)
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI})
			if err == nil {
				t.Errorf("Expected error, got empty")
			}
		})

		t.Run("API Key from environment", func(t *testing.T) {
			apiKey := "test-api-key-env"
			os.Setenv("GOOGLE_API_KEY", apiKey)
			t.Cleanup(func() { os.Unsetenv("GOOGLE_API_KEY") })
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})
	})

	t.Run("Project conflicts with APIKey", func(t *testing.T) {
		_, err := NewClient(ctx, &ClientConfig{Project: "test-project", APIKey: "test-api-key"})
		if err == nil {
			t.Errorf("Expected error, got empty")
		}
	})

	t.Run("Location conflicts with APIKey", func(t *testing.T) {
		_, err := NewClient(ctx, &ClientConfig{Location: "test-location", APIKey: "test-api-key"})
		if err == nil {
			t.Errorf("Expected error, got empty")
		}
	})

	t.Run("Check initialization of Models", func(t *testing.T) {
		client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client.Models == nil {
			t.Error("Expected Models to be initialized, but got nil")
		}
		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(ClientConfig{}),
		}
		if diff := cmp.Diff(*client.Models.apiClient.clientConfig, client.clientConfig, opts...); diff != "" {
			t.Errorf("Models.apiClient.clientConfig mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("HTTPClient is read from passed config", func(t *testing.T) {
		httpClient := &http.Client{}
		client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, APIKey: "test-api-key", HTTPClient: httpClient})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client.Models.apiClient.clientConfig.HTTPClient != httpClient {
			t.Errorf("HTTPClient want %#v, got %#v", httpClient, client.Models.apiClient.clientConfig.HTTPClient)
		}
	})

	t.Run("Pass nil config to NewClient", func(t *testing.T) {
		want := ClientConfig{
			Backend:    BackendGeminiAPI,
			Project:    "test-project-env",
			Location:   "test-location",
			APIKey:     "test-api-key",
			HTTPClient: &http.Client{},
			HTTPOptions: HTTPOptions{
				BaseURL:    "https://generativelanguage.googleapis.com/",
				APIVersion: "v1beta",
			},
		}
		os.Setenv("GOOGLE_CLOUD_PROJECT", want.Project)
		t.Cleanup(func() { os.Unsetenv("GOOGLE_CLOUD_PROJECT") })
		os.Setenv("GOOGLE_CLOUD_LOCATION", want.Location)
		t.Cleanup(func() { os.Unsetenv("GOOGLE_CLOUD_LOCATION") })
		os.Setenv("GOOGLE_API_KEY", want.APIKey)
		t.Cleanup(func() { os.Unsetenv("GOOGLE_API_KEY") })
		os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "0")
		if want.Backend == BackendVertexAI {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "1")
		}
		t.Cleanup(func() { os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI") })

		client, err := NewClient(ctx, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(ClientConfig{}),
		}
		if diff := cmp.Diff(want, *client.Models.apiClient.clientConfig, opts...); diff != "" {
			t.Errorf("Models.apiClient.clientConfig mismatch (-want +got):\n%s", diff)
		}
	})

}

func TestClientConfigHTTPOptions(t *testing.T) {
	tests := []struct {
		name               string
		clientConfig       ClientConfig
		expectedBaseURL    string
		expectedAPIVersion string
	}{
		{
			name: "Default Backend with base URL, API Version",
			clientConfig: ClientConfig{
				HTTPOptions: HTTPOptions{
					APIVersion: "v2",
					BaseURL:    "https://test-base-url.com/",
				},
				APIKey: "test-api-key",
			},
			expectedBaseURL:    "https://test-base-url.com/",
			expectedAPIVersion: "v2",
		},
		{
			name: "Google AI Backend with base URL, API Version",
			clientConfig: ClientConfig{
				Backend: BackendGeminiAPI,
				HTTPOptions: HTTPOptions{
					APIVersion: "v2",
					BaseURL:    "https://test-base-url.com/",
				},
				APIKey: "test-api-key",
			},
			expectedBaseURL:    "https://test-base-url.com/",
			expectedAPIVersion: "v2",
		},
		{
			name: "Vertex AI Backend with base URL, API Version",
			clientConfig: ClientConfig{
				Backend:  BackendVertexAI,
				Project:  "test-project",
				Location: "us-central1",
				HTTPOptions: HTTPOptions{
					APIVersion: "v2",
					BaseURL:    "https://test-base-url.com/",
				},
				Credentials: &google.Credentials{},
			},
			expectedBaseURL:    "https://test-base-url.com/",
			expectedAPIVersion: "v2",
		},
		{
			name: "Default Backend without API Version",
			clientConfig: ClientConfig{
				HTTPOptions: HTTPOptions{},
				APIKey:      "test-api-key",
			},
			expectedBaseURL:    "https://generativelanguage.googleapis.com/",
			expectedAPIVersion: "v1beta",
		},
		{
			name: "Google AI Backend without API Version",
			clientConfig: ClientConfig{
				HTTPOptions: HTTPOptions{},
				APIKey:      "test-api-key",
				Backend:     BackendGeminiAPI,
			},
			expectedBaseURL:    "https://generativelanguage.googleapis.com/",
			expectedAPIVersion: "v1beta",
		},
		{
			name: "Vertex AI Backend without API Version",
			clientConfig: ClientConfig{
				Backend:     BackendVertexAI,
				Project:     "test-project",
				Location:    "us-central1",
				HTTPOptions: HTTPOptions{},
				Credentials: &google.Credentials{},
			},
			expectedBaseURL:    "https://us-central1-aiplatform.googleapis.com/",
			expectedAPIVersion: "v1beta1",
		},
		{
			name: "Google AI Backend with HTTP Client Timeout and no HTTPOptions",
			clientConfig: ClientConfig{
				Backend:     BackendGeminiAPI,
				HTTPOptions: HTTPOptions{},
				APIKey:      "test-api-key",
				HTTPClient:  &http.Client{Timeout: 5000 * time.Millisecond},
			},
			expectedBaseURL:    "https://generativelanguage.googleapis.com/",
			expectedAPIVersion: "v1beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, &tt.clientConfig)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if client.clientConfig.HTTPOptions.BaseURL != tt.expectedBaseURL {
				t.Errorf("expected baseURL %s, got %s", tt.expectedBaseURL, client.clientConfig.HTTPOptions.BaseURL)
			}

			if client.clientConfig.HTTPOptions.APIVersion != tt.expectedAPIVersion {
				t.Errorf("expected apiVersion %s, got %s", tt.expectedAPIVersion, client.clientConfig.HTTPOptions.APIVersion)
			}
		})
	}
}
