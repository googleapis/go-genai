// Copyright 2025 Google LLC
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
	"os"
)

var defaultBaseGeminiURL *string = nil
var defaultBaseVertexURL *string = nil

// BaseURLParameters are parameters for setting the base URLs for the Gemini API and Vertex AI API.
type BaseURLParameters struct {
	GeminiURL *string
	VertexURL *string
}

// SetDefaultBaseURLs overrides the default base URLs for the Gemini API and Vertex AI API.
func SetDefaultBaseURLs(baseURLParams *BaseURLParameters) {
	if baseURLParams != nil {
		defaultBaseGeminiURL = baseURLParams.GeminiURL
		defaultBaseVertexURL = baseURLParams.VertexURL
	}
}

// GetDefaultBaseURLs returns the default base URLs for the Gemini API and Vertex AI API.
func GetDefaultBaseURLs() *BaseURLParameters {
	return &BaseURLParameters{
		GeminiURL: defaultBaseGeminiURL,
		VertexURL: defaultBaseVertexURL,
	}
}

// GetBaseURL returns the default base URL based on the following priority:
//
// 1. The base URL provided in the httpOptions.
// 2. The base URL set for Vertex AI.
// 3. The base URL set for Gemini.
func GetBaseURL(vertexai bool, httpOptions *HTTPOptions) *string {
	if httpOptions != nil && httpOptions.BaseURL != "" {
		return &httpOptions.BaseURL
	}
	baseURLs := GetDefaultBaseURLs()
	if vertexai {
		if baseURLs.VertexURL != nil {
			return baseURLs.VertexURL
		} else if v, ok := os.LookupEnv("GOOGLE_VERTEX_BASE_URL"); ok {
			return &v
		}
	} else {
		if baseURLs.GeminiURL != nil {
			return baseURLs.GeminiURL
		} else if v, ok := os.LookupEnv("GOOGLE_GEMINI_BASE_URL"); ok {
			return &v
		}
	}

	return nil
}
