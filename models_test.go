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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
						if testTableItem.SkipInAPIMode != "" {
							t.Skipf("Skipping because %s", testTableItem.SkipInAPIMode)
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
								} else if response.Candidates != nil && len(response.Candidates) == 0 {
									t.Errorf("expected at least one candidate, got none")
								} else if response.Candidates != nil && response.Candidates[0].Content != nil && len(response.Candidates[0].Content.Parts) == 0 {
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
					LanguageCode: "en-US",
				},
			}
			result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", Text("say something nice to me"), config)
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

func TestModelsGenerateContentStreamingFunctionCallJsonParamsWithoutHistory(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()

	var parameterSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"brightness": map[string]any{
				"type":        "number",
				"description": "Light level from 0 to 100. Zero is off and 100 is full brightness.",
			},
			"colorTemperature": map[string]any{
				"type":        "string",
				"description": "Color temperature of the light fixture which can be `daylight`, `cool` or `warm`.",
			},
		},
		"required": []string{"brightness", "colorTemperature"},
	}

	var tools = []*Tool{
		{
			FunctionDeclarations: []*FunctionDeclaration{
				{
					Name:                 "controlLight",
					Description:          "Set the brightness and color temperature of a room light.",
					ParametersJsonSchema: parameterSchema,
				},
			},
		},
	}
	var streamingArgument = true
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
			if client.ClientConfig().Backend == BackendVertexAI {
				t.Logf("Calling VertexAI Backend...")
			} else {
				t.Skip("Skip. GeminiAPI Backend does not support streaming function call.")
			}
			config := &GenerateContentConfig{
				Tools: tools,
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode:                        FunctionCallingConfigModeAny,
						StreamFunctionCallArguments: &streamingArgument,
					},
				},
			}
			for result, err := range client.Models.GenerateContentStream(
				ctx,
				"gemini-2.5-pro",
				Text("Control the light to 50% brightness and warm white color."),
				config,
			) {
				if err != nil {
					t.Errorf("GenerateContentStream failed unexpectedly: %v", err)
				}
				if result == nil {
					t.Fatalf("expected at least one response, got none")
				} else if result.Candidates != nil && len(result.Candidates) == 0 {
					t.Errorf("expected at least one candidate, got none")
				} else if result.Candidates != nil && result.Candidates[0].Content != nil && len(result.Candidates[0].Content.Parts) == 0 {
					t.Errorf("expected at least one part, got none")
				}
			}
		})
	}
}

func TestModelsGenerateContentStreamingFunctionCallGeminiParamsWithoutHistory(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()

	var parameterSchema = &Schema{
		Type: TypeObject,
		Properties: map[string]*Schema{
			"brightness": {
				Type:        TypeNumber,
				Description: "Light level from 0 to 100. Zero is off and 100 is full brightness.",
			},
			"colorTemperature": {
				Type:        TypeString,
				Description: "Color temperature of the light fixture which can be `daylight`, `cool` or `warm`.",
			},
		},
		Required: []string{"brightness", "colorTemperature"},
	}

	var tools = []*Tool{
		{
			FunctionDeclarations: []*FunctionDeclaration{
				{
					Name:        "controlLight",
					Description: "Set the brightness and color temperature of a room light.",
					Parameters:  parameterSchema,
				},
			},
		},
	}
	var streamingArgument = true
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
			if client.ClientConfig().Backend == BackendVertexAI {
				t.Logf("Calling VertexAI Backend...")
			} else {
				fmt.Println("Skip. GeminiAPI Backend does not support streaming function call.")
				return
			}
			config := &GenerateContentConfig{
				Tools: tools,
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode:                        FunctionCallingConfigModeAny,
						StreamFunctionCallArguments: &streamingArgument,
					},
				},
			}
			for result, err := range client.Models.GenerateContentStream(
				ctx,
				"gemini-2.5-pro",
				Text("Control the light to 50% brightness and warm white color."),
				config,
			) {
				if err != nil {
					t.Errorf("GenerateContentStream failed unexpectedly: %v", err)
				}
				if result == nil {
					t.Fatalf("expected at least one response, got none")
				} else if result.Candidates != nil && len(result.Candidates) == 0 {
					t.Errorf("expected at least one candidate, got none")
				} else if result.Candidates != nil && result.Candidates[0].Content != nil && len(result.Candidates[0].Content.Parts) == 0 {
					t.Errorf("expected at least one part, got none")
				}
			}
		})
	}
}

