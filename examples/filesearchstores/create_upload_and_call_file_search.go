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

//go:build ignore_vet

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.5-flash", "the model name, e.g. gemini-2.5-flash")

// Returns the location of the root directory of this repository.
func moduleRootDir() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Getcwd:", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			log.Fatal("unable to find root directory")
		}
		dir = parentDir
	}
}

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	// Create a new file search store.
	fileSearchStore, err := client.FileSearchStores.Create(ctx, &genai.CreateFileSearchStoreConfig{
		DisplayName: "My File Search Store",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("File search store created successfully: ", fileSearchStore.Name)

	// Upload a new file to the files service.
	var testDataDir = filepath.Join(moduleRootDir(), "testdata")
	filePath := filepath.Join(testDataDir, "story.txt")
	uploadedFile, err := client.Files.UploadFromPath(ctx, filePath, &genai.UploadFileConfig{MIMEType: "text/plain"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("File uploaded successfully: ", uploadedFile.Name)

	// Import the uploaded file to the file search store.
	importOperation, err := client.FileSearchStores.ImportFile(ctx, fileSearchStore.Name, uploadedFile.Name, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Import file operation: ", importOperation.Name)

	// Wait for the import file operation to complete.
	for !importOperation.Done {
		time.Sleep(5 * time.Second)
		fmt.Println("Waiting for import file operation to complete...")
		importOperation, err = client.Operations.GetImportFileOperation(ctx, importOperation, nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Upload a new file.
	uploadOperation, err := client.FileSearchStores.UploadToFileSearchStoreFromPath(ctx, filePath, fileSearchStore.Name, &genai.UploadToFileSearchStoreConfig{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Upload to file search store operation: ", uploadOperation.Name)

	// Wait for the operation to complete.
	for !uploadOperation.Done {
		time.Sleep(5 * time.Second)
		fmt.Println("Waiting for upload to file search store operation to complete...")
		uploadOperation, err = client.Operations.GetUploadToFileSearchStoreOperation(ctx, uploadOperation, nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	//Call generate content with file search tool
	var tools = []*genai.Tool{
		&genai.Tool{
			FileSearch: &genai.FileSearch{FileSearchStoreNames: []string{fileSearchStore.Name}},
		},
	}
	var config *genai.GenerateContentConfig = &genai.GenerateContentConfig{Tools: tools}
	result, err := client.Models.GenerateContent(ctx, *model, genai.Text("According to the story, how long has it been since Silas last saw a storm of this magnitude?"), config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Text response: ", result.Candidates[0].Content.Parts[0].Text)

	jsonData, err := json.MarshalIndent(result.Candidates[0].GroundingMetadata, "", "  ")
	fmt.Println("Grounding metadata: ", string(jsonData))

}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
