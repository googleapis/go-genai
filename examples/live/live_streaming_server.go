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
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	_ "embed"

	"github.com/gorilla/websocket"
	"google.golang.org/genai"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

//go:embed live_streaming.html
var homeTemplate string

func live(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("upgrade error: ", err)
		return
	}
	defer c.Close()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{Backend: genai.BackendGeminiAPI, HTTPOptions: genai.HTTPOptions{APIVersion: "v1beta"}})
	model := "gemini-2.0-flash-live-001"
	// Vertex AI client.
	// client, err := genai.NewClient(ctx, &genai.ClientConfig{Backend: genai.BackendVertexAI, HTTPOptions: genai.HTTPOptions{APIVersion: "v1beta"}})
	// model = "gemini-2.0-flash-live-preview-04-09"
	if err != nil {
		log.Fatal("create client error: ", err)
		return
	}

	session, err := client.Live.Connect(ctx, model, &genai.LiveConnectConfig{})
	if err != nil {
		log.Fatal("connect to model error: ", err)
	}

	// Get model's response
	go func() {
		for {
			message, err := session.Receive()
			if err != nil {
				log.Fatal("receive model response error: ", err)
			}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Fatal("marhal model response error: ", message, err)
			}
			err = c.WriteMessage(1, messageBytes)
			if err != nil {
				log.Println("write message error: ", err)
				break
			}
		}
	}()

	// Read from client and then forward to Google.
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read from client error: ", err)
			break
		}
		if len(message) > 0 {
			log.Printf(" bytes size received from client: %d", len(message))
		}

		var realtimeInput genai.LiveRealtimeInput
		if err := json.Unmarshal(message, &realtimeInput); err != nil {
			log.Fatal("unmarshal message error ", string(message), err)
		}
		session.SendRealtimeInput(realtimeInput)
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("home").Parse(homeTemplate)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	fmt.Println("ws://" + r.Host + "/live")
	err = tmpl.Execute(w, "ws://"+r.Host+"/live")
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func proxyVideo(w http.ResponseWriter, r *http.Request) {
	// Fetch the video from Google Cloud Storage.
	resp, err := http.Get("http://storage.googleapis.com/cloud-samples-data/video/animals.mp4")
	if err != nil {
		http.Error(w, "Error fetching video", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/", homePage)
	http.HandleFunc("/live", live)
	http.HandleFunc("/proxyVideo", proxyVideo)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
