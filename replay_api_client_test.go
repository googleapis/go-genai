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

func TestRedactVersionNumbers(t *testing.T) {
	testCases := []struct {
		desc  string
		param string
		want  string
	}{
		{
			desc:  "with genai",
			param: "google-genai-sdk/1.56.0 gl-go/go1.25.0",
			want:  "google-genai-sdk/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
		{
			desc:  "with genai prerelease version",
			param: "google-genai-sdk/1.56.0-20260427-RC04 gl-go/go1.25.0",
			want:  "google-genai-sdk/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
		{
			desc:  "with vertex",
			param: "google-genai-sdk/1.56.0 gl-go/go1.25.0",
			want:  "google-genai-sdk/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
		{
			desc:  "with genai and vertex",
			param: "google-genai-sdk/1.56.0+vertex-genai-modules/0.20.0 gl-go/go1.25.0",
			want:  "google-genai-sdk/{VERSION_NUMBER}+vertex-genai-modules/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
		{
			desc:  "with genai and vertex prerelease version",
			param: "google-genai-sdk/1.56.0+vertex-genai-modules/0.20.0-20260427-RC04 gl-go/go1.25.0",
			want:  "google-genai-sdk/{VERSION_NUMBER}+vertex-genai-modules/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
		{
			desc:  "with golang prerelease version",
			param: "google-genai-sdk/1.56.0+vertex-genai-modules/0.20.0 gl-go/go1.25.0-20260427-RC04",
			want:  "google-genai-sdk/{VERSION_NUMBER}+vertex-genai-modules/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
		{
			desc:  "with golang nightly build version",
			param: "google-genai-sdk/1.56.0 gl-go/go1.27-20260427-RC04 cl/906595525 +5fb2392a6f X:fieldtrack,boringcrypto,simd",
			want:  "google-genai-sdk/{VERSION_NUMBER} gl-go/{VERSION_NUMBER}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := redactVersionNumbers(tc.param)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("redactVersionNumbers() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
