// Copyright 2026 Google LLC
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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai/interactions/models/agents"
	"google.golang.org/genai/interactions/models/credentials"
	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
	"google.golang.org/genai/interactions/models/webhooks"
)

func TestInteractionsWorkflow(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/interactions" {
			t.Errorf("Path unexpected: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Method unexpected: %s", r.Method)
		}

		// Return a mock response
		resp := map[string]any{
			"id":      "mock_interaction_id",
			"created": "2026-03-30T22:20:00Z",
			"updated": "2026-03-30T22:20:00Z",
			"status":  "completed",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer ts.Close()

	// Create client pointing to the mock server
	client, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendGeminiAPI,
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
		APIKey: "dummy_key",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Interactions == nil {
		t.Fatalf("client.Interactions is nil")
	}

	// Call Create
	modelInput := interactions.NewInteractionsInput([]interactions.Content{{
		TextContent: &interactions.TextContent{
			Text: "Hello",
		},
	}})
	body := operations.NewCreateInteractionRequestBody(interactions.CreateModelInteraction{
		Model: interactions.Model("gemini-2.5-flash"),
		Input: &modelInput,
	})
	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{Body: body})
	if err != nil {
		t.Fatalf("Failed to call Create: %v", err)
	}

	if res.Interaction.ID == nil || *res.Interaction.ID != "mock_interaction_id" {
		t.Errorf("Expected ID 'mock_interaction_id', got '%v'", res.Interaction.ID)
	}
}

