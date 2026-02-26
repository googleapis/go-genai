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

var model = flag.String("model", "gemini-2.5-flash", "the model name, e.g. gemini-2.5-flash")
var parallelAPIKey = flag.String("parallel-api-key", "", "Parallel AI Search API key (required)")

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend != genai.BackendVertexAI {
		log.Fatal("Parallel AI Search is only supported with Vertex AI backend")
	}
	fmt.Println("Calling VertexAI Backend...")

	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{
			ParallelAiSearch: &genai.ParallelAiSearch{
				APIKey: *parallelAPIKey,
				CustomConfigs: &genai.ParallelAiSearchCustomConfigs{
					MaxResults: genai.Ptr[int32](5),
				},
			},
		}},
	}

	result, err := client.Models.GenerateContent(ctx, *model, genai.Text("What are the latest developments in AI?"), config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Response:", result.Text())
	if len(result.Candidates) > 0 && result.Candidates[0].GroundingMetadata != nil {
		jsonData, _ := json.MarshalIndent(result.Candidates[0].GroundingMetadata, "", "  ")
		fmt.Println("Grounding metadata:", string(jsonData))
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()
	if *parallelAPIKey == "" {
		log.Fatal("--parallel-api-key is required")
	}
	run(ctx)
}
