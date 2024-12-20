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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var fakeResponse = &GenerateContentResponse{
	Candidates: []*Candidate{
		{
			Content: &Content{
				Parts: []*Part{
					{Text: "This is a fake response"},
				},
			},
		},
	},
}

func vertexAIClient(ctx context.Context, t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client
}

func mlDevClient(ctx context.Context, t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(ctx, &ClientConfig{Backend: BackendGoogleAI})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client
}

// Creates a reusable test server setup.
func setupTestServer(t *testing.T, backend Backend) *httptest.Server {
	t.Helper()
	// Read expected request body from file
	goldenFileReq, err := os.Open(fmt.Sprintf("testdata/golden/%s.mldev.json", strings.ReplaceAll(t.Name(), "/", "_")))
	if backend == BackendVertexAI {
		goldenFileReq, err = os.Open(fmt.Sprintf("testdata/golden/%s.vertex.json", strings.ReplaceAll(t.Name(), "/", "_")))
	}
	if err != nil {
		t.Fatalf("Error opening golden file: %v", err)
	}
	defer goldenFileReq.Close()

	expectedRequestBody, err := io.ReadAll(goldenFileReq)
	if err != nil {
		t.Fatalf("Error reading golden file: %v", err)
	}

	fakeResponseBytes, err := json.Marshal(fakeResponse)
	if err != nil {
		t.Fatalf("marshal fake json failed at test %s", t.Name())
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert Request
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Error reading request body: %v", err)
		}
		if diff := cmp.Diff(string(reqBody), string(expectedRequestBody)); diff != "" {
			t.Fatalf("Request body mismatch (-want +got):\n%s\n got: %s", diff, string(reqBody))
		}

		// Mock Response
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.String(), "?alt=sse") {
			w.Write([]byte(fmt.Sprintf("data: %s\n\n", string(fakeResponseBytes))))
		} else {
			w.Write(fakeResponseBytes)
		}
	}))
}

func fakeClient(ctx context.Context, t *testing.T, server *httptest.Server, backend Backend) *Client {
	t.Helper()
	client, err := NewClient(ctx, &ClientConfig{
		Backend: backend,
		HTTPOptions: &HTTPOptions{
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		},
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client
}

func vertexAIFakeClient(ctx context.Context, t *testing.T) *Client {
	t.Helper()
	vertexServer := setupTestServer(t, BackendVertexAI)
	return fakeClient(ctx, t, vertexServer, BackendVertexAI)
}

func mldevFakeClient(ctx context.Context, t *testing.T) *Client {
	t.Helper()
	mlDevServer := setupTestServer(t, BackendGoogleAI)
	return fakeClient(ctx, t, mlDevServer, BackendGoogleAI)
}

type xpTestParams struct {
	platform string
	model    string
	client   *Client
	contents []*Content
	config   *GenerateContentConfig

	stream bool
}

func baseGenerateContentTest(ctx context.Context, t *testing.T, x xpTestParams) {
	t.Helper()
	if x.platform == "VertexAI" {
		if *mode == apiMode {
			x.client = vertexAIClient(ctx, t)
		} else {
			x.client = vertexAIFakeClient(ctx, t)
		}
	} else {
		if *mode == apiMode {
			x.client = mlDevClient(ctx, t)
		} else {
			x.client = mldevFakeClient(ctx, t)
		}
	}
	t.Run(x.platform, func(t *testing.T) {
		if x.stream {
			for _, err := range x.client.Models.GenerateContentStream(ctx, x.model, x.contents, x.config) {
				if err != nil {
					t.Errorf("GenerateContentStream failed: %v", err)
				}
			}
		} else {
			_, err := x.client.Models.GenerateContent(ctx, x.model, x.contents, x.config)
			if err != nil {
				t.Errorf("GenerateContent failed: %v", err)
			}
		}
	})
}

// Cross platform(XP) Vertex GenerateContent test
func xpVertexGenerateContentTest(ctx context.Context, t *testing.T, contents []*Content, config *GenerateContentConfig) {
	t.Helper()

	baseGenerateContentTest(ctx, t, xpTestParams{
		platform: "VertexAI",
		model:    "gemini-1.5-flash",
		contents: contents,
		config:   config,
	})
}

func xpMLDevGenerateContentTest(ctx context.Context, t *testing.T, contents []*Content) {
	t.Helper()

	baseGenerateContentTest(ctx, t, xpTestParams{
		platform: "MLDev",
		model:    "gemini-1.5-flash",
		contents: contents,
	})
}

// Cross platform(XP) GenerateContent test
func xpGenerateContentTest(ctx context.Context, t *testing.T, contents []*Content, config *GenerateContentConfig) {
	t.Helper()

	baseGenerateContentTest(ctx, t, xpTestParams{
		platform: "VertexAI",
		model:    "gemini-1.5-flash",
		contents: contents,
		config:   config,
	})
	baseGenerateContentTest(ctx, t, xpTestParams{
		platform: "MLDev",
		model:    "gemini-1.5-flash",
		contents: contents,
		config:   config,
	})
}

func xpGenerateContent20Test(ctx context.Context, t *testing.T, contents []*Content, config *GenerateContentConfig, stream bool) {
	t.Helper()
	baseGenerateContentTest(ctx, t, xpTestParams{
		platform: "VertexAI",
		model:    "gemini-2.0-flash-exp",
		contents: contents,
		config:   config,
		stream:   stream,
	})
}

func TestModelsGenerateContent(t *testing.T) {
	if *mode != apiMode && *mode != requestMode {
		t.Skip("Skipping tests in replay mode")
	}
	ctx := context.Background()

	t.Run("WithImageBytes", func(t *testing.T) {
		// Read image file
		imageFile, err := os.Open("testdata/google.jpg")
		if err != nil {
			t.Fatalf("Error opening image file: %v", err)
		}
		defer imageFile.Close()

		imageBytes, err := io.ReadAll(imageFile)
		if err != nil {
			t.Fatalf("Error reading image bytes: %v", err)
		}

		// Expected Request & Response
		contents := []*Content{
			{Parts: []*Part{
				{Text: "What's in this picture"},
				{InlineData: &Blob{Data: imageBytes, MIMEType: "image/png"}},
			}},
		}

		xpGenerateContentTest(ctx, t, contents, nil)
	})
}

// TODO (b/382689811): Use replays when replay supports streams.
func TestModelsGenerateContentStream(t *testing.T) {
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
				testTableFile := readTestTableFile(t, testFilePath)
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
						if isDisabledTest(t) || testTableItem.HasUnion || extractWantException(testTableItem, backend.backend) != "" {
							// Avoid skipping get a less noisy logs in the stream tests
							return
						}
						t.Run(testTableItem.Name, func(t *testing.T) {
							t.Parallel()
							client, err := NewClient(ctx, &ClientConfig{Backend: backend.backend})
							if err != nil {
								t.Fatalf("Error creating client: %v", err)
							}
							module := reflect.ValueOf(*client).FieldByName("Models")
							method := module.MethodByName("GenerateContentStream")
							args := extractArgs(ctx, t, method, testTableFile, testTableItem)
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
			client, err := NewClient(ctx, &ClientConfig{Backend: backend.backend})
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
