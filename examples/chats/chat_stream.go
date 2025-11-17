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

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.5-flash", "the model name, e.g. gemini-2.5-flash")

// mergeContents merges a slice of Content into a single Content by combining all parts.
func mergeContents(contents []*genai.Content) *genai.Content {
	if len(contents) == 0 {
		return nil
	}

	var allParts []*genai.Part
	var role string

	// Collect all parts from all contents
	for _, content := range contents {
		if content != nil {
			if role == "" {
				role = content.Role
			}
			allParts = append(allParts, content.Parts...)
		}
	}

	return &genai.Content{
		Role:  role,
		Parts: allParts,
	}
}

func chatStream(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}
	var config *genai.GenerateContentConfig = &genai.GenerateContentConfig{Temperature: genai.Ptr[float32](0.5)}

	// Create a new Chat.
	chat, err := client.Chats.Create(ctx, *model, config, nil)
	if err != nil {
		log.Fatal(err)
	}

	part := genai.Part{Text: "What is 1 + 2?"}
	p := make([]genai.Part, 1)
	p[0] = part

	// Send first chat message.
	for result, err := range chat.SendMessageStream(ctx, p...) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Result text: %s\n", result.Text())
	}

	// Send second chat message.
	part = genai.Part{Text: "Add 1 to the previous result."}

	for result, err := range chat.SendMessageStream(ctx, part) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Result text: %s\n", result.Text())
	}

	// Example: Merge streaming response contents into a single Content.
	fmt.Println("\n--- Merged Content Example ---")
	part = genai.Part{Text: "What are the first 5 prime numbers?"}

	var contents []*genai.Content
	for result, err := range chat.SendMessageStream(ctx, part) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Chunk text: %s\n", result.Text())
		// Collect content from each chunk
		if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
			contents = append(contents, result.Candidates[0].Content)
		}
	}

	// Merge all contents into one
	mergedContent := mergeContents(contents)
	if mergedContent != nil {
		fmt.Printf("\nMerged content has %d parts\n", len(mergedContent.Parts))
		// Print all text from merged content
		var fullText string
		for _, part := range mergedContent.Parts {
			if part.Text != "" {
				fullText += part.Text
			}
		}
		fmt.Printf("Merged full text: %s\n", fullText)
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()
	chatStream(ctx)
}
