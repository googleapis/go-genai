package genai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/auth"

	"github.com/google/go-cmp/cmp"
)

func TestFilesDownload(t *testing.T) {
	// Create a test server that returns different content based on the URI
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.Path
		switch uri {
		case "/test-version/files/filename:download":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("test download content"))
			if err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tests := []struct {
		name           string
		uri            DownloadURI
		config         *UploadFileConfig
		want           []byte
		wantVideoBytes []byte // Expected VideoBytes if uri is a Video type
		wantErr        bool
	}{
		{
			name: "SuccessfulFileDownload",
			uri: &File{
				DownloadURI: ts.URL + "test-version/files/filename",
			},
			want: []byte("test download content"),
		},
		{
			name: "SuccessfulFileDownload_ShortName",
			uri: &File{
				DownloadURI: "files/filename",
			},
			want: []byte("test download content"),
		},
		{
			name: "SuccessfulVideoDownload",
			uri: &Video{
				URI: ts.URL + "test-version/files/filename",
			},
			want:           []byte("test download content"),
			wantVideoBytes: []byte("test download content"),
		},
		{
			name: "SuccessfulGeneratedVideoDownload",
			uri: &GeneratedVideo{
				Video: &Video{
					URI: ts.URL + "test-version/files/filename",
				},
			},
			want:           []byte("test download content"),
			wantVideoBytes: []byte("test download content"),
		},
		{
			name:    "EmptyURI",
			uri:     &File{},
			wantErr: true,
		},
		{
			name: "InvalidPath1",
			uri: &File{
				DownloadURI: ts.URL + "test-version/invalid/filename",
			},
			wantErr: true,
		},
		{
			name: "InvalidPath2",
			uri: &File{
				DownloadURI: ts.URL + "test-version/files/-",
			},
			wantErr: true,
		},
	}

	mldevClient, err := NewClient(context.Background(), &ClientConfig{
		HTTPOptions: HTTPOptions{BaseURL: ts.URL, APIVersion: "test-version"},
		HTTPClient:  ts.Client(),
		Credentials: &auth.Credentials{}, // Replace with your actual credentials.
		envVarProvider: func() map[string]string {
			return map[string]string{
				"GOOGLE_API_KEY": "test-api-key",
			}
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mldevClient.Files.Download(context.Background(), tt.uri, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Files.Download() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("Files.Download() mismatch (-want +got):\n%s", diff)
				}
				if tt.wantVideoBytes != nil {
					switch v := tt.uri.(type) {
					case *Video:
						if diff := cmp.Diff(tt.wantVideoBytes, v.VideoBytes); diff != "" {
							t.Errorf("Video.VideoBytes mismatch (-want +got):\n%s", diff)
						}
					case *GeneratedVideo:
						if diff := cmp.Diff(tt.wantVideoBytes, v.Video.VideoBytes); diff != "" {
							t.Errorf("GeneratedVideo.Video.VideoBytes mismatch (-want +got):\n%s", diff)
						}
					}
				}
			}
		})
	}

	vertexClient, err := NewClient(context.Background(), &ClientConfig{
		Backend: BackendVertexAI,
		envVarProvider: func() map[string]string {
			return map[string]string{
				"GOOGLE_CLOUD_PROJECT":  "test-project",
				"GOOGLE_CLOUD_LOCATION": "test-location",
			}
		},
		Credentials: &auth.Credentials{},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("VertexFilesDownloadNotSupported", func(t *testing.T) {
		_, err := vertexClient.Files.Download(context.Background(), &File{DownloadURI: "something"}, nil)
		if err == nil {
			t.Errorf("Files.Download() succeeded, want error")
		}
		if !strings.Contains(err.Error(), "method Upload is only supported in the Gemini Developer client") {
			t.Errorf("Files.Download() error = %v, want error containing 'method Upload is only supported in the Gemini Developer client'", err)
		}
	})
}
