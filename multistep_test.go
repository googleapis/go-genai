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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var customTestMethods = map[string]func(ctx context.Context, client *Client, item *testTableItem) []reflect.Value{
	"shared/batches/create_delete":          createDelete,
	"shared/batches/create_get_cancel":      createGetCancelBatches,
	"shared/caches/create_get_delete":       createGetDelete,
	"shared/caches/create_update_get":       createUpdateGet,
	"shared/chats/send_message":             sendMessage,
	"shared/chats/send_message_stream":      sendMessageStream,
	"shared/files/upload_get_delete":        uploadGetDelete,
	"shared/models/generate_content_stream": generateContentStream,
	"shared/tunings/create_get_cancel":      createGetCancelTunings,
	"file_search_stores/multimodal_flow":    multimodalSearchFlow,
}

func wrapResults(resp any, err error) []reflect.Value {
	vResp := reflect.ValueOf(resp)
	if !vResp.IsValid() {
		vResp = reflect.Zero(reflect.TypeOf((*GenerateContentResponse)(nil)))
	}
	vErr := reflect.ValueOf(err)
	if !vErr.IsValid() {
		vErr = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())
	}
	return []reflect.Value{vResp, vErr}
}

func createDelete(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model  string                `json:"model"`
		Src    *BatchJobSource       `json:"src"`
		Config *CreateBatchJobConfig `json:"config"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	batchJob, err := client.Batches.Create(ctx, params.Model, params.Src, params.Config)
	if err != nil {
		return wrapResults(nil, err)
	}
	// if pending then don't delete to avoid error
	batchJob, err = client.Batches.Get(ctx, batchJob.Name, nil)
	if err != nil {
		return wrapResults(nil, err)
	}
	if batchJob.State != JobStatePending {
		return wrapResults(client.Batches.Delete(ctx, batchJob.Name, nil))
	}
	return wrapResults(batchJob, nil)
}

func createGetCancelBatches(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model  string                `json:"model"`
		Src    *BatchJobSource       `json:"src"`
		Config *CreateBatchJobConfig `json:"config"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	batchJob, err := client.Batches.Create(ctx, params.Model, params.Src, params.Config)
	if err != nil {
		return wrapResults(nil, err)
	}
	batchJob, err = client.Batches.Get(ctx, batchJob.Name, nil)
	if err != nil {
		return wrapResults(nil, err)
	}
	err = client.Batches.Cancel(ctx, batchJob.Name, nil)
	return wrapResults(nil, err)
}

func createGetCancelTunings(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		BaseModel       string                 `json:"baseModel"`
		TrainingDataset *TuningDataset         `json:"trainingDataset"`
		Config          *CreateTuningJobConfig `json:"config"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	tuningJob, err := client.Tunings.Tune(ctx, params.BaseModel, params.TrainingDataset, params.Config)
	if err != nil {
		return wrapResults(nil, err)
	}
	tuningJob, err = client.Tunings.Get(ctx, tuningJob.Name, nil)
	if err != nil {
		return wrapResults(nil, err)
	}
	_, err = client.Tunings.Cancel(ctx, tuningJob.Name, nil)
	return wrapResults(nil, err)
}

func createGetDelete(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model  string                     `json:"model"`
		Config *CreateCachedContentConfig `json:"config"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	var cache *CachedContent
	var err error
	if client.clientConfig.Backend == BackendVertexAI {
		cache, err = client.Caches.Create(ctx, params.Model, params.Config)
	} else {
		filePath := "tests/data/google.png"
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(filePath), 0755)            // nolint:errcheck
			os.WriteFile(filePath, []byte("fake content"), 0644) // nolint:errcheck
		}
		file, err := client.Files.UploadFromPath(ctx, filePath, nil)
		if err != nil {
			return wrapResults(nil, err)
		}
		parts := []*Part{}
		for i := 0; i < 5; i++ {
			parts = append(parts, NewPartFromFile(*file))
		}
		config := &CreateCachedContentConfig{
			Contents: []*Content{NewContentFromParts(parts, RoleUser)},
		}
		cache, err = client.Caches.Create(ctx, params.Model, config) // nolint:ineffassign,staticcheck
	}
	if err != nil {
		return wrapResults(cache, err)
	}
	gotCache, err := client.Caches.Get(ctx, cache.Name, nil)
	if err != nil {
		return wrapResults(gotCache, err)
	}
	return wrapResults(client.Caches.Delete(ctx, gotCache.Name, nil))
}

