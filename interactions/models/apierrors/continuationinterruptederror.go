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

package apierrors

import (
	"fmt"

	"google.golang.org/genai/interactions/models/interactions"
)

// ContinuationInterruptedError is returned when automatic continuation of an
// Interactions.Create call fails after progress.
//
// Interactions.Create with automatic continuation enabled (the default) may
// make several requests ("hops"), each one resuming the decode from the
// previous hop's continuation_token. If a hop fails after the server already
// produced (and billed) output in an earlier hop, or after a resumable
// checkpoint was reached, this error is returned instead of the underlying
// error so the work is not lost. The underlying error is Cause (also returned
// by Unwrap, so errors.Is and errors.As see through it).
//
// To resume, pass ContinuationToken as continuation_token to
// Interactions.Create: with store enabled (the default), also pass
// InteractionID as previous_interaction_id and no input; with store=false,
// resend the original input.
type ContinuationInterruptedError struct {
	// Message describes the failure.
	Message string
	// Cause is the underlying error (an API error, a transport error, or the
	// context error when the call's context expired).
	Cause error
	// PartialInteraction is, for unary calls, the interaction merged from the
	// hops that completed (status requires_continuation, with the resumable
	// ContinuationToken), or nil if no hop completed. Always nil for streams,
	// whose events were already yielded.
	PartialInteraction *interactions.Interaction
	// ContinuationToken is the latest resumable continuation_token. For
	// streams this may be a mid-hop checkpoint: resuming from it re-generates
	// any output yielded after the checkpoint.
	ContinuationToken string
	// InteractionID is the id of the interaction that issued
	// ContinuationToken (the value to send as previous_interaction_id), or ""
	// if unknown.
	InteractionID string
	// ContinuationIDs are the ids of every hop created during the call, in
	// order.
	ContinuationIDs []string
}

var _ error = &ContinuationInterruptedError{}

func (e *ContinuationInterruptedError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

// Unwrap returns the underlying error.
func (e *ContinuationInterruptedError) Unwrap() error {
	return e.Cause
}
