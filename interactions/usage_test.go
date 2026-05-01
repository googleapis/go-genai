// Copyright 2025 Google LLC
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

// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package interactions_test

import (
	"context"
	"os"
	"testing"

	"google.golang.org/genai/interactions"
	"google.golang.org/genai/interactions/internal/testutil"
	"google.golang.org/genai/interactions/option"
)

func TestUsage(t *testing.T) {
	t.Skip("Mock server tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := interactions.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	interaction, err := client.Interactions.NewModel(context.TODO(), interactions.NewModelParams{
		Input: interactions.Input{
			String: "Tell me a joke",
		},
		Model: "gemini-3-flash-preview",
	})
	if err != nil {
		t.Fatalf("err should be nil: %s", err.Error())
	}
	t.Logf("%+v\n", interaction.ID)
}
