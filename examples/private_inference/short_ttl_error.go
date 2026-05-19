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

// Package main demonstrates Private Inference short TTL timeout error handling in the Go GenAI SDK.
package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  project,
		Location: location,
	})
	if err != nil {
		log.Fatalf("NewClient failed: %v", err)
	}

	model := "gemini-2.5-flash-pie"
	caPool := "projects/cloud-llm-preview1/locations/us-central1/caPools/pie-ca-pool"
	rootCA := "examples/private_inference/test_root_ca.crt"

	log.Println("Starting secure session...")
	err = client.Models.StartSecureSession(ctx, model, caPool, rootCA)
	if err != nil {
		log.Fatalf("StartSecureSession failed: %v", err)
	}

	log.Println("Testing 1s TTL call (expect timeout/499 error)...")
	ttlConfig := &genai.GenerateContentConfig{
		RequestTTL: "1s",
	}
	_, ttlErr := client.Models.GenerateContent(ctx, model, genai.Text("Write a 10,000 word detailed chronological history of every country in Asia and Europe."), ttlConfig)
	if ttlErr != nil {
		log.Printf("Successfully caught expected TTL error: %v\n", ttlErr)
	} else {
		log.Fatalf("Expected TTL error for 1s request, but got successful response!")
	}
}
