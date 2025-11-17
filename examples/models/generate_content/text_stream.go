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

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI.GenerateContentStream API...")
	} else {
		fmt.Println("Calling GeminiAI.GenerateContentStream API...")
	}
	var config *genai.GenerateContentConfig = &genai.GenerateContentConfig{SystemInstruction: &genai.Content{Parts: []*genai.Part{&genai.Part{Text: "You are a story writer."}}}}
	// Call the GenerateContent method.
	for result, err := range client.Models.GenerateContentStream(ctx, *model, genai.Text("Tell me a story in 300 words."), config) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(result.Text())
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
