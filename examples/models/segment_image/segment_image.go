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
	"io"
	"log"
	"net/http"

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

	fmt.Println("Segment image example.")

	// Read the image data from a url.
	resp, err := http.Get("https://storage.googleapis.com/cloud-samples-data/generative-ai/image/scones.jpg")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	response1, err := client.Models.SegmentImage(
		ctx, "image-segmentation-001",
		/*source=*/ &genai.SegmentImageSource{
			Image: &genai.Image{ImageBytes: data},
		},
		&genai.SegmentImageConfig{
			Mode: genai.SegmentModeForeground,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response1)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
