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

// Unit tests for automatic continuation behaviors that the shared conformance
// cases (continuation_conformance_test.go) don't cover: the whole-call
// context deadline, per-hop operation timeouts and option/request handling.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"google.golang.org/genai/interactions/models/apierrors"
	"google.golang.org/genai/interactions/models/components"
	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
	"google.golang.org/genai/interactions/retry"
)

// scriptedHop is one scripted server answer (see scriptedServer).
type scriptedHop struct {
	// body is the unary JSON response, or nil for an SSE response made of
	// events.
	body map[string]any
	// events are SSE event payloads; the stream is closed cleanly after them.
	events []map[string]any
	// delay is how long the handler blocks before answering (it returns
	// early if the client goes away).
	delay time.Duration
	// status, when non-zero, is an HTTP error status.
	status int
}

type scriptedServer struct {
	t    *testing.T
	hops []scriptedHop

	mu       sync.Mutex
	requests []map[string]any
	// headers records the request headers, for option passthrough checks.
	headers []http.Header
}

func (s *scriptedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req map[string]any
	_ = json.Unmarshal(body, &req)
	s.mu.Lock()
	s.requests = append(s.requests, req)
	s.headers = append(s.headers, r.Header.Clone())
	n := len(s.requests)
	s.mu.Unlock()
	if n > len(s.hops) {
		s.t.Errorf("unexpected request #%d; the test scripts %d", n, len(s.hops))
		w.WriteHeader(http.StatusTeapot)
		return
	}
	hop := s.hops[n-1]
	if hop.delay > 0 {
		select {
		case <-time.After(hop.delay):
		case <-r.Context().Done():
			return
		}
	}
	switch {
	case hop.status != 0:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(hop.status)
		fmt.Fprintf(w, `{"error":{"code":%d,"message":"test","status":"ERROR"}}`, hop.status)
	case hop.body != nil:
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(hop.body)
		_, _ = w.Write(b)
	default:
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, e := range hop.events {
			b, _ := json.Marshal(e)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e["event_type"], b)
		}
		fmt.Fprint(w, "event: done\ndata: [DONE]\n\n")
	}
}

func (s *scriptedServer) requestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func unaryHop(id, status, token, text string) map[string]any {
	body := map[string]any{
		"id":     id,
		"status": status,
		"steps": []any{map[string]any{
			"type":    "model_output",
			"content": []any{map[string]any{"type": "text", "text": text}},
		}},
	}
	if token != "" {
		body["continuation_token"] = token
	}
	return body
}

func streamHopEvents(id, status, token, text string) []map[string]any {
	completed := map[string]any{"id": id, "status": status}
	if token != "" {
		completed["continuation_token"] = token
	}
	return []map[string]any{
		{"event_type": "interaction.created", "interaction": map[string]any{"id": id, "status": "in_progress"}},
		{"event_type": "step.delta", "index": 0, "delta": map[string]any{"type": "text", "text": text}},
		{"event_type": "interaction.completed", "interaction": completed},
	}
}

func newTestClient(ts *httptest.Server, opts ...SDKOption) *GenAI {
	return New(append([]SDKOption{
		WithServerURL(ts.URL),
		WithSecurity(components.Security{APIKey: String("test-key")}),
		WithRetryConfig(retry.Config{Strategy: "none"}),
	}, opts...)...)
}

func modelRequest(stream bool) operations.CreateInteractionRequest {
	input := interactions.NewInteractionsInput("Hello")
	body := interactions.CreateModelInteraction{Model: interactions.ModelGemini25Pro, Input: &input}
	if stream {
		body.Stream = Bool(true)
	}
	return operations.CreateInteractionRequest{Body: operations.NewCreateInteractionRequestBody(body)}
}