func createUpdateGet(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model  string                     `json:"model"`
		Config *CreateCachedContentConfig `json:"config"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	var cache *CachedContent
	var err error
	if client.clientConfig.Backend == BackendVertexAI {
		cache, err = client.Caches.Create(ctx, params.Model, params.Config)
	} else {
		filePath := "tests/data/google.png"
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(filePath), 0755)            // nolint:errcheck
			os.WriteFile(filePath, []byte("fake content"), 0644) // nolint:errcheck
		}
		file, err := client.Files.UploadFromPath(ctx, filePath, nil)
		if err != nil {
			return wrapResults(nil, err)
		}
		parts := []*Part{}
		for i := 0; i < 5; i++ {
			parts = append(parts, NewPartFromFile(*file))
		}
		config := &CreateCachedContentConfig{
			Contents: []*Content{NewContentFromParts(parts, RoleUser)},
		}
		cache, err = client.Caches.Create(ctx, params.Model, config) // nolint:ineffassign,staticcheck
	}
	if err != nil {
		return wrapResults(cache, err)
	}
	updatedCache, err := client.Caches.Update(ctx, cache.Name, &UpdateCachedContentConfig{TTL: 7200 * time.Second})
	if err != nil {
		return wrapResults(updatedCache, err)
	}
	return wrapResults(client.Caches.Get(ctx, updatedCache.Name, nil))
}

func sendMessage(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model   string `json:"model"`
		Message string `json:"message"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	chat, err := client.Chats.Create(ctx, params.Model, nil, nil)
	if err != nil {
		return wrapResults(nil, err)
	}
	return wrapResults(chat.SendMessage(ctx, Part{Text: params.Message}))
}

func sendMessageStream(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model   string `json:"model"`
		Message string `json:"message"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	chat, err := client.Chats.Create(ctx, params.Model, nil, nil)
	if err != nil {
		return wrapResults(nil, err)
	}
	iter := chat.SendMessageStream(ctx, Part{Text: params.Message})
	var lastResponse *GenerateContentResponse
	for resp, err := range iter {
		if err != nil {
			return wrapResults(nil, err)
		}
		lastResponse = resp
	}
	return wrapResults(lastResponse, nil)
}

func uploadGetDelete(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		FilePath string `json:"filePath"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	if _, err := os.Stat(params.FilePath); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(params.FilePath), 0755)            // nolint:errcheck
		os.WriteFile(params.FilePath, []byte("fake content"), 0644) // nolint:errcheck
	}

	file, err := client.Files.UploadFromPath(ctx, params.FilePath, nil)
	if err != nil {
		return wrapResults(nil, err)
	}
	gotFile, err := client.Files.Get(ctx, file.Name, nil)
	if err != nil {
		return wrapResults(gotFile, err)
	}
	return wrapResults(client.Files.Delete(ctx, gotFile.Name, nil))
}

