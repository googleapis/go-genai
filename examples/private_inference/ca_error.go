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

// Package main demonstrates Private Inference CA error handling for invalid CA configurations.
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

	model := "gemini-2.5-pro-pie"
	caPool := "projects/cloud-llm-preview1/locations/us-central1/caPools/pie-ca-pool"

	log.Println("CA Error Case 1: Starting secure session with NON-EXISTENT root CA file...")
	err = client.Models.StartSecureSession(ctx, model, caPool, "this/file/does/not/exist.crt")
	if err == nil {
		log.Fatalf("[Error] Expected StartSecureSession to fail, but it returned nil!")
	}
	log.Printf("[Success] StartSecureSession correctly failed with expected error: %v", err)

	log.Println("CA Error Case 2: Starting secure session with CORRUPTED root CA file...")
	tempFile, err := os.CreateTemp("", "corrupt_ca_*.crt")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	_, _ = tempFile.WriteString("-----BEGIN CERTIFICATE-----\nThis is not a valid base64 cert\n-----END CERTIFICATE-----\n")
	tempFile.Close()

	err = client.Models.StartSecureSession(ctx, model, caPool, tempFile.Name())
	if err == nil {
		log.Fatalf("[Error] Expected StartSecureSession to fail with corrupted certificate, but it returned nil!")
	}
	log.Printf("[Success] StartSecureSession correctly failed with expected error: %v", err)

	log.Println("All CA error cases validated successfully.")
}
