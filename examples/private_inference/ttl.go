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

// Package main demonstrates Private Inference with standard RequestTTL in the Go GenAI SDK.
package main

import (
	"context"
	"fmt"
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

	log.Println("Sending private inference request with RequestTTL: '120s'...")
	config := &genai.GenerateContentConfig{
		RequestTTL: "120s",
	}
	resp, err := client.Models.GenerateContent(ctx, model, genai.Text("Why is the sky blue?"), config)
	if err != nil {
		log.Fatalf("GenerateContent failed: %v", err)
	}

	fmt.Println("Secure Response:", resp.Text())
}