func TestModelsGenerateContentStreamingFunctionCallJsonParamsWithHistory(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()

	var parameterSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"brightness": map[string]any{
				"type":        "number",
				"description": "Light level from 0 to 100. Zero is off and 100 is full brightness.",
			},
			"colorTemperature": map[string]any{
				"type":        "string",
				"description": "Color temperature of the light fixture which can be `daylight`, `cool` or `warm`.",
			},
		},
		"required": []string{"brightness", "colorTemperature"},
	}

	var tools = []*Tool{
		{
			FunctionDeclarations: []*FunctionDeclaration{
				{
					Name:                 "controlLight",
					Description:          "Set the brightness and color temperature of a room light.",
					ParametersJsonSchema: parameterSchema,
				},
			},
		},
	}

	var willContinueFalse = false
	priorContent := []*Content{
		{
			Parts: []*Part{
				{Text: "Control the light in the living room to 50% brightness and warm white color."},
			},
			Role: "user",
		},
		{
			Parts: []*Part{
				{
					FunctionCall: &FunctionCall{
						Name: "controlLight",
						PartialArgs: []*PartialArg{
							{
								JsonPath:    "$.colorTemperature",
								StringValue: "warm",
							},
						},
						WillContinue: &willContinueFalse,
					},
				},
			},
			Role: "model",
		},
	}
	var streamingArgument = true
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
			if client.ClientConfig().Backend == BackendVertexAI {
				t.Logf("Calling VertexAI Backend...")
			} else {
				t.Skip("Skip. GeminiAPI Backend does not support streaming function call.")
			}
			config := &GenerateContentConfig{
				Tools: tools,
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode:                        FunctionCallingConfigModeAny,
						StreamFunctionCallArguments: &streamingArgument,
					},
				},
			}
			for result, err := range client.Models.GenerateContentStream(
				ctx,
				"gemini-2.5-pro",
				priorContent,
				config,
			) {
				if err != nil {
					t.Errorf("GenerateContentStream failed unexpectedly: %v", err)
				}
				if result == nil {
					t.Fatalf("expected at least one response, got none")
				} else if result.Candidates != nil && len(result.Candidates) == 0 {
					t.Errorf("expected at least one candidate, got none")
				} else if result.Candidates != nil && result.Candidates[0].Content != nil && len(result.Candidates[0].Content.Parts) == 0 {
					t.Errorf("expected at least one part, got none")
				}
			}
		})
	}
}

func TestModelsGenerateContentStreamingFunctionCallGeminiParamsWithHistory(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()

	var parameterSchema = &Schema{
		Type: TypeObject,
		Properties: map[string]*Schema{
			"brightness": {
				Type:        TypeNumber,
				Description: "Light level from 0 to 100. Zero is off and 100 is full brightness.",
			},
			"colorTemperature": {
				Type:        TypeString,
				Description: "Color temperature of the light fixture which can be `daylight`, `cool` or `warm`.",
			},
		},
		Required: []string{"brightness", "colorTemperature"},
	}

	var tools = []*Tool{
		{
			FunctionDeclarations: []*FunctionDeclaration{
				{
					Name:        "controlLight",
					Description: "Set the brightness and color temperature of a room light.",
					Parameters:  parameterSchema,
				},
			},
		},
	}

	var willContinueFalse = false
	priorContent := []*Content{
		{
			Parts: []*Part{
				{Text: "Control the light in the living room to 50% brightness and warm white color."},
			},
			Role: "user",
		},
		{
			Parts: []*Part{
				{
					FunctionCall: &FunctionCall{
						Name: "controlLight",
						PartialArgs: []*PartialArg{
							{
								JsonPath:    "$.colorTemperature",
								StringValue: "warm",
							},
						},
						WillContinue: &willContinueFalse,
					},
				},
			},
			Role: "model",
		},
	}
	var streamingArgument = true
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
			if client.ClientConfig().Backend == BackendVertexAI {
				t.Logf("Calling VertexAI Backend...")
			} else {
				t.Skip("Skip. GeminiAPI Backend does not support this test.")
				return
			}
			config := &GenerateContentConfig{
				Tools: tools,
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode:                        FunctionCallingConfigModeAny,
						StreamFunctionCallArguments: &streamingArgument,
					},
				},
			}
			for result, err := range client.Models.GenerateContentStream(
				ctx,
				"gemini-2.5-pro",
				priorContent,
				config,
			) {
				if err != nil {
					t.Errorf("GenerateContentStream failed unexpectedly: %v", err)
				}
				if result == nil {
					t.Fatalf("expected at least one response, got none")
				} else if result.Candidates != nil && len(result.Candidates) == 0 {
					t.Errorf("expected at least one candidate, got none")
				} else if result.Candidates != nil && result.Candidates[0].Content != nil && len(result.Candidates[0].Content.Parts) == 0 {
					t.Errorf("expected at least one part, got none")
				}
			}
		})
	}
}