func TestInteractionsWorkflow_Vertex(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1beta1/projects/my-project/locations/my-location/interactions"
		if r.URL.Path != expectedPath {
			t.Errorf("Path unexpected: got %s, want %s", r.URL.Path, expectedPath)
		}
		if r.Method != "POST" {
			t.Errorf("Method unexpected: %s", r.Method)
		}

		// Return a mock response
		resp := map[string]any{
			"id":      "mock_interaction_id",
			"created": "2026-03-30T22:20:00Z",
			"updated": "2026-03-30T22:20:00Z",
			"status":  "completed",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer ts.Close()

	// Create client pointing to the mock server
	client, err := NewClient(context.Background(), &ClientConfig{
		Backend:    BackendVertexAI,
		Project:    "my-project",
		Location:   "my-location",
		HTTPClient: &http.Client{},
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Interactions == nil {
		t.Fatalf("client.Interactions is nil")
	}

	// Call Create
	modelInput := interactions.NewInteractionsInput([]interactions.Content{{
		TextContent: &interactions.TextContent{
			Text: "Hello",
		},
	}})
	body := operations.NewCreateInteractionRequestBody(interactions.CreateModelInteraction{
		Model: interactions.Model("gemini-2.5-flash"),
		Input: &modelInput,
	})
	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{Body: body})
	if err != nil {
		t.Fatalf("Failed to call Create: %v", err)
	}

	if res.Interaction.ID == nil || *res.Interaction.ID != "mock_interaction_id" {
		t.Errorf("Expected ID 'mock_interaction_id', got '%v'", res.Interaction.ID)
	}
}

func TestInteractions_ClientOptions(t *testing.T) {
	// Set up a mock server that sleeps
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header
		if got := r.Header.Get("X-Custom-Header"); got != "custom-value" {
			t.Errorf("Header X-Custom-Header = %q, want %q", got, "custom-value")
		}

		// Sleep to trigger timeout
		time.Sleep(50 * time.Millisecond)

		resp := map[string]any{
			"id":      "mock_interaction_id",
			"created": "2026-03-30T22:20:00Z",
			"updated": "2026-03-30T22:20:00Z",
			"status":  "completed",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Configure client with custom headers, short timeout, and mock server base URL
	timeout := 10 * time.Millisecond
	client, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "dummy_key",
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
			Headers: http.Header{
				"X-Custom-Header": []string{"custom-value"},
			},
			Timeout: &timeout,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	modelInput := interactions.NewInteractionsInput([]interactions.Content{{
		TextContent: &interactions.TextContent{
			Text: "Hello",
		},
	}})
	body := operations.NewCreateInteractionRequestBody(interactions.CreateModelInteraction{
		Model: interactions.Model("gemini-2.5-flash"),
		Input: &modelInput,
	})

	_, err = client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{Body: body})
	if err == nil {
		t.Fatalf("Expected timeout error, but call succeeded")
	}

	// The error should be context.DeadlineExceeded or wrap it
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected error to contain 'context deadline exceeded', got: %v", err)
	}
}


func TestWebhooksWorkflow(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/webhooks") {
			resp := map[string]any{
				"id":   "webhook_123",
				"uri":  "https://example.com/callback",
				"name": "My Webhook",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/webhooks/webhook_123") {
			resp := map[string]any{
				"id":   "webhook_123",
				"uri":  "https://example.com/callback",
				"name": "My Webhook",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "dummy_key",
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	webhookInput := webhooks.WebhookInput{
		URI: "https://example.com/callback",
	}
	createRes, err := client.Webhooks.Create(context.Background(), operations.CreateWebhookRequest{Body: webhookInput})
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	if createRes.Webhook == nil || *createRes.Webhook.ID != "webhook_123" {
		t.Errorf("Expected webhook ID 'webhook_123', got %+v", createRes.Webhook)
	}

	getRes, err := client.Webhooks.Get(context.Background(), operations.GetWebhookRequest{ID: "webhook_123"})
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}

	if getRes.Webhook == nil || getRes.Webhook.URI != "https://example.com/callback" {
		t.Errorf("Expected webhook URI 'https://example.com/callback', got %+v", getRes.Webhook)
	}
}

func TestAgentsWorkflow(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/agents") {
			resp := map[string]any{
				"id":                 "agent_456",
				"base_agent":         "gemini-2.5-flash",
				"system_instruction": "You are a coding tutor.",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/agents/agent_456") {
			resp := map[string]any{
				"id":                 "agent_456",
				"base_agent":         "gemini-2.5-flash",
				"system_instruction": "You are a coding tutor.",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "dummy_key",
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	baseAgent := "gemini-2.5-flash"
	systemInstruction := "You are a coding tutor."
	agentInput := agents.Agent{
		BaseAgent:         &baseAgent,
		SystemInstruction: &systemInstruction,
	}

	createRes, err := client.Agents.Create(context.Background(), operations.CreateAgentRequest{Body: agentInput})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if createRes.Agent == nil || *createRes.Agent.ID != "agent_456" {
		t.Errorf("Expected agent ID 'agent_456', got %+v", createRes.Agent)
	}

	getRes, err := client.Agents.Get(context.Background(), operations.GetAgentRequest{ID: "agent_456"})
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if getRes.Agent == nil || *getRes.Agent.SystemInstruction != "You are a coding tutor." {
		t.Errorf("Expected system instruction 'You are a coding tutor.', got %+v", getRes.Agent)
	}
}

func TestCredentialsLifecycle(t *testing.T) {
	var captured []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = append(captured, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" && r.URL.Path == "/v1beta/credentials" {
			resp := map[string]any{
				"credentials": []map[string]any{
					{
						"id":          "cred_bearer_123",
						"type":        "bearer_token",
						"status":      "active",
						"create_time": "2026-07-22T15:18:38Z",
						"update_time": "2026-07-22T15:18:38Z",
					},
					{
						"id":          "cred_env_123",
						"type":        "environment_variable",
						"status":      "active",
						"create_time": "2026-07-22T15:18:38Z",
						"update_time": "2026-07-22T15:18:38Z",
					},
					{
						"id":          "cred_oauth_123",
						"type":        "oauth2",
						"status":      "active",
						"create_time": "2026-07-22T15:18:38Z",
						"update_time": "2026-07-22T15:18:38Z",
					},
				},
				"next_page_token": "token_next_123",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "DELETE" {
			_ = json.NewEncoder(w).Encode(map[string]any{})
			return
		}

		resp := map[string]any{
			"id":          "cred_bearer_123",
			"type":        "bearer_token",
			"status":      "active",
			"create_time": "2026-07-22T15:18:38Z",
			"update_time": "2026-07-22T15:18:38Z",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "dummy_key",
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Credentials == nil {
		t.Fatalf("client.Credentials is nil")
	}

	// 1. Bearer token credential creation with custom header and prefix
	headerName := "X-Custom-Auth"
	prefix := "Token"
	bearerCred, err := client.Credentials.Create(context.Background(), operations.CreateCredentialRequest{
		Body: credentials.NewCredentialCreateParams(credentials.HTTPBearerConfig{
			ID:         "cred_bearer_123",
			Token:      "test-bearer-token",
			HeaderName: &headerName,
			Prefix:     &prefix,
		}),
	})
	if err != nil {
		t.Fatalf("Failed to create bearer credential: %v", err)
	}
	if bearerCred.Credential == nil || bearerCred.Credential.ID != "cred_bearer_123" {
		t.Errorf("Expected credential ID 'cred_bearer_123', got %+v", bearerCred.Credential)
	}

	// 2. Environment variable credential creation with injection_location and trusted_domains
	envLocations := []credentials.InjectionLocationEnum{
		credentials.InjectionLocationEnumHeader,
		credentials.InjectionLocationEnumQuery,
	}
	_, err = client.Credentials.Create(context.Background(), operations.CreateCredentialRequest{
		Body: credentials.NewCredentialCreateParams(credentials.EnvironmentVariableConfig{
			ID:                "cred_env_123",
			Value:             "super-secret-key",
			InjectionLocation: credentials.NewEnvironmentVariableConfigInjectionLocation(envLocations),
			TrustedDomains:    []string{"api.example.com", "service.example.org"},
		}),
	})
	if err != nil {
		t.Fatalf("Failed to create env var credential: %v", err)
	}

	// 3. OAuth2 credential creation with scopes
	_, err = client.Credentials.Create(context.Background(), operations.CreateCredentialRequest{
		Body: credentials.NewCredentialCreateParams(credentials.OAuth2Config{
			ID:           "cred_oauth_123",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "test-refresh-token",
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes:       []string{"https://www.googleapis.com/auth/cloud-platform"},
		}),
	})
	if err != nil {
		t.Fatalf("Failed to create oauth2 credential: %v", err)
	}

	// 4. List credentials
	listRes, err := client.Credentials.List(context.Background(), operations.ListCredentialsRequest{})
	if err != nil {
		t.Fatalf("Failed to list credentials: %v", err)
	}
	if listRes.CredentialListResponse == nil || len(listRes.CredentialListResponse.Credentials) != 3 {
		t.Errorf("Expected 3 credentials, got %+v", listRes.CredentialListResponse)
	}

	// 5. Get credential
	getRes, err := client.Credentials.Get(context.Background(), operations.GetCredentialRequest{ID: "cred_bearer_123"})
	if err != nil {
		t.Fatalf("Failed to get credential: %v", err)
	}
	if getRes.Credential == nil || getRes.Credential.ID != "cred_bearer_123" {
		t.Errorf("Expected fetched credential ID 'cred_bearer_123', got %+v", getRes.Credential)
	}

	// 6. Update bearer token credential
	updatedToken := "updated-token"
	authHeader := "Authorization"
	bearerPrefix := "Bearer"
	_, err = client.Credentials.Update(context.Background(), operations.UpdateCredentialRequest{
		ID: "cred_bearer_123",
		Body: credentials.NewCredentialUpdate(credentials.HTTPBearerUpdateConfig{
			Token:      &updatedToken,
			HeaderName: &authHeader,
			Prefix:     &bearerPrefix,
		}),
	})
	if err != nil {
		t.Fatalf("Failed to update bearer credential: %v", err)
	}

	// 7. Update environment variable credential
	updatedValue := "updated-secret-key"
	singleLoc := credentials.InjectionLocationEnumHeader
	locUpdate := credentials.NewEnvironmentVariableUpdateConfigInjectionLocation(singleLoc)
	_, err = client.Credentials.Update(context.Background(), operations.UpdateCredentialRequest{
		ID: "cred_env_123",
		Body: credentials.NewCredentialUpdate(credentials.EnvironmentVariableUpdateConfig{
			Value:             &updatedValue,
			InjectionLocation: &locUpdate,
			TrustedDomains:    []string{"api.example.com"},
		}),
	})
	if err != nil {
		t.Fatalf("Failed to update env var credential: %v", err)
	}

	// 8. Update OAuth2 credential
	updatedSecret := "updated-secret"
	_, err = client.Credentials.Update(context.Background(), operations.UpdateCredentialRequest{
		ID: "cred_oauth_123",
		Body: credentials.NewCredentialUpdate(credentials.OAuth2UpdateConfig{
			ClientSecret: &updatedSecret,
			Scopes:       []string{"scope1", "scope2"},
		}),
	})
	if err != nil {
		t.Fatalf("Failed to update oauth2 credential: %v", err)
	}

	// 9. Delete credential
	_, err = client.Credentials.Delete(context.Background(), operations.DeleteCredentialRequest{ID: "cred_bearer_123"})
	if err != nil {
		t.Fatalf("Failed to delete credential: %v", err)
	}

	// Verify captured endpoints
	expectedCaptured := []string{
		"POST /v1beta/credentials",
		"POST /v1beta/credentials",
		"POST /v1beta/credentials",
		"GET /v1beta/credentials",
		"GET /v1beta/credentials/cred_bearer_123",
		"PATCH /v1beta/credentials/cred_bearer_123",
		"PATCH /v1beta/credentials/cred_env_123",
		"PATCH /v1beta/credentials/cred_oauth_123",
		"DELETE /v1beta/credentials/cred_bearer_123",
	}
	if len(captured) != len(expectedCaptured) {
		t.Fatalf("Expected %d requests, got %d: %v", len(expectedCaptured), len(captured), captured)
	}
	for i, exp := range expectedCaptured {
		if captured[i] != exp {
			t.Errorf("Captured request [%d]: got %q, want %q", i, captured[i], exp)
		}
	}
}
