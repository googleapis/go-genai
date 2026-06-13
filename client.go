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
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
)

// Client is the GenAI client. It provides access to the various GenAI services.
type Client struct {
	clientConfig ClientConfig
	// Models provides access to the Models service.
	Models *Models
	// Live provides access to the Live service.
	Live *Live
	// Caches provides access to the Caches service.
	Caches *Caches
	// Chats provides util functions for creating a new chat session.
	Chats *Chats
	// Files provides access to the Files service.
	Files *Files
	// Operations provides access to long-running operations.
	Operations *Operations
	// FileSearchStores provides access to the File Search Stores service.
	FileSearchStores *FileSearchStores
	// Batches provides access to the Batch service.
	Batches *Batches
	// Tunings provides access to the Tunings service.
	Tunings *Tunings
	// Tokens provides access to the Tokens service.
	AuthTokens *Tokens
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
	// BackendEnterprise is the Gemini Enterprise Agent Platform backend.
	BackendEnterprise
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

var multiRegionalLocations = map[string]bool{
	"us": true,
	"eu": true,
}

// ClientConfig is the configuration for the GenAI client.
type ClientConfig struct {
	// Optional. API Key for GenAI. Required for BackendGeminiAPI.
	// Can also be set via the GOOGLE_API_KEY or GEMINI_API_KEY environment variable.
	// Get a Gemini API key: https://ai.google.dev/gemini-api/docs/api-key
	APIKey string

	// Optional. Backend for GenAI. See Backend constants. Defaults to BackendGeminiAPI unless explicitly set to BackendVertexAI,
	// or the environment variable GOOGLE_GENAI_USE_VERTEXAI is set to "1" or "true".
	Backend Backend

	// Optional. GCP Project ID for Vertex AI. Required for BackendVertexAI.
	// Can also be set via the GOOGLE_CLOUD_PROJECT environment variable.
	// Find your Project ID: https://cloud.google.com/resource-manager/docs/creating-managing-projects#identifying_projects
	Project string

	// Optional. GCP Location/Region for Vertex AI. Required for BackendVertexAI.
	// Can also be set via the GOOGLE_CLOUD_LOCATION or GOOGLE_CLOUD_REGION environment variable.
	// Generative AI locations: https://cloud.google.com/vertex-ai/generative-ai/docs/learn/locations.
	Location string

	// Optional. Google credentials.  If not specified, [Application Default Credentials] will be used.
	//
	// [Application Default Credentials]: https://developers.google.com/accounts/docs/application-default-credentials
	Credentials *auth.Credentials

	// Optional HTTP client to use. If nil, a default client will be created.
	// For Vertex AI, this client must handle authentication appropriately.
	// Otherwise, call [UseDefaultCredentials] convenience method to add default credentials to the
	// client.
	HTTPClient *http.Client

	// Optional HTTP options to override.
	HTTPOptions HTTPOptions

	envVarProvider func() map[string]string
}

func defaultEnvVarProvider() map[string]string {
	vars := make(map[string]string)
	if v, ok := os.LookupEnv("GOOGLE_GENAI_USE_ENTERPRISE"); ok {
		vars["GOOGLE_GENAI_USE_ENTERPRISE"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_GENAI_USE_VERTEXAI"); ok {
		vars["GOOGLE_GENAI_USE_VERTEXAI"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_API_KEY"); ok {
		vars["GOOGLE_API_KEY"] = v
	}
	if v, ok := os.LookupEnv("GEMINI_API_KEY"); ok {
		vars["GEMINI_API_KEY"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_CLOUD_PROJECT"); ok {
		vars["GOOGLE_CLOUD_PROJECT"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_CLOUD_LOCATION"); ok {
		vars["GOOGLE_CLOUD_LOCATION"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_CLOUD_REGION"); ok {
		vars["GOOGLE_CLOUD_REGION"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_GEMINI_BASE_URL"); ok {
		vars["GOOGLE_GEMINI_BASE_URL"] = v
	}
	if v, ok := os.LookupEnv("GOOGLE_VERTEX_BASE_URL"); ok {
		vars["GOOGLE_VERTEX_BASE_URL"] = v
	}
	return vars
}

func getAPIKeyFromEnv(envVars map[string]string) string {
	googleAPIKey := envVars["GOOGLE_API_KEY"]
	geminiAPIKey := envVars["GEMINI_API_KEY"]
	if googleAPIKey != "" && geminiAPIKey != "" {
		log.Printf("Warning: Both GOOGLE_API_KEY and GEMINI_API_KEY are set. Using GOOGLE_API_KEY.")
	}
	if googleAPIKey != "" {
		return googleAPIKey
	}
	return geminiAPIKey
}

// NewInternalAPIClient creates a new internal API client.
//
// You can configure the client by passing in a ClientConfig struct.
//
// If a nil ClientConfig is provided, the client will be configured using
// default settings and environment variables:
//
//   - Environment Variables for BackendGeminiAPI:
//
//   - GEMINI_API_KEY: Specifies the API key for the Gemini API.
//
//   - GOOGLE_API_KEY: Can also be used to specify the API key for the Gemini API.
//     If both GOOGLE_API_KEY and GEMINI_API_KEY are set, GOOGLE_API_KEY will be used.
//
//   - Environment Variables for BackendVertexAI:
//
//   - GOOGLE_GENAI_USE_VERTEXAI: Must be set to "1" or "true" to use the Vertex AI
//     backend.
//
//   - GOOGLE_CLOUD_PROJECT: Required. Specifies the GCP project ID.
//
//   - GOOGLE_CLOUD_LOCATION or GOOGLE_CLOUD_REGION: Required. Specifies the GCP
//     location/region.
//
// If using the Vertex AI backend and no credentials are provided in the
// ClientConfig, the client will attempt to use application default credentials.
func NewInternalAPIClient(ctx context.Context, cc *ClientConfig) (*InternalAPIClient, error) {
	if cc == nil {
		cc = &ClientConfig{}
	}

	if cc.envVarProvider == nil {
		cc.envVarProvider = defaultEnvVarProvider
	}
	envVars := cc.envVarProvider()

	if cc.Project != "" && cc.APIKey != "" {
		return nil, fmt.Errorf("project and API key are mutually exclusive in the client initializer. ClientConfig: %#v", cc)
	}
	if cc.Location != "" && cc.APIKey != "" {
		return nil, fmt.Errorf("location and API key are mutually exclusive in the client initializer. ClientConfig: %#v", cc)
	}
	if cc.Credentials != nil && cc.APIKey != "" {
		return nil, fmt.Errorf("credentials and API key are mutually exclusive in the client initializer. ClientConfig: %#v", cc)
	}

	if cc.Backend == BackendEnterprise {
		cc.Backend = BackendVertexAI
	}

	if cc.Backend == BackendUnspecified {
		vEnterprise, enterpriseOK := envVars["GOOGLE_GENAI_USE_ENTERPRISE"]
		vVertex, vertexOK := envVars["GOOGLE_GENAI_USE_VERTEXAI"]

		isEnterprise := enterpriseOK && (strings.ToLower(vEnterprise) == "1" || strings.ToLower(vEnterprise) == "true")

		isVertexAI := vertexOK && (strings.ToLower(vVertex) == "1" || strings.ToLower(vVertex) == "true")

		if enterpriseOK && vertexOK {
			if isEnterprise != isVertexAI {
				log.Println("Warning: Both GOOGLE_GENAI_USE_ENTERPRISE and GOOGLE_GENAI_USE_VERTEXAI are set with conflicting values. The value of GOOGLE_GENAI_USE_ENTERPRISE will be used.")
			}
		}

		if enterpriseOK {
			if isEnterprise {
				cc.Backend = BackendVertexAI
			} else {
				cc.Backend = BackendGeminiAPI
			}
		} else if vertexOK {
			if isVertexAI {
				cc.Backend = BackendVertexAI
			} else {
				cc.Backend = BackendGeminiAPI
			}
		} else {
			cc.Backend = BackendGeminiAPI
		}
	}

	// Retrieve implicitly set values from the environment.
	envAPIKey := getAPIKeyFromEnv(envVars)
	envProject := envVars["GOOGLE_CLOUD_PROJECT"]
	envLocation := ""
	if location, ok := envVars["GOOGLE_CLOUD_LOCATION"]; ok {
		envLocation = location
	} else if location, ok := envVars["GOOGLE_CLOUD_REGION"]; ok {
		envLocation = location
	}
	configAPIKey := cc.APIKey
	configProject := cc.Project
	configLocation := cc.Location
	if cc.APIKey == "" {
		cc.APIKey = envAPIKey
	}
	if cc.Project == "" {
		cc.Project = envProject
	}
	if cc.Location == "" {
		cc.Location = envLocation
	}

	if cc.Backend == BackendVertexAI {
		// Handle when to use Vertex AI in express mode (api key).
		// Explicit initializer arguments are already validated above.
		if cc.Credentials != nil && envAPIKey != "" {
			log.Println("Warning: The user provided Google Cloud credentials will take precedence over the API key from the environment variable.")
			cc.APIKey = ""
		} else if configAPIKey != "" && (envProject != "" || envLocation != "") {
			// Explicit API key takes precedence over implicit project/location.
			log.Println("Warning: The user provided Vertex AI API key will take precedence over the project/location from the environment variables.")
			cc.Project = ""
			cc.Location = ""
		} else if (configProject != "" || configLocation != "") && envAPIKey != "" {
			// Explicit project/location takes precedence over implicit API key.
			log.Println("Warning: The user provided project/location will take precedence over the API key from the environment variable.")
			cc.APIKey = ""
		} else if (envProject != "" || envLocation != "") && envAPIKey != "" {
			// Implicit project/location takes precedence over implicit API key.
			log.Println("Warning: The project/location from the environment variables will take precedence over the API key from the environment variable.")
			cc.APIKey = ""
		}

		// Resolve BaseURL early to determine if we have a custom base URL.
		baseURL := getBaseURL(cc.Backend, &cc.HTTPOptions, envVars)
		if baseURL != "" {
			cc.HTTPOptions.BaseURL = baseURL
		}

		// If no location, no API key, and no custom Base URL, default to global.
		if cc.Location == "" && cc.APIKey == "" && cc.HTTPOptions.BaseURL == "" {
			cc.Location = "global"
		}

		configHasSufficientAuth := (configProject != "" && configLocation != "") || configAPIKey != ""
		hasSufficientAuth := (cc.Project != "" && cc.Location != "") || cc.APIKey != ""
		hasCustomBaseURL := cc.HTTPOptions.BaseURL != ""

		if !hasSufficientAuth && !hasCustomBaseURL {
			return nil, fmt.Errorf("project/location or API key must be set when using Vertex AI backend. ClientConfig: %#v", cc)
		}

		// If we have a custom Base URL and NO standard auth, clear Project/Location to avoid prefixing.
		// If we have a custom Base URL (and it's not the default googleapis ones effectively, though here we just check if it overrides),
		// and we DON'T have sufficient auth (Project/Location or APIKey), assume the Base URL handles it (e.g. proxy) or the user knows what they are doing.
		// In this case, we clear Project and Location to avoid 'projects//locations//' path construction if they were partial.
		if hasCustomBaseURL && !configHasSufficientAuth {
			cc.Project = ""
			cc.Location = ""
		}

		// Set default BaseURL if still empty.
		if cc.HTTPOptions.BaseURL == "" {
			if cc.Location == "global" || cc.APIKey != "" {
				cc.HTTPOptions.BaseURL = "https://aiplatform.googleapis.com/"
			} else if multiRegionalLocations[cc.Location] {
				cc.HTTPOptions.BaseURL = fmt.Sprintf("https://aiplatform.%s.rep.googleapis.com/", cc.Location)
			} else {
				cc.HTTPOptions.BaseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/", cc.Location)
			}
		}
	} else {
		// Mldev API
		// Resolve BaseURL for Gemini API
		baseURL := getBaseURL(cc.Backend, &cc.HTTPOptions, envVars)
		if baseURL != "" {
			cc.HTTPOptions.BaseURL = baseURL
		}
		if cc.HTTPOptions.BaseURL == "" {
			cc.HTTPOptions.BaseURL = "https://generativelanguage.googleapis.com/"
		}

		if cc.APIKey == "" {
			return nil, fmt.Errorf("api key is required for Google AI backend. ClientConfig: %#v.\nYou can get the API key from https://ai.google.dev/gemini-api/docs/api-key", cc)
		}
	}

	skipADC := cc.HTTPOptions.BaseURL != "" && cc.Project == "" && cc.Location == "" && cc.APIKey == ""
	if cc.Backend == BackendVertexAI && cc.Credentials == nil && cc.APIKey == "" && cc.HTTPClient == nil && !skipADC {
		cred, err := credentials.DetectDefault(&credentials.DetectOptions{
			Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		cc.Credentials = cred
	}

	if cc.HTTPOptions.APIVersion == "" && cc.Backend == BackendVertexAI {
		cc.HTTPOptions.APIVersion = "v1beta1"
	} else if cc.HTTPOptions.APIVersion == "" {
		cc.HTTPOptions.APIVersion = "v1beta"
	}

	if cc.HTTPClient == nil {
		// x-goog-api-key header is set for Express mode in api_client.go
		if cc.Backend == BackendVertexAI && cc.APIKey == "" && cc.Credentials != nil {
			quotaProjectID, err := cc.Credentials.QuotaProjectID(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get quota project ID: %w", err)
			}
			client, err := httptransport.NewClient(&httptransport.Options{
				Credentials: cc.Credentials,
				Headers: http.Header{
					"X-Goog-User-Project": []string{quotaProjectID},
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create HTTP client: %w", err)
			}
			cc.HTTPClient = client
		} else {
			// If credentials are provided for Gemini API, create an authenticated HTTP client
			if cc.Credentials != nil {
				client, err := httptransport.NewClient(&httptransport.Options{
					Credentials: cc.Credentials,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to create HTTP client: %w", err)
				}
				cc.HTTPClient = client
			} else {
				cc.HTTPClient = &http.Client{}
			}
		}
	}
	return &apiClient{clientConfig: cc}, nil
}

// NewClient creates a new GenAI client.
//
// You can configure the client by passing in a ClientConfig struct.
//
// If a nil ClientConfig is provided, the client will be configured using
// default settings and environment variables:
//
//   - Environment Variables for BackendGeminiAPI:
//
//   - GEMINI_API_KEY: Specifies the API key for the Gemini API.
//
//   - GOOGLE_API_KEY: Can also be used to specify the API key for the Gemini API.
//     If both GOOGLE_API_KEY and GEMINI_API_KEY are set, GOOGLE_API_KEY will be used.
//
//   - Environment Variables for BackendVertexAI:
//
//   - GOOGLE_GENAI_USE_VERTEXAI: Must be set to "1" or "true" to use the Vertex AI
//     backend.
//
//   - GOOGLE_CLOUD_PROJECT: Required. Specifies the GCP project ID.
//
//   - GOOGLE_CLOUD_LOCATION or GOOGLE_CLOUD_REGION: Required. Specifies the GCP
//     location/region.
//
// If using the Vertex AI backend and no credentials are provided in the
// ClientConfig, the client will attempt to use application default credentials.
func NewClient(ctx context.Context, cc *ClientConfig) (*Client, error) {
	if cc == nil {
		cc = &ClientConfig{}
	}
	ac, err := NewInternalAPIClient(ctx, cc)
	if err != nil {
		return nil, err
	}
	c := &Client{
		clientConfig:     *cc,
		Models:           &Models{apiClient: ac},
		Live:             &Live{apiClient: ac},
		Caches:           &Caches{apiClient: ac},
		Chats:            &Chats{apiClient: ac},
		Operations:       &Operations{apiClient: ac},
		FileSearchStores: &FileSearchStores{apiClient: ac, Documents: &Documents{apiClient: ac}},
		Files:            &Files{apiClient: ac},
		Batches:          &Batches{apiClient: ac},
		Tunings:          &Tunings{apiClient: ac},
		AuthTokens:       &Tokens{apiClient: ac},
	}
	return c, nil
}

// ClientConfig returns the ClientConfig for the client.
//
// The returned ClientConfig is a copy of the ClientConfig used to create the client.
func (c Client) ClientConfig() ClientConfig {
	return c.clientConfig
}

// UseDefaultCredentials sets the credentials to use default credentials and
// add authorization middleware to the HTTP client.
//
// If the ClientConfig already has credentials, this method will return an error.
//
// Use this method if your provided HTTPClient doesn't handles credentials.
func (cc *ClientConfig) UseDefaultCredentials() error {
	if cc.Credentials != nil {
		return fmt.Errorf("Credentials are already set")
	}
	if cc.Credentials == nil {
		cred, err := credentials.DetectDefault(&credentials.DetectOptions{
			Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		})
		if err != nil {
			return fmt.Errorf("failed to find default credentials: %w", err)
		}
		cc.Credentials = cred
	}
	if cc.HTTPClient != nil {
		err := httptransport.AddAuthorizationMiddleware(cc.HTTPClient, cc.Credentials)
		if err != nil {
			return fmt.Errorf("failed to create HTTP client: %w", err)
		}
	}
	return nil
}
