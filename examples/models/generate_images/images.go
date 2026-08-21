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
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"google.golang.org/genai"
)

func print(r any) {
	// Marshal the result to JSON.
	response, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend != genai.BackendVertexAI {
		log.Fatal("GenerateImages is only supported in the Vertex AI backend.")
	}

	fmt.Println("Calling VertexAI Backend...")
	fmt.Println("Generate image example.")
	response1, err := client.Models.GenerateImages(
		ctx, "imagen-4.0-generate-001",
		/*prompt=*/ "An umbrella in the foreground, and a rainy night sky in the background",
		&genai.GenerateImagesConfig{
			IncludeRAIReason:        true,
			IncludeSafetyAttributes: true,
			OutputMIMEType:          "image/jpeg",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response1)

	fmt.Println("Positive prompt safety attributes:")
	print(response1.PositivePromptSafetyAttributes)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
