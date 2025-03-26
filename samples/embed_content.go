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

// Package main contains the sample code for the GenerateContent API.
package main

/*
# For VertexAI Backend
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}

# For GeminiAPI Backend
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}

go run samples/embed_content.go --model=text-embedding-004
*/

import (
	"context"
	"flag"
	"fmt"
	"log"

	"google.golang.org/genai"
)

var model = flag.String("model", "text-embedding-004", "the model name, e.g. text-embedding-004")

func embedContent(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}
	// Call the GenerateContent method.
	result, err := client.Models.EmbedContent(ctx, *model, genai.Text("What is your name?"), &genai.EmbedContentConfig{TaskType: "RETRIEVAL_QUERY"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", result.Embeddings[0])

	result, err = client.Models.EmbedContent(ctx, *model, genai.Text("What is your name?"), &genai.EmbedContentConfig{TaskType: "RETRIEVAL_DOCUMENT"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", result.Embeddings[0])
}

func main() {
	ctx := context.Background()
	flag.Parse()
	embedContent(ctx)
}
