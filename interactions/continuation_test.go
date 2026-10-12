// Copyright 2026 Google LLC
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

package interactions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"google.golang.org/genai/interactions/models/components"
	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
	"google.golang.org/genai/interactions/retry"
)

func ptr[T any](v T) *T {
	return &v
}


type recordingHop struct {
	Request struct {
		BodySegments []json.RawMessage `json:"body_segments"`
	} `json:"request"`
	Response struct {
		StatusCode          int               `json:"status_code"`
		Headers             map[string]string `json:"headers"`
		BodySegments        []json.RawMessage `json:"body_segments"`
		SdkResponseSegments []json.RawMessage `json:"sdk_response_segments"`
	} `json:"response"`
}

type recordingData struct {
	Interactions []recordingHop `json:"interactions"`
}

type mockTransport struct {
	mu           sync.Mutex
	hops         []recordingHop
	hopIdx       int
	sentRequests []map[string]any
	rawRequests  []*http.Request
}

func (m *mockTransport) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var bodyMap map[string]any
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &bodyMap)
		}
	}
	m.sentRequests = append(m.sentRequests, bodyMap)
	m.rawRequests = append(m.rawRequests, req)

	if m.hopIdx >= len(m.hops) {
		return nil, fmt.Errorf("unexpected extra HTTP request #%d: %v", m.hopIdx+1, bodyMap)
	}
	hop := m.hops[m.hopIdx]
	m.hopIdx++

	contentType := "application/json"
	for k, v := range hop.Response.Headers {
		if strings.EqualFold(k, "content-type") {
			contentType = v
			break
		}
	}

	var rawBytes []byte
	if strings.Contains(contentType, "text/event-stream") {
		var sseBuilder strings.Builder
		for _, seg := range hop.Response.BodySegments {
			var evMap map[string]any
			_ = json.Unmarshal(seg, &evMap)
			evType := "message"
			if t, ok := evMap["event_type"].(string); ok && t != "" {
				evType = t
			}
			var compactBuf bytes.Buffer
			if err := json.Compact(&compactBuf, seg); err == nil {
				sseBuilder.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", evType, compactBuf.String()))
			} else {
				sseBuilder.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", evType, string(seg)))
			}
		}
		rawBytes = []byte(sseBuilder.String())
	} else {
		if len(hop.Response.BodySegments) > 0 {
			rawBytes = hop.Response.BodySegments[0]
		} else {
			rawBytes = []byte("{}")
		}
	}

	statusCode := hop.Response.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}

	resp := &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(rawBytes)),
		Request:    req,
	}
	resp.Header.Set("Content-Type", contentType)
	return resp, nil
}


func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}


// ---------------------------------------------------------------------------
// End-to-End Unit Tests via client.Interactions.Create
// ---------------------------------------------------------------------------

func makeTestClient(hops ...recordingHop) (*GenAI, *mockTransport) {
	tr := &mockTransport{hops: hops}
	client := New(
		WithClient(tr),
		WithSecurity(components.Security{APIKey: String("test-key")}),
		WithRetryConfig(retry.Config{Strategy: "none"}),
	)
	return client, tr
}

func makeUnaryHop(inter interactions.Interaction) recordingHop {
	b, _ := json.Marshal(inter)
	var h recordingHop
	h.Response.StatusCode = 200
	h.Response.Headers = map[string]string{"content-type": "application/json"}
	h.Response.BodySegments = []json.RawMessage{b}
	return h
}

func makeStreamingHop(events ...any) recordingHop {
	var h recordingHop
	h.Response.StatusCode = 200
	h.Response.Headers = map[string]string{"content-type": "text/event-stream"}
	for _, ev := range events {
		b, _ := json.Marshal(ev)
		h.Response.BodySegments = append(h.Response.BodySegments, b)
	}
	return h
}

func makeModelRequestBody(prompt string, opts ...func(*interactions.CreateModelInteraction)) operations.CreateInteractionRequestBody {
	cmi := interactions.CreateModelInteraction{
		Model: interactions.Model("gemini-2.5-flash"),
		Input: ptr(interactions.NewInteractionsInput(prompt)),
	}
	for _, opt := range opts {
		opt(&cmi)
	}
	return operations.NewCreateInteractionRequestBody(cmi)
}

type errorAfterReader struct {
	r   io.Reader
	err error
}

func (e *errorAfterReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF {
		return n, e.err
	}
	return n, err
}

