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

// Package main demonstrates Private Inference function calling in the Go GenAI SDK.
package main

import (
	"context"
	"encoding/json"
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

	log.Println("Sending private inference request with function calling...")
	tools := []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "get_weather",
					Description: "Get the weather for a given location.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"location": {
								Type:        genai.TypeString,
								Description: "The city and state, e.g. Tokyo",
							},
						},
						Required: []string{"location"},
					},
				},
			},
		},
	}
	fcConfig := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "Answer in 3 sentences."}},
		},
		RequestTTL: "120s",
		Tools:      tools,
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
			},
		},
	}
	fcResp, err := client.Models.GenerateContent(ctx, model, genai.Text("What's the weather like in Tokyo?"), fcConfig)
	if err != nil {
		log.Fatalf("GenerateContent for function calling failed: %v", err)
	}

	fcJsonData, err := json.MarshalIndent(fcResp, "", "  ")
	if err != nil {
		log.Fatalf("MarshalIndent failed: %v", err)
	}
	fmt.Println("Function Calling Response:\n", string(fcJsonData))
}
