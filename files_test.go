package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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
		config         *DownloadFileConfig
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
		if !strings.Contains(err.Error(), "method Download is only supported in the Gemini Developer client") {
			t.Errorf("Files.Download() error = %v, want error containing 'method Upload is only supported in the Gemini Developer client'", err)
		}
	})
}

func TestFilesAll(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name              string
		serverResponses   []map[string]any
		expectedFiles     []*File
		expectedNextPages []string
	}{
		{
			name: "Pagination_SinglePage",
			serverResponses: []map[string]any{
				{
					"files": []*File{
						{Name: "file1", DisplayName: "File 1"},
						{Name: "file2", DisplayName: "File 2"},
					},
					"nextPageToken": "",
				},
			},
			expectedFiles: []*File{
				{Name: "file1", DisplayName: "File 1"},
				{Name: "file2", DisplayName: "File 2"},
			},
		},
		{
			name: "Pagination_MultiplePages",
			serverResponses: []map[string]any{
				{
					"files": []*File{
						{Name: "file1", DisplayName: "File 1"},
					},
					"nextPageToken": "next_page_token",
				},
				{
					"files": []*File{
						{Name: "file2", DisplayName: "File 2"},
						{Name: "file3", DisplayName: "File 3"},
					},
					"nextPageToken": "",
				},
			},
			expectedFiles: []*File{
				{Name: "file1", DisplayName: "File 1"},
				{Name: "file2", DisplayName: "File 2"},
				{Name: "file3", DisplayName: "File 3"},
			},
		},
		{
			name:              "Empty_Response",
			serverResponses:   []map[string]any{{"files": []*File{}, "nextPageToken": ""}},
			expectedFiles:     []*File{},
			expectedNextPages: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseIndex := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if responseIndex > 0 && r.URL.Query().Get("pageToken") == "" {
					t.Errorf("Files.All() failed to pass pageToken in the request")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				response, err := json.Marshal(tt.serverResponses[responseIndex])
				if err != nil {
					t.Errorf("Failed to marshal response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, err = w.Write(response)
				if err != nil {
					t.Errorf("Failed to write response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				responseIndex++
			}))
			defer ts.Close()

			client, err := NewClient(ctx, &ClientConfig{HTTPOptions: HTTPOptions{BaseURL: ts.URL},
				envVarProvider: func() map[string]string {
					return map[string]string{
						"GOOGLE_API_KEY": "test-api-key",
					}
				},
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			gotFiles := []*File{}
			for file, err := range client.Files.All(ctx) {
				if err != nil {
					t.Errorf("Files.All() iteration error = %v", err)
					return
				}
				gotFiles = append(gotFiles, file)
			}

			if diff := cmp.Diff(tt.expectedFiles, gotFiles); diff != "" {
				t.Errorf("Files.All() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// MockUploadServer simulates the resumable upload process.
type MockUploadServer struct {
	t             *testing.T
	mu            sync.Mutex
	uploads       map[string]*uploadSession // Map upload URL path -> session
	nextUploadID  int
	baseURL       string
	createHandler http.HandlerFunc
	uploadHandler http.HandlerFunc
}

type uploadSession struct {
	totalSize    int64
	receivedSize int64
	finalized    bool
	fileMetadata File // Metadata sent in the 'start' request
}

func NewMockUploadServer(t *testing.T) *MockUploadServer {
	s := &MockUploadServer{
		t:       t,
		uploads: make(map[string]*uploadSession),
	}
	s.createHandler = s.handleCreate
	s.uploadHandler = s.handleUpload
	return s
}

func (s *MockUploadServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.t.Logf("Mock server received request: %s %s Headers: %v", r.Method, r.URL.Path, r.Header)
	if strings.HasPrefix(r.URL.Path, "/upload/v1beta/files") && r.Header.Get("X-Goog-Upload-Protocol") == "resumable" && r.Header.Get("X-Goog-Upload-Command") == "start" {
		s.createHandler(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/upload-session/") {
		s.uploadHandler(w, r)
	} else {
		s.t.Logf("Mock server: Unknown request path %s", r.URL.Path)
		http.NotFound(w, r)
	}
}

func (s *MockUploadServer) handleCreate(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.Method != http.MethodPost {
		s.t.Errorf("handleCreate: Expected POST, got %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		File File `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err != io.EOF { // Allow empty body
		s.t.Errorf("handleCreate: Failed to decode request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	uploadID := s.nextUploadID
	s.nextUploadID++
	uploadPath := fmt.Sprintf("/upload-session/%d", uploadID)
	uploadURL := s.baseURL + uploadPath

	contentLengthStr := r.Header.Get("X-Goog-Upload-Header-Content-Length")
	totalSize, _ := strconv.ParseInt(contentLengthStr, 10, 64) // Ignore error for testing simplicity if header missing

	s.uploads[uploadPath] = &uploadSession{
		totalSize:    totalSize,
		fileMetadata: reqBody.File,
	}

	s.t.Logf("handleCreate: Started upload session %s for file %+v, totalSize: %d", uploadPath, reqBody.File, totalSize)

	w.Header().Set("X-Goog-Upload-URL", uploadURL)
	w.WriteHeader(http.StatusOK)
	// No body needed for the create file response
}

func (s *MockUploadServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.Method != http.MethodPost {
		s.t.Errorf("handleUpload: Expected POST, got %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := s.uploads[r.URL.Path]
	if !ok {
		s.t.Errorf("handleUpload: Upload session not found for path %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	offsetStr := r.Header.Get("X-Goog-Upload-Offset")
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		s.t.Errorf("handleUpload: Invalid X-Goog-Upload-Offset: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if offset != session.receivedSize {
		s.t.Errorf("handleUpload: Mismatched offset. Expected %d, got %d", session.receivedSize, offset)
		http.Error(w, "Conflict", http.StatusConflict)
		return
	}

	chunkSize := r.ContentLength
	if chunkSize < 0 {
		s.t.Errorf("handleUpload: Invalid Content-Length: %d", chunkSize)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Simulate reading the chunk
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.t.Errorf("handleUpload: Failed to read request body: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if int64(len(bodyBytes)) != chunkSize {
		s.t.Errorf("handleUpload: Actual body size %d does not match Content-Length %d", len(bodyBytes), chunkSize)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	session.receivedSize += chunkSize
	s.t.Logf("handleUpload: Received chunk for %s. Offset: %d, Size: %d, Total Received: %d", r.URL.Path, offset, chunkSize, session.receivedSize)

	uploadCommand := r.Header.Get("X-Goog-Upload-Command")
	isFinalize := strings.Contains(uploadCommand, "finalize")

	if isFinalize {
		if session.totalSize > 0 && session.receivedSize != session.totalSize {
			s.t.Errorf("handleUpload: Finalize received but total size mismatch. Expected %d, got %d", session.totalSize, session.receivedSize)
			http.Error(w, "Bad Request - Size Mismatch on Finalize", http.StatusBadRequest)
			return
		}
		session.finalized = true
		s.t.Logf("handleUpload: Finalizing upload for %s", r.URL.Path)
		w.Header().Set("X-Goog-Upload-Status", "final")
		w.WriteHeader(http.StatusOK)

		// Return the final file metadata
		finalFile := session.fileMetadata
		if finalFile.Name == "" {
			// Assign a mock name if not provided
			finalFile.Name = fmt.Sprintf("files/generated-%s", filepath.Base(r.URL.Path))
		}
		finalFile.SizeBytes = Ptr[int64](session.receivedSize)
		finalFile.State = FileStateActive
		finalFile.CreateTime = time.Now().UTC()
		finalFile.UpdateTime = finalFile.CreateTime
		finalFile.MIMEType = r.Header.Get("X-Goog-Upload-Header-Content-Type")

		var respBody struct {
			File *File `json:"file"`
		}
		respBody.File = &finalFile
		if err := json.NewEncoder(w).Encode(respBody); err != nil {
			s.t.Errorf("handleUpload: Failed to encode final response: %v", err)
			// Header already sent, can't change status code here
		}
	} else {
		s.t.Logf("handleUpload: Upload active for %s", r.URL.Path)
		w.Header().Set("X-Goog-Upload-Status", "active")
		w.WriteHeader(http.StatusOK)
		// No body needed for active status
	}
}

func TestFilesUpload(t *testing.T) {
	ctx := context.Background()
	mockServer := NewMockUploadServer(t)
	ts := httptest.NewServer(mockServer)
	defer ts.Close()
	mockServer.baseURL = ts.URL

	client, err := NewClient(ctx, &ClientConfig{
		Backend: BackendGeminiAPI, // Upload only supported on Gemini API
		APIKey:  "test-api-key",
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
		HTTPClient: ts.Client(),
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name        string
		inputData   string
		config      *UploadFileConfig
		wantFile    *File // Only check key fields that the mock sets
		wantErr     bool
		wantErrMsg  string
		setupServer func(*MockUploadServer) // Optional setup for specific server behavior
	}{
		{
			name:      "Success - Small File (Single Chunk)",
			inputData: "This is a small file.",
			config: &UploadFileConfig{
				Name:        "files/small.txt",
				DisplayName: "My Small File",
				MIMEType:    "text/plain",
			},
			wantFile: &File{
				Name:        "files/small.txt",
				DisplayName: "My Small File",
				MIMEType:    "text/plain",
				SizeBytes:   Ptr(int64(len("This is a small file."))),
				State:       FileStateActive,
			},
			wantErr: false,
		},
		{
			name:      "Success - No Config",
			inputData: "Data with no config.",
			config:    nil, // Rely on mock server defaults
			wantFile: &File{
				Name:      "files/generated-0", // Mock generated name
				MIMEType:  "",                  // No MIME type provided
				SizeBytes: Ptr(int64(len("Data with no config."))),
				State:     FileStateActive,
			},
			wantErr: false,
		},
		{
			name:      "Success - Large File (Multiple Chunks)",
			inputData: strings.Repeat("A", int(maxChunkSize)+100), // Slightly larger than one chunk
			config: &UploadFileConfig{
				MIMEType: "application/octet-stream",
			},
			wantFile: &File{
				Name:      "files/generated-1", // Mock generated name
				MIMEType:  "application/octet-stream",
				SizeBytes: Ptr(int64(maxChunkSize + 100)),
				State:     FileStateActive,
			},
			wantErr: false,
		},
		{
			name:      "Success - Exact Multiple Chunks",
			inputData: strings.Repeat("B", int(maxChunkSize*2)), // Exactly two chunks
			config: &UploadFileConfig{
				MIMEType: "application/octet-stream",
			},
			wantFile: &File{
				Name:      "files/generated-2", // Mock generated name
				MIMEType:  "application/octet-stream",
				SizeBytes: Ptr(int64(maxChunkSize * 2)),
				State:     FileStateActive,
			},
			wantErr: false,
		},
		{
			name:      "Error - Create Fails (Server Error)",
			inputData: "data",
			config:    &UploadFileConfig{MIMEType: "text/plain"},
			wantErr:   true,
			setupServer: func(s *MockUploadServer) {
				s.createHandler = func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			},
			wantErrMsg: "Failed to create file. Ran into an error: Error 500",
		},
		{
			name:      "Error - Create Fails (No Upload URL)",
			inputData: "data",
			config:    &UploadFileConfig{MIMEType: "text/plain"},
			wantErr:   true,
			setupServer: func(s *MockUploadServer) {
				s.createHandler = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK) // No X-Goog-Upload-URL header
				}
			},
			wantErrMsg: "Failed to create file. Upload URL was not returned",
		},
		{
			name:      "Error - Upload Chunk Fails (Server Error)",
			inputData: "data",
			config:    &UploadFileConfig{MIMEType: "text/plain"},
			wantErr:   true,
			setupServer: func(s *MockUploadServer) {
				s.uploadHandler = func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Goog-Upload-Status", "active")
					w.WriteHeader(http.StatusOK)
					// Fail on the first actual upload chunk
					http.Error(w, "Chunk Upload Failed", http.StatusInternalServerError)
				}
			},
			wantErrMsg: "response body is invalid for chunk at offset 0",
		},
		{
			name:      "Error - Upload Finalize Fails (Wrong Status)",
			inputData: "data",
			config:    &UploadFileConfig{MIMEType: "text/plain"},
			wantErr:   true,
			setupServer: func(s *MockUploadServer) {
				originalUploadHandler := s.uploadHandler
				s.uploadHandler = func(w http.ResponseWriter, r *http.Request) {
					uploadCommand := r.Header.Get("X-Goog-Upload-Command")
					if strings.Contains(uploadCommand, "finalize") {
						w.Header().Set("X-Goog-Upload-Status", "active") // Wrong status
						w.WriteHeader(http.StatusOK)
					} else {
						originalUploadHandler(w, r) // Process normally if not finalize
					}
				}
			},
			wantErrMsg: "send finalize command but doesn't receive final status. Offset 4, Bytes read: 4, Upload status: active",
		},
		{
			name:       "Error - Reader Error",
			inputData:  "", // Input data doesn't matter here
			config:     &UploadFileConfig{MIMEType: "text/plain"},
			wantErr:    true,
			wantErrMsg: "Failed to read bytes from file at offset 0: intentional read error",
			// We'll inject the error reader directly in the test run
		},
	}

	for i, tt := range tests {
		// Reset server handlers for each test unless overridden
		mockServer.createHandler = mockServer.handleCreate
		mockServer.uploadHandler = mockServer.handleUpload
		if tt.setupServer != nil {
			tt.setupServer(mockServer)
		}
		// Reset upload ID for predictable generated names if needed
		mockServer.nextUploadID = i

		t.Run(tt.name, func(t *testing.T) {
			var reader io.Reader
			if strings.Contains(tt.name, "Error - Reader Error") {
				reader = &errorReader{}
			} else {
				reader = strings.NewReader(tt.inputData)
			}

			gotFile, err := client.Files.Upload(ctx, reader, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Upload() expected error, but got nil")
				} else if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Upload() error = %q, want error containing %q", err, tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Upload() unexpected error: %v", err)
				}
				if gotFile == nil {
					t.Fatal("Upload() returned nil file on success")
				}
				// Compare only the fields set by the mock
				opts := cmp.Options{
					cmp.FilterPath(func(p cmp.Path) bool {
						// Only compare fields explicitly set in wantFile
						field := p.Last().String()
						switch field {
						case ".Name", ".DisplayName", ".MIMEType", ".SizeBytes", ".State":
							return false // Keep these fields
						default:
							return true // Ignore other fields (like timestamps)
						}
					}, cmp.Ignore()),
				}
				if diff := cmp.Diff(tt.wantFile, gotFile, opts); diff != "" {
					t.Errorf("Upload() file mismatch (-want +got):\n%s", diff)
				}
				// Verify the upload session on the mock server
				mockServer.mu.Lock()
				sessionPath := fmt.Sprintf("/upload-session/%d", i) // Based on test index
				session, ok := mockServer.uploads[sessionPath]
				if !ok {
					t.Errorf("Mock server state error: session %s not found after successful upload", sessionPath)
				} else {
					if !session.finalized {
						t.Errorf("Mock server state error: session %s not finalized after successful upload", sessionPath)
					}
					if session.receivedSize != int64(len(tt.inputData)) {
						t.Errorf("Mock server state error: session %s received size %d, want %d", sessionPath, session.receivedSize, len(tt.inputData))
					}
				}
				mockServer.mu.Unlock()
			}
		})
	}
}

func TestFilesUploadFromPath(t *testing.T) {
	ctx := context.Background()
	mockServer := NewMockUploadServer(t)
	ts := httptest.NewServer(mockServer)
	defer ts.Close()
	mockServer.baseURL = ts.URL

	client, err := NewClient(ctx, &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "test-api-key",
		HTTPOptions: HTTPOptions{
			BaseURL: ts.URL,
		},
		HTTPClient: ts.Client(),
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Setup Temp File
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	fileContent := "Content for UploadFromPath test."
	err = os.WriteFile(filePath, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		config      *UploadFileConfig
		wantFile    *File
		wantErr     bool
		wantErrMsg  string
		setupServer func(*MockUploadServer)
	}{
		{
			name: "Success",
			path: filePath,
			config: &UploadFileConfig{
				Name:        "files/frompath.txt",
				DisplayName: "From Path",
			},
			wantFile: &File{
				Name:        "files/frompath.txt",
				DisplayName: "From Path",
				MIMEType:    "text/plain", // Auto-detected
				SizeBytes:   Ptr(int64(len(fileContent))),
				State:       FileStateActive,
			},
			wantErr: false,
		},
		{
			name: "Success - Config Overrides MIME",
			path: filePath,
			config: &UploadFileConfig{
				MIMEType: "application/custom",
			},
			wantFile: &File{
				Name:      "files/generated-1",  // Mock generated
				MIMEType:  "application/custom", // Overridden
				SizeBytes: Ptr(int64(len(fileContent))),
				State:     FileStateActive,
			},
			wantErr: false,
		},
		{
			name:       "Error - Invalid Path",
			path:       filepath.Join(tempDir, "nonexistent.file"),
			config:     nil,
			wantErr:    true,
			wantErrMsg: "is not a valid file path",
		},
		{
			name:       "Error - Directory Path",
			path:       tempDir, // Path is a directory
			config:     nil,
			wantErr:    true,
			wantErrMsg: "is not a valid file path",
		},
		{
			name: "Error - Unknown MIME Type",
			path: func() string { // Create a file with an unknown extension
				p := filepath.Join(tempDir, "file.unknownext")
				_ = os.WriteFile(p, []byte("data"), 0644)
				return p
			}(),
			config:     nil, // No MIME override
			wantErr:    true,
			wantErrMsg: "Unknown mime type",
		},
		{
			name: "Error - Upload Fails",
			path: filePath,
			config: &UploadFileConfig{
				Name: "files/fail.txt",
			},
			wantErr: true,
			setupServer: func(s *MockUploadServer) {
				s.createHandler = func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Create Failed", http.StatusInternalServerError)
				}
			},
			wantErrMsg: "Failed to create file",
		},
	}

	// Need to ensure mime type is registered for .txt
	_ = mime.AddExtensionType(".txt", "text/plain")

	for i, tt := range tests {
		// Reset server handlers
		mockServer.createHandler = mockServer.handleCreate
		mockServer.uploadHandler = mockServer.handleUpload
		if tt.setupServer != nil {
			tt.setupServer(mockServer)
		}
		mockServer.nextUploadID = i // Reset for predictable generated names

		t.Run(tt.name, func(t *testing.T) {
			gotFile, err := client.Files.UploadFromPath(ctx, tt.path, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UploadFromPath() expected error, but got nil")
				} else if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("UploadFromPath() error = %q, want error containing %q", err, tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("UploadFromPath() unexpected error: %v", err)
				}
				if gotFile == nil {
					t.Fatal("UploadFromPath() returned nil file on success")
				}
				// Compare relevant fields
				opts := cmp.Options{
					cmp.FilterPath(func(p cmp.Path) bool {
						field := p.Last().String()
						switch field {
						case ".Name", ".DisplayName", ".MIMEType", ".SizeBytes", ".State":
							return false // Keep
						default:
							return true // Ignore
						}
					}, cmp.Ignore()),
				}
				if diff := cmp.Diff(tt.wantFile, gotFile, opts); diff != "" {
					t.Errorf("UploadFromPath() file mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// errorReader is a helper for testing io errors during upload.
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("intentional read error")
}
