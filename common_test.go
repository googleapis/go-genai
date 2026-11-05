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
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	dummyExtrasProvider = func(body map[string]any) map[string]any { return body }
)

func TestMergeHTTPOptions(t *testing.T) {
	tests := []struct {
		name               string
		clientConfig       *ClientConfig
		requestHTTPOptions *HTTPOptions
		want               *HTTPOptions
	}{
		{
			name:               "both nil",
			clientConfig:       nil,
			requestHTTPOptions: nil,
			want:               nil,
		},
		{
			name:         "client nil and request not nil",
			clientConfig: nil,
			requestHTTPOptions: &HTTPOptions{
				BaseURL:    "https://example.com",
				APIVersion: "v1",
			},
			want: &HTTPOptions{
				BaseURL:    "https://example.com",
				APIVersion: "v1",
				Headers:    http.Header{},
			},
		},
		{
			name: "client not nil and request nil",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:    "https://client.com",
					APIVersion: "v2",
				},
			},
			requestHTTPOptions: nil,
			want: &HTTPOptions{
				BaseURL:    "https://client.com",
				APIVersion: "v2",
				Headers:    http.Header{},
			},
		},
		{
			name: "both have values, request overrides",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:    "https://client.com",
					APIVersion: "v2",
				},
			},
			requestHTTPOptions: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v3",
			},
			want: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v3",
				Headers:    http.Header{},
			},
		},
		{
			name: "both have values, request only updates some",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:    "https://client.com",
					APIVersion: "v2",
				},
			},
			requestHTTPOptions: &HTTPOptions{
				BaseURL: "https://request.com",
			},
			want: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v2",
				Headers:    http.Header{},
			},
		},
		{
			name: "client retry options are inherited when the request sets none",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:      "https://client.com",
					RetryOptions: &HTTPRetryOptions{Attempts: Ptr(int32(4))},
				},
			},
			requestHTTPOptions: &HTTPOptions{APIVersion: "v3"},
			want: &HTTPOptions{
				BaseURL:      "https://client.com",
				APIVersion:   "v3",
				RetryOptions: &HTTPRetryOptions{Attempts: Ptr(int32(4))},
				Headers:      http.Header{},
			},
		},
		{
			name: "request retry options override the client ones",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:      "https://client.com",
					RetryOptions: &HTTPRetryOptions{Attempts: Ptr(int32(4))},
				},
			},
			requestHTTPOptions: &HTTPOptions{
				RetryOptions: &HTTPRetryOptions{Attempts: Ptr(int32(2))},
			},
			want: &HTTPOptions{
				BaseURL:      "https://client.com",
				RetryOptions: &HTTPRetryOptions{Attempts: Ptr(int32(2))},
				Headers:      http.Header{},
			},
		},
		{
			name: "client config only",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:    "https://client.com",
					APIVersion: "v2",
				},
			},
			requestHTTPOptions: &HTTPOptions{},
			want: &HTTPOptions{
				BaseURL:    "https://client.com",
				APIVersion: "v2",
				Headers:    http.Header{},
			},
		},
		{
			name:         "empty client config",
			clientConfig: &ClientConfig{},
			requestHTTPOptions: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v3",
			},
			want: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v3",
				Headers:    http.Header{},
			},
		},
		{
			name:         "empty client and valid request",
			clientConfig: &ClientConfig{},
			requestHTTPOptions: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v3",
			},
			want: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v3",
				Headers:    http.Header{},
			},
		},
		{
			name: "merge headers",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:    "https://client.com",
					APIVersion: "v2",
					Headers: http.Header{
						"X-Client-Header-1": []string{"value1"},
						"X-Client-Header-2": []string{"value2"},
					},
				},
			},
			requestHTTPOptions: &HTTPOptions{
				BaseURL: "https://request.com",
				Headers: http.Header{
					"X-Request-Header-1": []string{"value3"},
					"X-Client-Header-2":  []string{"value4"},
				},
			},
			want: &HTTPOptions{
				BaseURL:    "https://request.com",
				APIVersion: "v2",
				Headers: http.Header{
					"X-Client-Header-1":  []string{"value1"},
					"X-Client-Header-2":  []string{"value2", "value4"},
					"X-Request-Header-1": []string{"value3"},
				},
			},
		},
		{
			name: "ExtrasRequestProvider in request only",
			requestHTTPOptions: &HTTPOptions{
				ExtrasRequestProvider: dummyExtrasProvider,
			},
			want: &HTTPOptions{
				ExtrasRequestProvider: dummyExtrasProvider,
				Headers:               http.Header{},
			},
		},
		{
			name: "ExtrasRequestProvider in client config only",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					ExtrasRequestProvider: dummyExtrasProvider,
				},
			},
			requestHTTPOptions: &HTTPOptions{},
			want: &HTTPOptions{
				ExtrasRequestProvider: dummyExtrasProvider,
				Headers:               http.Header{},
			},
		},
		{
			name: "ExtrasRequestProvider in both, request overrides",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					ExtrasRequestProvider: func(body map[string]any) map[string]any { return nil }, // Different provider
				},
			},
			requestHTTPOptions: &HTTPOptions{
				ExtrasRequestProvider: dummyExtrasProvider,
			},
			want: &HTTPOptions{
				ExtrasRequestProvider: dummyExtrasProvider,
				Headers:               http.Header{},
			},
		},
		{
			name: "ExtrasRequestProvider in neither",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{},
			},
			requestHTTPOptions: &HTTPOptions{},
			want: &HTTPOptions{
				ExtrasRequestProvider: nil,
				Headers:               http.Header{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeHTTPOptions(tt.clientConfig, tt.requestHTTPOptions)
			// Compare ExtrasRequestProvider by checking if they are both nil or both non-nil
			// as direct comparison of func pointers might not be reliable.
			opt := cmp.Comparer(func(x, y ExtrasRequestProvider) bool {
				return (x == nil && y == nil) || (x != nil && y != nil)
			})
			if diff := cmp.Diff(tt.want, got, opt); diff != "" {
				t.Errorf("mergeHTTPOptions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetValueByPath(t *testing.T) {
	tests := []struct {
		name  string
		data  map[string]any
		keys  []string
		value any
		want  map[string]any
	}{
		{
			name:  "Simple",
			data:  map[string]any{},
			keys:  []string{"a", "b"},
			value: "v",
			want:  map[string]any{"a": map[string]any{"b": "v"}},
		},
		{
			name:  "Empty_value",
			data:  map[string]any{},
			keys:  []string{"a", "b"},
			value: 0,
			want:  map[string]any{"a": map[string]any{"b": 0}},
		},
		{
			name:  "Nested",
			data:  map[string]any{"a": map[string]any{}},
			keys:  []string{"a", "b", "c"},
			value: "v",
			want:  map[string]any{"a": map[string]any{"b": map[string]any{"c": "v"}}},
		},
		{
			name:  "String_Array",
			data:  map[string]any{},
			keys:  []string{"b[]", "c"},
			value: []string{"v3", "v4"},
			want:  map[string]any{"b": []map[string]any{{"c": "v3"}, {"c": "v4"}}},
		},
		{
			name:  "Any_Array",
			data:  map[string]any{},
			keys:  []string{"a", "b[]", "c"},
			value: []any{"v1", "v2"},
			want:  map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
		},
		{
			name:  "Array_Existing",
			data:  map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
			keys:  []string{"a", "b[]", "d"},
			value: "v3",
			want:  map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1", "d": "v3"}, {"c": "v2", "d": "v3"}}}},
		},
		{
			name:  "Nil_value",
			data:  map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
			keys:  []string{"a", "b[]", "d"},
			value: nil,
			want:  map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
		},
		{
			name:  "Self_key",
			data:  map[string]any{"a": map[string]any{"b": map[string]any{}}},
			keys:  []string{"_self"},
			value: map[string]any{"c": "v"},
			want:  map[string]any{"a": map[string]any{"b": map[string]any{}}, "c": "v"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			setValueByPath(tt.data, tt.keys, tt.value)
			if diff := cmp.Diff(tt.data, tt.want); diff != "" {
				t.Errorf("setValueByPath() mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

func TestGetValueByPath(t *testing.T) {
	tests := []struct {
		name      string
		data      any
		keys      []string
		want      any
		wantPanic bool
	}{
		{
			name: "Simple",
			data: map[string]any{"a": map[string]any{"b": "v"}},
			keys: []string{"a", "b"},
			want: "v",
		},
		{
			name: "Array_Starting_Element",
			data: map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}},
			keys: []string{"b[]", "c"},
			want: []any{"v1", "v2"},
		},
		{
			name: "Array_Middle_Element",
			data: map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
			keys: []string{"a", "b[]", "c"},
			want: []any{"v1", "v2"},
		},
		{
			name: "KeyNotFound",
			data: map[string]any{"a": map[string]any{"b": "v"}},
			keys: []string{"a", "c"},
			want: nil,
		},
		{
			name: "NilData",
			data: nil,
			keys: []string{"a", "b"},
			want: nil,
		},
		{
			name: "WrongData",
			data: "data",
			keys: []string{"a", "b"},
			want: nil,
		},
		{
			name: "Self",
			data: map[string]any{"a": map[string]any{"b": "v"}},
			keys: []string{"_self"},
			want: map[string]any{"a": map[string]any{"b": "v"}},
		},
		{
			name: "empty key",
			data: map[string]any{"a": map[string]any{"b": "v"}},
			keys: []string{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("The code panicked unexpectedly: %v", r)
					}
				} else {
					if tt.wantPanic {
						t.Errorf("The code did not panic as expected")
					}
				}
			}()

			if tt.wantPanic {
				_ = getValueByPath(tt.data, tt.keys) // This should panic
			} else {
				got := getValueByPath(tt.data, tt.keys)
				if diff := cmp.Diff(got, tt.want); diff != "" {
					t.Errorf("getValueByPath() mismatch (-want +got):\n%s", diff)
				}
			}

		})
	}
}

func TestGetValueByPathOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		data         any
		keys         []string
		defaultValue any
		want         any
	}{
		{
			name:         "Simple_Exists",
			data:         map[string]any{"a": map[string]any{"b": "v"}},
			keys:         []string{"a", "b"},
			defaultValue: "default",
			want:         "v",
		},
		{
			name:         "Simple_Not_Exists",
			data:         map[string]any{"a": map[string]any{"b": "v"}},
			keys:         []string{"a", "c"},
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "Array_Exists",
			data:         map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
			keys:         []string{"a", "b[]", "c"},
			defaultValue: "default",
			want:         []any{"v1", "v2"},
		},
		{
			name:         "Array_Not_Exists",
			data:         map[string]any{"a": map[string]any{"b": []map[string]any{{"c": "v1"}, {"c": "v2"}}}},
			keys:         []string{"a", "b[]", "d"},
			defaultValue: "default",
			want:         []any{"default", "default"},
		},
		{
			name:         "Nil_Value_Returns_Default",
			data:         map[string]any{"a": map[string]any{"b": nil}},
			keys:         []string{"a", "b"},
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "Self_Key",
			data:         map[string]any{"a": "v"},
			keys:         []string{"_self"},
			defaultValue: "default",
			want:         map[string]any{"a": "v"},
		},
		{
			name:         "Empty_Keys",
			data:         map[string]any{"a": "v"},
			keys:         []string{},
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "Integer_Default",
			data:         map[string]any{"a": "v"},
			keys:         []string{"b"},
			defaultValue: 123,
			want:         123,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getValueByPathOrDefault(tt.data, tt.keys, tt.defaultValue)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("getValueByPathOrDefault() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMoveValueByPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		paths    map[string]string
		expected map[string]any
	}{
		{
			name: "DocstringExample",
			data: map[string]any{
				"requests": []any{
					map[string]any{"content": "v1"},
					map[string]any{"content": "v2"},
				},
			},
			paths: map[string]string{"requests[].*": "requests[].request.*"},
			expected: map[string]any{
				"requests": []any{
					map[string]any{"request": map[string]any{"content": "v1"}},
					map[string]any{"request": map[string]any{"content": "v2"}},
				},
			},
		},
		{
			name: "ComplexNestedStructure",
			data: map[string]any{
				"requests": []any{
					map[string]any{
						"request": map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{
										"text": "1",
									},
								},
							},
						},
						"outputDimensionality": 64,
					},
					map[string]any{
						"request": map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{
										"text": "2",
									},
								},
							},
						},
						"outputDimensionality": 64,
					},
					map[string]any{
						"request": map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{
										"text": "3",
									},
								},
							},
						},
						"outputDimensionality": 64,
					},
				},
			},
			paths: map[string]string{"requests[].*": "requests[].request.*"},
			expected: map[string]any{
				"requests": []any{
					map[string]any{
						"request": map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{
										"text": "1",
									},
								},
							},
							"outputDimensionality": 64,
						},
					},
					map[string]any{
						"request": map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{
										"text": "2",
									},
								},
							},
							"outputDimensionality": 64,
						},
					},
					map[string]any{
						"request": map[string]any{
							"content": map[string]any{
								"parts": []any{
									map[string]any{
										"text": "3",
									},
								},
							},
							"outputDimensionality": 64,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moveValueByPath(tt.data, tt.paths)
			if diff := cmp.Diff(tt.expected, tt.data); diff != "" {
				t.Errorf("moveValueByPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// copyConverter returns a shallow copy of fromObject.
func copyConverter(fromObject, _, _ map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range fromObject {
		out[k] = v
	}
	return out, nil
}

var errTestConverter = errors.New("converter failed")

func failingConverter(_, _, _ map[string]any) (map[string]any, error) {
	return nil, errTestConverter
}

// applyConverterToSliceTests pins how each slice element reaches the converter:
// a null element becomes an empty object at the same index, and a non-object
// element is an error rather than a panic.
var applyConverterToSliceTests = []struct {
	name      string
	inputs    []any
	converter converterFuncWithRoot
	want      []map[string]any
	wantErr   []string
	wantErrIs error
}{
	{
		name:      "all objects",
		inputs:    []any{map[string]any{"a": "b"}, map[string]any{"c": "d"}},
		converter: copyConverter,
		want:      []map[string]any{{"a": "b"}, {"c": "d"}},
	},
	{
		name:      "nil element becomes empty object",
		inputs:    []any{map[string]any{"a": "b"}, nil},
		converter: copyConverter,
		want:      []map[string]any{{"a": "b"}, {}},
	},
	{
		name:      "empty input",
		inputs:    []any{},
		converter: copyConverter,
		want:      nil,
	},
	{
		name:      "string element returns error",
		inputs:    []any{"x"},
		converter: copyConverter,
		wantErr:   []string{"element 0", "string"},
	},
	{
		name:      "number element returns error",
		inputs:    []any{map[string]any{}, float64(1)},
		converter: copyConverter,
		wantErr:   []string{"element 1", "float64"},
	},
	{
		name:      "converter error propagates",
		inputs:    []any{map[string]any{"a": "b"}},
		converter: failingConverter,
		wantErrIs: errTestConverter,
	},
}

func checkApplyConverterToSliceResult(t *testing.T, got []map[string]any, err error, want []map[string]any, wantErr []string, wantErrIs error) {
	t.Helper()
	if wantErr == nil && wantErrIs == nil {
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		return
	}
	if err == nil {
		t.Fatalf("err = nil, want an error")
	}
	if got != nil {
		t.Errorf("result = %v, want nil", got)
	}
	if wantErrIs != nil && !errors.Is(err, wantErrIs) {
		t.Errorf("err = %v, want %v", err, wantErrIs)
	}
	for _, s := range wantErr {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("err = %q, want it to contain %q", err.Error(), s)
		}
	}
}

func TestApplyConverterToSliceWithRoot(t *testing.T) {
	for _, tt := range applyConverterToSliceTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyConverterToSliceWithRoot(tt.inputs, tt.converter, nil)
			checkApplyConverterToSliceResult(t, got, err, tt.want, tt.wantErr, tt.wantErrIs)
		})
	}
}

func TestApplyConverterToSliceWithClientWithRoot(t *testing.T) {
	for _, tt := range applyConverterToSliceTests {
		t.Run(tt.name, func(t *testing.T) {
			converter := func(_ *apiClient, fromObject, parentObject, rootObject map[string]any) (map[string]any, error) {
				return tt.converter(fromObject, parentObject, rootObject)
			}
			got, err := applyConverterToSliceWithClientWithRoot(nil, tt.inputs, converter, nil)
			checkApplyConverterToSliceResult(t, got, err, tt.want, tt.wantErr, tt.wantErrIs)
		})
	}
}
