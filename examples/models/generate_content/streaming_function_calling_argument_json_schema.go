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
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-3-pro-preview", "the model name, e.g. gemini-2.0-flash")

func run(ctx context.Context) {
	var parameterSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"brightness": map[string]any{
				"type":        "number",
				"description": "Light level from 0 to 100. Zero is off and 100 is full brightness.",
			},
			"colorTemperature": map[string]any{
				"type":        "string",
				"description": "Color temperature of the light fixture which can be `daylight`, `cool` or `warm`.",
			},
		},
		"required": []string{"brightness", "colorTemperature"},
	}

	var tools = []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:                 "controlLight",
					Description:          "Set the brightness and color temperature of a room light.",
					ParametersJsonSchema: parameterSchema,
				},
			},
		},
	}
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		println(" GeminiAPI Backend doesn't support streaming function call arguments API.")
		return
	}

	var streamingArgument = true
	var config *genai.GenerateContentConfig = &genai.GenerateContentConfig{
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode:                        genai.FunctionCallingConfigModeAny,
				StreamFunctionCallArguments: &streamingArgument,
			},
		},
		Temperature: genai.Ptr[float32](0),
		Tools:       tools,
	}

	for result, err := range client.Models.GenerateContentStream(
		ctx,
		*model,
		genai.Text("Control the light in the living room to 50% brightness and warm white color."),
		config,
	) {
		if err != nil {
			log.Fatal(err)
		}
		jsonData, _ := json.MarshalIndent(result.Candidates[0].Content, "", "  ")
		fmt.Println(string(jsonData))
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
