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

func TestModelTransformer(t *testing.T) {
	tests := []struct {
		name         string
		backend      Backend
		input        string
		want         string
		wantErr      bool
		wantFullName string
	}{
		{
			name:         "VertexAI_Model",
			backend:      BackendVertexAI,
			input:        "gemini-2.0-flash",
			want:         "publishers/google/models/gemini-2.0-flash",
			wantFullName: "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
		},
		{
			name:         "VertexAI_Model_Models_Prefix",
			backend:      BackendVertexAI,
			input:        "models/gemini-2.0-flash",
			want:         "models/gemini-2.0-flash",
			wantFullName: "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
		},
		{
			name:         "VertexAI_Model_Publisher",
			backend:      BackendVertexAI,
			input:        "google/gemini-2.0-flash",
			want:         "publishers/google/models/gemini-2.0-flash",
			wantFullName: "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
		},
		{
			name:         "VertexAI_Model_Publisher_Prefix",
			backend:      BackendVertexAI,
			input:        "publishers/google/models/gemini-2.0-flash",
			want:         "publishers/google/models/gemini-2.0-flash",
			wantFullName: "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
		},
		{
			name:         "VertexAI_Model_Project_Prefix",
			backend:      BackendVertexAI,
			input:        "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
			want:         "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
			wantFullName: "projects/test-project/locations/test-location/publishers/google/models/gemini-2.0-flash",
		},

		{
			name:         "GoogleAI_Model_Short",
			backend:      BackendGeminiAPI,
			input:        "gemini-2.0-flash",
			want:         "models/gemini-2.0-flash",
			wantFullName: "models/gemini-2.0-flash",
		},
		{
			name:         "GoogleAI_Model_Full",
			backend:      BackendGeminiAPI,
			input:        "models/gemini-2.0-flash",
			want:         "models/gemini-2.0-flash",
			wantFullName: "models/gemini-2.0-flash",
		},
		{
			name:         "GoogleAI_Model_TunedModel",
			backend:      BackendGeminiAPI,
			input:        "tunedModels/your-tuned-model",
			want:         "tunedModels/your-tuned-model",
			wantFullName: "tunedModels/your-tuned-model",
		},
		{
			name:    "Empty_Model",
			backend: BackendVertexAI,
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &apiClient{clientConfig: &ClientConfig{
				Backend:  tt.backend,
				Project:  "test-project",
				Location: "test-location",
			}}
			got, err := tModel(ac, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("tModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !cmp.Equal(got, tt.want) {
				t.Errorf("tModel() got = %v, want %v", got, tt.want)
			}

			if !tt.wantErr {
				gotFullName, err := tModelFullName(ac, tt.input)
				if err != nil {
					t.Errorf("tModelFullName() error = %v", err)
					return
				}
				if !cmp.Equal(gotFullName, tt.wantFullName) {
					t.Errorf("tModelFullName() got = %v, want %v", gotFullName, tt.wantFullName)
				}
			}
		})
	}
	t.Run("Invalid_Model_Type", func(t *testing.T) {
		_, err := tModel(&apiClient{}, 123) // Invalid input type (int)
		if err == nil {
			t.Error("tModel() expected error for invalid input type, got nil")
		}
	})
	t.Run("Invalid_ModelFullName_Type", func(t *testing.T) {
		_, err := tModelFullName(&apiClient{}, 123) // Invalid input type (int)
		if err == nil {
			t.Error("tModelFullName() expected error for invalid input type, got nil")
		}
	})
}

func TestSchemaTransformer(t *testing.T) {
	tests := []struct {
		name    string
		backend Backend
		input   map[string]any
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "GeminiAPI_Schema_With_Default",
			backend: BackendGeminiAPI,
			input: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"name": map[string]any{
						"type":    "STRING",
						"default": "test",
					},
				},
			},
			want: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"name": map[string]any{
						"type":    "STRING",
						"default": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "VertexAI_Schema_With_Default",
			backend: BackendVertexAI,
			input: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"type":    "STRING",
					"default": "test",
				},
			},
			want: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"type":    "STRING",
					"default": "test",
				},
			},
			wantErr: false,
		},
		{
			name:    "VertexAI_Schema_With_AnyOf",
			backend: BackendVertexAI,
			input: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"type": "STRING",
					"anyOf": []any{
						map[string]any{
							"type": "STRING",
						},
						map[string]any{
							"type": "NUMBER",
						},
					},
				},
			},
			want: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"type": "STRING",
					"anyOf": []any{
						map[string]any{
							"type": "STRING",
						},
						map[string]any{
							"type": "NUMBER",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "VertexAI_Schema_With_Items",
			backend: BackendVertexAI,
			input: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"type": "STRING",
					"items": map[string]any{
						"type": "STRING",
					},
				},
			},
			want: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"type": "STRING",
					"items": map[string]any{
						"type": "STRING",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tSchema(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("tSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !cmp.Equal(got, tt.want) {
				t.Errorf("tSchema() got = %v, want %v", got, tt.want)
			}
		})
	}
}
