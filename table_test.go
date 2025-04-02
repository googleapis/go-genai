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
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/auth"
	"github.com/google/go-cmp/cmp"
)

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if i == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

// methodParamType is extra mapping of method param name to its param type because reflect module cannot process private struct.
var methodParamType = map[string]map[string]reflect.Type{
	"editImage": map[string]reflect.Type{
		"referenceImages": reflect.TypeOf(([]ReferenceImage)(nil)),
	},
}

type interfaceDeserialize func([]byte) (reflect.Value, error)

// methodParamType is dedicated deserializer for each interface type because json string cannot be unmarshalled to
// interface type directly.
var interfaceDeserializer = map[string]interfaceDeserialize{
	"[]genai.ReferenceImage": func(s []byte) (reflect.Value, error) {
		var images1 []*referenceImageAPI
		if err := json.Unmarshal(s, &images1); err != nil {
			return reflect.Value{}, err
		}
		// Need to match the method type because reflect method.Call() cannot process interface type.
		var images2 []ReferenceImage
		for _, image := range images1 {
			images2 = append(images2, image)
		}
		return reflect.ValueOf(images2), nil
	},
}

func sanitizeGotSDKResponses(t *testing.T, responses []map[string]any) {
	t.Helper()
	for _, response := range responses {
		if _, ok := response["NextPageToken"].(string); ok {
			if response["NextPageToken"] != "" {
				response["nextPageToken"] = response["NextPageToken"]
			}
			delete(response, "NextPageToken")
		}
		if _, ok := response["Name"].(string); ok {
			response[response["Name"].(string)] = response["Items"]
			delete(response, "Items")
			delete(response, "Name")
		}

	}
}

