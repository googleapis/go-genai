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

	mcpServer := gaos_interactions.MCPServer{
		Name: genai.Ptr("weather_service"),
		URL:  genai.Ptr("https://gemini-api-demos.uc.r.appspot.com/mcp"),
	}

	tool := gaos_interactions.NewTool(mcpServer)

	body := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model:             gaos_interactions.Model("gemini-flash-latest"),
		Input:             genai.Ptr(gaos_interactions.NewInteractionsInput("What is the temperature today in London?")),
		SystemInstruction: genai.Ptr("Today is 9-23-2025. Any dates before this are in the past, and any dates after this are in the future."),
		Tools:             []gaos_interactions.Tool{tool},
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
		if res.Interaction.OutputText != nil {
			fmt.Println("Output:", *res.Interaction.OutputText)
		}
	}
}
