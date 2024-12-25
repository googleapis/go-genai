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
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type unionDeserialize func([]byte) (reflect.Value, error)

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func readTestTableFile(t *testing.T, dir string) *testTableFile {
	t.Helper()
	data, err := os.ReadFile(dir)
	if err != nil {
		t.Error("Error reading file:", err)
	}
	var testTableFile testTableFile
	if err = json.Unmarshal(data, &testTableFile); err != nil {
		t.Error("Error unmarshalling JSON:", err)
	}
	return &testTableFile
}

func extractArgs(ctx context.Context, t *testing.T, method reflect.Value, testTableFile *testTableFile, testTableItem *testTableItem) []reflect.Value {
	t.Helper()
	args := []reflect.Value{
		reflect.ValueOf(ctx),
	}
	fromParams := []any{ctx}
	for i := 1; i < method.Type().NumIn(); i++ {
		parameterName := testTableFile.ParameterNames[i-1]
		parameterValue, ok := testTableItem.Parameters[parameterName]
		if ok {
			convertedJSON, err := json.Marshal(parameterValue)
			if err != nil {
				t.Error("ExtractArgs: error marshalling:", err)
			}
			paramType := method.Type().In(i)
			convertedValue := reflect.New(paramType).Elem()
			if err = json.Unmarshal(convertedJSON, convertedValue.Addr().Interface()); err != nil {
				t.Error("ExtractArgs: error unmarshalling:", err, string(convertedJSON))
			}
			args = append(args, convertedValue)
		} else {
			args = append(args, reflect.New(method.Type().In(i)).Elem())
		}
	}
	numParams := method.Type().NumIn()
	for i := 1; i < numParams; i++ {
		if i >= len(fromParams) {
			break
		}
	}
	return args
}

func extractMethod(t *testing.T, testTableFile *testTableFile, client *Client) reflect.Value {
	t.Helper()
	// Gets module name and method name.
	segments := strings.Split(testTableFile.TestMethod, ".")
	if len(segments) != 2 {
		t.Error("Invalid test method: " + testTableFile.TestMethod)
	}
	moduleName := segments[0]
	methodName := segments[1]

	// Finds the module and method.
	module := reflect.ValueOf(*client).FieldByName(snakeToPascal(moduleName))
	if !module.IsValid() {
		t.Skipf("Skipping module: %s.%s, not supported in Go", moduleName, methodName)
	}
	method := module.MethodByName(snakeToPascal(methodName))
	if !method.IsValid() {
		t.Skipf("Skipping method: %s.%s, not supported in Go", moduleName, methodName)
	}
	return method
}

func extractWantException(testTableItem *testTableItem, backend Backend) string {
	if backend == BackendVertexAI {
		return testTableItem.ExceptionIfVertex
	}
	return testTableItem.ExceptionIfMLDev
}

func createReplayAPIClient(t *testing.T, testTableDirectory string, testTableItem *testTableItem, backendName string) *replayAPIClient {
	t.Helper()
	replayAPIClient := newReplayAPIClient(t)
	replayFileName := testTableItem.Name
	if testTableItem.OverrideReplayID != "" {
		replayFileName = testTableItem.OverrideReplayID
	}
	replayFilePath := path.Join(testTableDirectory, fmt.Sprintf("%s.%s.json", replayFileName, backendName))
	replayAPIClient.LoadReplay(replayFilePath)
	return replayAPIClient
}

