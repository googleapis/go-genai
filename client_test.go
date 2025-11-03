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

	"cloud.google.com/go/auth"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestNewClient only runs in replay mode.
func TestNewClient(t *testing.T) {

	ctx := context.Background()
	t.Run("VertexAI with default credentials", func(t *testing.T) {
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
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, envVarProvider: func() map[string]string { return map[string]string{} }})
			if err == nil {
				t.Errorf("Expected error, got empty")
			}
		})

		t.Run("Credentials is read from passed config", func(t *testing.T) {
			creds := &auth.Credentials{}
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Credentials: creds, Project: "test-project", Location: "test-location"})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.Models.apiClient.clientConfig.Credentials != creds {
				t.Errorf("Credentials want %#v, got %#v", creds, client.Models.apiClient.clientConfig.Credentials)
			}
		})

		t.Run("Credentials and API key are mutually exclusive", func(t *testing.T) {
			creds := &auth.Credentials{}
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Credentials: creds, APIKey: "test-api-key"})
			if err == nil {
				t.Fatalf("Expected error, got empty")
			}
		})

		t.Run("Explicit project and location takes precedence over project and location from environment when set VertexAI", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "constructor-project", Location: "constructor-location",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_CLOUD_PROJECT":  "env-project-id",
						"GOOGLE_CLOUD_LOCATION": "env-location",
						"GOOGLE_API_KEY":        "",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "constructor-project" {
				t.Errorf("Expected project %q, got %q", "constructor-project", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "constructor-location" {
				t.Errorf("Expected location %q, got %q", "constructor-location", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API key to be empty, got %q", client.clientConfig.APIKey)
			}
		})

		t.Run("API key from config when set VertexAI", func(t *testing.T) {
			apiKey := "test-api-key-constructor"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, APIKey: apiKey,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "test-api-key-env",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "" {
				t.Errorf("Expected project to be empty, got %q", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "" {
				t.Errorf("Expected location to be empty, got %q", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("API key from environment when set VertexAI", func(t *testing.T) {
			apiKey := "test-api-key-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": apiKey,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "" {
				t.Errorf("Expected project to be empty, got %q", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "" {
				t.Errorf("Expected location to be empty, got %q", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("Project from environment", func(t *testing.T) {
			projectID := "test-project-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Location: "test-location",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_CLOUD_PROJECT": projectID,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Project != projectID {
				t.Errorf("Expected project %q, got %q", projectID, client.clientConfig.Project)
			}
		})

		t.Run("Location from GOOGLE_CLOUD_REGION environment", func(t *testing.T) {
			location := "test-region-env"
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_CLOUD_REGION": location,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location %q, got %q", location, client.clientConfig.Location)
			}
		})

		t.Run("Location from GOOGLE_CLOUD_LOCATION environment", func(t *testing.T) {
			location := "test-location-env"
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_CLOUD_LOCATION": location,
					}
				}})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location %q, got %q", location, client.clientConfig.Location)
			}
		})

		t.Run("VertexAI set from environment", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Location: "test-location",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_GENAI_USE_VERTEXAI": "true",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected location %s, got %s", BackendVertexAI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI false from environment", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_GENAI_USE_VERTEXAI": "false",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendGeminiAPI {
				t.Errorf("Expected location %s, got %s", BackendGeminiAPI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI from config", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "test-project", Location: "test-location",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_GENAI_USE_VERTEXAI": "false",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %s, got %s", BackendVertexAI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI is unset from config and environment is false", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_GENAI_USE_VERTEXAI": "false",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendGeminiAPI {
				t.Errorf("Expected Backend %s, got %s", BackendGeminiAPI, client.clientConfig.Backend)
			}
		})

		t.Run("VertexAI is unset from config but environment is true", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, APIKey: "test-api-key",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_GENAI_USE_VERTEXAI": "true",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendGeminiAPI {
				t.Errorf("Expected Backend %s, got %s", BackendGeminiAPI, client.clientConfig.Backend)
			}
		})

		t.Run("API key from constructor takes precedence over proj/location from environment", func(t *testing.T) {
			// Vertex AI API key combo 1
			apiKey := "vertexai-api-key"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, APIKey: apiKey,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY":        "",
						"GOOGLE_CLOUD_PROJECT":  "test-project-env",
						"GOOGLE_CLOUD_LOCATION": "test-location-env",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			// Explicit API key takes precedence over implicit project/location.
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "" {
				t.Errorf("Expected project to be empty, got %q", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "" {
				t.Errorf("Expected location to be empty, got %q", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
			expectedBaseURL := "https://aiplatform.googleapis.com/"
			if client.clientConfig.HTTPOptions.BaseURL != expectedBaseURL {
				t.Errorf("Expected base URL to be %q, got %q", expectedBaseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Proj/location from constructor takes precedence over API key from environment", func(t *testing.T) {
			// Vertex AI API key combo 2
			project := "test-project"
			location := "test-location"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: project, Location: location,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY":        "vertexai-api-key-env",
						"GOOGLE_CLOUD_PROJECT":  "",
						"GOOGLE_CLOUD_LOCATION": "",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			// Explicit project/location takes precedence over implicit API key.
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != project {
				t.Errorf("Expected project to be %q, got %q", project, client.clientConfig.Project)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location to be %q, got %q", location, client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API key to be empty, got %q", client.clientConfig.APIKey)
			}
			expectedBaseURL := "https://test-location-aiplatform.googleapis.com/"
			if client.clientConfig.HTTPOptions.BaseURL != expectedBaseURL {
				t.Errorf("Expected base URL to be %q, got %q", expectedBaseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Proj/location from environment takes precedence over API key from environment", func(t *testing.T) {
			// Vertex AI API key combo 3
			project := "test-project-env"
			location := "test-location-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY":        "vertexai-api-key-env",
						"GOOGLE_CLOUD_PROJECT":  project,
						"GOOGLE_CLOUD_LOCATION": location,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			// Implicit project/location takes precedence over implicit API key.
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != project {
				t.Errorf("Expected project to be %q, got %q", project, client.clientConfig.Project)
			}
			if client.clientConfig.Location != location {
				t.Errorf("Expected location to be %q, got %q", location, client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API key to be empty, got %q", client.clientConfig.APIKey)
			}
			expectedBaseURL := "https://test-location-env-aiplatform.googleapis.com/"
			if client.clientConfig.HTTPOptions.BaseURL != expectedBaseURL {
				t.Errorf("Expected base URL to be %q, got %q", expectedBaseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Base URL from HTTPOptions", func(t *testing.T) {
			baseURL := "https://test-base-url.com/"
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Location: "test-location", Backend: BackendVertexAI,
				HTTPOptions: HTTPOptions{
					BaseURL: baseURL,
				}})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.HTTPOptions.BaseURL != baseURL {
				t.Errorf("Expected base URL %q, got %q", baseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Base URL from SetDefaultBaseURLs", func(t *testing.T) {
			baseURL := "https://test-base-url.com/"
			SetDefaultBaseURLs(BaseURLParameters{
				VertexURL: baseURL,
			})
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Location: "test-location", Backend: BackendVertexAI})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.HTTPOptions.BaseURL != baseURL {
				t.Errorf("Expected base URL %q, got %q", baseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
			SetDefaultBaseURLs(BaseURLParameters{
				GeminiURL: "",
				VertexURL: "",
			})
		})

		t.Run("Base URL from environment", func(t *testing.T) {
			baseURL := "https://test-base-url.com/"
			client, err := NewClient(ctx, &ClientConfig{Project: "test-project", Location: "test-location", Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_VERTEX_BASE_URL": baseURL,
					}
				}})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.HTTPOptions.BaseURL != baseURL {
				t.Errorf("Expected base URL %q, got %q", baseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Default location to global when only project is provided", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "fake-project-id",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "env-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "fake-project-id" {
				t.Errorf("Expected project %q, got %q", "fake-project-id", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "global" {
				t.Errorf("Expected location %q, got %q", "global", client.clientConfig.Location)
			}
		})

		t.Run("Default location to global when credentials are provided with project but no location", func(t *testing.T) {
			creds := &auth.Credentials{}
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Credentials: creds, Project: "fake-project-id",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "env-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "fake-project-id" {
				t.Errorf("Expected project %q, got %q", "fake-project-id", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "global" {
				t.Errorf("Expected location %q, got %q", "global", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API key to be empty, got %q", client.clientConfig.APIKey)
			}
		})

		t.Run("Default location to global when explicit project takes precedence over env api key", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "explicit-project-id",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "env-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "explicit-project-id" {
				t.Errorf("Expected project %q, got %q", "explicit-project-id", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "global" {
				t.Errorf("Expected location %q, got %q", "global", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API key to be empty, got %q", client.clientConfig.APIKey)
			}
		})

		t.Run("Default location to global when env project takes precedence over env api key", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY":       "env-api-key",
						"GOOGLE_CLOUD_PROJECT": "env-project-id",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "env-project-id" {
				t.Errorf("Expected project %q, got %q", "env-project-id", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "global" {
				t.Errorf("Expected location %q, got %q", "global", client.clientConfig.Location)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API key to be empty, got %q", client.clientConfig.APIKey)
			}
		})

		t.Run("No default location to global when explicit location is set", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "fake-project-id", Location: "us-central1",
				envVarProvider: func() map[string]string {
					return map[string]string{}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "fake-project-id" {
				t.Errorf("Expected project %q, got %q", "fake-project-id", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "us-central1" {
				t.Errorf("Expected location %q, got %q", "us-central1", client.clientConfig.Location)
			}
		})

		t.Run("No default location to global when env location is set", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_CLOUD_PROJECT":  "fake-project-id",
						"GOOGLE_CLOUD_LOCATION": "us-west1",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.Project != "fake-project-id" {
				t.Errorf("Expected project %q, got %q", "fake-project-id", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "us-west1" {
				t.Errorf("Expected location %q, got %q", "us-west1", client.clientConfig.Location)
			}
		})

		t.Run("No default location when using api key only mode", func(t *testing.T) {
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, APIKey: "vertexai-api-key",
				envVarProvider: func() map[string]string {
					return map[string]string{}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.Backend != BackendVertexAI {
				t.Errorf("Expected Backend %q, got %q", BackendVertexAI, client.clientConfig.Backend)
			}
			if client.clientConfig.APIKey != "vertexai-api-key" {
				t.Errorf("Expected API key %q, got %q", "vertexai-api-key", client.clientConfig.APIKey)
			}
			if client.clientConfig.Project != "" {
				t.Errorf("Expected project to be empty, got %q", client.clientConfig.Project)
			}
			if client.clientConfig.Location != "" {
				t.Errorf("Expected location to be empty, got %q", client.clientConfig.Location)
			}
		})

		t.Run("Credentials empty when providing http client", func(t *testing.T) {
			cc := &ClientConfig{Backend: BackendVertexAI, HTTPClient: &http.Client{}, Project: "test-project", Location: "test-location"}
			// Because the above http.Client doesn't handle credentials, we call UseDefaultCredentials()
			// so that the http client will have authorization headers in the requests.
			err := cc.UseDefaultCredentials()
			if err != nil {
				t.Fatalf("Expected no error, got error %v", err)
			}
			_, err = NewClient(ctx, cc)
			if err != nil {
				t.Fatalf("Expected no error, got error %v", err)
			}
		})
	})

	t.Run("VertexAI without default credentials", func(t *testing.T) {
		t.Run("Credentials empty when providing http client", func(t *testing.T) {
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, HTTPClient: &http.Client{}, Project: "test-project", Location: "test-location"})
			// Verify client creation should not fail when no default credential file exists.
			if err != nil {
				t.Fatalf("Expected no error, got error %v", err)
			}
		})
	})

	t.Run("GoogleAI", func(t *testing.T) {
		t.Run("API Key from config", func(t *testing.T) {
			apiKey := "test-api-key"
			client, err := NewClient(ctx, &ClientConfig{APIKey: apiKey, envVarProvider: func() map[string]string { return map[string]string{} }})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("API Key from config", func(t *testing.T) {
			apiKey := "test-constructor-api-key"
			client, err := NewClient(ctx, &ClientConfig{APIKey: apiKey,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "test-env-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("No api key when using GoogleAI", func(t *testing.T) {
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, envVarProvider: func() map[string]string { return map[string]string{} }})
			if err == nil {
				t.Errorf("Expected error, got empty")
			}
		})

		t.Run("API Key from GOOGLE_API_KEY only", func(t *testing.T) {
			apiKey := "test-api-key-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": apiKey,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})
		t.Run("API Key from GEMINI_API_KEY only", func(t *testing.T) {
			apiKey := "test-api-key-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GEMINI_API_KEY": apiKey,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("API Key from GEMINI_API_KEY and GOOGLE_API_KEY as empty string", func(t *testing.T) {
			apiKey := "test-api-key-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "",
						"GEMINI_API_KEY": apiKey,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != apiKey {
				t.Errorf("Expected API key %q, got %q", apiKey, client.clientConfig.APIKey)
			}
		})

		t.Run("API Key both GEMINI_API_KEY and GOOGLE_API_KEY", func(t *testing.T) {
			geminiAPIKey := "gemini-api-key-env"
			googleAPIKey := "google-api-key-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": googleAPIKey,
						"GEMINI_API_KEY": geminiAPIKey,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != googleAPIKey {
				t.Errorf("Expected APIggcg key %q, got %q", googleAPIKey, client.clientConfig.APIKey)
			}
		})

		t.Run("Base URL from HTTPOptions", func(t *testing.T) {
			baseURL := "https://test-base-url.com/"
			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key", Backend: BackendGeminiAPI,
				HTTPOptions: HTTPOptions{
					BaseURL: baseURL,
				}})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.HTTPOptions.BaseURL != baseURL {
				t.Errorf("Expected base URL %q, got %q", baseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Base URL from SetDefaultBaseURLs", func(t *testing.T) {
			baseURL := "https://test-base-url.com/"
			SetDefaultBaseURLs(BaseURLParameters{
				GeminiURL: baseURL,
			})
			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key", Backend: BackendGeminiAPI})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.HTTPOptions.BaseURL != baseURL {
				t.Errorf("Expected base URL %q, got %q", baseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
			SetDefaultBaseURLs(BaseURLParameters{
				GeminiURL: "",
				VertexURL: "",
			})
		})

		t.Run("Base URL from environment", func(t *testing.T) {
			baseURL := "https://test-base-url.com/"
			client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key", Backend: BackendGeminiAPI,
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_GEMINI_BASE_URL": baseURL,
					}
				}})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.HTTPOptions.BaseURL != baseURL {
				t.Errorf("Expected base URL %q, got %q", baseURL, client.clientConfig.HTTPOptions.BaseURL)
			}
		})

		t.Run("Credentials empty when providing http client", func(t *testing.T) {
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, HTTPClient: &http.Client{}, APIKey: "test-api-key"})
			if err != nil {
				t.Fatalf("Expected no error, got error %v", err)
			}
		})
	})

	t.Run("Project conflicts with APIKey", func(t *testing.T) {
		_, err := NewClient(ctx, &ClientConfig{Project: "test-project", APIKey: "test-api-key", envVarProvider: func() map[string]string { return map[string]string{} }})
		if err == nil {
			t.Errorf("Expected error, got empty")
		}
	})

	t.Run("Location conflicts with APIKey", func(t *testing.T) {
		_, err := NewClient(ctx, &ClientConfig{Location: "test-location", APIKey: "test-api-key", envVarProvider: func() map[string]string { return map[string]string{} }})
		if err == nil {
			t.Errorf("Expected error, got empty")
		}
	})

	t.Run("Check initialization of Models", func(t *testing.T) {
		client, err := NewClient(ctx, &ClientConfig{APIKey: "test-api-key", envVarProvider: func() map[string]string { return map[string]string{} }})
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
		client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, APIKey: "test-api-key", HTTPClient: httpClient, envVarProvider: func() map[string]string { return map[string]string{} }})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if client.Models.apiClient.clientConfig.HTTPClient != httpClient {
			t.Errorf("HTTPClient want %#v, got %#v", httpClient, client.Models.apiClient.clientConfig.HTTPClient)
		}
	})

	t.Run("Pass empty config to NewClient", func(t *testing.T) {
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
		client, err := NewClient(ctx, &ClientConfig{
			envVarProvider: func() map[string]string {
				return map[string]string{
					"GOOGLE_CLOUD_PROJECT":      want.Project,
					"GOOGLE_CLOUD_LOCATION":     want.Location,
					"GOOGLE_API_KEY":            want.APIKey,
					"GOOGLE_GENAI_USE_VERTEXAI": "0",
				}
			},
		})
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
				envVarProvider: func() map[string]string {
					return map[string]string{}
				},
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
				Credentials: &auth.Credentials{},
			},
			expectedBaseURL:    "https://test-base-url.com/",
			expectedAPIVersion: "v2",
		},
		{
			name: "Default Backend without API Version",
			clientConfig: ClientConfig{
				HTTPOptions: HTTPOptions{},
				APIKey:      "test-api-key",
				envVarProvider: func() map[string]string {
					return map[string]string{}
				},
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
				Credentials: &auth.Credentials{},
			},
			expectedBaseURL:    "https://us-central1-aiplatform.googleapis.com/",
			expectedAPIVersion: "v1beta1",
		},
		{
			name: "Vertex AI Backend with global location",
			clientConfig: ClientConfig{
				Backend:     BackendVertexAI,
				Project:     "test-project",
				Location:    "global",
				HTTPOptions: HTTPOptions{},
				Credentials: &auth.Credentials{},
			},
			expectedBaseURL:    "https://aiplatform.googleapis.com/",
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
