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
	"log"
	"strings"
	"sync"

	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
)

const (
	IncompleteContinuationWarning = "Interaction stopped before completion (status='continuation_required') with a continuation_token because the per-request decode token limit was reached. To resume decoding automatically, pass automatic_continuation=true (note: each continuation hop re-prefills and bills input tokens), or use background=true (the most cost-efficient option for server-managed continuation), or pass continuation_token to interactions.create()."
)

var (
	warnIncompleteOnce sync.Once
)

func warnIncompleteContinuation() {
	warnIncompleteOnce.Do(func() {
		log.Println("WARNING:", IncompleteContinuationWarning)
	})
}

func resetContinuationWarningsForTesting() {
	warnIncompleteOnce = sync.Once{}
}

func withOutputText(in *interactions.Interaction) *interactions.Interaction {
	if in == nil {
		return nil
	}
	cp := *in
	if cp.OutputText == nil || *cp.OutputText == "" {
		cp.OutputText = computeOutputText(cp.Steps)
	}
	return &cp
}

func computeOutputText(steps []interactions.Step) *string {
	if len(steps) == 0 {
		return nil
	}

	var textParts []string
	collecting := false

outer:
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]

		if step.Type == interactions.StepTypeUserInput {
			break
		}

		// Skip thought steps without terminating text collection across continuation hops
		if step.Type == interactions.StepTypeThought {
			continue
		}

		if step.Type != interactions.StepTypeModelOutput {
			if collecting {
				break
			}
			continue
		}

		modelOutput := step.ModelOutputStep
		if modelOutput == nil {
			if collecting {
				break
			}
			continue
		}

		contents := modelOutput.GetContent()
		for j := len(contents) - 1; j >= 0; j-- {
			c := contents[j]
			if c.TextContent != nil {
				collecting = true
				textParts = append(textParts, c.TextContent.Text)
			} else if collecting {
				break outer
			}
		}
	}

	if len(textParts) == 0 {
		return nil
	}

	// Reverse since we collected backwards
	for i, j := 0, len(textParts)-1; i < j; i, j = i+1, j-1 {
		textParts[i], textParts[j] = textParts[j], textParts[i]
	}

	joined := strings.Join(textParts, "")
	return &joined
}

func shouldContinueUnary(interaction *interactions.Interaction) (string, bool) {
	if interaction == nil {
		return "", false
	}
	if string(interaction.Status) != "continuation_required" {
		return "", false
	}
	if interaction.ContinuationToken == nil || *interaction.ContinuationToken == "" {
		return "", false
	}
	return *interaction.ContinuationToken, true
}

func isStreaming(body operations.CreateInteractionRequestBody) bool {
	if body.CreateModelInteraction != nil && body.CreateModelInteraction.Stream != nil {
		return *body.CreateModelInteraction.Stream
	}
	if body.CreateAgentInteraction != nil && body.CreateAgentInteraction.Stream != nil {
		return *body.CreateAgentInteraction.Stream
	}
	return false
}

func isBackground(body operations.CreateInteractionRequestBody) bool {
	if body.CreateModelInteraction != nil && body.CreateModelInteraction.Background != nil {
		return *body.CreateModelInteraction.Background
	}
	if body.CreateAgentInteraction != nil && body.CreateAgentInteraction.Background != nil {
		return *body.CreateAgentInteraction.Background
	}
	return false
}

func isStoreFalse(body operations.CreateInteractionRequestBody) bool {
	if body.CreateModelInteraction != nil && body.CreateModelInteraction.Store != nil {
		return !*body.CreateModelInteraction.Store
	}
	if body.CreateAgentInteraction != nil && body.CreateAgentInteraction.Store != nil {
		return !*body.CreateAgentInteraction.Store
	}
	return false
}

func hasContinuationToken(body operations.CreateInteractionRequestBody) bool {
	if body.CreateModelInteraction != nil && body.CreateModelInteraction.ContinuationToken != nil {
		return *body.CreateModelInteraction.ContinuationToken != ""
	}
	if body.CreateAgentInteraction != nil && body.CreateAgentInteraction.ContinuationToken != nil {
		return *body.CreateAgentInteraction.ContinuationToken != ""
	}
	return false
}

func extractHopContinuationState(
	event *interactions.InteractionSSEStreamEvent,
	hopStatus string,
	hopContinuationToken string,
) (string, string) {
	if event == nil {
		return hopStatus, hopContinuationToken
	}
	if event.GetDataError() != nil {
		return "error", ""
	}
	if created := event.GetDataInteractionCreated(); created != nil {
		evInter := created.GetInteraction()
		if evInter.GetStatus() != "" {
			hopStatus = string(evInter.GetStatus())
		}
		if tok := evInter.GetContinuationToken(); tok != nil && *tok != "" {
			hopContinuationToken = *tok
		}
	}
	if statusUpdate := event.GetDataInteractionStatusUpdate(); statusUpdate != nil {
		if statusUpdate.GetStatus() != "" {
			hopStatus = string(statusUpdate.GetStatus())
		}
		if tok := statusUpdate.GetContinuationToken(); tok != nil && *tok != "" {
			hopContinuationToken = *tok
		}
	}
	if completed := event.GetDataInteractionCompleted(); completed != nil {
		evInter := completed.GetInteraction()
		if evInter.GetStatus() != "" {
			hopStatus = string(evInter.GetStatus())
		}
		if tok := evInter.GetContinuationToken(); tok != nil && *tok != "" {
			hopContinuationToken = *tok
		}
	}
	return hopStatus, hopContinuationToken
}

func canResumeStreamHop(hopStatus, hopContinuationToken string) bool {
	if hopContinuationToken == "" {
		return false
	}
	switch hopStatus {
	case "", "in_progress", "continuation_required":
		return true
	default:
		return false
	}
}

func withContinuationToken(body operations.CreateInteractionRequestBody, token string) operations.CreateInteractionRequestBody {
	if body.CreateModelInteraction != nil {
		cp := *body.CreateModelInteraction
		cp.ContinuationToken = &token
		return operations.NewCreateInteractionRequestBody(cp)
	}
	if body.CreateAgentInteraction != nil {
		cp := *body.CreateAgentInteraction
		cp.ContinuationToken = &token
		return operations.NewCreateInteractionRequestBody(cp)
	}
	return body
}
