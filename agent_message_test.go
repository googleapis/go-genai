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

// Parsing and dedup of the Agent-Message header are tested in
// internal/agentmessage. These tests cover the wiring into the
// GenerateContent and Interactions clients.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/genai/interactions/models/apierrors"
	"google.golang.org/genai/interactions/models/operations"
	"google.golang.org/genai/interactions/retry"
)

// captureLog redirects the standard logger to a buffer for the test duration.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	origOut, origFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(origOut)
		log.SetFlags(origFlags)
	})
	return &buf
}

// newAgentMessageServer returns a server that sets the given Agent-Message
// header on every response.
func newAgentMessageServer(t *testing.T, msg string) *httptest.Server {
	return newAgentMessageServerWithStatus(t, msg, http.StatusOK)
}

// newAgentMessageServerWithStatus is like newAgentMessageServer but responds
// with the given status code.
func newAgentMessageServerWithStatus(t *testing.T, msg string, status int) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Agent-Message", fmt.Sprintf(`field_ignored;field="temperature";msg=%%"%s"`, msg))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusOK {
			fmt.Fprintln(w, `{"response": "ok"}`)
		} else {
			fmt.Fprintf(w, `{"error": {"code": %d, "message": "test error", "status": "TEST"}}`+"\n", status)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

func newAgentMessageTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := NewClient(context.Background(), &ClientConfig{
		Backend:     BackendGeminiAPI,
		HTTPOptions: HTTPOptions{BaseURL: baseURL},
		APIKey:      "dummy_key",
	})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	return client
}

func TestInteractionsHookEmitsAgentMessageWarning(t *testing.T) {
	buf := captureLog(t)
	ts := newAgentMessageServer(t, "interactions hook warning")
	client := newAgentMessageTestClient(t, ts.URL)

	for range 2 {
		if _, err := client.Environments.Files.Download(context.Background(), "env-1", "test.txt"); err != nil {
			t.Fatalf("Download() failed: %v", err)
		}
	}

	if got, want := strings.TrimSpace(buf.String()), "Warning: [Gemini API] interactions hook warning"; got != want {
		t.Errorf("logged warning = %q, want %q", got, want)
	}
}

func TestInteractionsEmitsAgentMessageWarningOnError(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"4xx", http.StatusBadRequest},
		{"5xx", http.StatusServiceUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := captureLog(t)
			msg := "interactions " + tc.name + " warning"
			ts := newAgentMessageServerWithStatus(t, msg, tc.status)
			client := newAgentMessageTestClient(t, ts.URL)

			// Disable retries so the 5xx case doesn't back off.
			_, err := client.Voices.List(context.Background(), operations.ListVoicesRequest{}, operations.WithRetries(retry.Config{Strategy: "none"}))
			var apiErr *apierrors.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tc.status || !strings.Contains(apiErr.Body, "test error") {
				t.Fatalf("Voices.List() error = %v, want APIError with status %d and server body", err, tc.status)
			}

			if got, want := strings.TrimSpace(buf.String()), "Warning: [Gemini API] "+msg; got != want {
				t.Errorf("logged warning = %q, want %q", got, want)
			}
		})
	}
}

func TestInteractionsTransportErrorPassesThrough(t *testing.T) {
	buf := captureLog(t)
	ts := newAgentMessageServer(t, "unused")
	client := newAgentMessageTestClient(t, ts.URL)
	ts.Close() // Every request now fails with a transport error and no response.

	_, err := client.Voices.List(context.Background(), operations.ListVoicesRequest{}, operations.WithRetries(retry.Config{Strategy: "none"}))
	if err == nil || !strings.Contains(err.Error(), "error sending request") {
		t.Fatalf("Voices.List() error = %v, want transport error", err)
	}
	if got := buf.String(); got != "" {
		t.Errorf("logged %q, want nothing", got)
	}
}
