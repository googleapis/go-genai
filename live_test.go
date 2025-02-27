// Copyright 2024 Google LLC
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
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
)

type mockTokenSource struct {
	MockToken *oauth2.Token
}

func (mts mockTokenSource) Token() (*oauth2.Token, error) {
	return mts.MockToken, nil
}

func TestLiveConnect(t *testing.T) {
	ctx := context.Background()
	const model = "test-model"

	mldevClient, err := NewClient(ctx, &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "test-api-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Needed for account default credential.
	// Usually this file is in ~/.config/gcloud/application_default_credentials.json
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "testdata/credentials.json")
	vertexClient, err := NewClient(ctx, &ClientConfig{
		Backend:  BackendVertexAI,
		Project:  "test-project",
		Location: "test-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	mockToken := &oauth2.Token{
		AccessToken: "fake_access_token",
	}
	mts := mockTokenSource{MockToken: mockToken}

	connectTests := []struct {
		desc            string
		client          *Client
		config          *LiveConnectConfig
		wantRequestBody string
		wantErr         bool
	}{
		{
			desc:            "successful connection mldev",
			client:          mldevClient,
			wantRequestBody: `{"setup":{"model":"models/test-model"}}`,
		},
		{
			desc:   "successful connection with config mldev",
			client: mldevClient,
			config: &LiveConnectConfig{
				GenerationConfig:  &GenerationConfig{Temperature: Ptr(0.5)},
				SystemInstruction: &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:             []*Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			wantRequestBody: `{"setup":{"generationConfig":{"temperature":0.5},"model":"models/test-model","systemInstruction":{"parts":[{"text":"test instruction"}]},"tools":[{"googleSearch":{}}]}}`,
		},
		{
			desc:            "successful connection vertex",
			client:          vertexClient,
			wantRequestBody: `{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`,
		},
		{
			desc:   "successful connection with config vertex",
			client: vertexClient,
			config: &LiveConnectConfig{
				GenerationConfig:  &GenerationConfig{Temperature: Ptr(0.5)},
				SystemInstruction: &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:             []*Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			wantRequestBody: `{"setup":{"generationConfig":{"temperature":0.5},"model":"projects/test-project/locations/test-location/publishers/google/models/test-model","systemInstruction":{"parts":[{"text":"test instruction"}]},"tools":[{"googleSearch":{}}]}}`,
		},
	}

	for _, tt := range connectTests {
		t.Run(tt.desc, func(t *testing.T) {
			var upgrader = websocket.Upgrader{}
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, _ := upgrader.Upgrade(w, r, nil)
				defer conn.Close()

				mt, message, err := conn.ReadMessage()
				if err != nil {
					if tt.wantErr {
						return
					}
					t.Fatalf("ReadMessage: %v", err)
				}
				if diff := cmp.Diff(string(message), tt.wantRequestBody); diff != "" {
					t.Errorf("Request message mismatch (-want +got):\n%s", diff)
				}

				response := &LiveServerMessage{}
				if err := json.Unmarshal([]byte(`{"setupComplete":{}}`), response); err != nil {
					t.Fatalf("Unmarshal: %v", err)
				}
				responseBytes, err := json.Marshal(response)
				if err != nil {
					t.Fatalf("Marshal: %v", err)
				}

				conn.WriteMessage(mt, responseBytes)
			}))
			defer ts.Close()

			tt.client.Live.apiClient.clientConfig.HTTPOptions.BaseURL = strings.Replace(ts.URL, "http", "ws", 1)
			tt.client.Live.apiClient.clientConfig.HTTPClient = ts.Client()
			if tt.client.Live.apiClient.clientConfig.Backend == BackendVertexAI {
				tt.client.Live.apiClient.clientConfig.Credentials.TokenSource = mts
			}
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}
			session, err := tt.client.Live.Connect(model, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer session.Close()
		})
	}

	t.Run("Send and Receive", func(t *testing.T) {
		sendReceiveTests := []struct {
			desc                  string
			client                *Client
			wantRequestBodySlice  []string
			fakeResponseBodySlice []string
			wantErr               bool
		}{
			{
				desc:                  "send clientContent to Google AI",
				client:                mldevClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"clientContent":{"turns":[{"parts":[{"text":"client test message"}],"role":"user"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "send clientContent to Vertex AI",
				client:                vertexClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`, `{"clientContent":{"turns":[{"parts":[{"text":"client test message"}],"role":"user"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "received error in response",
				client:                mldevClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"clientContent":{"turns":[{"parts":[{"text":"client test message"}],"role":"user"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"error":{"code":400,"message":"test error message","status":"INVALID_ARGUMENT"}}`},
				wantErr:               true,
			},
		}

		for _, tt := range sendReceiveTests {
			t.Run(tt.desc, func(t *testing.T) {
				ts := setupTestWebsocketServer(t, tt.wantRequestBodySlice, tt.fakeResponseBodySlice)
				defer ts.Close()

				tt.client.Live.apiClient.clientConfig.HTTPOptions.BaseURL = strings.Replace(ts.URL, "http", "ws", 1)
				tt.client.Live.apiClient.clientConfig.HTTPClient = ts.Client()
				if tt.client.Live.apiClient.clientConfig.Backend == BackendVertexAI {
					tt.client.Live.apiClient.clientConfig.Credentials.TokenSource = mts
				}

				session, err := tt.client.Live.Connect("test-model", &LiveConnectConfig{})
				if err != nil {
					t.Fatalf("Connect failed: %v", err)
				}
				defer session.Close()

				// Construct a test message
				clientMessage := &LiveClientMessage{
					ClientContent: &LiveClientContent{Turns: Text("client test message")},
				}

				// Test sending the message
				err = session.Send(clientMessage)
				if err != nil {
					t.Errorf("Send failed : %v", err)
				}

				// Construct the expected response
				serverMessage := &LiveServerMessage{ServerContent: &LiveServerContent{ModelTurn: Text("server test message")[0]}}
				// Test receiving the response
				gotMessage, err := session.Receive()
				if err != nil {
					if tt.wantErr {
						return
					}
					t.Errorf("Receive failed: %v", err)
				}
				if diff := cmp.Diff(gotMessage, serverMessage); diff != "" {
					t.Errorf("Response message mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})
}

// Helper function to set up a test websocket server.
func setupTestWebsocketServer(t *testing.T, wantRequestBodySlice []string, fakeResponseBodySlice []string) *httptest.Server {
	t.Helper()

	var upgrader = websocket.Upgrader{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()

		index := 0

		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				t.Logf("read error: %v", err)
				break
			}
			if diff := cmp.Diff(string(message), wantRequestBodySlice[index]); diff != "" {
				t.Errorf("Request message mismatch (-want +got):\n%s", diff)
			}
			err = conn.WriteMessage(mt, []byte(fakeResponseBodySlice[index]))
			index++
			if err != nil {
				t.Logf("write error: %v", err)
				break
			}
		}
	}))

	return ts
}
