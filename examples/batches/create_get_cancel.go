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
	// Create a Gemini Developer API client.
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
		var model = flag.String("model", "gemini-1.5-flash-002", "the model name, e.g. gemini-1.5-pro-002")
		// Create a batch job.
		result, err := client.Batches.Create(
			ctx,
			*model,
			&genai.BatchJobSource{
				GCSURI: []string{
					"gs://unified-genai-tests/batches/input/generate_content_requests.jsonl",
				},
				Format: "jsonl",
			},
			&genai.CreateBatchJobConfig{
				DisplayName: "test-batch-job",
				Dest: &genai.BatchJobDestination{
					Format: "jsonl",
					GCSURI: "gs://unified-genai-tests/batches/output",
				},
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Created batch job:")
		print(result)
		// Get the batch job.
		result, err = client.Batches.Get(ctx, result.Name, &genai.GetBatchJobConfig{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Retrieved batch job:")
		print(result)

		// Cancel the batch job.
		err = client.Batches.Cancel(ctx, result.Name, &genai.CancelBatchJobConfig{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Cancelled batch job:", result.Name)
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
		var model = flag.String("model", "gemini-2.0-flash", "the model name, e.g. gemini-1.5-pro-002")
		// Create a batch job.
		result, err := client.Batches.Create(
			ctx,
			*model,
			&genai.BatchJobSource{
				InlinedRequests: []*genai.InlinedRequest{
					{
						Contents: []*genai.Content{
							{
								Role: "user",
								Parts: []*genai.Part{
									{
										Text: "Hello!",
									},
								},
							},
						},
					},
				},
			},
			&genai.CreateBatchJobConfig{
				DisplayName: "test-batch-job",
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Created batch job:")
		print(result)
		// Get the batch job.
		result, err = client.Batches.Get(ctx, result.Name, &genai.GetBatchJobConfig{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Retrieved batch job:")
		print(result)

		// Cancel the batch job.
		err = client.Batches.Cancel(ctx, result.Name, &genai.CancelBatchJobConfig{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Cancelled batch job:", result.Name)
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
