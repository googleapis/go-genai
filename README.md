![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/googleapis/go-genai)
[![Go Reference](https://pkg.go.dev/badge/google.golang.org/genai.svg)](https://pkg.go.dev/google.golang.org/genai)

# Google Gen AI Go SDK

The Google Gen AI Go SDK provides an interface for developers to integrate
Google's generative models into their Go applications. It supports the
[Gemini Developer API](https://ai.google.dev/gemini-api/docs) and
[Gemini Enterprise Agent Platform](https://docs.cloud.google.com/gemini-enterprise-agent-platform)
APIs.

> [!NOTE]
> The GenAI SDK now supports the [Interactions API](#interactions) (Experimental).


The Google Gen AI Go SDK enables developers to use Google's state-of-the-art
generative AI models (like Gemini) to build AI-powered features and applications.
This SDK supports use cases like:
- Generate text from text-only input
- Generate text from text-and-images input (multimodal)
- ...

For example, with just a few lines of code, you can access Gemini's multimodal
capabilities to generate text from text-and-image input.

```go
parts := []*genai.Part{
  {Text: "What's this image about?"},
  {InlineData: &genai.Blob{Data: imageBytes, MIMEType: "image/jpeg"}},
}
result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{{Parts: parts}}, nil)
```

## Installation and usage

Add the SDK to your module with `go get google.golang.org/genai`.

## Create Clients

### Imports
```go
import "google.golang.org/genai"
```

### Gemini API Client:
```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
	APIKey:   apiKey,
	Backend:  genai.BackendGeminiAPI,
})
```

### Gemini Enterprise Agent Platform Client:
```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
	Project:  project,
	Location: location,
	Backend:  genai.BackendEnterprise,
})
```

### (Optional) Using environment variables:

You can create a client by configuring the necessary environment variables.
Configuration setup instructions depends on whether you're using the Gemini
Developer API or the Gemini API in Gemini Enterprise Agent Platform.

**Gemini Developer API:** Set `GOOGLE_API_KEY` as shown below:

```bash
export GOOGLE_API_KEY='your-api-key'
```

**Gemini API on Gemini Enterprise Agent Platform:** Set `GOOGLE_GENAI_USE_ENTERPRISE`,
`GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION`, as shown below:

```bash
export GOOGLE_GENAI_USE_ENTERPRISE=true
export GOOGLE_CLOUD_PROJECT='your-project-id'
export GOOGLE_CLOUD_LOCATION='us-central1'
```

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{})
```

## Interactions

The Interactions API allows you to interact with agents and models in a multi-turn conversation.

Here is a simple example of creating a new interaction with a model:

```go
package main

import (
	"context"
	"log"

	"google.golang.org/genai"
	"google.golang.org/genai/interactions"
)

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	params := interactions.NewModelParams{
		Model: "gemini-2.5-flash",
		Input: interactions.Input{String: ptr("Tell me a short joke about programming.")},
	}

	interaction, err := client.Interactions.NewModel(ctx, params)
	if err != nil {
		log.Fatal(err)
	}

	for _, output := range interaction.Outputs {
		if output.Text != nil {
			println(output.Text.Text)
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}
```

## License

The contents of this repository are licensed under the
[Apache License, version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
