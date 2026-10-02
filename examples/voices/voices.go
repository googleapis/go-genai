// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
	"google.golang.org/genai/interactions/models/operations"
	"google.golang.org/genai/interactions/models/voices"
)

func main() {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if client.ClientConfig().Backend == genai.BackendVertexAI || os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "true" {
		fmt.Println("Voices API is currently supported on Gemini API (MLDev). Skipping on Vertex.")
		return
	}

	fmt.Println("Using Gemini Developer API")

	fmt.Println("\n--- 1. Listing Voices ---")
	listResp, err := client.Voices.List(ctx, operations.ListVoicesRequest{})
	if err != nil {
		log.Fatalf("Failed to list voices: %v", err)
	}
	if listResp.ListVoicesResponse != nil {
		for _, v := range listResp.ListVoicesResponse.Voices {
			var id, displayName string
			if v.ID != nil {
				id = *v.ID
			}
			if v.DisplayName != nil {
				displayName = *v.DisplayName
			}
			fmt.Printf(" - %s (%s)\n", id, displayName)
		}
	}

	fmt.Println("\n--- 2. Creating a Custom Prompted Voice ---")
	createResp, err := client.Voices.Create(ctx, operations.CreateVoiceRequest{
		Body: voices.CreateVoiceRequest{
			Store: genai.Ptr(true),
			Voice: voices.VoiceInput{
				Type:         voices.VoiceTypePrompted,
				DisplayName:  genai.Ptr("Warm Narrator"),
				LanguageCode: genai.Ptr("en-US"),
				Prompted: &voices.PromptedVoice{
					Input: "A warm, friendly narrator voice with a calm pace.",
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create voice: %v", err)
	}
	if createResp.Voice == nil || createResp.Voice.ID == nil {
		log.Fatalf("No voice ID returned")
	}
	voiceID := *createResp.Voice.ID
	fmt.Printf("Created voice ID: %s\n", voiceID)

	defer func() {
		fmt.Printf("\n--- 4. Deleting Voice ID: %s ---\n", voiceID)
		_, err := client.Voices.Delete(ctx, operations.DeleteVoiceRequest{
			ID: voiceID,
		})
		if err != nil {
			log.Printf("Failed to delete voice: %v", err)
		} else {
			fmt.Println("Voice deleted successfully.")
		}
	}()

	fmt.Printf("\n--- 3. Getting Voice ID: %s ---\n", voiceID)
	getResp, err := client.Voices.Get(ctx, operations.GetVoiceRequest{
		ID: voiceID,
	})
	if err != nil {
		log.Fatalf("Failed to get voice: %v", err)
	}
	if getResp.Voice != nil {
		var id, displayName string
		if getResp.Voice.ID != nil {
			id = *getResp.Voice.ID
		}
		if getResp.Voice.DisplayName != nil {
			displayName = *getResp.Voice.DisplayName
		}
		fmt.Printf("Retrieved voice: %s (%s)\n", id, displayName)
	}
}
