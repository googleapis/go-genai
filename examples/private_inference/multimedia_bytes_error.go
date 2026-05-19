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

// Package main demonstrates Private Inference multimedia error handling for bytes / InlineData in the Go GenAI SDK.
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

	log.Println("Reading sample image file...")
	imageBytes, err := os.ReadFile("examples/private_inference/google.jpg")
	if err != nil {
		imageBytes, err = os.ReadFile("examples/private_inference/test_root_ca.crt")
		if err != nil {
			log.Fatalf("Failed to read image file: %v", err)
		}
	}

	log.Println("Sending private inference request with Multimedia bytes (InlineData) and RequestTTL: '60s'...")
	parts := []*genai.Part{
		{Text: "What is shown in this image?"},
		genai.NewPartFromBytes(imageBytes, "image/jpeg"),
	}
	contents := []*genai.Content{{Parts: parts}}
	mmConfig := &genai.GenerateContentConfig{
		RequestTTL: "60s",
	}

	mmResp, mmErr := client.Models.GenerateContent(ctx, model, contents, mmConfig)
	if mmErr != nil {
		log.Printf("Successfully caught multimedia error: %v\n", mmErr)
	} else {
		fmt.Println("Multimedia Response:", mmResp.Text())
	}
}
