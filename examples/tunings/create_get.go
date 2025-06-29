// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"time"

	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.5-flash", "the model name, e.g. gemini-2.5-flash")

var inlineTrainingData = [][]string{
	{"1", "2"},
	{"3", "4"},
	{"-3", "-2"},
	{"twenty two", "twenty three"},
	{"two hundred", "two hundred one"},
	{"ninety nine", "one hundred"},
	{"8", "9"},
	{"-98", "-97"},
	{"1,000", "1,001"},
	{"10,100,000", "10,100,001"},
	{"thirteen", "fourteen"},
	{"eighty", "eighty one"},
	{"one", "two"},
	{"three", "four"},
	{"seven", "eight"},
}

func print(r any) {
	// Marshal the result to JSON.
	response, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	// Log the output.
	fmt.Println(string(response))
}

func delay(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func run(ctx context.Context) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{Backend: genai.BackendVertexAI})
	if err != nil {
		log.Fatal(err)
	}

	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("Calling VertexAI Backend...")
	} else {
		fmt.Println("Calling GeminiAPI Backend...")
	}

	fmt.Println("Tuning example. Tuning job creation is currently only supported in BackendVertexAI.")

	if client.ClientConfig().Backend == genai.BackendVertexAI {
		tuningJob, err := client.Tunings.Tune(ctx, *model, &genai.TuningDataset{
			GCSURI: "gs://cloud-samples-data/ai-platform/generative_ai/gemini-1_5/text/sft_train_data.jsonl",
		}, nil)
		if err != nil {
			log.Fatal(err)
		}

		// Create the tuning job, and let it complete.
		fmt.Println("Creating tuning job: ")
		print(tuningJob)
		tuningJobName := tuningJob.Name

		tunedModel := ""
		for tunedModel == "" {
			fmt.Println("Waiting for tuned model to be available")
			delay(10000)
			// Get the tuning job.
			fetchedTuningJob, err := client.Tunings.Get(ctx, tuningJobName, nil)
			if err != nil {
				log.Fatal(err)
			}
			if fetchedTuningJob.TunedModel != nil {
				tunedModel = fetchedTuningJob.TunedModel.Model
			}
		}
		fmt.Println("Tuned model: ", tunedModel)

		getModelResponse, err := client.Models.Get(ctx, tunedModel, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Fetch tuned model: ")
		print(getModelResponse)
	}
}

func main() {
	ctx := context.Background()
	flag.Parse()
	run(ctx)
}
