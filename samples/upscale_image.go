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

# For Gemini AI API (upscale is not supported in Gemini AI API)
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}

go run samples/upscale_image.go --model=imagen-3.0-generate-001
*/

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"google.golang.org/genai"
)

var model = flag.String("model", "imagen-3.0-generate-001", "the model name, e.g. imagen-3.0-generate-001")

func upscaleImage(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI UpscaleImage API...")
	} else {
		fmt.Println("Calling GeminiAI UpscaleImage API...")
	}

	// First, generate an image.
	var generateImagesConfig *genai.GenerateImagesConfig = &genai.GenerateImagesConfig{
		NumberOfImages:   1,
		OutputMIMEType:   "image/jpeg",
		IncludeRAIReason: true,
	}
	generationResult, err := client.Models.GenerateImages(ctx, *model, "Create a blue circle", generateImagesConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Call the UpscaleImage method.
	var config *genai.UpscaleImageConfig = &genai.UpscaleImageConfig{
		OutputMIMEType:   "image/jpeg",
		IncludeRAIReason: true,
	}
	var image *genai.Image = generationResult.GeneratedImages[0].Image
	result, err := client.Models.UpscaleImage(ctx, *model, image, "x2", config)
	if err != nil {
		log.Fatal(err)
	}
	// Marshal the result to JSON and pretty-print it to a byte array.
	response, err := json.MarshalIndent(*result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}

func main() {
	ctx := context.Background()
	flag.Parse()
	upscaleImage(ctx)
}
