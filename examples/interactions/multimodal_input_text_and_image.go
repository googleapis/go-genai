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
	"encoding/base64"
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

	candidatePaths := []string{
		"testdata/google.jpg",
		"../testdata/google.jpg",
		"../../testdata/google.jpg",
		"third_party/golang/google_golang_org/genai/v/v0/google/genai/testdata/google.jpg",
	}
	var imageBytes []byte
	for _, p := range candidatePaths {
		imageBytes, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Fatal("Failed to read google.jpg from any candidate path. Last error:", err)
	}
	imageBase64 := base64.StdEncoding.EncodeToString(imageBytes)

	contents := []gaos_interactions.Content{
		gaos_interactions.NewContent(gaos_interactions.TextContent{Text: "What is shown in this image?"}),
		gaos_interactions.NewContent(gaos_interactions.ImageContent{
			Data:     ptr(imageBase64),
			MimeType: gaos_interactions.ImageContentMimeTypeImageJpeg.ToPointer(),
		}),
	}

	body := operations.NewCreateInteractionRequestBody(gaos_interactions.CreateModelInteraction{
		Model: gaos_interactions.Model("gemini-flash-latest"),
		Input: ptr(gaos_interactions.NewInteractionsInput(contents)),
	})

	res, err := client.Interactions.Create(ctx, operations.CreateInteractionRequest{Body: body})
	if err != nil {
		log.Fatal(err)
	}

	if res.Interaction != nil {
		if res.Interaction.ID != nil {
			fmt.Println("Interaction ID:", *res.Interaction.ID)
		}
		if res.Interaction.OutputText != nil {
			fmt.Printf("Output: %s\n", *res.Interaction.OutputText)
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}
