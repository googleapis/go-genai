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
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, envVarProvider: func() map[string]string { return map[string]string{} }})
			if err == nil {
				t.Errorf("Expected error, got empty")
			}
		})

		t.Run("Missing location", func(t *testing.T) {
			_, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "test-project", envVarProvider: func() map[string]string { return map[string]string{} }})
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

		t.Run("API Key from environment ignored when set VertexAI", func(t *testing.T) {
			apiKey := "test-api-key-env"
			client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI, Project: "test-project", Location: "test-location",
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": apiKey,
					}
				},
			})
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if client.clientConfig.APIKey != "" {
				t.Errorf("Expected API ignored, got %q", client.clientConfig.APIKey)
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
