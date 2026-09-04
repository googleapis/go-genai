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
	"os"
	"strings"
	"testing"
	"time"
)

// API-mode integration tests for the live (bidirectional WebSocket) module. The
// live module opens a WebSocket, so it cannot be replayed and only runs against
// the real backend. See go/genai-sdk:integration-testing.

// liveBackend describes one backend under test. Live models are backend
// specific, and both are audio-native and reject a TEXT response modality, so
// these tests request AUDIO and enable output transcription.
type liveBackend struct {
	name    string
	backend Backend
	model   string
	// location pins the Vertex client to a region, overriding the
	// GOOGLE_CLOUD_LOCATION the Agent Platform wrapper exports. Empty means
	// take the environment as-is.
	location string
	isVertex bool
}

var liveBackends = []liveBackend{
	{name: "mldev", backend: BackendGeminiAPI, model: "gemini-3.1-flash-live-preview"},
	// gemini-live-2.5-flash-native-audio is not served on the global endpoint:
	// a setup there is rejected with 1008 "Publisher model ... was not found".
	// It is served in us-central1, us-east5 and europe-west4, so the client is
	// pinned to a region even though the shared table tests run at global.
	{
		name:     "vertex",
		backend:  BackendVertexAI,
		model:    "gemini-live-2.5-flash-native-audio",
		location: "us-central1",
		isVertex: true,
	},
}

// skipDisabledLiveBackend skips when the running job has selected the other
// backend, mirroring the shared table tests (table_test.go).
func skipDisabledLiveBackend(t *testing.T, isVertex bool) {
	t.Helper()
	runVertexOnly := os.Getenv("GOOGLE_GENAI_RUN_VERTEX_ONLY_IN_API_MODE") != ""
	runGeminiOnly := os.Getenv("GOOGLE_GENAI_RUN_GEMINI_ONLY_IN_API_MODE") != ""
	if isVertex && runGeminiOnly {
		t.Skip("Skipping Vertex AI live tests (GEMINI ONLY config enabled)")
	}
	if !isVertex && runVertexOnly {
		t.Skip("Skipping Gemini API live tests (VERTEX ONLY config enabled)")
	}
}

// liveTurnTimeout bounds a single model turn. Session.Receive blocks
// indefinitely, so without this a wedged turn would hang the nightly.
const liveTurnTimeout = 90 * time.Second

// liveTurn is everything a single model turn produced.
type liveTurn struct {
	audioBytes int
	transcript string
	toolCalls  []*FunctionCall
}

func newLiveConfig() *LiveConnectConfig {
	return &LiveConnectConfig{
		ResponseModalities:       []Modality{ModalityAudio},
		OutputAudioTranscription: &AudioTranscriptionConfig{},
	}
}

// receiveLiveTurn drains exactly one model turn, or the tool call that
// interrupts it, and fails the test if that takes longer than liveTurnTimeout.
// Receive cannot be cancelled, so it runs on its own goroutine and the session
// is closed on timeout to unblock it.
func receiveLiveTurn(t *testing.T, session *Session) *liveTurn {
	t.Helper()

	type receiveResult struct {
		turn *liveTurn
		err  error
	}
	done := make(chan receiveResult, 1)

	go func() {
		turn := &liveTurn{}
		var transcript strings.Builder
		for {
			message, err := session.Receive()
			if err != nil {
				done <- receiveResult{err: err}
				return
			}
			if message.ToolCall != nil && len(message.ToolCall.FunctionCalls) > 0 {
				turn.toolCalls = append(turn.toolCalls, message.ToolCall.FunctionCalls...)
				turn.transcript = transcript.String()
				done <- receiveResult{turn: turn}
				return
			}
			// Connect re-buffers the setup message, so skip anything with no
			// server content.
			if message.ServerContent == nil {
				continue
			}
			if message.ServerContent.OutputTranscription != nil {
				transcript.WriteString(message.ServerContent.OutputTranscription.Text)
			}
			if message.ServerContent.ModelTurn != nil {
				for _, part := range message.ServerContent.ModelTurn.Parts {
					if part.InlineData != nil {
						turn.audioBytes += len(part.InlineData.Data)
					}
				}
			}
			if message.ServerContent.TurnComplete {
				turn.transcript = transcript.String()
				done <- receiveResult{turn: turn}
				return
			}
		}
	}()

	select {
	case result := <-done:
		if result.err != nil {
			if isLiveQuotaError(result.err) {
				t.Skipf("Resource Exhausted (429). Skipping test instead of failing: %v", result.err)
			}
			t.Fatalf("Receive failed unexpectedly: %v", result.err)
		}
		return result.turn
	case <-time.After(liveTurnTimeout):
		session.Close() // nolint:errcheck
		t.Fatalf("Timed out after %v waiting for the model turn", liveTurnTimeout)
		return nil
	}
}

// isLiveQuotaError reports whether the error is a 429. A quota response still
// proves the SDK reached and parsed a reply from the live endpoint, so it is
// not a regression. See go/genai-sdk:integration-testing section 4.4.
func isLiveQuotaError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "429")
}