func TestUnaryContextDeadlineBoundsWholeCall(t *testing.T) {
	// Hop 1 answers quickly, hop 2 would take longer than the remaining
	// deadline: the call must stop with a ContinuationInterruptedError
	// wrapping context.DeadlineExceeded and carrying hop 1 as partial result.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{body: unaryHop("i1", "requires_continuation", "t1", "A")},
		{body: unaryHop("i2", "completed", "", "B"), delay: 5 * time.Second},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := client.Interactions.Create(ctx, modelRequest(false))

	var interrupted *apierrors.ContinuationInterruptedError
	if !errors.As(err, &interrupted) {
		t.Fatalf("Create() error = %v (%T), want a *apierrors.ContinuationInterruptedError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Create() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if interrupted.ContinuationToken != "t1" || interrupted.InteractionID != "i1" {
		t.Errorf("error token/interaction = %q/%q, want t1/i1", interrupted.ContinuationToken, interrupted.InteractionID)
	}
	if got := interrupted.PartialInteraction; got == nil || deref(got.ID) != "i1" || deref(got.OutputText) != "A" {
		t.Errorf("PartialInteraction = %+v, want hop 1 (id i1, text A)", got)
	}
	if n := server.requestCount(); n != 2 {
		t.Errorf("requests = %d, want 2", n)
	}
}

// cancelAfterFirstResponse is an HTTPClient that cancels the call's context as
// soon as the first response has been received, so the context is already
// done when the SDK would start hop 2.
type cancelAfterFirstResponse struct {
	inner  HTTPClient
	cancel context.CancelFunc
	once   sync.Once
}

func (c *cancelAfterFirstResponse) Do(req *http.Request) (*http.Response, error) {
	res, err := c.inner.Do(req)
	c.once.Do(c.cancel)
	return res, err
}

func TestUnaryExpiredContextDoesNotStartNextHop(t *testing.T) {
	// The context is done by the time hop 1 has been received: no second
	// request is sent and the failure is reported as interrupted after
	// progress, wrapping the context error.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{body: unaryHop("i1", "requires_continuation", "t1", "A")},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := newTestClient(ts, WithClient(&cancelAfterFirstResponse{inner: http.DefaultClient, cancel: cancel}))

	_, err := client.Interactions.Create(ctx, modelRequest(false))
	var interrupted *apierrors.ContinuationInterruptedError
	if !errors.As(err, &interrupted) {
		t.Fatalf("Create() error = %v (%T), want a *apierrors.ContinuationInterruptedError", err, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Create() error = %v, want it to wrap context.Canceled", err)
	}
	if interrupted.ContinuationToken != "t1" || interrupted.InteractionID != "i1" {
		t.Errorf("error token/interaction = %q/%q, want t1/i1", interrupted.ContinuationToken, interrupted.InteractionID)
	}
	if n := server.requestCount(); n != 1 {
		t.Errorf("requests = %d, want 1 (no hop after the context is done)", n)
	}
}

func TestUnaryFirstHopDeadlineReturnsRawError(t *testing.T) {
	// Expiry before any hop completed is not "after progress": the raw error
	// is returned, not a ContinuationInterruptedError.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{body: unaryHop("i1", "completed", "", "A"), delay: 5 * time.Second},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := client.Interactions.Create(ctx, modelRequest(false))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Create() error = %v, want context.DeadlineExceeded", err)
	}
	var interrupted *apierrors.ContinuationInterruptedError
	if errors.As(err, &interrupted) {
		t.Errorf("Create() error = %v, want the raw error, not a ContinuationInterruptedError", err)
	}
}

func TestUnaryOperationTimeoutAppliesPerHop(t *testing.T) {
	// WithOperationTimeout is an HTTP request setting: each hop gets the full
	// timeout, so two hops that each take 60% of it succeed.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{body: unaryHop("i1", "requires_continuation", "t1", "A"), delay: 300 * time.Millisecond},
		{body: unaryHop("i2", "completed", "", "B"), delay: 300 * time.Millisecond},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	res, err := client.Interactions.Create(context.Background(), modelRequest(false),
		operations.WithOperationTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := deref(res.Interaction.OutputText); got != "AB" {
		t.Errorf("OutputText = %q, want AB", got)
	}
	if got := res.Interaction.ContinuationIds; len(got) != 2 || got[0] != "i1" || got[1] != "i2" {
		t.Errorf("ContinuationIds = %v, want [i1 i2]", got)
	}
}

func TestUnaryOperationTimeoutExpiryIsInterrupted(t *testing.T) {
	// A per-hop timeout that expires on hop 2 is a failure after progress.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{body: unaryHop("i1", "requires_continuation", "t1", "A")},
		{body: unaryHop("i2", "completed", "", "B"), delay: 5 * time.Second},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	_, err := client.Interactions.Create(context.Background(), modelRequest(false),
		operations.WithOperationTimeout(300*time.Millisecond))
	var interrupted *apierrors.ContinuationInterruptedError
	if !errors.As(err, &interrupted) {
		t.Fatalf("Create() error = %v (%T), want a *apierrors.ContinuationInterruptedError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Create() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if interrupted.ContinuationToken != "t1" {
		t.Errorf("ContinuationToken = %q, want t1", interrupted.ContinuationToken)
	}
}

func TestStreamContextDeadlineBetweenHopsIsInterrupted(t *testing.T) {
	// Hop 1 completes with requires_continuation; the deadline passes before
	// hop 2 is requested: the stream ends with a ContinuationInterruptedError
	// wrapping context.DeadlineExceeded and hop 1's token.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{events: streamHopEvents("i1", "requires_continuation", "t1", "A")},
		{events: streamHopEvents("i2", "completed", "", "B"), delay: 5 * time.Second},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	res, err := client.Interactions.Create(ctx, modelRequest(true))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	es := res.InteractionSSEStreamEvent
	defer es.Close()
	var types []interactions.InteractionSSEEventType
	for es.Next() {
		types = append(types, es.Value().Data.Type)
	}
	err = es.Err()
	var interrupted *apierrors.ContinuationInterruptedError
	if !errors.As(err, &interrupted) {
		t.Fatalf("stream error = %v (%T), want a *apierrors.ContinuationInterruptedError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("stream error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if interrupted.ContinuationToken != "t1" || interrupted.InteractionID != "i1" || interrupted.PartialInteraction != nil {
		t.Errorf("error = %+v, want token t1 from i1 and no partial interaction", interrupted)
	}
	if len(types) != 3 {
		t.Errorf("yielded %d events (%v), want hop 1's 3 events", len(types), types)
	}
}

func TestStreamCloseStopsContinuation(t *testing.T) {
	// Closing the stream after hop 1's events must not start hop 2.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{events: streamHopEvents("i1", "requires_continuation", "t1", "A")},
		{events: streamHopEvents("i2", "completed", "", "B")},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	res, err := client.Interactions.Create(context.Background(), modelRequest(true))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	es := res.InteractionSSEStreamEvent
	for i := 0; i < 3; i++ {
		if !es.Next() {
			t.Fatalf("Next() = false after %d events: %v", i, es.Err())
		}
	}
	if err := es.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if es.Next() {
		t.Errorf("Next() = true after Close()")
	}
	if n := server.requestCount(); n != 1 {
		t.Errorf("requests = %d, want 1 (no hop after Close)", n)
	}
}

func TestOptionsAreForwardedToEveryHop(t *testing.T) {
	// Request options (e.g. headers) apply to each hop, and the request's
	// objects are not mutated by the SDK.
	server := &scriptedServer{t: t, hops: []scriptedHop{
		{body: unaryHop("i1", "requires_continuation", "t1", "A")},
		{body: unaryHop("i2", "completed", "", "B")},
	}}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	request := modelRequest(false)
	request.Body.CreateModelInteraction.GenerationConfig = &interactions.GenerationConfig{Seed: Int(7)}
	res, err := client.Interactions.Create(context.Background(), request,
		operations.WithSetHeaders(map[string]string{"X-Test": "yes"}))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := deref(res.Interaction.OutputText); got != "AB" {
		t.Errorf("OutputText = %q, want AB", got)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	for i, h := range server.headers {
		if h.Get("X-Test") != "yes" {
			t.Errorf("request %d: X-Test header = %q, want yes", i+1, h.Get("X-Test"))
		}
	}
	for i, r := range server.requests {
		gc, _ := r["generation_config"].(map[string]any)
		if _, ok := gc["max_output_tokens"]; gc["seed"] != float64(7) || ok {
			t.Errorf("request %d: generation_config = %v, want seed 7 and max_output_tokens unset", i+1, gc)
		}
	}
	if gc := request.Body.CreateModelInteraction.GenerationConfig; gc.Seed == nil || *gc.Seed != 7 || gc.MaxOutputTokens != nil {
		t.Errorf("the caller's GenerationConfig was mutated: %+v", gc)
	}
	if request.Body.CreateModelInteraction.ContinuationToken != nil || request.Body.CreateModelInteraction.PreviousInteractionID != nil {
		t.Errorf("the caller's request body was mutated: %+v", request.Body.CreateModelInteraction)
	}
}

func TestWithAutomaticContinuationIsRejectedByOtherOperations(t *testing.T) {
	server := &scriptedServer{t: t}
	ts := httptest.NewServer(server)
	defer ts.Close()
	client := newTestClient(ts)

	_, err := client.Interactions.Get(context.Background(), operations.GetInteractionByIDRequest{ID: "i1"},
		operations.WithAutomaticContinuation(false))
	if !errors.Is(err, operations.ErrUnsupportedOption) {
		t.Errorf("Get() with WithAutomaticContinuation error = %v, want ErrUnsupportedOption", err)
	}
	if n := server.requestCount(); n != 0 {
		t.Errorf("requests = %d, want 0", n)
	}
}

func TestMergeUsagesSumsCountsAndBreakdowns(t *testing.T) {
	text := interactions.ResponseModalityText
	image := interactions.ResponseModalityImage
	first := &interactions.Usage{
		TotalInputTokens:  Int(10),
		TotalOutputTokens: Int(5),
		TotalTokens:       Int(15),
		OutputTokensByModality: []interactions.ModalityTokens{
			{Modality: &text, Tokens: Int(5)},
		},
	}
	second := &interactions.Usage{
		TotalInputTokens:   Int(15),
		TotalOutputTokens:  Int(3),
		TotalThoughtTokens: Int(2),
		TotalTokens:        Int(20),
		InitialInputTokens: Int(10),
		OutputTokensByModality: []interactions.ModalityTokens{
			{Modality: &text, Tokens: Int(2)},
			{Modality: &image, Tokens: Int(1)},
		},
	}
	merged := mergeUsages([]*interactions.Usage{first, second})
	want := map[string]int{"input": 25, "output": 8, "thought": 2, "total": 35, "initial": 10}
	got := map[string]int{
		"input":   *merged.TotalInputTokens,
		"output":  *merged.TotalOutputTokens,
		"thought": *merged.TotalThoughtTokens,
		"total":   *merged.TotalTokens,
		"initial": *merged.InitialInputTokens,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("merged usage %s = %d, want %d", k, got[k], w)
		}
	}
	if len(merged.OutputTokensByModality) != 2 || *merged.OutputTokensByModality[0].Tokens != 7 || *merged.OutputTokensByModality[1].Tokens != 1 {
		t.Errorf("OutputTokensByModality = %+v, want text 7, image 1", merged.OutputTokensByModality)
	}
	if first.TotalInputTokens == nil || *first.TotalInputTokens != 10 || *first.OutputTokensByModality[0].Tokens != 5 {
		t.Errorf("the first hop's usage was mutated: %+v", first)
	}
}
