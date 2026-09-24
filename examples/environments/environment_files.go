// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
	"google.golang.org/genai/interactions/models/environments"
	"google.golang.org/genai/interactions/models/interactions"
	"google.golang.org/genai/interactions/models/operations"
)

func ptr[T any](v T) *T {
	return &v
}

func main() {
	ctx := context.Background()

	cfg := &genai.ClientConfig{
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1alpha",
		},
	}
	if baseURL := os.Getenv("GOOGLE_GENAI_BASE_URL"); baseURL != "" {
		cfg.HTTPOptions.BaseURL = baseURL
	}
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Environment Files API is currently supported on Gemini API (MLDev). Skipping on Vertex.")
		return
	}

	fmt.Println("Using Gemini Developer API")

	fmt.Println("\n--- 1. Creating Environment with Workspace Files ---")
	createReq := operations.CreateEnvironmentRequest{
		Body: environments.CreateEnvironmentRequest{
			Sources: []interactions.Source{
				{
					Type:    interactions.SourceTypeInline.ToPointer(),
					Target:  ptr("main.py"),
					Content: ptr("print(\"Hello from Go Environment Files demo!\")\n"),
				},
				{
					Type:    interactions.SourceTypeInline.ToPointer(),
					Target:  ptr("config.json"),
					Content: ptr("{\"version\": \"1.0\", \"debug\": true}\n"),
				},
				{
					Type:    interactions.SourceTypeInline.ToPointer(),
					Target:  ptr("src/utils.py"),
					Content: ptr("def greet(name: str) -> str:\n    return f\"Hello, {name}!\"\n"),
				},
			},
		},
	}

	createResp, err := client.Environments.CreateEnvironment(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create environment: %v", err)
	}
	if createResp.Environment == nil || createResp.Environment.ID == "" {
		log.Fatalf("No environment ID returned")
	}
	envID := createResp.Environment.ID
	fmt.Printf("Environment created successfully! ID: %s\n", envID)

	defer func() {
		fmt.Printf("\n--- 7. Cleaning up Environment ID: %s ---\n", envID)
		_, err := client.Environments.DeleteEnvironment(ctx, operations.DeleteEnvironmentRequest{
			ID: envID,
		})
		if err != nil {
			log.Printf("Failed to delete environment: %v", err)
		} else {
			fmt.Println("Environment deleted successfully.")
		}
	}()

	fmt.Println("\n--- 2. Listing Files at Root Directory (path=\".\") ---")
	rootFilesResp, err := client.Environments.Files.List(ctx, operations.GetEnvironmentFilesRequest{
		Environment: envID,
		Path:        ".",
	})
	if err != nil {
		log.Fatalf("Failed to list root files: %v", err)
	}
	for _, f := range rootFilesResp.GetEnvironmentFilesResponse.Files {
		var name, size, fType string
		if f.Name != nil {
			name = *f.Name
		}
		if f.SizeBytes != nil {
			size = *f.SizeBytes
		}
		if f.Type != nil {
			fType = string(*f.Type)
		}
		fmt.Printf(" - %s (type=%s, size=%s bytes)\n", name, fType, size)
	}

	fmt.Println("\n--- 3. Querying Subdirectory (path=\"src\", recursive=true) ---")
	srcFilesResp, err := client.Environments.Files.List(ctx, operations.GetEnvironmentFilesRequest{
		Environment: envID,
		Path:        "src",
		Recursive:   ptr(true),
	})
	if err != nil {
		log.Fatalf("Failed to query subdirectory: %v", err)
	}
	for _, f := range srcFilesResp.GetEnvironmentFilesResponse.Files {
		var name, path string
		if f.Name != nil {
			name = *f.Name
		}
		if f.Path != nil {
			path = *f.Path
		}
		fmt.Printf(" - %s (path=%s)\n", name, path)
	}

	fmt.Println("\n--- 4. Querying Specific File Path (path=\"main.py\") ---")
	mainFileResp, err := client.Environments.Files.List(ctx, operations.GetEnvironmentFilesRequest{
		Environment: envID,
		Path:        "main.py",
	})
	if err != nil {
		log.Fatalf("Failed to query main.py: %v", err)
	}
	for _, f := range mainFileResp.GetEnvironmentFilesResponse.Files {
		var size string
		if f.SizeBytes != nil {
			size = *f.SizeBytes
		}
		fmt.Printf("main.py file size: %s bytes\n", size)
	}

	fmt.Println("\n--- 5. Uploading a New File (path=\"uploaded.txt\") ---")
	contentToUpload := []byte("Hello from Go Environment Files upload demo!\n")
	uploadResp, err := client.Environments.Files.UploadBytes(ctx, envID, "uploaded.txt", contentToUpload)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}
	uploadedName := "unknown"
	if uploadResp.Files != nil && len(uploadResp.Files.Files) > 0 && uploadResp.Files.Files[0].Name != nil {
		uploadedName = *uploadResp.Files.Files[0].Name
	}
	fmt.Printf("Uploaded file successfully: %s\n", uploadedName)

	fmt.Println("\n--- 6. Downloading File Content (path=\"uploaded.txt\") ---")
	downloadedBytes, err := client.Environments.Files.Download(ctx, envID, "uploaded.txt")
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	fmt.Printf("Downloaded uploaded.txt content:\n%s\n", string(downloadedBytes))
}
