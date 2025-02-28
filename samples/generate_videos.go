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
# For Vertex AI API
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}

# For Gemini AI API
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}

go run samples/generate_videos.go --model=veo-2.0-generate-001
*/

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/genai"
)

var model = flag.String("model", "veo-2.0-generate-001", "the model name, e.g. veo-2.0-generate-001")

func generateVideos(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI GenerateVideo API...")
	} else {
		fmt.Println("Calling GeminiAI GenerateVideo API...")
	}
	// Pass in basic config
	var config *genai.GenerateVideosConfig = &genai.GenerateVideosConfig{
		OutputGCSURI: "gs://unified-genai-tests/tmp/genai/video/outputs",
	}
	// Call the GenerateVideo method.
	operation, err := client.Models.GenerateVideos(ctx, *model, "A neon hologram of a cat driving at top speed", nil, config)
	if err != nil {
		log.Fatal(err)
	}

	for !operation.Done {
		fmt.Println("Waiting for operation to complete...")
		time.Sleep(20 * time.Second)
		operation, err = client.Operations.Get(ctx, operation, nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Marshal the result to JSON and pretty-print it to a byte array.
	response, err := json.MarshalIndent(*operation, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}

func main() {
	ctx := context.Background()
	flag.Parse()
	generateVideos(ctx)
}