func sayLive(t *testing.T, session *Session, text string) {
	t.Helper()
	err := session.SendClientContent(LiveClientContentInput{
		Turns:        []*Content{{Role: RoleUser, Parts: []*Part{{Text: text}}}},
		TurnComplete: Ptr(true),
	})
	if err != nil {
		if isLiveQuotaError(err) {
			t.Skipf("Resource Exhausted (429). Skipping test instead of failing: %v", err)
		}
		t.Fatalf("SendClientContent failed unexpectedly: %v", err)
	}
}

func connectLive(t *testing.T, ctx context.Context, be liveBackend, config *LiveConnectConfig) *Session {
	t.Helper()
	client, err := NewClient(ctx, &ClientConfig{Backend: be.backend, Location: be.location})
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}
	session, err := client.Live.Connect(ctx, be.model, config)
	if err != nil {
		if isLiveQuotaError(err) {
			t.Skipf("Resource Exhausted (429). Skipping test instead of failing: %v", err)
		}
		t.Fatalf("Live.Connect failed unexpectedly: %v", err)
	}
	return session
}

func TestLiveAPI(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()

	for _, be := range liveBackends {
		be := be
		t.Run(be.name, func(t *testing.T) {
			runLiveAPISubtests(t, ctx, be)
		})
	}
}

func runLiveAPISubtests(t *testing.T, ctx context.Context, be liveBackend) {
	skipDisabledLiveBackend(t, be.isVertex)

	t.Run("text_input", func(t *testing.T) {
		session := connectLive(t, ctx, be, newLiveConfig())
		defer session.Close() // nolint:errcheck

		sayLive(t, session, "Say hello.")
		turn := receiveLiveTurn(t, session)

		if turn.audioBytes == 0 {
			t.Errorf("Expected audio output from the model, got none")
		}
		if strings.TrimSpace(turn.transcript) == "" {
			t.Errorf("Expected an output transcription, got none")
		}
	})

	t.Run("multi_turn", func(t *testing.T) {
		session := connectLive(t, ctx, be, newLiveConfig())
		defer session.Close() // nolint:errcheck

		sayLive(t, session, "Remember the number 42. Just acknowledge it.")
		if first := receiveLiveTurn(t, session); strings.TrimSpace(first.transcript) == "" {
			t.Errorf("Expected a response to the first turn, got none")
		}

		sayLive(t, session, "What number did I ask you to remember?")
		second := receiveLiveTurn(t, session)

		if second.audioBytes == 0 {
			t.Errorf("Expected audio output on the second turn, got none")
		}
		if !strings.Contains(second.transcript, "42") {
			t.Errorf("Expected the second turn to recall context from the first, transcript was %q", second.transcript)
		}
	})

	t.Run("function_calling", func(t *testing.T) {
		config := newLiveConfig()
		config.Tools = []*Tool{{
			FunctionDeclarations: []*FunctionDeclaration{{
				Name:        "turn_on_the_lights",
				Description: "Turns the lights on in the room.",
				Parameters:  &Schema{Type: TypeObject, Properties: map[string]*Schema{}},
			}},
		}}
		session := connectLive(t, ctx, be, config)
		defer session.Close() // nolint:errcheck

		sayLive(t, session, "Please turn on the lights.")
		turn := receiveLiveTurn(t, session)

		if len(turn.toolCalls) == 0 {
			t.Fatalf("Expected the model to request the tool, got no tool calls")
		}
		call := turn.toolCalls[0]
		if call.Name != "turn_on_the_lights" {
			t.Errorf("Tool call name = %q, want %q", call.Name, "turn_on_the_lights")
		}

		err := session.SendToolResponse(LiveToolResponseInput{
			FunctionResponses: []*FunctionResponse{{
				ID:       call.ID,
				Name:     call.Name,
				Response: map[string]any{"result": "ok"},
			}},
		})
		if err != nil {
			t.Fatalf("SendToolResponse failed unexpectedly: %v", err)
		}

		// Both backends must accept the tool result and complete the turn, but
		// only the Gemini API returns assertable content: Vertex emits an empty
		// transcription.
		followUp := receiveLiveTurn(t, session)
		if !be.isVertex && strings.TrimSpace(followUp.transcript) == "" {
			t.Errorf("Expected the model to respond after the tool result, got no transcription")
		}
	})

	// This SDK does not validate FunctionResponse IDs, so the error pathway here
	// is a rejected setup instead: connecting with a model that does not exist
	// must surface a failure rather than hang.
	t.Run("invalid_model_fails_to_connect", func(t *testing.T) {
		// Same Location override as connectLive: without it the Vertex case would
		// probe the global endpoint rather than the one the rest of the suite uses.
		client, err := NewClient(ctx, &ClientConfig{Backend: be.backend, Location: be.location})
		if err != nil {
			t.Fatalf("Error creating client: %v", err)
		}
		session, err := client.Live.Connect(ctx, "gemini-nonexistent-live-model", newLiveConfig())
		if err == nil {
			session.Close() // nolint:errcheck
			t.Fatalf("Expected Live.Connect to fail for a nonexistent model, but it succeeded")
		}
		if isLiveQuotaError(err) {
			t.Skipf("Resource Exhausted (429). Skipping test instead of failing: %v", err)
		}
	})
}
