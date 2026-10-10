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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"google.golang.org/genai"
)

var (
	model = flag.String(
		"model",
		"gemini-2.5-flash",
		"the model name, e.g. gemini-2.5-flash",
	)
	prompt = flag.String(
		"prompt",
		"Write an exhaustive, multi-chapter textbook on compiler design that is around 40,000 tokens long.",
		"the prompt to send to the model to trigger multi-hop continuation",
	)
)

func runGenerateContent(ctx context.Context, client *genai.Client) {
	// Automatic continuation is enabled by default.
	result, err := client.Models.GenerateContent(ctx, *model, genai.Text(*prompt), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text())
}

func runChat(ctx context.Context, client *genai.Client) {
	// Automatic continuation is enabled by default in Chat sessions.
	chat, err := client.Chats.Create(ctx, *model, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := chat.SendMessage(ctx, genai.Part{Text: *prompt})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text())
}

func main() {
	ctx := context.Background()
	flag.Parse()

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	runGenerateContent(ctx, client)
	runChat(ctx, client)
}
