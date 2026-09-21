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

	parametersSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"attendees": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of people attending the meeting.",
			},
			"date": map[string]any{
				"type":        "string",
				"description": "Date of the meeting (e.g., 2024-07-29)",
			},
			"time": map[string]any{
				"type":        "string",
				"description": "Time of the meeting (e.g., 15:00)",
			},
			"topic": map[string]any{
				"type":        "string",
				"description": "The subject or topic of the meeting.",
			},
		},
		"required": []string{"attendees", "date", "time", "topic"},
	}

	function := gaos_interactions.Function{
		Name:        ptr("schedule_meeting"),
		Description: ptr("Schedules a meeting with specified attendees at a given time and date."),
		Parameters:  parametersSchema,
	}

	tool := gaos_interactions.NewTool(function)

	body := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model("gemini-flash-latest"),
		Input: ptr(gaos_interactions.NewInteractionsInput("Schedule a meeting for 10/06/2028 at 10 am with Peter and Amir about the Next Gen API")),
		Tools: []gaos_interactions.Tool{tool},
	})

	res, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body})
	if err != nil {
		log.Fatal(err)
	}

	if res.Interaction != nil {
		if res.Interaction.ID != nil {
			fmt.Println("Interaction ID:", *res.Interaction.ID)
		}
		fmt.Println("Status:", res.Interaction.Status)

		if res.Interaction.OutputText != nil && *res.Interaction.OutputText != "" {
			fmt.Println("Output Text:", *res.Interaction.OutputText)
		}
		for _, step := range res.Interaction.Steps {
			if step.FunctionCallStep != nil {
				fmt.Println("Function Call:", step.FunctionCallStep.Name)
				fmt.Println("Arguments:", step.FunctionCallStep.Arguments)
			}
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}
