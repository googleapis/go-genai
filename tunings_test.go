package genai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestTuningsTuneUnit(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		baseModel       string
		trainingDataset *TuningDataset
		config          *CreateTuningJobConfig
		expectedPath    string
		expectedBody    map[string]any
		backend         Backend
		envVarProvider  func() map[string]string
	}{
		{
			name:      "VertexAI_BaseModel_DPO",
			baseModel: "gemini-2.5-flash",
			trainingDataset: &TuningDataset{
				GCSURI: "gs://test-bucket/train.jsonl",
			},
			config: &CreateTuningJobConfig{
				TunedModelDisplayName: "Test Tuned Model",
				Method:                TuningMethodPreferenceTuning,
				AdapterSize:           AdapterSizeOne,
			},
			expectedPath: "/tuningJobs",
			expectedBody: map[string]any{
				"baseModel":             "gemini-2.5-flash",
				"tunedModelDisplayName": "Test Tuned Model",
				"preferenceOptimizationSpec": map[string]any{
					"trainingDatasetUri": "gs://test-bucket/train.jsonl",
					"hyperParameters": map[string]any{
						"adapterSize": "ADAPTER_SIZE_ONE",
					},
				},
			},
			backend: BackendVertexAI,
			envVarProvider: func() map[string]string {
				return map[string]string{
					"GOOGLE_API_KEY": "test-api-key",
				}
			},
		},
		{
			name:      "VertexAI_BaseModel",
			baseModel: "gemini-1.5-pro-001",
			trainingDataset: &TuningDataset{
				GCSURI: "gs://test-bucket/train.jsonl",
			},
			config: &CreateTuningJobConfig{
				TunedModelDisplayName: "Test Tuned Model",
			},
			expectedPath: "/tuningJobs",
			expectedBody: map[string]any{
				"baseModel":             "gemini-1.5-pro-001",
				"tunedModelDisplayName": "Test Tuned Model",
				"supervisedTuningSpec": map[string]any{
					"trainingDatasetUri": "gs://test-bucket/train.jsonl",
				},
			},
			backend: BackendVertexAI,
			envVarProvider: func() map[string]string {
				return map[string]string{
					"GOOGLE_API_KEY": "test-api-key",
				}
			},
		},
		{
			name:      "VertexAI_PreTunedModel",
			baseModel: "projects/123/locations/us-central1/models/456",
			trainingDataset: &TuningDataset{
				GCSURI: "gs://test-bucket/train.jsonl",
			},
			expectedPath: "/tuningJobs",
			expectedBody: map[string]any{
				"preTunedModel": map[string]any{
					"tunedModelName": "projects/123/locations/us-central1/models/456",
				},
				"supervisedTuningSpec": map[string]any{
					"trainingDatasetUri": "gs://test-bucket/train.jsonl",
				},
			},
			backend: BackendVertexAI,
			envVarProvider: func() map[string]string {
				return map[string]string{
					"GOOGLE_API_KEY": "test-api-key",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected method %s, got %s", http.MethodPost, r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, tt.expectedPath) {
					t.Errorf("Expected path suffix %s, got %s", tt.expectedPath, r.URL.Path)
				}

				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				if diff := cmp.Diff(tt.expectedBody, body); diff != "" {
					t.Errorf("Request body mismatch (-want +got):\n%s", diff)
				}

				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(map[string]any{
					"name":  "projects/123/locations/us-central1/tuningJobs/789",
					"state": "JOB_STATE_SUCCEEDED",
					"tunedModel": map[string]any{
						"model": "projects/123/locations/us-central1/models/abc",
					},
				}); err != nil {
					t.Fatalf("Failed to encode response: %v", err)
				}
			}))
			defer ts.Close()

			client, err := NewClient(ctx, &ClientConfig{Backend: tt.backend, HTTPOptions: HTTPOptions{BaseURL: ts.URL}, envVarProvider: tt.envVarProvider})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			_, err = client.Tunings.Tune(ctx, tt.baseModel, tt.trainingDataset, tt.config)
			if err != nil {
				t.Errorf("Tunings.Tune() failed unexpectedly: %v", err)
			}
		})
	}
}
func TestTuningsTuneAPIMode(t *testing.T) {
	if *mode != apiMode {
		t.Skip("Skip. This test is only in the API mode")
	}
	ctx := context.Background()

	t.Run("VertexAI", func(t *testing.T) {
		if isDisabledTest(t) {
			t.Skip("Skip: disabled test")
		}
		client, err := NewClient(ctx, &ClientConfig{Backend: BackendVertexAI})
		if err != nil {
			t.Fatal(err)
		}

		trainingDataset := &TuningDataset{
			GCSURI: "gs://cloud-samples-data/ai-platform/generative_ai/gemini-2_0/text/sft_train_data.jsonl",
		}

		// Test tuning with a base model.
		baseModel := "gemini-2.5-flash"
		baseJob, err := client.Tunings.Tune(ctx, baseModel, trainingDataset, nil)
		if err != nil {
			t.Fatalf("Tunings.Tune() with base model failed: %v", err)
		}

		if baseJob.State != JobStatePending && baseJob.State != JobStateRunning && baseJob.State != JobStateQueued {
			t.Errorf("Expected base job state to be PENDING, RUNNING or QUEUED, but got %s", baseJob.State)
		}

		// Wait for the base tuning job to start running (not waiting for completion)
		for baseJob.State != JobStateRunning && baseJob.State != JobStateSucceeded && baseJob.State != JobStateFailed && baseJob.State != JobStateCancelled {
			time.Sleep(10 * time.Second) // Poll every 10 seconds
			baseJob, err = client.Tunings.Get(ctx, baseJob.Name, nil)
			if err != nil {
				t.Fatalf("Failed to get status of base tuning job: %v", err)
			}
		}

		if baseJob.State != JobStateRunning && baseJob.State != JobStateSucceeded {
			t.Fatalf("Base tuning job did not start running successfully. State: %s", baseJob.State)
		}

		preTunedModelName := "projects/801452371447/locations/us-central1/models/3399218262595076096"

		// Test tuning with a pre-tuned model.
		continuousJob, err := client.Tunings.Tune(ctx, preTunedModelName, trainingDataset, nil)

		if err != nil {
			t.Fatalf("Tunings.Tune() with pre-tuned model failed: %v", err)
		}

		if continuousJob.State != JobStatePending && continuousJob.State != JobStateRunning && continuousJob.State != JobStateQueued {
			t.Errorf("Expected continuous job state to be PENDING, RUNNING or QUEUED, but got %s", continuousJob.State)
		}

		// Wait for the continuous tuning job to start running (not waiting for completion)
		for continuousJob.State != JobStateRunning && continuousJob.State != JobStateSucceeded && continuousJob.State != JobStateFailed && continuousJob.State != JobStateCancelled {
			time.Sleep(10 * time.Second) // Poll every 10 seconds
			continuousJob, err = client.Tunings.Get(ctx, continuousJob.Name, nil)
			if err != nil {
				t.Fatalf("Failed to get status of continuous tuning job: %v", err)
			}
		}
	})
}