func TestStreamingContinuationResumesOnDeadConnectionWithLastToken(t *testing.T) {
	hop1Events := []any{
		map[string]any{
			"event_type":         "interaction.status_update",
			"status":             "in_progress",
			"continuation_token": "tok_checkpoint_1",
		},
		map[string]any{
			"event_type": "step.delta",
			"index":      0,
			"delta": map[string]any{
				"text": "Part 1 ",
				"type": "text",
			},
		},
		map[string]any{
			"event_type":         "interaction.status_update",
			"status":             "in_progress",
			"continuation_token": "tok_checkpoint_2",
		},
		map[string]any{
			"event_type": "step.delta",
			"index":      0,
			"delta": map[string]any{
				"text": "Part 2 ",
				"type": "text",
			},
		},
	}
	hop2Events := []any{
		map[string]any{
			"event_type": "step.delta",
			"index":      0,
			"delta": map[string]any{
				"text": "Part 3",
				"type": "text",
			},
		},
		map[string]any{
			"event_type": "interaction.completed",
			"interaction": map[string]any{
				"id":     "int_dead_conn",
				"status": "completed",
			},
		},
	}

	buildSSE := func(events []any) string {
		var sb strings.Builder
		for _, ev := range events {
			b, _ := json.Marshal(ev)
			var evMap map[string]any
			_ = json.Unmarshal(b, &evMap)
			evType, _ := evMap["event_type"].(string)
			sb.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", evType, string(b)))
		}
		return sb.String()
	}

	// 1. Dead connection mid-stream (both read error and premature EOF) after checkpoint tokens
	for _, dropErr := range []error{fmt.Errorf("peer closed connection"), io.EOF} {
		t.Run(fmt.Sprintf("resumes_with_last_token/err=%v", dropErr), func(t *testing.T) {
			var sentRequests []map[string]any
			callCount := 0
			tr := &customHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					callCount++
					var bodyMap map[string]any
					if req.Body != nil {
						b, _ := io.ReadAll(req.Body)
						_ = json.Unmarshal(b, &bodyMap)
					}
					sentRequests = append(sentRequests, bodyMap)

					if callCount == 1 {
						return &http.Response{
							StatusCode: 200,
							Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
							Body: io.NopCloser(&errorAfterReader{
								r:   strings.NewReader(buildSSE(hop1Events)),
								err: dropErr,
							}),
							Request: req,
						}, nil
					}
					return &http.Response{
						StatusCode: 200,
						Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
						Body:       io.NopCloser(strings.NewReader(buildSSE(hop2Events))),
						Request:    req,
					}, nil
				},
			}

			client := New(
				WithClient(tr),
				WithSecurity(components.Security{APIKey: String("test-key")}),
				WithRetryConfig(retry.Config{Strategy: "none"}),
			)

			res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
				Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
					c.Stream = ptr(true)
				}),
				AutomaticContinuation: ptr(true),
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			stream := res.InteractionSSEStreamEvent
			if stream == nil {
				t.Fatal("expected non-nil stream")
			}
			defer stream.Close()

			var textParts []string
			var eventCount int
			for stream.Next() {
				eventCount++
				ev := stream.Value()
				if deltaEv := ev.GetDataStepDelta(); deltaEv != nil && deltaEv.Delta.TextDelta != nil {
					textParts = append(textParts, deltaEv.Delta.TextDelta.Text)
				}
			}
			if err := stream.Err(); err != nil {
				t.Fatalf("expected stream to recover without error, got: %v", err)
			}
			if len(sentRequests) != 2 {
				t.Fatalf("expected 2 HTTP requests, got %d", len(sentRequests))
			}
			if tok := sentRequests[0]["continuation_token"]; tok != nil && tok != "" {
				t.Errorf("expected no continuation_token in hop 1 request, got %v", tok)
			}
			if tok := sentRequests[1]["continuation_token"]; tok != "tok_checkpoint_2" {
				t.Errorf("expected tok_checkpoint_2 in hop 2 request, got %v", tok)
			}
			if eventCount != 6 {
				t.Errorf("expected 6 total events across both hops, got %d", eventCount)
			}
			if gotText := strings.Join(textParts, ""); gotText != "Part 1 Part 2 Part 3" {
				t.Errorf("expected combined text %q, got %q", "Part 1 Part 2 Part 3", gotText)
			}
		})
	}

	// 2. Dead connection before any continuation_token is received -> surfaces error without reconnecting
	t.Run("no_token_surfaces_error", func(t *testing.T) {
		callCount := 0
		noTokenEvents := []any{
			map[string]any{
				"event_type": "step.delta",
				"index":      0,
				"delta": map[string]any{
					"text": "Partial",
					"type": "text",
				},
			},
		}
		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body: io.NopCloser(&errorAfterReader{
						r:   strings.NewReader(buildSSE(noTokenEvents)),
						err: fmt.Errorf("connection reset before checkpoint"),
					}),
					Request: req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
			WithRetryConfig(retry.Config{Strategy: "none"}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(true),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		stream := res.InteractionSSEStreamEvent
		defer stream.Close()
		for stream.Next() {
		}
		if stream.Err() == nil || !strings.Contains(stream.Err().Error(), "connection reset before checkpoint") {
			t.Fatalf("expected connection reset error, got: %v", stream.Err())
		}
		if callCount != 1 {
			t.Errorf("expected 1 request when no token was received, got %d", callCount)
		}
	})

	// 3. Dead connection when AutomaticContinuation=false -> surfaces error without reconnecting
	t.Run("automatic_continuation_false_surfaces_error", func(t *testing.T) {
		callCount := 0
		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body: io.NopCloser(&errorAfterReader{
						r:   strings.NewReader(buildSSE(hop1Events)),
						err: fmt.Errorf("connection reset with auto=false"),
					}),
					Request: req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
			WithRetryConfig(retry.Config{Strategy: "none"}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(false),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		stream := res.InteractionSSEStreamEvent
		defer stream.Close()
		for stream.Next() {
		}
		if stream.Err() == nil || !strings.Contains(stream.Err().Error(), "connection reset with auto=false") {
			t.Fatalf("expected connection reset error, got: %v", stream.Err())
		}
		if callCount != 1 {
			t.Errorf("expected 1 request when AutomaticContinuation=false, got %d", callCount)
		}
	})

	// 4. Consecutive failure on resumed hop before a new token is emitted -> stops without infinite loop
	t.Run("stops_if_resumed_hop_dies_before_new_token", func(t *testing.T) {
		callCount := 0
		hop2NoTokenEvents := []any{
			map[string]any{
				"event_type": "step.delta",
				"index":      0,
				"delta": map[string]any{
					"text": "Still no new token",
					"type": "text",
				},
			},
		}
		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				if callCount == 1 {
					return &http.Response{
						StatusCode: 200,
						Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
						Body: io.NopCloser(&errorAfterReader{
							r:   strings.NewReader(buildSSE(hop1Events)),
							err: fmt.Errorf("hop 1 died"),
						}),
						Request: req,
					}, nil
				}
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body: io.NopCloser(&errorAfterReader{
						r:   strings.NewReader(buildSSE(hop2NoTokenEvents)),
						err: fmt.Errorf("hop 2 died before new token"),
					}),
					Request: req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
			WithRetryConfig(retry.Config{Strategy: "none"}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(true),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		stream := res.InteractionSSEStreamEvent
		defer stream.Close()
		for stream.Next() {
		}
		if stream.Err() == nil || !strings.Contains(stream.Err().Error(), "hop 2 died before new token") {
			t.Fatalf("expected hop 2 error, got: %v", stream.Err())
		}
		if callCount != 2 {
			t.Errorf("expected exactly 2 requests without infinite loop, got %d", callCount)
		}
	})

	// 5. Connection drops mid-SSE frame at EOF (truncated JSON payload) -> resumes with last checkpoint token
	t.Run("resumes_on_truncated_mid_sse_frame", func(t *testing.T) {
		var sentRequests []map[string]any
		callCount := 0
		truncatedHop1SSE := buildSSE(hop1Events) + "event: step.delta\ndata: {\"event_type\":\"step.delta\",\"delta\":{\"text\":\"truncated"
		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				var bodyMap map[string]any
				if req.Body != nil {
					b, _ := io.ReadAll(req.Body)
					_ = json.Unmarshal(b, &bodyMap)
				}
				sentRequests = append(sentRequests, bodyMap)

				if callCount == 1 {
					return &http.Response{
						StatusCode: 200,
						Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
						Body:       io.NopCloser(strings.NewReader(truncatedHop1SSE)),
						Request:    req,
					}, nil
				}
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body:       io.NopCloser(strings.NewReader(buildSSE(hop2Events))),
					Request:    req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
			WithRetryConfig(retry.Config{Strategy: "none"}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(true),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		stream := res.InteractionSSEStreamEvent
		defer stream.Close()

		var textParts []string
		for stream.Next() {
			ev := stream.Value()
			if deltaEv := ev.GetDataStepDelta(); deltaEv != nil && deltaEv.Delta.TextDelta != nil {
				textParts = append(textParts, deltaEv.Delta.TextDelta.Text)
			}
		}
		if err := stream.Err(); err != nil {
			t.Fatalf("expected truncated mid-frame stream to recover without error, got: %v", err)
		}
		if len(sentRequests) != 2 {
			t.Fatalf("expected 2 HTTP requests, got %d", len(sentRequests))
		}
		if tok := sentRequests[1]["continuation_token"]; tok != "tok_checkpoint_2" {
			t.Errorf("expected tok_checkpoint_2 in hop 2 request, got %v", tok)
		}
		if gotText := strings.Join(textParts, ""); gotText != "Part 1 Part 2 Part 3" {
			t.Errorf("expected combined text %q, got %q", "Part 1 Part 2 Part 3", gotText)
		}
	})

	// 6. Server emits SSE event: error after checkpoint token and closes stream -> does not reconnect
	t.Run("does_not_reconnect_after_sse_error_event", func(t *testing.T) {
		callCount := 0
		eventsWithErr := append([]any{}, hop1Events...)
		eventsWithErr = append(eventsWithErr, map[string]any{
			"event_type": "error",
			"error": map[string]any{
				"code":    "resource_exhausted",
				"message": "Quota exceeded mid-stream",
			},
		})
		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body:       io.NopCloser(strings.NewReader(buildSSE(eventsWithErr))),
					Request:    req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
			WithRetryConfig(retry.Config{Strategy: "none"}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(true),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		stream := res.InteractionSSEStreamEvent
		defer stream.Close()

		var sawErrorEvent bool
		for stream.Next() {
			if stream.Value().GetDataError() != nil {
				sawErrorEvent = true
			}
		}
		if !sawErrorEvent {
			t.Error("expected to receive ErrorEvent before stream ended")
		}
		if callCount != 1 {
			t.Errorf("expected 1 request (no reconnect after SSE error event), got %d", callCount)
		}
	})
}

func TestAutomaticContinuationOptionPrecedence(t *testing.T) {
	hop1 := interactions.Interaction{
		ID:                ptr("id1"),
		Status:            interactions.InteractionStatus("continuation_required"),
		ContinuationToken: ptr("tok1"),
	}
	hop2 := interactions.Interaction{
		ID:     ptr("id2"),
		Status: interactions.InteractionStatusCompleted,
	}

	// Case 1: Request.AutomaticContinuation = false, but Option = WithAutomaticContinuation(true)
	client, tr := makeTestClient(makeUnaryHop(hop1), makeUnaryHop(hop2))
	res, err := client.Interactions.Create(
		context.Background(),
		operations.CreateInteractionRequest{
			Body:                  makeModelRequestBody("test"),
			AutomaticContinuation: ptr(false),
		},
		operations.WithAutomaticContinuation(true),
	)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(tr.sentRequests) != 2 {
		t.Errorf("expected option to override request and make 2 requests, got %d", len(tr.sentRequests))
	}
	if res.Interaction.Status != interactions.InteractionStatusCompleted {
		t.Errorf("expected completed status, got %v", res.Interaction.Status)
	}

	// Case 2: Request.AutomaticContinuation = true, but Option = WithAutomaticContinuation(false)
	client2, tr2 := makeTestClient(makeUnaryHop(hop1), makeUnaryHop(hop2))
	res2, err := client2.Interactions.Create(
		context.Background(),
		operations.CreateInteractionRequest{
			Body:                  makeModelRequestBody("test"),
			AutomaticContinuation: ptr(true),
		},
		operations.WithAutomaticContinuation(false),
	)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(tr2.sentRequests) != 1 {
		t.Errorf("expected option to override request and stop after 1 request, got %d", len(tr2.sentRequests))
	}
	if string(res2.Interaction.Status) != "continuation_required" {
		t.Errorf("expected continuation_required status, got %v", res2.Interaction.Status)
	}
}

func TestRejectAutomaticContinuationWithBackground(t *testing.T) {
	client, tr := makeTestClient()

	// Unary: automatic_continuation=false with background=true -> error
	_, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Background = ptr(true)
		}),
		AutomaticContinuation: ptr(false),
	})
	if err == nil {
		t.Fatal("expected error when automatic_continuation=false with background=true (unary)")
	}
	if !strings.Contains(err.Error(), "automatic_continuation=false cannot be used with background=true") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Unary via option: WithAutomaticContinuation(false) with background=true -> error
	_, errOpt := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Background = ptr(true)
		}),
	}, operations.WithAutomaticContinuation(false))
	if errOpt == nil {
		t.Fatal("expected error when WithAutomaticContinuation(false) with background=true")
	}
	if !strings.Contains(errOpt.Error(), "automatic_continuation=false cannot be used with background=true") {
		t.Errorf("unexpected error message: %v", errOpt)
	}

	// Streaming: automatic_continuation=false with background=true -> error
	_, errStream := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Background = ptr(true)
			c.Stream = ptr(true)
		}),
		AutomaticContinuation: ptr(false),
	})
	if errStream == nil {
		t.Fatal("expected error when automatic_continuation=false with background=true (streaming)")
	}
	if !strings.Contains(errStream.Error(), "automatic_continuation=false cannot be used with background=true") {
		t.Errorf("unexpected error message: %v", errStream)
	}

	if len(tr.sentRequests) != 0 {
		t.Errorf("expected 0 HTTP requests for rejected background calls, got %d", len(tr.sentRequests))
	}
}

