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

	body := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model:  gaos_interactions.Model("gemini-flash-latest"),
		Input:  gaos_interactions.NewInteractionsInput("Tell me a story"),
		Stream: ptr(true),
	})

	res, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body})
	if err != nil {
		log.Fatal(err)
	}

	stream := res.InteractionSSEStreamEvent
	if stream == nil {
		log.Fatal("Response was not a stream")
	}
	defer stream.Close()

	for stream.Next() {
		event := stream.Value()
		if event != nil {
			stepDelta := event.GetDataStepDelta()
			if stepDelta != nil {
				textDelta := stepDelta.GetDeltaText()
				if textDelta != nil {
					fmt.Print(textDelta.GetText())
					os.Stdout.Sync()
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		fmt.Printf("\nError during streaming: %v\n", err)
	}
	fmt.Println()
}

func ptr[T any](v T) *T {
	return &v
}