func extractArgs(ctx context.Context, t *testing.T, method reflect.Value, testTableFile *testTableFile, testTableItem *testTableItem) []reflect.Value {
	t.Helper()
	args := []reflect.Value{
		reflect.ValueOf(ctx),
	}
	fromParams := []any{ctx}
	for i := 1; i < method.Type().NumIn(); i++ {
		parameterName := snakeToCamel(testTableFile.ParameterNames[i-1])
		parameterValue, ok := testTableItem.Parameters[parameterName]
		if ok {
			paramType := method.Type().In(i)
			if paramType == nil {
				_, methodName := moduleAndMethodName(t, testTableFile)
				if pt, ok := methodParamType[methodName][parameterName]; ok {
					paramType = pt
				}
			}

			paramTypeName := paramType.String()
			if paramTypeName == "[]genai.ReferenceImage" {
				sanitizeMapWithSourceType(t, reflect.TypeOf(([]*referenceImageAPI)(nil)), parameterValue)
			} else {
				sanitizeMapWithSourceType(t, paramType, parameterValue)
			}
			sanitizeMapByPath(parameterValue, "httpOptions.headers", func(data any, path string) any {
				if _, ok := data.(map[string]any); !ok {
					log.Printf("convertStringMapToHeaderMap: data is not map[string]any: %s %T\n", data, data)
					return data
				}
				m := data.(map[string]any)
				result := make(map[string][]string)
				for k, v := range m {
					result[k] = []string{v.(string)}
				}
				return result
			}, false)
			convertedJSON, err := json.Marshal(parameterValue)
			if err != nil {
				t.Error("ExtractArgs: error marshalling:", err)
			}

			if deserializer, ok := interfaceDeserializer[paramTypeName]; ok {
				// interface types.
				convertedValue, err := deserializer(convertedJSON)
				if err != nil {
					t.Fatalf("ExtractArgs: error unmarshalling slice: %v, json: %s", err, string(convertedJSON))
				}
				args = append(args, convertedValue)
			} else {
				// struct types.
				convertedValue := reflect.New(paramType).Elem()
				if err = json.Unmarshal(convertedJSON, convertedValue.Addr().Interface()); err != nil {
					t.Error("ExtractArgs: error unmarshalling:", err, string(convertedJSON))
				}
				args = append(args, convertedValue)
			}
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

func moduleAndMethodName(t *testing.T, testTableFile *testTableFile) (string, string) {
	t.Helper()
	// Gets module name and method name.
	segments := strings.Split(testTableFile.TestMethod, ".")
	if len(segments) != 2 {
		t.Error("Invalid test method: " + testTableFile.TestMethod)
	}
	moduleName := segments[0]
	methodName := segments[1]
	return moduleName, methodName
}

func extractMethod(t *testing.T, testTableFile *testTableFile, client *Client) reflect.Value {
	t.Helper()
	// Gets module name and method name.
	segments := strings.Split(testTableFile.TestMethod, ".")
	if len(segments) != 2 {
		t.Error("Invalid test method: " + testTableFile.TestMethod)
	}
	moduleName, methodName := moduleAndMethodName(t, testTableFile)

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
	if *mode == unitMode {
		t.Skipf("Skipping test, mode is %s", *mode)
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
					var testTableFile testTableFile
					if err := readFileForReplayTest(testFilePath, &testTableFile, false); err != nil {
						t.Errorf("error loading test table file, %v", err)
					}
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
							if testTableItem.SkipInAPIMode != "" {
								t.Skipf("Skipping because %s", testTableItem.SkipInAPIMode)
							}
							config := ClientConfig{Backend: backend.Backend}
							replayClient := createReplayAPIClient(t, testTableDirectory, testTableItem, backend.name)
							if *mode == replayMode {
								config.HTTPOptions.BaseURL = replayClient.GetBaseURL()
								config.HTTPClient = replayClient.server.Client()
								if backend.Backend == BackendVertexAI {
									config.Project = "fake-project"
									config.Location = "fake-location"
									config.Credentials = &auth.Credentials{}
								} else {
									config.APIKey = "fake-api-key"
								}
							}
							client, err := NewClient(ctx, &config)
							if err != nil {
								t.Fatalf("Error creating client: %v", err)
							}
							method := extractMethod(t, &testTableFile, client)
							args := extractArgs(ctx, t, method, &testTableFile, testTableItem)

							// Inject unknown fields to the replay file to simulate the case where the SDK adds
							// unknown fields to the response.
							// For forward compatibility tests.
							if testName == "TestTable/vertex/models/generate_content" {
								injectUnknownFields(t, replayClient)
							}
							response := method.Call(args)
							if *mode == apiMode {
								return
							}
							// TODO(b/399217361): Re-enable response checks after replay files record using extras.
							if *mode == replayMode && testName == "models/generate_images" &&
								(testTableItem.Name == "test_all_vertexai_config_parameters" ||
									testTableItem.Name == "test_all_mldev_config_parameters") {
								return
							}
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
								var resp any
								if response[0].Kind() == reflect.Ptr {
									resp = response[0].Elem().Interface()
								} else {
									resp = response[0].Interface()
								}
								got := convertSDKResponseToMatchReplayType(t, resp, testTableItem.IgnoreKeys)
								sanitizeGotSDKResponses(t, got)
								for _, v := range got {
									sanitizeMapWithSourceType(t, reflect.TypeOf(resp), v)
								}
								want := replayClient.LatestInteraction().Response.SDKResponseSegments
								if testTableItem.IgnoreKeys != nil {
									for _, keyToIgnore := range testTableItem.IgnoreKeys {
										for _, m := range want {
											delete(m, keyToIgnore)
										}
									}
								}
								for _, v := range want {
									_ = convertFloat64ToString(v)
								}
								for _, v := range got {
									_ = convertFloat64ToString(v)
								}
								opts := cmp.Options{stringComparator, floatComparator}
								if diff := cmp.Diff(got, want, opts); diff != "" {
									t.Errorf("Responses had diff (-got +want):\n%v\n %v\n\n %v", diff, got, want)
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

func convertSDKResponseToMatchReplayType(t *testing.T, response any, ignoreKeys []string) []map[string]any {
	t.Helper()
	responseJSON, err := json.MarshalIndent([]any{response}, "", "  ")
	if err != nil {
		t.Fatal("Error marshalling gotJSON:", err)
	}
	responseMap := []map[string]any{}
	if err = json.Unmarshal(responseJSON, &responseMap); err != nil {
		t.Fatal("Error unmarshalling want:", err)
	}
	omitEmptyValues(responseMap)
	// Remove the keys in ignoreKeys
	if ignoreKeys != nil {
		for _, m := range responseMap {
			for _, keyToIgnore := range ignoreKeys {
				delete(m, keyToIgnore)
			}
		}
	}
	return responseMap
}

func injectUnknownFields(t *testing.T, replayClient *replayAPIClient) {
	t.Helper()
	var inject func(in any) int
	inject = func(in any) int {
		counter := 0
		switch in := in.(type) {
		case map[string]any:
			for _, v := range in {
				inject(v)
			}
			in["unknownFieldString"] = "unknownValue"
			in["unknownFieldNumber"] = 0
			in["unknownFieldMap"] = map[string]any{"unknownFieldString": "unknownValue"}
			in["unknownFieldArray"] = []any{map[string]any{"unknownFieldString": "unknownValue"}}
			counter++
		case []any:
			for _, v := range in {
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
