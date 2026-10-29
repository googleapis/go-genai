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

// Automatic continuation for Interactions.Create.
//
// A foreground model interaction stops at the per-request decode token limit
// with status "requires_continuation" and a continuation_token. With
// automatic continuation enabled (the default), Create resumes the decode
// across several requests ("hops") and, for unary calls, merges the hops into
// one Interaction. This file is the hand-written counterpart of the Python
// SDK's _interactions_continuation.py; the generated per-request call is
// createOnce in interactions.go.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"google.golang.org/genai/interactions/models/apierrors"
	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
	"google.golang.org/genai/interactions/types/stream"
)

// MaxStreamReconnectsWithoutProgress is the maximum number of consecutive
// reconnects after a mid-hop disconnect that made no progress (no new
// checkpoint token since the hop's start token).
const MaxStreamReconnectsWithoutProgress = 3

// ErrAutomaticContinuationWithBackground is returned by Create, before any
// request is sent, when WithAutomaticContinuation(false) is combined with
// background=true: background interactions are always continued by the
// server.
var ErrAutomaticContinuationWithBackground = errors.New("interactions: WithAutomaticContinuation(false) cannot be used with background=true")

const incompleteContinuationWarning = "Interaction stopped before completion (status='requires_continuation') " +
	"with a continuation_token because the per-request decode token limit was " +
	"reached. To resume decoding automatically, leave automatic continuation " +
	"enabled (note: each continuation hop re-prefills and bills input tokens), " +
	"or use background=true (the most cost-efficient option for server-managed " +
	"continuation), or resume manually by passing continuation_token to " +
	"Interactions.Create (with previous_interaction_id set to this " +
	"interaction's id and no input, or with the original input when store=false)."

var incompleteContinuationWarnOnce sync.Once

func warnIncompleteContinuation() {
	incompleteContinuationWarnOnce.Do(func() {
		log.Printf("WARNING: %s", incompleteContinuationWarning)
	})
}

