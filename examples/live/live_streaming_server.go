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

// live handles incoming WebSocket requests for the live streaming example.
// It upgrades the HTTP connection to a WebSocket connection, establishes a
// connection with the configured GenAI model (Gemini API or Vertex AI),
// and then proxies messages between the client WebSocket and the GenAI service.
// It runs two goroutines: one to receive messages from the GenAI service and
// forward them to the client, and another to read messages from the client
// and send them to the GenAI service.
func live(w http.ResponseWriter, r *http.Request) {
	// Attempt to upgrade the HTTP connection to a WebSocket connection.
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Log fatal error if the WebSocket upgrade fails (e.g., invalid request headers).
		log.Fatal("upgrade error: ", err)
		return
	}
	defer c.Close()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		// Log fatal error if client creation fails (e.g., invalid config, authentication issues).
		log.Fatal("create client error: ", err)
		return
	}

	var model string
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		model = "gemini-2.0-flash-live-preview-04-09"
	} else {
		model = "gemini-live-2.5-flash-preview"
	}

	// Establish the live WebSocket connection with the specified GenAI model.
	session, err := client.Live.Connect(ctx, model, &genai.LiveConnectConfig{})
	if err != nil {
		// Log fatal error if connecting to the model fails (e.g., network issues, invalid model name).
		log.Fatal("connect to model error: ", err)
	}
	defer session.Close() // Ensure session is closed when the handler exits

	// Goroutine to receive messages from the GenAI service and send to the client
	go func() {
		for {
			// Receive the next message from the GenAI service session.
			message, err := session.Receive()
			if err != nil {
				// Log fatal error if receiving from the GenAI service fails (e.g., connection closed, network error).
				log.Fatal("receive model response error: ", err)
			}
			// Marshal the received message into JSON format.
			messageBytes, err := json.Marshal(message)
			if err != nil {
				// Log fatal error if marshaling the message to JSON fails.
				log.Fatal("marhal model response error: ", message, err)
			}
			// Send the JSON message to the client WebSocket.
			err = c.WriteMessage(websocket.TextMessage, messageBytes) // Use TextMessage type for JSON
			if err != nil {
				// Log error and break the loop if writing to the client WebSocket fails (e.g., client disconnected).
				log.Println("write message error: ", err)
				break
			}
		}
	}()

	// Main loop to read messages from the client and send to the GenAI service
	for {
		// Read the next message from the client WebSocket.
		_, message, err := c.ReadMessage()
		if err != nil {
			// Log error and break the loop if reading from the client WebSocket fails (e.g., client disconnected).
			log.Println("read from client error: ", err)
			break // Exit loop on error
		}
		if len(message) > 0 {
			log.Printf(" bytes size received from client: %d", len(message))
		}

		var realtimeInput genai.LiveRealtimeInput
		// Unmarshal the received client message into a LiveRealtimeInput struct.
		if err := json.Unmarshal(message, &realtimeInput); err != nil {
			// Log fatal error if unmarshaling the client message fails (e.g., invalid JSON format).
			log.Fatal("unmarshal message error ", string(message), err)
		}
		// Send the unmarshaled realtime input to the GenAI service session.
		// Note: This currently doesn't handle potential errors from SendRealtimeInput.
		// Consider adding error handling here if needed.
		session.SendRealtimeInput(realtimeInput)
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	// Parse the embedded HTML template.
	tmpl, err := template.New("home").Parse(homeTemplate)
	if err != nil {
		// Return an internal server error if the template parsing fails.
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	fmt.Println("ws://" + r.Host + "/live")
	// Execute the template, passing the WebSocket URL to it.
	err = tmpl.Execute(w, "ws://"+r.Host+"/live")
	if err != nil {
		// Return an internal server error if executing the template fails.
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func proxyVideo(w http.ResponseWriter, r *http.Request) {
	// Fetch the video from Google Cloud Storage.
	resp, err := http.Get("http://storage.googleapis.com/cloud-samples-data/video/animals.mp4")
	if err != nil {
		// Return an internal server error if fetching the video fails.
		http.Error(w, "Error fetching video", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Set CORS header to allow requests from the frontend origin.
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080") // Adjust if your frontend runs elsewhere
	// Set the Content-Type header based on the original video response.
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	// Copy the video content from the GCS response to the client response.
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		// Log error if copying the video content fails.
		log.Printf("Error copying video content: %v", err)
		// It might be too late to send an HTTP error header here if data was already sent.
	}
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
	// Log fatal error if the HTTP server fails to start (e.g., port already in use).
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
