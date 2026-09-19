// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interactions

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/genai/interactions/models/components"
	"google.golang.org/genai/interactions/models/operations"
)

func TestFilesUploadAndDownload(t *testing.T) {
	var uploadURL string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/files/") {
			w.Header().Set("X-Goog-Upload-URL", uploadURL)
			w.Header().Set("X-Goog-Upload-Status", "active")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/scotty/upload/session" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"files": [{"name": "environments/env-1/files/test.txt", "path": "test.txt", "size_bytes": "12"}]}`))
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/test.txt") {
			if r.URL.Query().Get("alt") != "media" {
				t.Errorf("Query alt = %s; want media", r.URL.Query().Get("alt"))
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("downloaded content"))
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files") {
			resp := map[string]any{
				"files": []map[string]any{
					{
						"name":       "environments/env-1/files/test.txt",
						"path":       "test.txt",
						"type":       "file",
						"size_bytes": "12",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.String(), http.StatusBadRequest)
	}))
	defer ts.Close()

	uploadURL = ts.URL + "/scotty/upload/session"

	client := New(
		WithServerURL(ts.URL),
		WithSecurity(components.Security{
			APIKey: String("dummy_key"),
		}),
	)

	ctx := context.Background()

	// Test Upload
	uploadResp, err := client.Environments.Files.Upload(ctx, operations.UploadEnvironmentFileRequest{
		Environment: "env-1",
		Path:        "test.txt",
		Content:     []byte("hello world!"),
	})
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if uploadResp.Files == nil || len(uploadResp.Files.Files) == 0 {
		t.Fatalf("Expected files in upload response, got none")
	}

	// Test Download bytes
	data, err := client.Environments.Files.Download(ctx, "env-1", "test.txt")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if string(data) != "downloaded content" {
		t.Errorf("Download content = %s; want 'downloaded content'", string(data))
	}

	// Test Download stream
	rc, err := client.Environments.Files.DownloadStream(ctx, "env-1", "test.txt")
	if err != nil {
		t.Fatalf("DownloadStream failed: %v", err)
	}
	defer rc.Close()
	streamData, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(streamData) != "downloaded content" {
		t.Errorf("DownloadStream content = %s; want 'downloaded content'", string(streamData))
	}

	// Test List
	listResp, err := client.Environments.Files.List(ctx, operations.GetEnvironmentFilesRequest{
		Environment: "env-1",
		Path:        "",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if listResp.GetEnvironmentFilesResponse == nil || len(listResp.GetEnvironmentFilesResponse.Files) == 0 {
		t.Fatalf("Expected files in list response, got none")
	}
}

func TestFilesUploadDifferentInputs(t *testing.T) {
	type capturedReq struct {
		method  string
		path    string
		headers http.Header
		body    []byte
	}
	var captured []capturedReq
	var uploadURL string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = append(captured, capturedReq{
			method:  r.Method,
			path:    r.URL.RequestURI(),
			headers: r.Header.Clone(),
			body:    body,
		})

		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/files/") {
			w.Header().Set("X-Goog-Upload-URL", uploadURL)
			w.Header().Set("X-Goog-Upload-Status", "active")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/scotty/upload/session" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"files": [{"name": "environments/env-1/files/output.txt", "path": "output.txt", "size_bytes": "20"}]}`))
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/download.txt") {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("saved file content"))
			return
		}
		http.Error(w, "unexpected", http.StatusBadRequest)
	}))
	defer ts.Close()

	uploadURL = ts.URL + "/scotty/upload/session"
	client := New(
		WithServerURL(ts.URL),
		WithSecurity(components.Security{
			APIKey: String("dummy_key"),
		}),
	)
	ctx := context.Background()

	// 1. Test UploadFile from temp file path
	tmpDir := t.TempDir()
	filePath := tmpDir + "/test_upload.py"
	if err := os.WriteFile(filePath, []byte("print('from file')"), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	captured = nil
	fileResp, err := client.Environments.Files.UploadFile(ctx, "env-1", "test_upload.py", filePath)
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if fileResp.Files == nil || len(fileResp.Files.Files) == 0 {
		t.Fatalf("Expected file in UploadFile response")
	}
	if len(captured) < 2 {
		t.Fatalf("Expected at least 2 captured requests (handshake + chunk), got %d", len(captured))
	}
	handshake := captured[0]
	if handshake.method != http.MethodPut || !strings.Contains(handshake.path, "/files/test_upload.py") {
		t.Errorf("Handshake method = %s, path = %s", handshake.method, handshake.path)
	}
	if handshake.headers.Get("X-Goog-Upload-Protocol") != "resumable" {
		t.Errorf("Handshake protocol = %s; want resumable", handshake.headers.Get("X-Goog-Upload-Protocol"))
	}
	if handshake.headers.Get("X-Goog-Upload-Header-Content-Type") != "text/x-python" {
		t.Errorf("Handshake content type = %s; want text/x-python", handshake.headers.Get("X-Goog-Upload-Header-Content-Type"))
	}
	chunk := captured[1]
	if chunk.method != http.MethodPost || chunk.headers.Get("X-Goog-Upload-Command") != "upload, finalize" {
		t.Errorf("Chunk command = %s; want 'upload, finalize'", chunk.headers.Get("X-Goog-Upload-Command"))
	}
	if string(chunk.body) != "print('from file')" {
		t.Errorf("Chunk body = %s; want print('from file')", string(chunk.body))
	}

	// 2. Test UploadStream from io.Reader
	captured = nil
	streamContent := "streamed content data"
	streamResp, err := client.Environments.Files.UploadStream(ctx, "env-1", "stream.txt", strings.NewReader(streamContent), int64(len(streamContent)))
	if err != nil {
		t.Fatalf("UploadStream failed: %v", err)
	}
	if streamResp.Files == nil || len(streamResp.Files.Files) == 0 {
		t.Fatalf("Expected file in UploadStream response")
	}
	if len(captured) < 2 {
		t.Fatalf("Expected 2 captured requests for UploadStream, got %d", len(captured))
	}
	if string(captured[1].body) != streamContent {
		t.Errorf("UploadStream body = %s; want %s", string(captured[1].body), streamContent)
	}

	// 3. Test 0-byte file upload
	captured = nil
	zeroResp, err := client.Environments.Files.UploadBytes(ctx, "env-1", "empty.txt", []byte{})
	if err != nil {
		t.Fatalf("0-byte upload failed: %v", err)
	}
	if zeroResp.Files == nil || len(zeroResp.Files.Files) == 0 {
		t.Fatalf("Expected file in 0-byte upload response")
	}
	if len(captured) < 2 {
		t.Fatalf("Expected 2 captured requests for 0-byte upload, got %d", len(captured))
	}
	if captured[1].headers.Get("X-Goog-Upload-Command") != "upload, finalize" {
		t.Errorf("0-byte chunk command = %s; want 'upload, finalize'", captured[1].headers.Get("X-Goog-Upload-Command"))
	}
	if captured[1].headers.Get("X-Goog-Upload-Offset") != "0" {
		t.Errorf("0-byte chunk offset = %s; want 0", captured[1].headers.Get("X-Goog-Upload-Offset"))
	}

	// 4. Test DownloadToFile
	destPath := tmpDir + "/downloaded.txt"
	if err := client.Environments.Files.DownloadToFile(ctx, "env-1", "download.txt", destPath); err != nil {
		t.Fatalf("DownloadToFile failed: %v", err)
	}
	savedData, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(savedData) != "saved file content" {
		t.Errorf("DownloadToFile content = %s; want 'saved file content'", string(savedData))
	}
}

