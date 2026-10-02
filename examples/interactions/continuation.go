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

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
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

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-flash-latest"
	}

	fmt.Println("--- Unary Continuation ---")
	runUnaryContinuation(ctx, client, model)

	fmt.Println("\n--- Streaming Continuation ---")
	runStreamingContinuation(ctx, client, model)
}

func printInteractionText(interaction *gaos_interactions.Interaction) {
	if interaction.OutputText != nil {
		fmt.Print(*interaction.OutputText)
		return
	}
	for _, step := range interaction.GetSteps() {
		if modelOutput := step.ModelOutputStep; modelOutput != nil {
			for _, content := range modelOutput.GetContent() {
				if textContent := content.TextContent; textContent != nil {
					fmt.Print(textContent.Text)
				}
			}
		}
	}
}

func runUnaryContinuation(ctx context.Context, client *genai.Client, model string) {
	body := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model(model),
		Input: genai.Ptr(gaos_interactions.NewInteractionsInput("Write a 500 word story about a robot.")),
	})

	res, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body})
	if err != nil {
		log.Fatal(err)
	}

	interaction := res.Interaction
	if interaction == nil {
		log.Fatal("No interaction returned")
	}

	printInteractionText(interaction)

	for interaction.Status == gaos_interactions.InteractionStatusIncomplete && interaction.ContinuationToken != nil {
		contBody := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
			Model:                 gaos_interactions.Model(model),
			PreviousInteractionID: interaction.ID,
			ContinuationToken:     interaction.ContinuationToken,
		})

		res, err = client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: contBody})
		if err != nil {
			log.Fatal(err)
		}
		interaction = res.Interaction
		if interaction == nil {
			log.Fatal("No interaction returned on continuation")
		}
		printInteractionText(interaction)
	}
	fmt.Printf("\nFinal status: %s\n", interaction.Status)
}

func runStreamingContinuation(ctx context.Context, client *genai.Client, model string) {
	var interactionID *string
	var continuationToken *string

	for {
		reqModel := gaos_interactions.CreateModelInteraction{
			Model:                 gaos_interactions.Model(model),
			PreviousInteractionID: interactionID,
			ContinuationToken:     continuationToken,
			Stream:                genai.Ptr(true),
		}
		if continuationToken == nil {
			reqModel.Input = genai.Ptr(gaos_interactions.NewInteractionsInput("Write a 500 word story about a robot."))
		}

		res, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{
			Body: operations.NewCreateInteractionRequestBody(reqModel),
		})
		if err != nil {
			log.Fatal(err)
		}

		stream := res.InteractionSSEStreamEvent
		if stream == nil {
			log.Fatal("Response was not a stream")
		}

		var status gaos_interactions.InteractionSseEventInteractionStatus
		for stream.Next() {
			event := stream.Value()
			if event == nil {
				continue
			}
			if stepDelta := event.GetDataStepDelta(); stepDelta != nil {
				if textDelta := stepDelta.GetDeltaText(); textDelta != nil {
					fmt.Print(textDelta.GetText())
					os.Stdout.Sync()
				}
			} else if completed := event.GetDataInteractionCompleted(); completed != nil {
				evInteraction := completed.GetInteraction()
				id := evInteraction.GetID()
				interactionID = &id
				status = evInteraction.GetStatus()
				continuationToken = evInteraction.GetContinuationToken()
			}
		}

		if err := stream.Err(); err != nil {
			stream.Close()
			log.Fatalf("\nError during streaming: %v\n", err)
		}
		stream.Close()

		if status != gaos_interactions.InteractionSseEventInteractionStatusIncomplete || continuationToken == nil {
			fmt.Printf("\nFinal stream status: %s\n", status)
			break
		}
	}
}