func TestRejectAutomaticContinuationWithStoreFalseUnary(t *testing.T) {
	client, tr := makeTestClient()

	// Unary: automatic_continuation=true with store=false -> error
	_, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Store = ptr(false)
		}),
		AutomaticContinuation: ptr(true),
	})
	if err == nil {
		t.Fatal("expected error when automatic_continuation=true with store=false (unary)")
	}
	if !strings.Contains(err.Error(), "automatic_continuation=true cannot be used with store=false in unary mode") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Unary via option: WithAutomaticContinuation(true) with store=false -> error
	_, errOpt := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Store = ptr(false)
		}),
	}, operations.WithAutomaticContinuation(true))
	if errOpt == nil {
		t.Fatal("expected error when WithAutomaticContinuation(true) with store=false (unary)")
	}
	if !strings.Contains(errOpt.Error(), "automatic_continuation=true cannot be used with store=false in unary mode") {
		t.Errorf("unexpected error message: %v", errOpt)
	}

	if len(tr.sentRequests) != 0 {
		t.Errorf("expected 0 HTTP requests for rejected store=false calls, got %d", len(tr.sentRequests))
	}
}

func TestStopConditionsUnaryAndStreaming(t *testing.T) {
	// 1. Stop on requires_action even if continuation_token is set (streaming)
	t.Run("stop_on_requires_action_streaming", func(t *testing.T) {
		hopEvents := []any{
			map[string]any{
				"event_type": "interaction.completed",
				"interaction": map[string]any{
					"id":                 "act1",
					"status":             "requires_action",
					"continuation_token": "tok_act",
				},
			},
		}
		client, tr := makeTestClient(makeStreamingHop(hopEvents...))
		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(true),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		stream := res.InteractionSSEStreamEvent
		for stream.Next() {
		}
		if err := stream.Err(); err != nil {
			t.Fatalf("stream err: %v", err)
		}
		stream.Close()
		if len(tr.sentRequests) != 1 {
			t.Errorf("expected to stop on requires_action, got %d requests", len(tr.sentRequests))
		}
	})

	// 2. Stop when continuation_token is empty or nil (unary)
	t.Run("stop_on_empty_continuation_token_unary", func(t *testing.T) {
		for _, tok := range []*string{nil, ptr("")} {
			hopNoTok := interactions.Interaction{
				ID:                ptr("notok1"),
				Status:            interactions.InteractionStatus("continuation_required"),
				ContinuationToken: tok,
			}
			client, tr := makeTestClient(makeUnaryHop(hopNoTok))
			res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
				Body:                  makeModelRequestBody("test"),
				AutomaticContinuation: ptr(true),
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			if len(tr.sentRequests) != 1 {
				t.Errorf("expected to stop on missing/empty token, got %d requests", len(tr.sentRequests))
			}
			if string(res.Interaction.Status) != "continuation_required" {
				t.Errorf("expected continuation_required status, got %v", res.Interaction.Status)
			}
		}
	})

	// 3. Stop when continuation_token is empty or nil (streaming)
	t.Run("stop_on_empty_continuation_token_streaming", func(t *testing.T) {
		for _, tok := range []any{nil, ""} {
			hopEvents := []any{
				map[string]any{
					"event_type": "interaction.completed",
					"interaction": map[string]any{
						"id":                 "notok_stream",
						"status":             "continuation_required",
						"continuation_token": tok,
					},
				},
			}
			client, tr := makeTestClient(makeStreamingHop(hopEvents...))
			res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
				Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
					c.Stream = ptr(true)
				}),
				AutomaticContinuation: ptr(true),
			})
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			stream := res.InteractionSSEStreamEvent
			for stream.Next() {
			}
			if err := stream.Err(); err != nil {
				t.Fatalf("stream err: %v", err)
			}
			stream.Close()
			if len(tr.sentRequests) != 1 {
				t.Errorf("expected to stop on missing/empty token in streaming, got %d requests", len(tr.sentRequests))
			}
		}
	})
}

