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
	"fmt"
	"log"
	"time"

	"google.golang.org/genai"
)

func print(r any) {
	response, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(response))
}

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Inlined requests are not supported for Vertex AI backend.")
		return
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	inlineRequests := []*genai.InlinedRequest{
		{
			Contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						{
							Text: "Tell me a one-sentence joke.",
						},
					},
					Role: "user",
				},
			},
			Config: &genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{
						{
							Text: "You are a funny comedian. Always respond with humor and wit.",
						},
					},
				},
				Temperature: genai.Ptr[float32](0.5),
			},
		},
		{
			Contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						{
							Text: "Why is the sky blue?",
						},
					},
					Role: "user",
				},
			},
			Config: &genai.GenerateContentConfig{
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{
						{
							Text: "You are a helpful science teacher. Explain complex concepts in simple terms.",
						},
					},
				},
				Temperature: genai.Ptr[float32](0.5),
			},
		},
	}

	inlineBatchJob, err := client.Batches.Create(
		ctx,
		"models/gemini-2.5-flash",
		&genai.BatchJobSource{
			InlinedRequests: inlineRequests,
		},
		&genai.CreateBatchJobConfig{
			DisplayName: "inlined-requests-job-1",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created batch job: %s\n", inlineBatchJob.Name)

	model := "models/gemini-embedding-2-preview"
	embedBatchJob, err := client.Batches.CreateEmbeddings(
		ctx,
		&model,
		&genai.EmbeddingsBatchJobSource{
			InlinedRequests: &genai.EmbedContentBatch{
				Contents: []*genai.Content{
					{Parts: []*genai.Part{{Text: "first text to embed"}}},
					{Parts: []*genai.Part{{Text: "second text to embed"}}},
				},
			},
		},
		&genai.CreateEmbeddingsBatchJobConfig{
			DisplayName: "gemini-embed-2-job",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created embedding batch job: %s\n", embedBatchJob.Name)
	fmt.Println("Polling for completion...")

	// Poll until the job finishes
	for {
		job, err := client.Batches.Get(ctx, embedBatchJob.Name, nil)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Status: %s\n", job.State)
		if job.State == genai.JobStateSucceeded {
			fmt.Println("Job Succeeded! Responses:")
			print(job.Dest.InlinedEmbedContentResponses)
			break
		}
		if job.State == genai.JobStateFailed || job.State == genai.JobStateCancelled {
			fmt.Println("Job failed or was cancelled.")
			break
		}

		time.Sleep(10 * time.Second)
	}
}

func main() {
	ctx := context.Background()
	run(ctx)
}
