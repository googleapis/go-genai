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

func print(r any) {
	// Marshal the result to JSON.
	response, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	fmt.Println("Edit image example. Only supported in BackendVertexAI.")
	contentRefImg := &genai.ContentReferenceImage{
		ReferenceImage: &genai.Image{GCSURI: "gs://genai-sdk-tests/inputs/images/dog.jpg"},
		ReferenceID:    1,
	}
	styleRefImg := &genai.StyleReferenceImage{
		ReferenceID:    2,
		ReferenceImage: &genai.Image{GCSURI: "gs://genai-sdk-tests/inputs/images/cyberpunk.jpg"},
		Config: &genai.StyleReferenceConfig{
			StyleDescription: "cyberpunk style",
		},
	}
	response, err := client.Models.EditImage(
		ctx, "imagen-4.0-ingredients-preview",
		/*prompt=*/ "Dog in [1] sleeping on the ground at the bottom of the image with the cyberpunk city landscape in [2] in the background visible on the side of the mug.",
		[]genai.ReferenceImage{contentRefImg, styleRefImg},
		&genai.EditImageConfig{
			IncludeRAIReason: true,
			OutputMIMEType:   "image/jpeg",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