func TestInitialCallerContinuationToken(t *testing.T) {
	hop1 := interactions.Interaction{
		ID:                ptr("id1"),
		Status:            interactions.InteractionStatus("continuation_required"),
		ContinuationToken: ptr("server_tok_1"),
	}
	hop2 := interactions.Interaction{
		ID:     ptr("id2"),
		Status: interactions.InteractionStatusCompleted,
	}

	client, tr := makeTestClient(makeUnaryHop(hop1), makeUnaryHop(hop2))

	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.ContinuationToken = ptr("caller_initial_tok")
		}),
		AutomaticContinuation: ptr(true),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(tr.sentRequests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(tr.sentRequests))
	}
	if tok := tr.sentRequests[0]["continuation_token"]; tok != "caller_initial_tok" {
		t.Errorf("hop 1 should have caller_initial_tok, got %v", tok)
	}
	if tok := tr.sentRequests[1]["continuation_token"]; tok != "server_tok_1" {
		t.Errorf("hop 2 should have server_tok_1, got %v", tok)
	}
	if res.Interaction.Status != interactions.InteractionStatusCompleted {
		t.Errorf("expected completed status, got %v", res.Interaction.Status)
	}
}

func TestTransientHTTPRetryHop2(t *testing.T) {
	hop1 := interactions.Interaction{
		ID:                ptr("id1"),
		Status:            interactions.InteractionStatus("continuation_required"),
		ContinuationToken: ptr("tok1"),
	}
	hop2 := interactions.Interaction{
		ID:     ptr("id2"),
		Status: interactions.InteractionStatusCompleted,
	}

	var hop2Attempt int32
	tr := &mockTransport{}
	tr.hops = []recordingHop{
		makeUnaryHop(hop1),
	}

	// Custom transport that serves hop 1, fails on hop 2 attempt 1 (HTTP 503), succeeds on hop 2 attempt 2
	retryTransport := &mockTransportRetry{
		hop1:         makeUnaryHop(hop1),
		hop2:         makeUnaryHop(hop2),
		hop2Attempts: &hop2Attempt,
	}

	client := New(
		WithClient(retryTransport),
		WithSecurity(components.Security{APIKey: String("test-key")}),
		WithRetryConfig(retry.Config{
			Strategy: "backoff",
			Backoff: &retry.BackoffStrategy{
				InitialInterval: 1,
				MaxInterval:     5,
				Exponent:        1.0,
				MaxElapsedTime:  500,
			},
			RetryConnectionErrors: true,
		}),
	)

	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body:                  makeModelRequestBody("test"),
		AutomaticContinuation: ptr(true),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if atomic.LoadInt32(&hop2Attempt) != 2 {
		t.Errorf("expected 2 attempts on hop 2, got %d", hop2Attempt)
	}
	if res.Interaction.Status != interactions.InteractionStatusCompleted {
		t.Errorf("expected completed status, got %v", res.Interaction.Status)
	}
}

