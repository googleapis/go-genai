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

package genai

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Stream test runs in api mode but read _test_table.json for retrieving test params.
// TODO (b/382689811): Use replays when replay supports streams.
func TestModelsGenerateContentStream(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	replayPath := newReplayAPIClient(t).ReplaysDirectory

	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			err := filepath.Walk(replayPath, func(testFilePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.Name() != "_test_table.json" {
					return nil
				}
				var testTableFile testTableFile
				if err := readFileForReplayTest(testFilePath, &testTableFile, false); err != nil {
					t.Errorf("error loading test table file, %v", err)
				}
				if strings.Contains(testTableFile.TestMethod, "stream") {
					t.Fatal("Replays supports generate_content_stream now. Revitis these tests and use the replays instead.")
				}
				// We only want `generate_content` method to test the generate_content_stream API.
				if testTableFile.TestMethod != "models.generate_content" {
					return nil
				}
				testTableDirectory := filepath.Dir(strings.TrimPrefix(testFilePath, replayPath))
				testName := strings.TrimPrefix(testTableDirectory, "/tests/")
				t.Run(testName, func(t *testing.T) {
					for _, testTableItem := range testTableFile.TestTable {
						t.Logf("testTableItem: %v", t.Name())
						if isDisabledTest(t) || testTableItem.HasUnion || extractWantException(testTableItem, backend.Backend) != "" {
							// Avoid skipping get a less noisy logs in the stream tests
							return
						}
						t.Run(testTableItem.Name, func(t *testing.T) {
							t.Parallel()
							client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
							if err != nil {
								t.Fatalf("Error creating client: %v", err)
							}
							module := reflect.ValueOf(*client).FieldByName("Models")
							method := module.MethodByName("GenerateContentStream")
							args := extractArgs(ctx, t, method, &testTableFile, testTableItem)
							method.Call(args)
							model := args[1].Interface().(string)
							contents := args[2].Interface().([]*Content)
							config := args[3].Interface().(*GenerateContentConfig)
							for response, err := range client.Models.GenerateContentStream(ctx, model, contents, config) {
								if err != nil {
									t.Errorf("GenerateContentStream failed unexpectedly: %v", err)
								}
								if response == nil {
									t.Fatalf("expected at least one response, got none")
								}
								if len(response.Candidates) == 0 {
									t.Errorf("expected at least one candidate, got none")
								}
								if len(response.Candidates[0].Content.Parts) == 0 {
									t.Errorf("expected at least one part, got none")
								}
							}
						})
					}
				})
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestModelsGenerateContentAudio(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			config := &GenerateContentConfig{
				ResponseModalities: []string{"AUDIO"},
				SpeechConfig: &SpeechConfig{
					VoiceConfig: &VoiceConfig{
						PrebuiltVoiceConfig: &PrebuiltVoiceConfig{
							VoiceName: "Aoede",
						},
					},
				},
			}
			result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash-exp", Text("say something nice to me"), config)
			if err != nil {
				t.Errorf("GenerateContent failed unexpectedly: %v", err)
			}
			if result == nil {
				t.Fatalf("expected at least one response, got none")
			}
			if len(result.Candidates) == 0 {
				t.Errorf("expected at least one candidate, got none")
			}
		})
	}
}
