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
			name:         "client nil",
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
			name: "request nil",
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
			name: "empty request",
			clientConfig: &ClientConfig{
				HTTPOptions: HTTPOptions{
					BaseURL:    "",
					APIVersion: "",
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
			name:         "empty client and request",
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