type mockTransportRetry struct {
	mu           sync.Mutex
	hop1         recordingHop
	hop2         recordingHop
	hop1Done     bool
	hop2Attempts *int32
	requests     []map[string]any
}

func (m *mockTransportRetry) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var bodyMap map[string]any
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(b))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(b)), nil
		}
		_ = json.Unmarshal(b, &bodyMap)
	}
	m.requests = append(m.requests, bodyMap)

	if !m.hop1Done {
		m.hop1Done = true
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(m.hop1.Response.BodySegments[0])),
			Request:    req,
		}, nil
	}

	attempt := atomic.AddInt32(m.hop2Attempts, 1)
	if attempt == 1 {
		return &http.Response{
			StatusCode: 503,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": {"code": 503, "message": "Service Unavailable"}}`))),
			Request:    req,
		}, nil
	}

	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(m.hop2.Response.BodySegments[0])),
		Request:    req,
	}, nil
}

func TestStreamingStatusUpdateCheckpointToken(t *testing.T) {
	// Hop 1 has status_update checkpoint token, and terminal completed omits token
	hop1Events := []any{
		map[string]any{
			"event_type":         "interaction.status_update",
			"status":             "in_progress",
			"continuation_token": "checkpoint_token_abc",
		},
		map[string]any{
			"event_type": "step.delta",
			"index":      0,
			"delta": map[string]any{
				"text": "Text",
				"type": "text",
			},
		},
		map[string]any{
			"event_type": "interaction.completed",
			"interaction": map[string]any{
				"id":     "hop1_id",
				"status": "continuation_required",
				// continuation_token is omitted here!
			},
		},
	}
	hop2Events := []any{
		map[string]any{
			"event_type": "interaction.completed",
			"interaction": map[string]any{
				"id":     "hop2_id",
				"status": "completed",
			},
		},
	}

	client, tr := makeTestClient(makeStreamingHop(hop1Events...), makeStreamingHop(hop2Events...))
	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Stream = ptr(true)
		}),
		AutomaticContinuation: ptr(true),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stream := res.InteractionSSEStreamEvent
	for stream.Next() {
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err: %v", err)
	}
	stream.Close()

	if len(tr.sentRequests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(tr.sentRequests))
	}
	if tok := tr.sentRequests[1]["continuation_token"]; tok != "checkpoint_token_abc" {
		t.Errorf("expected checkpoint_token_abc in hop 2 request, got %v", tok)
	}
}