func TestModelsGenerateContentMultiSpeakerVoiceConfigAudio(t *testing.T) {
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
					MultiSpeakerVoiceConfig: &MultiSpeakerVoiceConfig{
						SpeakerVoiceConfigs: []*SpeakerVoiceConfig{
							{
								Speaker: "Alice",
								VoiceConfig: &VoiceConfig{
									PrebuiltVoiceConfig: &PrebuiltVoiceConfig{
										VoiceName: "Aoede",
									},
								},
							},
							{
								Speaker: "Bob",
								VoiceConfig: &VoiceConfig{
									PrebuiltVoiceConfig: &PrebuiltVoiceConfig{
										VoiceName: "Kore",
									},
								},
							},
						},
					},
					LanguageCode: "en-US",
				},
			}
			result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", Text("say something nice to me"), config)
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

func TestModelsGenerateVideosText2VideoPoll(t *testing.T) {
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
			operation, err := client.Models.GenerateVideos(ctx, "veo-2.0-generate-001", "A neon hologram of a cat driving at top speed", nil, nil)
			if err != nil {
				t.Errorf("GenerateVideos failed unexpectedly: %v", err)
			}
			for !operation.Done {
				fmt.Println("Waiting for operation to complete...")
				time.Sleep(20 * time.Second)
				operation, err = client.Operations.GetVideosOperation(ctx, operation, nil)
				if err != nil {
					log.Fatal(err)
				}
			}
			if operation == nil || operation.Response == nil {
				t.Fatalf("expected at least one response, got none")
			}
			if operation.Response.GeneratedVideos[0].Video.URI == "" && operation.Response.GeneratedVideos[0].Video.VideoBytes == nil {
				t.Fatalf("expected generated video to have either URI or video bytes")
			}
		})
	}
}

func TestModelsGenerateVideosFromSource(t *testing.T) {
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
			video := &Video{
				URI:      "gs://genai-sdk-tests/inputs/videos/cat_driving.mp4",
				MIMEType: "video/mp4",
			}
			outputGCSURI := "gs://genai-sdk-tests/outputs/videos"
			if backend.Backend != BackendVertexAI {
				// Not supported in MLDev.
				video = nil
				outputGCSURI = ""
			}
			generateVideosSource := &GenerateVideosSource{
				Prompt: "Driving across a bridge.",
				Video:  video,
			}
			config := &GenerateVideosConfig{
				NumberOfVideos: 1,
				OutputGCSURI:   outputGCSURI,
			}
			operation, err := client.Models.GenerateVideosFromSource(ctx, "veo-2.0-generate-001", generateVideosSource, config)
			if err != nil {
				t.Errorf("GenerateVideos failed unexpectedly: %v", err)
			}
			for !operation.Done {
				fmt.Println("Waiting for operation to complete...")
				time.Sleep(20 * time.Second)
				operation, err = client.Operations.GetVideosOperation(ctx, operation, nil)
				if err != nil {
					log.Fatal(err)
				}
			}
			if operation == nil || operation.Response == nil {
				t.Fatalf("expected at least one response, got none")
			}
			if operation.Response.GeneratedVideos[0].Video.URI == "" && operation.Response.GeneratedVideos[0].Video.VideoBytes == nil {
				t.Fatalf("expected generated video to have either URI or video bytes")
			}
		})
	}
}

