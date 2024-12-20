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
	ClientConfig *ClientConfig
	Models       *Models
}

// Backend is the GenAI backend to use for the client.
type Backend int

const (
	// BackendUnspecified causes the backend determined automatically. If the
	// GOOGLE_GENAI_USE_VERTEXAI environment variable is set to "1" or "true", then
	// the backend is `BackendVertexAI`. Otherwise, if GOOGLE_GENAI_USE_VERTEXAI
	// is unset or set to any other value, then `BackendGoogleAI` is used.  Explicitly
	// setting the backend in ClientConfig overrides the environment variable.
	BackendUnspecified Backend = iota
	// BackendGoogleAI is the Google AI backend.
	BackendGoogleAI
	// BackendVertexAI is the Vertex AI backend.
	BackendVertexAI
)

// The Stringer interface for Backend.
func (t Backend) String() string {
	switch t {
	case BackendGoogleAI:
		return "BackendGoogleAI"
	case BackendVertexAI:
		return "BackendVertexAI"
	default:
		return "BackendUnspecified"
	}
}

// ClientConfig is the configuration for the GenAI client.
type ClientConfig struct {
	APIKey      string              // API Key for GenAI. Required for BackendGoogleAI.
	Backend     Backend             // Backend for GenAI. See Backend constants. Defaults to BackendGoogleAI unless explicitly set to BackendVertexAI, or the environment variable GOOGLE_GENAI_USE_VERTEXAI is set to "1" or "true".
	Project     string              // GCP Project ID for Vertex AI. Required for BackendVertexAI.
	Location    string              // GCP Location/Region for Vertex AI. Required for BackendVertexAI. See https://cloud.google.com/vertex-ai/docs/general/locations
	Credentials *google.Credentials // Optional. Google credentials.  If not specified, application default credentials will be used.
	HTTPOptions *HTTPOptions        // Optional HTTP configuration.
}

// HTTPOptions is the HTTP configuration for the GenAI client.
type HTTPOptions struct {
	BaseURL    string            // The base URL for the AI platform service endpoint.
	APIVersion string            // Specifies the version of the API to use.
	Headers    map[string]string // Additional HTTP headers to be sent with the request.
	HTTPClient *http.Client      // Optional HTTP client to use. If nil, a default client will be created. For Vertex AI, this client must handle authentication appropriately.
}

// NewClient creates a new GenAI client.
//
// You can configure the client by passing in a ClientConfig struct.
func NewClient(ctx context.Context, cc *ClientConfig) (*Client, error) {
	if cc == nil {
		cc = &ClientConfig{}
	}
	if cc.HTTPOptions == nil {
		cc.HTTPOptions = &HTTPOptions{}
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
				cc.Backend = BackendGoogleAI
			}
		} else {
			cc.Backend = BackendGoogleAI
		}
	}

	// Only set the API key for MLDev API.
	if cc.APIKey == "" && cc.Backend == BackendGoogleAI {
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
			return nil, fmt.Errorf("api key is required for Google AI backend. ClientConfig: %v", cc)
		}
	}

	if cc.Backend == BackendVertexAI && cc.Credentials == nil {
		cred, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		cc.Credentials = cred
	}

	if cc.HTTPOptions.BaseURL == "" {
		if cc.Backend == BackendVertexAI {
			cc.HTTPOptions.BaseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/", cc.Location)
		} else {
			cc.HTTPOptions.BaseURL = "https://generativelanguage.googleapis.com/"
		}
	}

	if cc.Backend == BackendVertexAI && cc.HTTPOptions.APIVersion == "" {
		cc.HTTPOptions.APIVersion = "v1beta1"
	} else if cc.Backend == BackendGoogleAI && cc.HTTPOptions.APIVersion == "" {
		cc.HTTPOptions.APIVersion = "v1beta"
	}

	if cc.HTTPOptions.HTTPClient == nil {
		if cc.Backend == BackendVertexAI {
			cc.HTTPOptions.HTTPClient = oauth2.NewClient(ctx, oauth2.ReuseTokenSource(nil, cc.Credentials.TokenSource))
		} else {
			cc.HTTPOptions.HTTPClient = &http.Client{}
		}
	}

	ac := &apiClient{ClientConfig: *cc}
	c := &Client{
		ClientConfig: cc,
		Models:       &Models{apiClient: ac},
	}
	return c, nil
}