func TestCloseStreamingClosesUnderlyingStream(t *testing.T) {
	t.Run("single_hop_close", func(t *testing.T) {
		var closed bool
		cr := &closableReader{
			Reader: strings.NewReader("event: interaction.created\ndata: {}\n\n"),
			onClose: func() {
				closed = true
			},
		}

		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body:       cr,
					Request:    req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		stream := res.InteractionSSEStreamEvent
		if stream == nil {
			t.Fatal("expected non-nil stream")
		}

		_ = stream.Close()
		if !closed {
			t.Error("expected stream.Close() to close underlying HTTP response body")
		}
	})

	t.Run("multi_hop_close_after_first_event_of_hop_2", func(t *testing.T) {
		var hop1Closed, hop2Closed bool
		var callCount int

		hop1SSE := "event: interaction.completed\ndata: {\"event_type\":\"interaction.completed\",\"interaction\":{\"id\":\"h1\",\"status\":\"continuation_required\",\"continuation_token\":\"tok1\"}}\n\n"
		hop2SSE := "event: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"part2\"}}\n\nevent: interaction.completed\ndata: {\"event_type\":\"interaction.completed\",\"interaction\":{\"id\":\"h2\",\"status\":\"completed\"}}\n\n"

		tr := &customHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				callCount++
				if callCount == 1 {
					return &http.Response{
						StatusCode: 200,
						Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
						Body: &closableReader{
							Reader:  strings.NewReader(hop1SSE),
							onClose: func() { hop1Closed = true },
						},
						Request: req,
					}, nil
				}
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body: &closableReader{
						Reader:  strings.NewReader(hop2SSE),
						onClose: func() { hop2Closed = true },
					},
					Request: req,
				}, nil
			},
		}

		client := New(
			WithClient(tr),
			WithSecurity(components.Security{APIKey: String("test-key")}),
		)

		res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.Stream = ptr(true)
			}),
			AutomaticContinuation: ptr(true),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		stream := res.InteractionSSEStreamEvent
		if stream == nil {
			t.Fatal("expected non-nil stream")
		}

		// Consume hop 1's interaction.completed event and hop 2's first event (step.delta)
		if !stream.Next() {
			t.Fatalf("expected hop 1 event, err: %v", stream.Err())
		}
		if !stream.Next() {
			t.Fatalf("expected hop 2 first event, err: %v", stream.Err())
		}
		if !hop1Closed {
			t.Error("expected hop 1 stream to be closed when transitioning to hop 2")
		}

		// Close early during hop 2
		_ = stream.Close()
		if !hop2Closed {
			t.Error("expected stream.Close() during hop 2 to close hop 2's active HTTP response body")
		}
	})
}