func TestModelsGenerateVideosExtensionFromSource(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			// Gemini only test
			if isDisabledTest(t) || backend.Backend == BackendVertexAI {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}

			// Generate first video
			operation1, err := client.Models.GenerateVideosFromSource(
				ctx,
				"veo-3-exp",
				&GenerateVideosSource{
					Prompt: "Rain",
				},
				&GenerateVideosConfig{
					NumberOfVideos: 1,
				},
			)
			if err != nil {
				t.Errorf("GenerateVideos failed unexpectedly: %v", err)
			}
			for !operation1.Done {
				fmt.Println("Waiting for operation to complete...")
				time.Sleep(20 * time.Second)
				operation1, err = client.Operations.GetVideosOperation(ctx, operation1, nil)
				if err != nil {
					log.Fatal(err)
				}
			}
			if operation1 == nil || operation1.Response == nil {
				t.Fatalf("expected at least one response, got none")
			}
			for _, v := range operation1.Response.GeneratedVideos {
				data, err := client.Files.Download(ctx, NewDownloadURIFromGeneratedVideo(v), nil)
				if err != nil {
					log.Println(err)
					continue
				}
				fmt.Printf("Video file %s downloaded. Data size: %d. \n", v.Video.URI, len(data))
			}
			if operation1.Response.GeneratedVideos[0].Video.URI == "" || operation1.Response.GeneratedVideos[0].Video.VideoBytes == nil {
				t.Fatalf("expected generated video to have both URI and video bytes after downloading")
			}

			// Extend the first video
			operation2, err := client.Models.GenerateVideosFromSource(
				ctx,
				"veo-3-exp",
				&GenerateVideosSource{
					Prompt: "Sun",
					Video:  operation1.Response.GeneratedVideos[0].Video,
				},
				&GenerateVideosConfig{
					NumberOfVideos: 1,
				},
			)
			if err != nil {
				t.Errorf("GenerateVideos failed unexpectedly: %v", err)
			}
			for !operation2.Done {
				fmt.Println("Waiting for operation to complete...")
				time.Sleep(20 * time.Second)
				operation2, err = client.Operations.GetVideosOperation(ctx, operation2, nil)
				if err != nil {
					log.Fatal(err)
				}
			}
			if operation2 == nil || operation2.Response == nil {
				t.Fatalf("expected at least one response, got none")
			}
			if operation2.Response.GeneratedVideos[0].Video.URI == "" && operation2.Response.GeneratedVideos[0].Video.VideoBytes == nil {
				t.Fatalf("expected generated video to have either URI or video bytes")
			}
		})
	}
}

func TestModelsGenerateVideosEditOutpaint(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Parallel()
			if isDisabledTest(t) || backend.Backend != BackendVertexAI {
				t.Skip("Skip: disabled test")
			}
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.Backend})
			if err != nil {
				t.Fatal(err)
			}
			video := &Video{
				URI:      "gs://genai-sdk-tests/inputs/videos/editing_demo.mp4",
				MIMEType: "video/mp4",
			}
			outputGCSURI := "gs://genai-sdk-tests/outputs/videos"
			generateVideosSource := &GenerateVideosSource{
				Prompt: "A mountain landscape",
				Video:  video,
			}
			config := &GenerateVideosConfig{
				NumberOfVideos: 1,
				OutputGCSURI:   outputGCSURI,
				AspectRatio:    "16:9",
				Mask: &VideoGenerationMask{
					Image: &Image{
						GCSURI:   "gs://genai-sdk-tests/inputs/videos/video_outpaint_mask.png",
						MIMEType: "image/png",
					},
					MaskMode: VideoGenerationMaskModeOutpaint,
				},
			}
			operation, err := client.Models.GenerateVideosFromSource(ctx, "veo-2.0-generate-exp", generateVideosSource, config)
			if err != nil {
				t.Errorf("GenerateVideos failed unexpectedly: %v", err)
			}
			for !operation.Done {
				fmt.Println("Waiting for operation to complete...")
				time.Sleep(20 * time.Second)
				operation, err = client.Operations.GetVideosOperation(ctx, operation, nil)
				if err != nil {
					log.Fatal(err)
				}
			}
			if operation == nil || operation.Response == nil {
				t.Fatalf("expected at least one response, got none")
			}
			if operation.Response.GeneratedVideos[0].Video.URI == "" && operation.Response.GeneratedVideos[0].Video.VideoBytes == nil {
				t.Fatalf("expected generated video to have either URI or video bytes")
			}
		})
	}
}