func TestFilesUploadMultiChunk(t *testing.T) {
	const chunkSize = 8 * 1024 * 1024 // 8MB
	totalSize := int64(chunkSize + 1024*1024) // 9MB total (2 chunks)
	payload := make([]byte, totalSize)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	var uploadURL string
	var chunkCommands []string
	var chunkOffsets []string
	var chunkLengths []int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/files/large.bin") {
			w.Header().Set("X-Goog-Upload-URL", uploadURL)
			w.Header().Set("X-Goog-Upload-Status", "active")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/scotty/upload/multichunk" {
			cmd := r.Header.Get("X-Goog-Upload-Command")
			offset := r.Header.Get("X-Goog-Upload-Offset")
			body, _ := io.ReadAll(r.Body)

			chunkCommands = append(chunkCommands, cmd)
			chunkOffsets = append(chunkOffsets, offset)
			chunkLengths = append(chunkLengths, len(body))

			if cmd == "upload, finalize" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"files": [{"name": "environments/env-1/files/large.bin", "path": "large.bin", "size_bytes": "9437184"}]}`))
				return
			}
			w.Header().Set("X-Goog-Upload-Status", "active")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.String(), http.StatusBadRequest)
	}))
	defer ts.Close()

	uploadURL = ts.URL + "/scotty/upload/multichunk"
	client := New(
		WithServerURL(ts.URL),
		WithSecurity(components.Security{
			APIKey: String("dummy_key"),
		}),
	)

	resp, err := client.Environments.Files.UploadBytes(context.Background(), "env-1", "large.bin", payload)
	if err != nil {
		t.Fatalf("UploadBytes multi-chunk failed: %v", err)
	}
	if resp.Files == nil || len(resp.Files.Files) == 0 {
		t.Fatalf("Expected file in multi-chunk response, got none")
	}

	if len(chunkCommands) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(chunkCommands))
	}
	// Chunk 1
	if chunkCommands[0] != "upload" {
		t.Errorf("Chunk 1 command = %s; want 'upload'", chunkCommands[0])
	}
	if chunkOffsets[0] != "0" {
		t.Errorf("Chunk 1 offset = %s; want '0'", chunkOffsets[0])
	}
	if chunkLengths[0] != chunkSize {
		t.Errorf("Chunk 1 length = %d; want %d", chunkLengths[0], chunkSize)
	}
	// Chunk 2 (Final)
	if chunkCommands[1] != "upload, finalize" {
		t.Errorf("Chunk 2 command = %s; want 'upload, finalize'", chunkCommands[1])
	}
	if chunkOffsets[1] != "8388608" {
		t.Errorf("Chunk 2 offset = %s; want '8388608'", chunkOffsets[1])
	}
	if chunkLengths[1] != 1024*1024 {
		t.Errorf("Chunk 2 length = %d; want %d", chunkLengths[1], 1024*1024)
	}
}