func TestGetAndCancelPopulateOutputText(t *testing.T) {
	interWithSteps := interactions.Interaction{
		ID:     ptr("int_get_cancel"),
		Status: interactions.InteractionStatusCompleted,
		Steps: []interactions.Step{
			interactions.NewStep(interactions.ModelOutputStep{
				Content: []interactions.Content{
					interactions.NewContent(interactions.TextContent{Text: "Hello from "}),
					interactions.NewContent(interactions.TextContent{Text: "Get/Cancel"}),
				},
			}),
		},
	}

	client, _ := makeTestClient(makeUnaryHop(interWithSteps), makeUnaryHop(interWithSteps))

	getRes, err := client.Interactions.Get(context.Background(), operations.GetInteractionByIDRequest{
		ID: "int_get_cancel",
	})
	if err != nil {
		t.Fatalf("Interactions.Get failed: %v", err)
	}
	if getRes.Interaction == nil || derefStr(getRes.Interaction.OutputText) != "Hello from Get/Cancel" {
		t.Errorf("Get OutputText mismatch: got %q, want %q", derefStr(getRes.Interaction.OutputText), "Hello from Get/Cancel")
	}

	cancelRes, err := client.Interactions.Cancel(context.Background(), operations.CancelInteractionByIDRequest{
		ID: "int_get_cancel",
	})
	if err != nil {
		t.Fatalf("Interactions.Cancel failed: %v", err)
	}
	if cancelRes.Interaction == nil || derefStr(cancelRes.Interaction.OutputText) != "Hello from Get/Cancel" {
		t.Errorf("Cancel OutputText mismatch: got %q, want %q", derefStr(cancelRes.Interaction.OutputText), "Hello from Get/Cancel")
	}
}

type closableReader struct {
	io.Reader
	onClose func()
}

