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

// Package main contains the sample code for the Imagen.
package main

/*
# For VertexAI Backend
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}

# For GeminiAPI Backend
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}

go run samples/imagen.go
*/

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

func imagen(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	fmt.Println("Generate image example.")
	response1, err := client.Models.GenerateImages(
		ctx, "imagen-3.0-generate-002",
		/*prompt=*/ "An umbrella in the foreground, and a rainy night sky in the background",
		&genai.GenerateImagesConfig{
			IncludeRAIReason:        true,
			IncludeSafetyAttributes: true,
			OutputMIMEType:          "image/jpeg",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response1)
	print(response1.PositivePromptSafetyAttributes)

	fmt.Println("Upscale image example. Only supported in BackendVertexAI.")
	response2, err := client.Models.UpscaleImage(
		ctx, "imagen-3.0-generate-002",
		response1.GeneratedImages[0].Image,
		/*upscaleFactor=*/ "x2",
		&genai.UpscaleImageConfig{
			IncludeRAIReason: true,
			OutputMIMEType:   "image/jpeg",
		})
	if err != nil {
		log.Fatal(err)
	}
	print(response2)

	fmt.Println("Edit image example. Only supported in BackendVertexAI.")
	rawRefImg := &genai.RawReferenceImage{
		ReferenceImage: response1.GeneratedImages[0].Image,
		ReferenceID:    1,
	}
	maskRefImg := &genai.MaskReferenceImage{
		ReferenceID: 2,
		Config: &genai.MaskReferenceConfig{
			MaskMode:     "MASK_MODE_BACKGROUND",
			MaskDilation: genai.Ptr[float32](0.0),
		},
	}
	response3, err := client.Models.EditImage(
		ctx, "imagen-3.0-capability-001",
		/*prompt=*/ "Sunlight and clear sky",
		[]genai.ReferenceImage{rawRefImg, maskRefImg},
		&genai.EditImageConfig{
			EditMode:         "EDIT_MODE_INPAINT_INSERTION",
			IncludeRAIReason: true,
			OutputMIMEType:   "image/jpeg",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	print(response3)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	imagen(ctx)
}
