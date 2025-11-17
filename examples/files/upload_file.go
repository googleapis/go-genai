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
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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

func calculateSHA256(filePath string) (string, error) {
	r, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Failed to open file attempting to calculate checksum")
	}
	defer r.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("failed to copy data to hasher: %w", err)
	}
	hashBytes := hasher.Sum(nil)

	return hex.EncodeToString(hashBytes), nil
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
	// Upload a new file.
	var testDataDir = filepath.Join(moduleRootDir(), "testdata")
	filePath := filepath.Join(testDataDir, "google.jpg")
	file, err := client.Files.UploadFromPath(ctx, filePath, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("File uploaded successfully: ", file.Name)
	checksum, err := calculateSHA256(filePath)
	if err != nil {
		log.Fatal("Failed to calculate checksum: ", err)
	}
	uploadedFileHash, err := b64.StdEncoding.DecodeString(file.Sha256Hash)
	if err != nil {
		log.Fatal("Failed to decode uploaded file hash: ", err)
	}
	fmt.Println("Calculated file checksum: ", checksum)
	fmt.Println("Uploaded file checksum: ", string(uploadedFileHash))
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