func (c *closableReader) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return nil
}

type customHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (c *customHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.doFunc(req)
}

func TestIncompleteContinuationWarningUnary(t *testing.T) {
	// Subtest 1: warns when continuation_required without automatic_continuation
	t.Run("warns_when_incomplete", func(t *testing.T) {
		resetContinuationWarningsForTesting()
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		hop := interactions.Interaction{
			ID:                ptr("id1"),
			Status:            interactions.InteractionStatus("continuation_required"),
			ContinuationToken: ptr("tok1"),
		}
		client, _ := makeTestClient(makeUnaryHop(hop))
		_, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body:                  makeModelRequestBody("test"),
			AutomaticContinuation: ptr(false),
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if !strings.Contains(logBuf.String(), "status='continuation_required'") {
			t.Errorf("expected incomplete continuation warning, got: %s", logBuf.String())
		}
	})

	// Subtest 2: warns only once
	t.Run("warns_only_once", func(t *testing.T) {
		resetContinuationWarningsForTesting()
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		hop := interactions.Interaction{
			ID:                ptr("id1"),
			Status:            interactions.InteractionStatus("continuation_required"),
			ContinuationToken: ptr("tok1"),
		}
		client, _ := makeTestClient(makeUnaryHop(hop), makeUnaryHop(hop))
		_, _ = client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test"),
		})
		_, _ = client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test"),
		})

		count := strings.Count(logBuf.String(), "status='continuation_required'")
		if count != 1 {
			t.Errorf("expected warning to be logged exactly once, got %d times", count)
		}
	})

	// Subtest 3: suppressed when caller passes continuation token
	t.Run("suppressed_with_caller_token", func(t *testing.T) {
		resetContinuationWarningsForTesting()
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		hop := interactions.Interaction{
			ID:                ptr("id1"),
			Status:            interactions.InteractionStatus("continuation_required"),
			ContinuationToken: ptr("tok1"),
		}
		client, _ := makeTestClient(makeUnaryHop(hop))
		_, _ = client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
			Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
				c.ContinuationToken = ptr("caller_tok")
			}),
		})
		if strings.Contains(logBuf.String(), "status='continuation_required'") {
			t.Errorf("warning should be suppressed when caller provides token, got: %s", logBuf.String())
		}
	})
}

func TestIncompleteContinuationWarningStreaming(t *testing.T) {
	resetContinuationWarningsForTesting()
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	hopEvents := []any{
		map[string]any{
			"event_type": "interaction.completed",
			"interaction": map[string]any{
				"id":                 "hop1_id",
				"status":             "continuation_required",
				"continuation_token": "stream_tok",
			},
		},
	}

	client, _ := makeTestClient(makeStreamingHop(hopEvents...))
	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test", func(c *interactions.CreateModelInteraction) {
			c.Stream = ptr(true)
		}),
		AutomaticContinuation: ptr(false),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stream := res.InteractionSSEStreamEvent
	for stream.Next() {
	}
	stream.Close()

	if !strings.Contains(logBuf.String(), "status='continuation_required'") {
		t.Errorf("expected incomplete continuation warning on streaming, got: %s", logBuf.String())
	}
}

func TestOutputTextBreaksOnToolCall(t *testing.T) {
	hop := interactions.Interaction{
		ID:     ptr("int_tool_steps"),
		Status: interactions.InteractionStatusCompleted,
		Steps: []interactions.Step{
			interactions.NewStep(interactions.ModelOutputStep{
				Content: []interactions.Content{
					interactions.NewContent(interactions.TextContent{Text: "Pre-tool text that should be ignored"}),
				},
			}),
			interactions.NewStep(interactions.FunctionCallStep{
				Name: "search",
			}),
			interactions.NewStep(interactions.ModelOutputStep{
				Content: []interactions.Content{
					interactions.NewContent(interactions.TextContent{Text: "Post-tool part 1, "}),
				},
			}),
			interactions.NewStep(interactions.ThoughtStep{
				Signature: ptr("sig"),
			}),
			interactions.NewStep(interactions.ModelOutputStep{
				Content: []interactions.Content{
					interactions.NewContent(interactions.TextContent{Text: "post-tool part 2"}),
				},
			}),
		},
	}

	client, _ := makeTestClient(makeUnaryHop(hop))
	res, err := client.Interactions.Create(context.Background(), operations.CreateInteractionRequest{
		Body: makeModelRequestBody("test"),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if res.Interaction == nil || derefStr(res.Interaction.OutputText) != "Post-tool part 1, post-tool part 2" {
		t.Errorf("expected OutputText %q, got %q", "Post-tool part 1, post-tool part 2", derefStr(res.Interaction.OutputText))
	}
}
