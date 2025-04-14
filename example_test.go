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

package genai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"google.golang.org/genai"
)

// Your GCP project
const project = "your-project"

// A GCP location like "us-central1"
const location = "some-gcp-location"

// Your Google API key
const apiKey = "your-api-key"

// This example shows how to create a new client for Vertex AI.
func ExampleNewClient_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	fmt.Println(client.ClientConfig().Backend)
}

// This example shows how to create a new client for Gemini API.
func ExampleNewClient_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	fmt.Println(client.ClientConfig().APIKey)
}

// This example shows how to call the GenerateContent method with a simple text to Vertex AI.
func ExampleModels_GenerateContent_text_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", genai.Text("Tell me about New York?"), nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a simple text to Gemini API.
func ExampleModels_GenerateContent_text_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", genai.Text("Tell me about New York?"), nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with multiple texts to Vertex AI.
func ExampleModels_GenerateContent_texts_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "Tell me about New York?"},
		{Text: "And how about San Francison?"},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with multiple texts to Gemini API.
func ExampleModels_GenerateContent_texts_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "Tell me about New York?"},
		{Text: "And how about San Francison?"},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with inline image to Vertex AI.
func ExampleModels_GenerateContent_inlineImage_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

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

	parts := []*genai.Part{
		{Text: "What's this image about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "image/jpeg"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with inline image to Gemini API.
func ExampleModels_GenerateContent_inlineImage_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

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

	parts := []*genai.Part{
		{Text: "What's this image about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "image/jpeg"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a inline pdf file to Vertex AI.
func ExampleModels_GenerateContent_inlinePDF_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read the pdf file.
	resp, err := http.Get("your pdf url")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "What's this pdf about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "application/pdf"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a inline pdf file to Gemini API.
func ExampleModels_GenerateContent_inlinePDF_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read the pdf file.
	resp, err := http.Get("your pdf url")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "What's this pdf about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "application/pdf"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a inline audio file to Vertex AI.
func ExampleModels_GenerateContent_inlineAudio_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read the audio file.
	resp, err := http.Get("your audio url")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "What's this music about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "audio/mp3"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a inline audio file to Gemini API.
func ExampleModels_GenerateContent_inlineAudio_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Get("your audio url")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "What's this music about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "audio/mp3"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a inline video file to Vertex AI.
func ExampleModels_GenerateContent_inlineVideo_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read the video file.
	resp, err := http.Get("your video url")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "What's this video about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "video/mp4"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with a inline video file to Gemini API.
func ExampleModels_GenerateContent_inlineVideo_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read the video file.
	resp, err := http.Get("your video url")
	if err != nil {
		fmt.Println("Error fetching image:", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	parts := []*genai.Part{
		{Text: "What's this video about?"},
		{InlineData: &genai.Blob{Data: data, MIMEType: "video/mp4"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with GCS URI to Vertex AI.
func ExampleModels_GenerateContent_gcsURI_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	parts := []*genai.Part{
		{Text: "What's this video about?"},
		{FileData: &genai.FileData{FileURI: "gs://cloud-samples-data/video/animals.mp4", MIMEType: "video/mp4"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with HTTP URL to Vertex AI.
func ExampleModels_GenerateContent_httpURL_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	parts := []*genai.Part{
		{Text: "What's this picture about?"},
		{FileData: &genai.FileData{FileURI: "https://storage.googleapis.com/cloud-samples-data/generative-ai/image/scones.jpg", MIMEType: "image/jpeg"}},
	}
	contents := []*genai.Content{{Parts: parts}}

	result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash", contents, nil)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with Google Search Retrieval to Vertex AI.
func ExampleModels_GenerateContent_googleSearchRetrieval_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	parts := []*genai.Part{{Text: "Tell me about New York?"}}
	contents := []*genai.Content{{Parts: parts}}

	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		contents,
		&genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearchRetrieval: &genai.GoogleSearchRetrieval{}},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with Google Search Retrieval to Gemini API.
func ExampleModels_GenerateContent_googleSearchRetrieval_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("Tell me about New York?"),
		&genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearchRetrieval: &genai.GoogleSearchRetrieval{}},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with code execution to Vertex AI.
func ExampleModels_GenerateContent_codeExecution_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("What is the sum of the first 50 prime numbers? Generate and run code for the calculation, and make sure you get all 50."),
		&genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{CodeExecution: &genai.ToolCodeExecution{}},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with code execution to Gemini API.
func ExampleModels_GenerateContent_codeExecution_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("What is the sum of the first 50 prime numbers? Generate and run code for the calculation, and make sure you get all 50."),
		&genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{CodeExecution: &genai.ToolCodeExecution{}},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with GenerateContentConfig to Vertex AI.
func ExampleModels_GenerateContent_config_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("Tell me about New York?"),
		&genai.GenerateContentConfig{
			Temperature:      genai.Ptr[float32](0.5),
			TopP:             genai.Ptr[float32](0.5),
			TopK:             genai.Ptr[float32](2.0),
			ResponseMIMEType: "application/json",
			StopSequences:    []string{"\n"},
			CandidateCount:   2,
			Seed:             genai.Ptr[int32](42),
			MaxOutputTokens:  128,
			PresencePenalty:  genai.Ptr[float32](0.5),
			FrequencyPenalty: genai.Ptr[float32](0.5),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with GenerateContentConfig to Gemini API.
func ExampleModels_GenerateContent_config_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("Tell me about New York?"),
		&genai.GenerateContentConfig{
			Temperature:      genai.Ptr[float32](0.5),
			TopP:             genai.Ptr[float32](0.5),
			TopK:             genai.Ptr[float32](2.0),
			ResponseMIMEType: "application/json",
			StopSequences:    []string{"\n"},
			CandidateCount:   2,
			Seed:             genai.Ptr[int32](42),
			MaxOutputTokens:  128,
			PresencePenalty:  genai.Ptr[float32](0.5),
			FrequencyPenalty: genai.Ptr[float32](0.5),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with system instruction to Vertex AI.
func ExampleModels_GenerateContent_systemInstruction_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("Tell me about New York?"),
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: "You are a helpful assistant."}}},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with system instruction to Gemini API.
func ExampleModels_GenerateContent_systemInstruction_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text("Tell me about New York?"),
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: "You are a helpful assistant."}}},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContent method with third party model to Vertex AI.
func ExampleModels_GenerateContent_thirdPartyModel_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContent method.
	result, err := client.Models.GenerateContent(ctx,
		"meta/llama-3.2-90b-vision-instruct-maas",
		genai.Text("Tell me about New York?"),
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

// This example shows how to call the GenerateContentStream method with a simple text to Vertex AI.
func ExampleModels_GenerateContentStream_text_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContentStream method.
	for result, err := range client.Models.GenerateContentStream(
		ctx,
		"gemini-2.0-flash",
		genai.Text("Give me top 3 indoor kids friendly ideas."),
		nil,
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(result.Candidates[0].Content.Parts[0].Text)
	}
}

// This example shows how to call the GenerateContentStream method with a simple text to Gemini API.
func ExampleModels_GenerateContentStream_text_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Call the GenerateContentStream method.
	for result, err := range client.Models.GenerateContentStream(
		ctx,
		"gemini-2.0-flash",
		genai.Text("Give me top 3 indoor kids friendly ideas."),
		nil,
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(result.Candidates[0].Content.Parts[0].Text)
	}
}

func ExampleChats_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := chat.SendMessage(ctx, genai.Part{Text: "What's the weather in New York?"})
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)

	result, err = chat.SendMessage(ctx, genai.Part{Text: "How about San Francisco?"})
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

func ExampleChats_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := chat.SendMessage(ctx, genai.Part{Text: "What's the weather in New York?"})
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)

	result, err = chat.SendMessage(ctx, genai.Part{Text: "How about San Francisco?"})
	if err != nil {
		log.Fatal(err)
	}
	debugPrint(result)
}

func ExampleChats_stream_geminiapi() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "What's the weather in New York?"}) {
		if err != nil {
			log.Fatal(err)
		}
		debugPrint(result)
	}

	for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "How about San Francisco?"}) {
		if err != nil {
			log.Fatal(err)
		}
		debugPrint(result)
	}
}

func ExampleChats_stream_vertexai() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal(err)
	}

	chat, err := client.Chats.Create(ctx, "gemini-2.0-flash", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "What's the weather in New York?"}) {
		if err != nil {
			log.Fatal(err)
		}
		debugPrint(result)
	}

	for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "How about San Francisco?"}) {
		if err != nil {
			log.Fatal(err)
		}
		debugPrint(result)
	}
}

func debugPrint[T any](r *T) {
	// Marshal the result to JSON.
	response, err := json.MarshalIndent(*r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}
