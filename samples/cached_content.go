// Copyright 2024 Google LLC
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

// Package main contains the sample code for the GenerateContent API.
package main

/*
# For Vertex AI API
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}

# For Gemini AI API
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}

go run samples/generate_text.go --model=gemini-1.5-pro-002
*/

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-1.5-pro-002", "the model name, e.g. gemini-1.5-pro-002")

func debugPrint(r any) {
	// Marshal the result to JSON.
	response, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}

func createCachedContent(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}
	// Create a cached content.
	result, err := client.Caches.Create(ctx, *model, &genai.CreateCachedContentConfig{
		TTL: "86400s",
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{
						FileData: &genai.FileData{
							MIMEType: "application/pdf",
							FileURI:  "gs://cloud-samples-data/generative-ai/pdf/2312.11805v3.pdf",
						},
					},
					{
						FileData: &genai.FileData{
							MIMEType: "application/pdf",
							FileURI:  "gs://cloud-samples-data/generative-ai/pdf/2312.11805v3.pdf",
						},
					},
				},
			},
		}})
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)

	// Iterate over the cached contents.
	// Option 1: using the All method.
	for item, err := range client.Caches.All(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		debugPrint(item)
	}

	// Iterate over the cached contents.
	// Option 2: using the List method for more control.
	// Example 2.1 - List the first page.
	page, err := client.Caches.List(ctx, &genai.ListCachedContentsConfig{PageSize: 2})
	// Example 2.2 - Continue to the next page.
	page, err = page.Next(ctx)
	// Example 2.3 - Resume the page iteration using the next page token.
	page, err = client.Caches.List(ctx, &genai.ListCachedContentsConfig{PageSize: 2, PageToken: page.NextPageToken})
	if err == genai.PageDone {
		fmt.Println("No more cached content to retrieve.")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(page.Items)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	createCachedContent(ctx)
}
