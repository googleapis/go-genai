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

//go:build ignore_vet

// Automatic continuation: a long unary decode is transparently continued
// across several server requests ("hops") and merged into one Interaction.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
	"google.golang.org/genai/interactions/models/apierrors"
	gaos_interactions "google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
)

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "true" {
		fmt.Println("Interactions API is not yet supported on Vertex AI")
		return
	}

	fmt.Println("Using Gemini Developer API")

	body := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model("gemini-flash-latest"),
		Input: genai.Ptr(gaos_interactions.NewInteractionsInput(
			"Write a very long, detailed story about a lighthouse keeper.")),
	})

	// Automatic continuation is on by default: if the model stops at the
	// per-request decode limit (status requires_continuation), the SDK sends
	// further requests and merges the hops.
	res, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body})
	if err != nil {
		// A failure after some hops completed keeps the work done so far.
		var interrupted *apierrors.ContinuationInterruptedError
		if errors.As(err, &interrupted) {
			fmt.Printf("Interrupted after %d hop(s): %v\n", len(interrupted.ContinuationIDs), interrupted.Cause)
			if interrupted.PartialInteraction != nil && interrupted.PartialInteraction.OutputText != nil {
				fmt.Printf("Partial output: %s\n", *interrupted.PartialInteraction.OutputText)
			}
			// Resume manually: continuation_token + previous_interaction_id
			// (store=true) and no input.
			fmt.Printf("Resume with continuation_token=%q previous_interaction_id=%q\n",
				interrupted.ContinuationToken, interrupted.InteractionID)
			return
		}
		log.Fatal(err)
	}

	interaction := res.Interaction
	fmt.Printf("Status: %s\n", interaction.Status)
	// Every hop's id, in order; the last one is interaction.ID and can be used
	// as previous_interaction_id for the next turn.
	fmt.Printf("Hops: %v\n", interaction.ContinuationIds)
	if interaction.Usage != nil && interaction.Usage.TotalOutputTokens != nil {
		fmt.Printf("Output tokens (summed across hops): %d\n", *interaction.Usage.TotalOutputTokens)
	}
	if interaction.OutputText != nil {
		fmt.Printf("Output: %s\n", *interaction.OutputText)
	}

	// Opting out returns the first server response as-is; if it stopped at
	// requires_continuation, continue it yourself with its continuation_token.
	raw, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body},
		operations.WithAutomaticContinuation(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Without automatic continuation: status=%s continuation_token set=%v\n",
		raw.Interaction.Status, raw.Interaction.ContinuationToken != nil)
}