func generateContentStream(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		Model    string `json:"model"`
		Contents any    `json:"contents"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	var contents []*Content
	switch v := params.Contents.(type) {
	case string:
		contents = Text(v)
	case []any:
		contentsJSON, _ := json.Marshal(v)
		if err := json.Unmarshal(contentsJSON, &contents); err != nil {
			return wrapResults(nil, err)
		}
	}

	iter := client.Models.GenerateContentStream(ctx, params.Model, contents, nil)
	var lastResp *GenerateContentResponse
	for resp, err := range iter {
		if err != nil {
			return wrapResults(nil, err)
		}
		lastResp = resp
	}
	return wrapResults(lastResp, nil)
}

func multimodalSearchFlow(ctx context.Context, client *Client, item *testTableItem) []reflect.Value {
	params := struct {
		DisplayName       string `json:"displayName"`
		Query             string `json:"query"`
		TextContent       string `json:"textContent"`
		ImageRelativePath string `json:"imageRelativePath"`
	}{}
	paramsJSON, _ := json.Marshal(item.Parameters)
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return wrapResults(nil, err)
	}

	store, err := client.FileSearchStores.Create(ctx, &CreateFileSearchStoreConfig{
		DisplayName:    params.DisplayName,
	})
	if err != nil {
		return wrapResults(nil, err)
	}

	defer func() {
		trueVar := true
		client.FileSearchStores.Delete(ctx, store.Name, &DeleteFileSearchStoreConfig{Force: &trueVar}) // nolint:errcheck
	}()

	// Upload Text
	textFilePath := "tests/data/test_file.txt"
	if err := os.MkdirAll(filepath.Dir(textFilePath), 0755); err != nil {
		return wrapResults(nil, err)
	}
	if err := os.WriteFile(textFilePath, []byte(params.TextContent), 0644); err != nil {
		return wrapResults(nil, err)
	}
	defer os.Remove(textFilePath)

	opText, err := client.FileSearchStores.UploadToFileSearchStoreFromPath(ctx, textFilePath, store.Name, &UploadToFileSearchStoreConfig{
		MIMEType: "text/plain",
	})
	if err != nil {
		return wrapResults(nil, err)
	}

	// Upload Image
	// Resolve path relative to google3
	currentDir, _ := os.Getwd()
	google3Path := ""
	lastIndex := strings.LastIndex(currentDir, "google3/")
	if lastIndex != -1 {
		google3Path = currentDir[:lastIndex+len("google3/")]
	}
	resolvedImagePath := filepath.Join(google3Path, "third_party/py/google/genai/tests/data/dog.jpg")

	opImage, err := client.FileSearchStores.UploadToFileSearchStoreFromPath(ctx, resolvedImagePath, store.Name, &UploadToFileSearchStoreConfig{
		MIMEType: "image/png",
	})
	if err != nil {
		return wrapResults(nil, err)
	}

	// Wait for operations
	for !opText.Done {
		time.Sleep(1 * time.Second)
		opText, err = client.Operations.GetUploadToFileSearchStoreOperation(ctx, opText, nil)
		if err != nil {
			return wrapResults(nil, err)
		}
	}

	for !opImage.Done {
		time.Sleep(1 * time.Second)
		opImage, err = client.Operations.GetUploadToFileSearchStoreOperation(ctx, opImage, nil)
		if err != nil {
			return wrapResults(nil, err)
		}
	}

	// Search
	response, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", Text(params.Query), &GenerateContentConfig{
		Tools: []*Tool{
			{
				FileSearch: &FileSearch{
					FileSearchStoreNames: []string{store.Name},
				},
			},
		},
	})
	if err != nil {
		return wrapResults(nil, err)
	}

	// Verify response has grounding metadata
	if len(response.Candidates) == 0 || response.Candidates[0].GroundingMetadata == nil {
		return wrapResults(nil, fmt.Errorf("no grounding metadata in response"))
	}

	// Download Media
	var blobMediaId string
	for _, chunk := range response.Candidates[0].GroundingMetadata.GroundingChunks {
		if chunk.RetrievedContext != nil && chunk.RetrievedContext.MediaID != "" {
			blobMediaId = chunk.RetrievedContext.MediaID
			break
		}
	}

	if client.clientConfig.Backend != BackendVertexAI {
		if blobMediaId == "" {
			return wrapResults(nil, fmt.Errorf("no mediaId found in grounding metadata to test download"))
		}
		content, err := client.FileSearchStores.DownloadMedia(ctx, blobMediaId, nil)
		if err != nil {
			return wrapResults(nil, err)
		}
		if content == nil {
			return wrapResults(nil, fmt.Errorf("downloaded content is null"))
		}
	}

	return wrapResults(response, nil)
}
