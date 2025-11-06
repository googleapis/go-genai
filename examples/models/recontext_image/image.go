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

	fmt.Println("Product recontext example. Only supported in BackendVertexAI.")
	productImage := &genai.ProductImage{
		ProductImage: &genai.Image{GCSURI: "gs://genai-sdk-tests/inputs/images/backpack1.png"},
	}
	productImages := []*genai.ProductImage{productImage}
	prompt := "On a school desk."
	response1, err := client.Models.RecontextImage(
		ctx, "imagen-product-recontext-preview-06-30",
		&genai.RecontextImageSource{
			Prompt:        prompt,
			PersonImage:   nil,
			ProductImages: productImages},
		&genai.RecontextImageConfig{
			OutputMIMEType: "image/jpeg",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response1)

	fmt.Println("Virtual try-on example. Only supported in BackendVertexAI.")
	productImagePants := &genai.ProductImage{
		ProductImage: &genai.Image{GCSURI: "gs://genai-sdk-tests/inputs/images/pants.jpg"},
	}
	personImage := &genai.Image{GCSURI: "gs://genai-sdk-tests/inputs/images/man.jpg"}
	productImages2 := []*genai.ProductImage{productImagePants}
	response2, err := client.Models.RecontextImage(
		ctx, "virtual-try-on-exp-05-31",
		&genai.RecontextImageSource{
			Prompt:        "",
			PersonImage:   personImage,
			ProductImages: productImages2},
		&genai.RecontextImageConfig{
			OutputMIMEType: "image/jpeg",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response2)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