// TestTable only runs in apiMode or replayMode.
func TestTable(t *testing.T) {
	if *mode != apiMode && *mode != replayMode {
		t.Skipf("Skipping test table because client env mode is enabled and affect environment variables")
	}
	ctx := context.Background()
	// Read the replaypath from the ReplayAPIClient instead of the env variable to avoid future
	// breakages if the behavior of the ReplayAPIClient changes, e.g. takes the replay directory
	// from a different source, as the tests must read the replay files from the same source.
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
				testTableDirectory := filepath.Dir(strings.TrimPrefix(testFilePath, replayPath))
				testName := strings.TrimPrefix(testTableDirectory, "/tests/")
				t.Run(testName, func(t *testing.T) {
					testTableFile := readTestTableFile(t, testFilePath)
					for _, testTableItem := range testTableFile.TestTable {
						t.Run(testTableItem.Name, func(t *testing.T) {
							t.Parallel()
							if isDisabledTest(t) {
								t.Skipf("Skipping disabled test")
							}
							if testTableItem.HasUnion {
								// TODO(b/377989301): Handle unions.
								t.Skipf("Skipping because it has union")
							}
							config := ClientConfig{Backend: backend.Backend}
							replayClient := createReplayAPIClient(t, testTableDirectory, testTableItem, backend.name)
							if *mode == "replay" {
								config.baseURL = replayClient.GetBaseURL()
								config.HTTPClient, err = replayClient.CreateClient(ctx)
							}
							if backend.Backend == BackendVertexAI {
								config.Project = "fake-project"
								config.Location = "fake-location"
							} else {
								config.APIKey = "fake-api-key"
							}
							client, err := NewClient(ctx, &config)
							if err != nil {
								t.Fatalf("Error creating client: %v", err)
							}
							method := extractMethod(t, testTableFile, client)
							args := extractArgs(ctx, t, method, testTableFile, testTableItem)

							// Inject unknown fields to the replay file to simulate the case where the SDK adds
							// unknown fields to the response.
							injectUnknownFields(t, replayClient)

							response := method.Call(args)
							wantException := extractWantException(testTableItem, backend.Backend)
							if wantException != "" {
								if response[1].IsNil() {
									t.Fatalf("Calling method expected to fail but it didn't, err: %v", wantException)
								}
								gotException := response[1].Interface().(error).Error()
								if diff := cmp.Diff(gotException, wantException, cmp.Comparer(func(x, y string) bool {
									// Check the contains on both sides (x->y || y->x) because comparer has to be
									// symmetric (https://pkg.go.dev/github.com/google/go-cmp/cmp#Comparer)
									return strings.Contains(x, y) || strings.Contains(y, x)
								})); diff != "" {
									t.Errorf("Exceptions had diff (-got +want):\n%v", diff)
								}
							} else {
								// Assert there was no error when the call is successful.
								if !response[1].IsNil() {
									t.Fatalf("Calling method failed unexpectedly, err: %v", response[1].Interface().(error).Error())
								}
								// Assert the response when the call is successful.
								got := convertSDKResponseToMatchReplayType(t, response[0].Elem().Interface())
								want := replayClient.LatestInteraction().Response.SDKResponseSegments
								if diff := cmp.Diff(got, want); diff != "" {
									t.Errorf("Responses had diff (-got +want):\n%v", diff)
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

func convertSDKResponseToMatchReplayType(t *testing.T, response any) []map[string]any {
	t.Helper()
	if reflect.ValueOf(response).IsZero() {
		return []map[string]any{}
	}
	responseJSON, err := json.MarshalIndent([]any{response}, "", "  ")
	if err != nil {
		t.Fatal("Error marshalling gotJSON:", err)
	}
	responseMap := []map[string]any{}
	if err = json.Unmarshal(responseJSON, &responseMap); err != nil {
		t.Fatal("Error unmarshalling want:", err)
	}
	return responseMap
}

func injectUnknownFields(t *testing.T, replayClient *replayAPIClient) {
	t.Helper()
	var inject func(in any) int
	inject = func(in any) int {
		counter := 0
		switch in.(type) {
		case map[string]any:
			m := in.(map[string]any)
			for _, v := range m {
				inject(v)
			}
			m["unknownFieldString"] = "unknownValue"
			m["unknownFieldNumber"] = 0
			m["unknownFieldMap"] = map[string]any{"unknownFieldString": "unknownValue"}
			m["unknownFieldArray"] = []any{map[string]any{"unknownFieldString": "unknownValue"}}
			counter++
		case []any:
			for _, v := range in.([]any) {
				inject(v)
			}
		}
		return counter
	}
	for _, interaction := range replayClient.ReplayFile.Interactions {
		for _, bodySegment := range interaction.Response.BodySegments {
			// This ensures that the injection actually happened to avoid false positives test results.
			if inject(bodySegment) == 0 {
				t.Fatal("No unknown fields were injected. There must be at least one unknown field added to the body segments.")
			}
		}
	}
}