func TestModelsGenerateContentImage(t *testing.T) {
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
				ResponseModalities: []string{"IMAGE", "TEXT"},
			}
			result, err := client.Models.GenerateContent(ctx, "gemini-2.0-flash-preview-image-generation",
				Text("Generate an image of the Eiffel tower with fireworks in the background."), config)
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

func TestModelsAll(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		serverResponses []map[string]any
		expectedModels  []*Model
	}{
		{
			name: "Pagination_SinglePage",
			serverResponses: []map[string]any{
				{
					"models": []*Model{
						{Name: "model1", DisplayName: "Model 1"},
						{Name: "model2", DisplayName: "Model 2"},
					},
					"nextPageToken": "",
				},
			},
			expectedModels: []*Model{
				{Name: "model1", DisplayName: "Model 1", TunedModelInfo: &TunedModelInfo{}},
				{Name: "model2", DisplayName: "Model 2", TunedModelInfo: &TunedModelInfo{}},
			},
		},
		{
			name: "Pagination_MultiplePages",
			serverResponses: []map[string]any{
				{
					"models": []*Model{
						{Name: "model1", DisplayName: "Model 1"},
					},
					"nextPageToken": "next_page_token",
				},
				{
					"models": []*Model{
						{Name: "model2", DisplayName: "Model 2"},
						{Name: "model3", DisplayName: "Model 3"},
					},
					"nextPageToken": "",
				},
			},
			expectedModels: []*Model{
				{Name: "model1", DisplayName: "Model 1", TunedModelInfo: &TunedModelInfo{}},
				{Name: "model2", DisplayName: "Model 2", TunedModelInfo: &TunedModelInfo{}},
				{Name: "model3", DisplayName: "Model 3", TunedModelInfo: &TunedModelInfo{}},
			},
		},
		{
			name:            "Empty_Response",
			serverResponses: []map[string]any{{"models": []*Model{}, "nextPageToken": ""}},
			expectedModels:  []*Model{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseIndex := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if responseIndex > 0 && r.URL.Query().Get("pageToken") == "" {
					t.Errorf("Models.All() failed to pass pageToken in the request")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				response, err := json.Marshal(tt.serverResponses[responseIndex])
				if err != nil {
					t.Errorf("Failed to marshal response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err = w.Write(response)
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				responseIndex++
			}))
			defer ts.Close()

			client, err := NewClient(ctx, &ClientConfig{HTTPOptions: HTTPOptions{BaseURL: ts.URL},
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "test-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			gotModels := []*Model{}
			for model, err := range client.Models.All(ctx) {
				if err != nil {
					t.Errorf("Models.All() iteration error = %v", err)
					return
				}
				gotModels = append(gotModels, model)
			}

			if diff := cmp.Diff(tt.expectedModels, gotModels); diff != "" {
				t.Errorf("Models.All() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModelsAllEmptyResponse(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		serverResponses []func(w http.ResponseWriter)
	}{
		{
			name: "Empty_JSON_Payload",
			serverResponses: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{}`))
					if err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				},
			},
		},
		{
			name: "JSON_Payload_With_Unknown_Fields",
			serverResponses: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"unknownField": "value", "models": []}`))
					if err != nil {
						t.Errorf("Failed to write response: %v", err)
					}
				},
			},
		},
		{
			name: "Entirely_Empty_Response_Body",
			serverResponses: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					w.WriteHeader(http.StatusOK)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseIndex := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.serverResponses[responseIndex](w)
				responseIndex++
			}))
			defer ts.Close()

			client, err := NewClient(ctx, &ClientConfig{HTTPOptions: HTTPOptions{BaseURL: ts.URL},
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "test-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			gotModels := []*Model{}
			for model, err := range client.Models.All(ctx) {
				if err != nil {
					t.Errorf("Models.All() iteration error = %v", err)
					return
				}
				gotModels = append(gotModels, model)
			}

			if len(gotModels) != 0 {
				t.Errorf("Models.All() expected empty list, got: %v", gotModels)
			}
		})
	}
}
