// Copyright 2025 Google LLC
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

	conversationHistory := []gaos_interactions.Step{
		gaos_interactions.NewStep(gaos_interactions.UserInputStep{
			Content: []gaos_interactions.Content{
				gaos_interactions.NewContent(gaos_interactions.TextContent{Text: "What are the three largest cities in Spain?"}),
			},
		}),
	}

	fmt.Println("User: What are the three largest cities in Spain?")

	body1 := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model("gemini-flash-latest"),
		Input: ptr(gaos_interactions.NewInteractionsInput(conversationHistory)),
		Store: ptr(false),
	})

	res1, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body1})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Model: ")
	if res1.Interaction != nil {
		if res1.Interaction.OutputText != nil {
			fmt.Println(*res1.Interaction.OutputText)
		}

		// Add model response to history
		for _, step := range res1.Interaction.Steps {
			if step.ModelOutputStep != nil {
				conversationHistory = append(conversationHistory, gaos_interactions.NewStep(*step.ModelOutputStep))
			}
		}
	}

	// Add next user message
	conversationHistory = append(conversationHistory, gaos_interactions.NewStep(gaos_interactions.UserInputStep{
		Content: []gaos_interactions.Content{
			gaos_interactions.NewContent(gaos_interactions.TextContent{Text: "What is the most famous landmark in the second one?"}),
		},
	}))

	fmt.Println("\nUser: What is the most famous landmark in the second one?")

	body2 := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model("gemini-flash-latest"),
		Input: ptr(gaos_interactions.NewInteractionsInput(conversationHistory)),
		Store: ptr(false),
	})

	res2, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body2})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Model: ")
	if res2.Interaction != nil && res2.Interaction.OutputText != nil {
		fmt.Println(*res2.Interaction.OutputText)
	}
}

func ptr[T any](v T) *T {
	return &v
}
