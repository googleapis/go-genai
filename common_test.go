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
	"testing"

	"github.com/google/go-cmp/cmp"
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeHTTPOptions(tt.clientConfig, tt.requestHTTPOptions)
			if diff := cmp.Diff(tt.want, got); diff != "" {
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
