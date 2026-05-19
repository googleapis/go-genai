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

// Package main demonstrates Private Inference multimedia error handling in the Go GenAI SDK.
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

	log.Println("Sending private inference request with Multimedia (Video + Text)...")
	parts := []*genai.Part{
		{Text: "summarize the following video."},
		{FileData: &genai.FileData{FileURI: "gs://cloud-samples-data/video/animals.mp4", MIMEType: "video/mp4"}},
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
