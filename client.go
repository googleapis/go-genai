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
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Client is the GenAI client.
type Client struct {
	clientConfig ClientConfig
	Models       *Models
	Live         *Live
	Caches       *Caches
}

// Backend is the GenAI backend to use for the client.
type Backend int

const (
	// BackendUnspecified causes the backend determined automatically. If the
	// GOOGLE_GENAI_USE_VERTEXAI environment variable is set to "1" or "true", then
	// the backend is BackendVertexAI. Otherwise, if GOOGLE_GENAI_USE_VERTEXAI
	// is unset or set to any other value, then BackendGeminiAPI is used.  Explicitly
	// setting the backend in ClientConfig overrides the environment variable.
	BackendUnspecified Backend = iota
	// BackendGeminiAPI is the Gemini API backend.
	BackendGeminiAPI
	// BackendVertexAI is the Vertex AI backend.
	BackendVertexAI
)

// The Stringer interface for Backend.
func (t Backend) String() string {
	switch t {
	case BackendGeminiAPI:
		return "BackendGeminiAPI"
	case BackendVertexAI:
		return "BackendVertexAI"
	default:
		return "BackendUnspecified"
	}
}

// ClientConfig is the configuration for the GenAI client.
type ClientConfig struct {
	APIKey      string              // API Key for GenAI. Required for BackendGeminiAPI.
	Backend     Backend             // Backend for GenAI. See Backend constants. Defaults to BackendGeminiAPI unless explicitly set to BackendVertexAI, or the environment variable GOOGLE_GENAI_USE_VERTEXAI is set to "1" or "true".
	Project     string              // GCP Project ID for Vertex AI. Required for BackendVertexAI.
	Location    string              // GCP Location/Region for Vertex AI. Required for BackendVertexAI. See https://cloud.google.com/vertex-ai/docs/general/locations
	Credentials *google.Credentials // Optional. Google credentials.  If not specified, application default credentials will be used.
	HTTPClient  *http.Client        // Optional HTTP client to use. If nil, a default client will be created. For Vertex AI, this client must handle authentication appropriately.
	HTTPOptions HTTPOptions         // Optional HTTP options to override.
}

// NewClient creates a new GenAI client.
//
// You can configure the client by passing in a ClientConfig struct.
func NewClient(ctx context.Context, cc *ClientConfig) (*Client, error) {
	if cc == nil {
		cc = &ClientConfig{}
	}

	if cc.Project != "" && cc.APIKey != "" {
		return nil, fmt.Errorf("project and API key are mutually exclusive in the client initializer. ClientConfig: %v", cc)
	}
	if cc.Location != "" && cc.APIKey != "" {
		return nil, fmt.Errorf("location and API key are mutually exclusive in the client initializer. ClientConfig: %v", cc)
	}

	if cc.Backend == BackendUnspecified {
		if v, ok := os.LookupEnv("GOOGLE_GENAI_USE_VERTEXAI"); ok {
			v = strings.ToLower(v)
			if v == "1" || v == "true" {
				cc.Backend = BackendVertexAI
			} else {
				cc.Backend = BackendGeminiAPI
			}
		} else {
			cc.Backend = BackendGeminiAPI
		}
	}

	// Only set the API key for MLDev API.
	if cc.APIKey == "" && cc.Backend == BackendGeminiAPI {
		cc.APIKey = os.Getenv("GOOGLE_API_KEY")
	}
	if cc.Project == "" {
		cc.Project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if cc.Location == "" {
		if location, ok := os.LookupEnv("GOOGLE_CLOUD_LOCATION"); ok {
			cc.Location = location
		} else if location, ok := os.LookupEnv("GOOGLE_CLOUD_REGION"); ok {
			cc.Location = location
		}
	}

	if cc.Backend == BackendVertexAI {
		if cc.Project == "" {
			return nil, fmt.Errorf("project is required for Vertex AI backend. ClientConfig: %v", cc)
		}
		if cc.Location == "" {
			return nil, fmt.Errorf("location is required for Vertex AI backend. ClientConfig: %v", cc)
		}
	} else {
		if cc.APIKey == "" {
			return nil, fmt.Errorf("api key is required for Google AI backend. ClientConfig: %v.\nYou can get the API key from https://ai.google.dev/gemini-api/docs/api-key", cc)
		}
	}

	if cc.Backend == BackendVertexAI && cc.Credentials == nil {
		cred, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		cc.Credentials = cred
	}

	if cc.HTTPOptions.BaseURL == "" && cc.Backend == BackendVertexAI {
		if cc.Location == "global" {
			cc.HTTPOptions.BaseURL = "https://aiplatform.googleapis.com/"
		} else {
			cc.HTTPOptions.BaseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/", cc.Location)
		}
	} else if cc.HTTPOptions.BaseURL == "" {
		cc.HTTPOptions.BaseURL = "https://generativelanguage.googleapis.com/"
	}

	if cc.HTTPOptions.APIVersion == "" && cc.Backend == BackendVertexAI {
		cc.HTTPOptions.APIVersion = "v1beta1"
	} else if cc.HTTPOptions.APIVersion == "" {
		cc.HTTPOptions.APIVersion = "v1beta"
	}

	if cc.HTTPClient == nil {
		if cc.Backend == BackendVertexAI {
			cc.HTTPClient = oauth2.NewClient(ctx, oauth2.ReuseTokenSource(nil, cc.Credentials.TokenSource))
		} else {
			cc.HTTPClient = &http.Client{}
		}
	}

	ac := &apiClient{clientConfig: cc}
	c := &Client{
		clientConfig: *cc,
		Models:       &Models{apiClient: ac},
		Live:         &Live{apiClient: ac},
		Caches:       &Caches{apiClient: ac},
	}
	return c, nil
}

// ClientConfig returns the ClientConfig for the client.
//
// The returned ClientConfig is a copy of the ClientConfig used to create the client.
func (c Client) ClientConfig() ClientConfig {
	return c.clientConfig
}