// Create creates a new interaction.
//
// By default the SDK automatically continues the interaction: a foreground
// model interaction that stops at the per-request decode token limit (status
// "requires_continuation" with a continuation_token) is resumed with further
// requests ("hops") until it completes, becomes "incomplete" (e.g. reaches
// generation_config.max_output_tokens), requires action, or fails. Pass
// operations.WithAutomaticContinuation(false) to disable it and get the
// server response as-is (so you can drive continuation yourself).
//
//   - For stream=true, the events of all hops are yielded sequentially,
//     verbatim, on the returned event stream (each hop's interaction.created
//     and intermediate interaction.completed with status
//     "requires_continuation" included). If a hop's connection drops before
//     any output was yielded since its latest checkpoint token, the hop is
//     resumed from that token (at most MaxStreamReconnectsWithoutProgress
//     times without progress).
//   - For stream=false, each hop is a separate interaction on the server that
//     only contains that hop's output. The SDK merges the hops into one
//     Interaction: Steps are concatenated in hop order, Usage token counts are
//     summed across hops (with Usage.InitialInputTokens set to the first hop's
//     input tokens), and all other fields (ID, Status, ...) come from the
//     final hop; the final hop's ID can be passed as previous_interaction_id
//     to continue the conversation. ContinuationIds lists the ids of every hop
//     in order (the last one is ID). PreviousInteractionID, Created and Input
//     come from the first hop. Note that Interactions.Get and
//     Interactions.Delete act on a single hop.
//   - With store=true (the default), each continuation hop sends the hop that
//     issued the continuation_token as previous_interaction_id and omits input
//     (the server already has it). With store=false, the original input is
//     resent with the token.
//   - Client-side continuation is skipped for background=true and agent
//     interactions, whose loops are managed by the server. background=true
//     with WithAutomaticContinuation(false) returns
//     ErrAutomaticContinuationWithBackground.
//   - Timeouts: the ctx deadline (or cancellation) bounds the whole call,
//     across hops: no further hop starts once it has passed, and a running
//     hop or stream is cut. operations.WithOperationTimeout, the SDK-level
//     WithTimeout and the retry configuration apply to each hop (HTTP request)
//     separately; transient errors on a continuation hop retry that hop using
//     the latest continuation_token without restarting from the initial
//     request.
//   - Billing note: with background=false, each continuation hop re-prefills
//     the conversation and generated prefix, so input tokens are billed on
//     every hop (and summed in Usage.TotalInputTokens). If an unrecoverable
//     error occurs after one or more hops have completed, those hops are still
//     billed. For server-managed continuation without client-side hop loops,
//     use background=true (with store=true).
//   - Errors: a failure on the first request is returned as usual. A failure
//     after progress (a later hop, or a stream after a resumable checkpoint)
//     is returned as an *apierrors.ContinuationInterruptedError wrapping the
//     failure, carrying ContinuationToken, InteractionID and ContinuationIDs,
//     plus PartialInteraction (the completed hops merged, for stream=false),
//     so the output already produced is not lost and the call can be resumed
//     manually. For streams the error is reported by the event stream's Err
//     method. Running out of the ctx deadline after progress is reported the
//     same way (wrapping context.DeadlineExceeded).
func (s *Interactions) Create(ctx context.Context, request operations.CreateInteractionRequest, opts ...operations.Option) (*operations.CreateInteractionResponse, error) {
	o := operations.Options{}
	supportedOptions := []string{
		operations.SupportedOptionRetries,
		operations.SupportedOptionTimeout,
		operations.SupportedOptionAcceptHeaderOverride,
		operations.SupportedOptionAutomaticContinuation,
	}
	for _, opt := range opts {
		if err := opt(&o, supportedOptions...); err != nil {
			return nil, fmt.Errorf("error applying option: %w", err)
		}
	}

	prepared, err := prepareContinuation(request, o.AutomaticContinuation)
	if err != nil {
		return nil, err
	}

	if !prepared.enabled {
		res, err := s.createOnce(ctx, prepared.request, opts...)
		if err != nil {
			return nil, err
		}
		if prepared.callerToken == "" {
			if res.InteractionSSEStreamEvent != nil {
				res.InteractionSSEStreamEvent = wrapStreamIncompleteWarning(res.InteractionSSEStreamEvent)
			} else if res.Interaction != nil && nextUnaryHopToken(res.Interaction, "") != "" {
				warnIncompleteContinuation()
			}
		}
		return res, nil
	}

	if prepared.stream {
		return s.createStreamWithContinuation(ctx, prepared, opts)
	}
	return s.createUnaryWithContinuation(ctx, prepared, opts)
}

// continuationRequest is a prepared Create request and its per-hop builder.
type continuationRequest struct {
	// request is the initial hop's request.
	request operations.CreateInteractionRequest
	// model is request.Body.CreateModelInteraction; nil for agent requests.
	model *interactions.CreateModelInteraction
	// stream is whether the request asks for an event stream.
	stream bool
	// store is whether the interaction is persisted on the server (default true).
	store bool
	// callerToken is the continuation_token the caller passed, if any.
	callerToken string
	// enabled is whether the SDK runs the continuation loop.
	enabled bool
}

// prepareContinuation validates and normalizes Create's arguments.
func prepareContinuation(request operations.CreateInteractionRequest, automaticContinuation *bool) (*continuationRequest, error) {
	p := &continuationRequest{request: request, store: true}
	var background bool
	switch {
	case request.Body.CreateModelInteraction != nil:
		m := request.Body.CreateModelInteraction
		background = m.Background != nil && *m.Background
		p.stream = m.Stream != nil && *m.Stream
		p.store = m.Store == nil || *m.Store
		if m.ContinuationToken != nil {
			p.callerToken = *m.ContinuationToken
		}
		p.model = m
	case request.Body.CreateAgentInteraction != nil:
		a := request.Body.CreateAgentInteraction
		background = a.Background != nil && *a.Background
		p.stream = a.Stream != nil && *a.Stream
		if a.ContinuationToken != nil {
			p.callerToken = *a.ContinuationToken
		}
	}

	if background && automaticContinuation != nil && !*automaticContinuation {
		return nil, ErrAutomaticContinuationWithBackground
	}

	// Agent interactions (server-managed loops) and background interactions
	// are never continued by the SDK.
	p.enabled = (automaticContinuation == nil || *automaticContinuation) && !background && p.model != nil
	return p, nil
}

