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
	"log"

	"github.com/sanity-io/litter"
	"google.golang.org/genai"
	"google.golang.org/genai/interactions"
)

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{HTTPOptions: genai.HTTPOptions{BaseURL: "https://staging-generativelanguage.sandbox.googleapis.com/"}})
	if err != nil {
		log.Fatal(err)
	}

	litter.Config.HideZeroValues = true // cleaner output

	params := interactions.NewModelParams{
		Model: "gemini-2.5-flash",
		Input: interactions.Input{String: ptr("Tell me a short joke about programming.")},
	}

	interaction, err := client.Interactions.NewModel(ctx, params)
	if err != nil {
		log.Fatal(err)
	}

	litter.Dump(interaction)

	for _, output := range interaction.Outputs {
		if output.Text != nil {
			println(output.Text.Text)
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}

}
