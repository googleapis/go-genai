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
	"flag"
	"fmt"
	"log"

	"cloud.google.com/go/auth/credentials"
	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.5-flash", "the model name, e.g. gemini-2.5-flash")
var gcsURI = flag.String("gcs-uri", "gs://cloud-samples-data/generative-ai/pdf/2312.11805v3.pdf", "the gcs uri of the pdf file")

// This example shows how to register a file from GCS and use it in the Gemini API.
// Setup instructions: https://ai.google.dev/gemini-api/docs/file-input-methods#registration
// GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json GEMINI_API_KEY=<your-api-key> go run gcs_reference.go
func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatalln(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		log.Fatalln("Not supported for VertexAI backend")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes: []string{"https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/devstorage.read_only"},
	})
	if err != nil {
		log.Fatal(err)
	}

	registeredFiles, err := client.Files.RegisterFiles(ctx, creds, []string{*gcsURI}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Registered files:", registeredFiles.Files)

	result, err := client.Models.GenerateContent(ctx, *model, []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: "What's this pdf about?"},
				{
					FileData: &genai.FileData{
						FileURI:  registeredFiles.Files[0].URI,
						MIMEType: "application/pdf",
					},
				},
			},
		},
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated content:", result.Text())
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