// hopRequest returns the request for a continuation hop resuming from token.
//
// With store enabled (the default) the server rebuilds the context from the
// stored interaction that issued token, so the hop sends that interaction
// (tokenOwnerID) as previous_interaction_id (the server checks the token
// belongs to it) and omits input. With store=false nothing is stored, so the
// original input is resent with the token instead.
//
// tokenOwnerID is "" when the token did not come from an earlier hop (e.g.
// the caller's own continuation_token); the original request is then resent
// as-is with token.
func (p *continuationRequest) hopRequest(token, tokenOwnerID string) operations.CreateInteractionRequest {
	req := p.request
	m := *p.model
	m.ContinuationToken = &token
	if tokenOwnerID != "" && p.store {
		m.PreviousInteractionID = &tokenOwnerID
		m.Input = nil
	}
	req.Body = operations.NewCreateInteractionRequestBody(m)
	return req
}

// isAPIError reports whether err is an API (HTTP status or response
// validation) error, as opposed to a transport error.
func isAPIError(err error) bool {
	var apiErr *apierrors.APIError
	var clientErr *apierrors.CreateInteractionClientError
	var serverErr *apierrors.CreateInteractionServerError
	var validationErr *apierrors.ResponseValidationError
	return errors.As(err, &apiErr) || errors.As(err, &clientErr) || errors.As(err, &serverErr) || errors.As(err, &validationErr)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// --- Unary ---

// nextUnaryHopToken returns the token to continue from after hop, or "" to
// stop: the hop must have status requires_continuation and a non-empty
// continuation_token different from sentToken (the token the hop was started
// with: continuing from the same token would loop forever without progress).
func nextUnaryHopToken(hop *interactions.Interaction, sentToken string) string {
	if hop == nil || hop.Status != interactions.InteractionStatusRequiresContinuation {
		return ""
	}
	token := deref(hop.ContinuationToken)
	if token == "" || token == sentToken {
		return ""
	}
	return token
}

func (s *Interactions) createUnaryWithContinuation(ctx context.Context, p *continuationRequest, opts []operations.Option) (*operations.CreateInteractionResponse, error) {
	res, err := s.createOnce(ctx, p.request, opts...)
	if err != nil {
		return nil, err
	}
	if res.Interaction == nil {
		return res, nil
	}
	hops := []*interactions.Interaction{res.Interaction}
	sentToken := p.callerToken
	for {
		last := hops[len(hops)-1]
		token := nextUnaryHopToken(last, sentToken)
		if token == "" {
			break
		}
		var hop *operations.CreateInteractionResponse
		if err := ctx.Err(); err != nil {
			return nil, interruptedUnary(err, hops, token, p.store)
		}
		hop, err = s.createOnce(ctx, p.hopRequest(token, deref(last.ID)), opts...)
		if err != nil {
			return nil, interruptedUnary(err, hops, token, p.store)
		}
		if hop.Interaction == nil {
			return nil, interruptedUnary(errors.New("continuation hop returned no interaction"), hops, token, p.store)
		}
		res = hop
		hops = append(hops, hop.Interaction)
		sentToken = token
	}
	res.Interaction = mergeUnaryHops(hops)
	return res, nil
}

// interruptedUnary builds the error for a continuation hop that failed after
// hops completed.
func interruptedUnary(cause error, hops []*interactions.Interaction, token string, store bool) error {
	interactionID := ""
	if store {
		interactionID = deref(hops[len(hops)-1].ID)
	}
	return &apierrors.ContinuationInterruptedError{
		Message: fmt.Sprintf("automatic continuation was interrupted after %d completed hop(s); "+
			"the partial interaction (status requires_continuation) and its continuation_token can be used to resume", len(hops)),
		Cause:              cause,
		PartialInteraction: mergeUnaryHops(hops),
		ContinuationToken:  token,
		InteractionID:      interactionID,
		ContinuationIDs:    mergeContinuationIDs(hops),
	}
}

// mergeContinuationIDs returns the ordered, de-duplicated ids of every hop:
// each hop's server-populated continuation_ids (if any) followed by its id.
func mergeContinuationIDs(hops []*interactions.Interaction) []string {
	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, hop := range hops {
		for _, id := range hop.ContinuationIds {
			add(id)
		}
		add(deref(hop.ID))
	}
	return ids
}

// mergeUnaryHops merges automatic continuation hops into a single interaction.
//
// Each hop is its own interaction on the server, and only contains the output
// generated during that hop. The merged result is the final hop (so ID,
// Status, etc. refer to it) with the steps of every hop concatenated, usage
// summed across hops, and ContinuationIds listing every hop's id. Fields
// describing the turn's request (PreviousInteractionID, Created, Input) come
// from the first hop: later hops point back at the hop before them, not at
// the caller's previous turn.
func mergeUnaryHops(hops []*interactions.Interaction) *interactions.Interaction {
	merged := *hops[len(hops)-1]
	var steps []interactions.Step
	for i, hop := range hops {
		hopSteps := hop.Steps
		if i > 0 {
			// Later hops continue the same turn; a repeated user_input step
			// would duplicate the input and cut OutputText (which stops at the
			// latest user_input step) short.
			for len(hopSteps) > 0 && hopSteps[0].Type == interactions.StepTypeUserInput {
				hopSteps = hopSteps[1:]
			}
		}
		steps = append(steps, hopSteps...)
	}
	merged.Steps = steps
	merged.ContinuationIds = mergeContinuationIDs(hops)
	if len(hops) > 1 {
		merged.PreviousInteractionID = hops[0].PreviousInteractionID
		merged.Created = hops[0].Created
		merged.Input = hops[0].Input
	}
	var usages []*interactions.Usage
	for _, hop := range hops {
		if hop.Usage != nil {
			usages = append(usages, hop.Usage)
		}
	}
	if usage := mergeUsagesWithFirst(usages, hops[0].Usage); usage != nil {
		merged.Usage = usage
	}
	merged.OutputText = nil
	merged.OutputImage = nil
	merged.OutputAudio = nil
	merged.OutputVideo = nil
	merged.PopulateOutputProperties()
	return &merged
}

// mergeUsages sums token usage across hops, keeping the first hop's input
// tokens as InitialInputTokens (falling back to its TotalInputTokens).
//
// Every integer count is summed, and the per-modality / per-tool breakdowns
// are summed per key, so the merged usage matches what was billed across the
// hops. Other fields come from the last hop.
func mergeUsages(usages []*interactions.Usage) *interactions.Usage {
	if len(usages) == 0 {
		return nil
	}
	return mergeUsagesWithFirst(usages, usages[0])
}

func mergeUsagesWithFirst(usages []*interactions.Usage, first *interactions.Usage) *interactions.Usage {
	if len(usages) == 0 {
		return nil
	}
	merged := *usages[len(usages)-1]
	mv := reflect.ValueOf(&merged).Elem()
	for i := 0; i < mv.NumField(); i++ {
		field := mv.Type().Field(i)
		if !field.IsExported() || field.Name == "InitialInputTokens" {
			continue
		}
		switch fv := mv.Field(i); fv.Interface().(type) {
		case *int:
			var sum int
			present := false
			for _, u := range usages {
				if v := reflect.ValueOf(u).Elem().Field(i).Interface().(*int); v != nil {
					sum += *v
					present = true
				}
			}
			if present {
				fv.Set(reflect.ValueOf(&sum))
			}
		case []interactions.ModalityTokens:
			var lists [][]interactions.ModalityTokens
			for _, u := range usages {
				lists = append(lists, reflect.ValueOf(u).Elem().Field(i).Interface().([]interactions.ModalityTokens))
			}
			fv.Set(reflect.ValueOf(mergeModalityTokens(lists)))
		case []interactions.GroundingToolCount:
			var lists [][]interactions.GroundingToolCount
			for _, u := range usages {
				lists = append(lists, reflect.ValueOf(u).Elem().Field(i).Interface().([]interactions.GroundingToolCount))
			}
			fv.Set(reflect.ValueOf(mergeGroundingToolCounts(lists)))
		}
	}
	switch {
	case first != nil && first.InitialInputTokens != nil:
		merged.InitialInputTokens = Int(*first.InitialInputTokens)
	case first != nil && first.TotalInputTokens != nil:
		merged.InitialInputTokens = Int(*first.TotalInputTokens)
	default:
		merged.InitialInputTokens = nil
	}
	return &merged
}

func mergeModalityTokens(lists [][]interactions.ModalityTokens) []interactions.ModalityTokens {
	var merged []interactions.ModalityTokens
	index := map[interactions.ResponseModality]int{}
	for _, items := range lists {
		for _, item := range items {
			var key interactions.ResponseModality
			if item.Modality != nil {
				key = *item.Modality
			}
			i, ok := index[key]
			if !ok {
				index[key] = len(merged)
				merged = append(merged, item)
				continue
			}
			if merged[i].Tokens != nil && item.Tokens != nil {
				merged[i].Tokens = Int(*merged[i].Tokens + *item.Tokens)
			}
		}
	}
	return merged
}

func mergeGroundingToolCounts(lists [][]interactions.GroundingToolCount) []interactions.GroundingToolCount {
	var merged []interactions.GroundingToolCount
	index := map[interactions.GroundingToolCountType]int{}
	for _, items := range lists {
		for _, item := range items {
			var key interactions.GroundingToolCountType
			if item.Type != nil {
				key = *item.Type
			}
			i, ok := index[key]
			if !ok {
				index[key] = len(merged)
				merged = append(merged, item)
				continue
			}
			if merged[i].Count != nil && item.Count != nil {
				merged[i].Count = Int(*merged[i].Count + *item.Count)
			}
		}
	}
	return merged
}

// --- Streaming ---

type sseEvent = interactions.InteractionSSEStreamEvent

// isResumableStreamHopStatus reports whether a hop that ended with status can
// be resumed from a token ("" is a hop that ended before any status arrived).
func isResumableStreamHopStatus(status string) bool {
	switch status {
	case "", string(interactions.InteractionStatusInProgress), string(interactions.InteractionStatusRequiresContinuation):
		return true
	}
	return false
}

// hopContinuationState extracts the (status, continuation_token) updates from
// one SSE event. The interaction.completed snapshot is authoritative: a
// missing token clears the hop's seed token and any earlier checkpoint.
func hopContinuationState(ev *sseEvent, status, token string) (string, string) {
	d := ev.Data
	switch d.Type {
	case interactions.InteractionSSEEventTypeError:
		return "error", ""
	case interactions.InteractionSSEEventTypeInteractionCreated:
		if e := d.InteractionCreatedEvent; e != nil {
			if e.Interaction.Status != "" {
				status = string(e.Interaction.Status)
			}
			if t := deref(e.Interaction.ContinuationToken); t != "" {
				token = t
			}
		}
	case interactions.InteractionSSEEventTypeInteractionCompleted:
		if e := d.InteractionCompletedEvent; e != nil {
			if e.Interaction.Status != "" {
				status = string(e.Interaction.Status)
			}
			token = deref(e.Interaction.ContinuationToken)
		}
	case interactions.InteractionSSEEventTypeInteractionStatusUpdate:
		if e := d.InteractionStatusUpdate; e != nil {
			if e.Status != "" {
				status = string(e.Status)
			}
			if t := deref(e.ContinuationToken); t != "" {
				token = t
			}
		}
	}
	return status, token
}

// eventInteractionID returns the id of the interaction (hop) an SSE event
// belongs to, if present.
func eventInteractionID(ev *sseEvent) string {
	d := ev.Data
	switch d.Type {
	case interactions.InteractionSSEEventTypeInteractionCreated:
		if d.InteractionCreatedEvent != nil {
			return d.InteractionCreatedEvent.Interaction.ID
		}
	case interactions.InteractionSSEEventTypeInteractionCompleted:
		if d.InteractionCompletedEvent != nil {
			return d.InteractionCompletedEvent.Interaction.ID
		}
	case interactions.InteractionSSEEventTypeInteractionStatusUpdate:
		if d.InteractionStatusUpdate != nil {
			return d.InteractionStatusUpdate.InteractionID
		}
	}
	return ""
}

// hasContentSinceCheckpoint tracks whether output was yielded since the latest
// resume point. Events that carry no model output (interaction.created,
// interaction.status_update) don't count; a checkpoint token or the terminal
// snapshot's token is a fresh resume point.
func hasContentSinceCheckpoint(ev *sseEvent, contentSinceCheckpoint bool) bool {
	d := ev.Data
	switch d.Type {
	case interactions.InteractionSSEEventTypeInteractionStatusUpdate:
		if d.InteractionStatusUpdate != nil && deref(d.InteractionStatusUpdate.ContinuationToken) != "" {
			return false
		}
		return contentSinceCheckpoint
	case interactions.InteractionSSEEventTypeInteractionCompleted:
		return false
	case interactions.InteractionSSEEventTypeInteractionCreated:
		return contentSinceCheckpoint
	}
	return true
}

// streamHop is what was observed during one hop (HTTP request) of a
// continuation stream.
type streamHop struct {
	// startToken and startTokenOwnerID are the continuation_token and owner id
	// this hop was requested with.
	startToken        string
	startTokenOwnerID string
	// token is seeded with startToken so a disconnect before the hop's first
	// checkpoint retries from the hop start.
	token string
	// tokenOwnerID is the interaction that issued token. With store=true the
	// server only accepts a token together with the interaction that issued
	// it as previous_interaction_id, so a retry of the seed token keeps the
	// seed's owner even after this hop's own interaction was created.
	tokenOwnerID string
	// interactionID is this hop's own interaction id, once known.
	interactionID string
	// status is the latest status observed during the hop ("" if none yet).
	status string
	// opened is whether the HTTP stream was opened (response headers
	// received).
	opened bool
	// contentSinceCheckpoint is whether output events were yielded since the
	// latest checkpoint token.
	contentSinceCheckpoint bool
}

// streamContinuationState is the state of a streaming Create call with
// continuation. Per-hop state lives in a streamHop replaced wholesale by
// startHop, so per-hop fields cannot leak across hops. Loop state (the next
// resume token and owner id, the no-progress reconnect counter, and the
// pending transport error) persists across hops.
type streamContinuationState struct {
	nextToken        string
	nextTokenOwnerID string
	// initialToken is the caller's own token: a failure that leaves nothing
	// newer to resume from is not "after progress" and is returned as-is.
	initialToken string
	store        bool
	// hopIDs are the ids of every hop's interaction, in order.
	hopIDs               []string
	noProgressReconnects int
	// lastErr is the latest transport error not followed by progress;
	// returned if the loop gives up so the failure is not hidden by later
	// clean closes.
	lastErr error
	hop     streamHop
}

func newStreamContinuationState(initialToken string, store bool) *streamContinuationState {
	return &streamContinuationState{
		nextToken:    initialToken,
		initialToken: initialToken,
		store:        store,
		hop:          streamHop{startToken: initialToken, token: initialToken},
	}
}

// startHop starts a new hop and returns the (token, tokenOwnerID) to request.
func (st *streamContinuationState) startHop(opened bool) (string, string) {
	st.hop = streamHop{
		startToken:        st.nextToken,
		startTokenOwnerID: st.nextTokenOwnerID,
		token:             st.nextToken,
		tokenOwnerID:      st.nextTokenOwnerID,
		opened:            opened,
	}
	return st.nextToken, st.nextTokenOwnerID
}

// observe records one streamed SSE event.
func (st *streamContinuationState) observe(ev *sseEvent) {
	hop := &st.hop
	hop.opened = true
	if id := eventInteractionID(ev); id != "" {
		hop.interactionID = id
		known := false
		for _, h := range st.hopIDs {
			if h == id {
				known = true
				break
			}
		}
		if !known {
			st.hopIDs = append(st.hopIDs, id)
		}
	}
	previousToken := hop.token
	hop.status, hop.token = hopContinuationState(ev, hop.status, hop.token)
	if hop.token != "" && hop.token != previousToken {
		hop.tokenOwnerID = hop.interactionID
	}
	hop.contentSinceCheckpoint = hasContentSinceCheckpoint(ev, hop.contentSinceCheckpoint)
}

func (st *streamContinuationState) canResumeHop() bool {
	// Resuming from the checkpoint would re-generate (and duplicate) any
	// output yielded since it.
	hop := &st.hop
	return !hop.contentSinceCheckpoint && hop.token != "" && isResumableStreamHopStatus(hop.status)
}

// fail returns the error to report when the stream gives up with err: if the
// server issued a resumable token during this call (a completed
// requires_continuation hop, or a mid-hop checkpoint), err is wrapped in an
// *apierrors.ContinuationInterruptedError carrying that token; otherwise err
// is returned as-is.
func (st *streamContinuationState) fail(err error) error {
	hop := &st.hop
	token := ""
	if hop.token != "" && isResumableStreamHopStatus(hop.status) {
		token = hop.token
	}
	if token == "" || token == st.initialToken {
		return err
	}
	interactionID := ""
	if st.store {
		interactionID = hop.tokenOwnerID
	}
	return &apierrors.ContinuationInterruptedError{
		Message: "automatic continuation of the stream was interrupted; the continuation_token can be used to resume " +
			"(output yielded after a mid-hop checkpoint is re-generated)",
		Cause:             err,
		ContinuationToken: token,
		InteractionID:     interactionID,
		ContinuationIDs:   append([]string(nil), st.hopIDs...),
	}
}

// endHop ends the current hop and reports whether another hop should start.
// It returns hopErr (or the pending prior disconnect error) when the stream
// cannot safely resume or exceeds MaxStreamReconnectsWithoutProgress, wrapped
// by fail. terminal marks hopErr as not retryable (the call's context expired
// or was cancelled).
func (st *streamContinuationState) endHop(hopErr error, terminal bool) (bool, error) {
	hop := &st.hop
	if hopErr != nil && terminal {
		return false, st.fail(hopErr)
	}
	if hopErr != nil && !hop.opened && isAPIError(hopErr) {
		// Pre-stream HTTP status or validation errors (already retried by the
		// retry configuration) propagate immediately; only transport errors
		// resume.
		return false, st.fail(hopErr)
	}
	if !st.canResumeHop() {
		if hopErr != nil {
			return false, st.fail(hopErr)
		}
		return false, nil
	}
	if hop.status == string(interactions.InteractionStatusRequiresContinuation) && hop.token == hop.startToken {
		// The hop finished but handed back the token it was started with:
		// resuming would loop forever without progress.
		if hopErr != nil {
			return false, st.fail(hopErr)
		}
		return false, nil
	}

	switch {
	case hop.status == string(interactions.InteractionStatusRequiresContinuation):
		// Clean hop boundary (per-request decode limit reached).
		st.noProgressReconnects = 0
	case hop.token != hop.startToken:
		// Dropped mid-hop after a new checkpoint token was received.
		st.noProgressReconnects = 0
	default:
		// Dropped mid-hop before any new checkpoint: retry the hop's own token.
		st.noProgressReconnects++
	}
	if st.noProgressReconnects == 0 {
		st.lastErr = nil
	}
	if hopErr != nil {
		st.lastErr = hopErr
	}
	if st.noProgressReconnects > MaxStreamReconnectsWithoutProgress {
		if st.lastErr != nil {
			return false, st.fail(st.lastErr)
		}
		return false, nil
	}
	st.nextToken = hop.token
	st.nextTokenOwnerID = hop.tokenOwnerID
	return true, nil
}

// streamContinuation drives the hops of a streaming Create call and presents
// them as one event stream.
type streamContinuation struct {
	s       *Interactions
	ctx     context.Context
	p       *continuationRequest
	opts    []operations.Option
	state   *streamContinuationState
	current *stream.EventStream[sseEvent]
	done    bool
	mu      sync.Mutex
}

func (s *Interactions) createStreamWithContinuation(ctx context.Context, p *continuationRequest, opts []operations.Option) (*operations.CreateInteractionResponse, error) {
	res, err := s.createOnce(ctx, p.request, opts...)
	if err != nil {
		return nil, err
	}
	if res.InteractionSSEStreamEvent == nil {
		return res, nil
	}
	c := &streamContinuation{
		s:       s,
		ctx:     ctx,
		p:       p,
		opts:    opts,
		state:   newStreamContinuationState(p.callerToken, p.store),
		current: res.InteractionSSEStreamEvent,
	}
	c.state.startHop(true)
	res.InteractionSSEStreamEvent = stream.NewEventStreamFromFunc(c.next, c.close)
	return res, nil
}

// next returns the next event across hops, (nil, nil, nil) at the end of the
// stream, or the error ending it.
func (c *streamContinuation) next() (*sseEvent, json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for !c.done {
		if c.current == nil {
			token, owner := c.state.startHop(false)
			var hopErr error
			if err := c.ctx.Err(); err != nil {
				hopErr = err
			} else {
				res, err := c.s.createOnce(c.ctx, c.p.hopRequest(token, owner), c.opts...)
				switch {
				case err != nil:
					hopErr = err
				case res.InteractionSSEStreamEvent == nil:
					// Not a stream: nothing more to yield.
					c.done = true
					return nil, nil, nil
				default:
					c.current = res.InteractionSSEStreamEvent
					c.state.hop.opened = true
					continue
				}
			}
			if err := c.endHop(hopErr); err != nil {
				return nil, nil, err
			}
			continue
		}

		if c.current.Next() {
			ev := c.current.Value()
			c.state.observe(ev)
			return ev, c.current.RawValue(), nil
		}
		hopErr := c.current.Err()
		_ = c.current.Close()
		c.current = nil
		if err := c.endHop(hopErr); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

// endHop ends the current hop; it sets done when the stream is over and
// returns the error to report, if any.
func (c *streamContinuation) endHop(hopErr error) error {
	more, err := c.state.endHop(hopErr, c.ctx.Err() != nil)
	if err != nil {
		c.done = true
		return err
	}
	if !more {
		c.done = true
	}
	return nil
}

func (c *streamContinuation) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.done = true
	if c.current != nil {
		err := c.current.Close()
		c.current = nil
		return err
	}
	return nil
}

// wrapStreamIncompleteWarning wraps an uncontinued stream to emit a one-time
// warning when it ends at requires_continuation with a continuation_token.
func wrapStreamIncompleteWarning(es *stream.EventStream[sseEvent]) *stream.EventStream[sseEvent] {
	var status, token string
	next := func() (*sseEvent, json.RawMessage, error) {
		if es.Next() {
			ev := es.Value()
			status, token = hopContinuationState(ev, status, token)
			return ev, es.RawValue(), nil
		}
		err := es.Err()
		if err == nil && status == string(interactions.InteractionStatusRequiresContinuation) && token != "" {
			warnIncompleteContinuation()
		}
		return nil, nil, err
	}
	return stream.NewEventStreamFromFunc(next, es.Close)
}
