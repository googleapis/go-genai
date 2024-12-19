package genai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		Backend: BackendGoogleAI,
		APIKey:  "test-api-key",
	})
	if err != nil {
		t.Fatal(err)
	}
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

	tests := []struct {
		desc        string
		backend     Backend
		client      *Client
		config      *LiveConnectConfig
		requestBody string
		wantErr     bool
	}{
		{
			desc:        "successful connection mldev",
			backend:     BackendGoogleAI,
			client:      mldevClient,
			requestBody: `{"setup":{"model":"models/test-model"}}`,
			wantErr:     false,
		},
		{
			desc:    "successful connection with config mldev",
			backend: BackendGoogleAI,
			client:  mldevClient,
			config: &LiveConnectConfig{
				GenerationConfig:  &GenerationConfig{Temperature: Ptr(0.5)},
				SystemInstruction: &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:             []*Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			requestBody: `{"setup":{"model":"models/test-model","generationConfig":{"temperature":0.5},"systemInstruction":{"parts":[{"text":"test instruction"}]},"tools":[{"googleSearch":{}}]}}`,
			wantErr:     false,
		},
		{
			desc:        "successful connection vertex",
			backend:     BackendVertexAI,
			client:      vertexClient,
			requestBody: `{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`,
			wantErr:     false,
		},
		{
			desc:    "successful connection with config vertex",
			backend: BackendVertexAI,
			client:  vertexClient,
			config: &LiveConnectConfig{
				GenerationConfig:  &GenerationConfig{Temperature: Ptr(0.5)},
				SystemInstruction: &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:             []*Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			requestBody: `{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model","generationConfig":{"temperature":0.5},"systemInstruction":{"parts":[{"text":"test instruction"}]},"tools":[{"googleSearch":{}}]}}`,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
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
				if diff := cmp.Diff(string(message), tt.requestBody); diff != "" {
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

			// if tt.backend == BackendVertexAI {
			// 	return
			// }
			tt.client.Live.apiClient.clientConfig.baseURL = strings.Replace(ts.URL, "http", "ws", 1)
			tt.client.Live.apiClient.clientConfig.HTTPClient = ts.Client()
			if tt.backend == BackendVertexAI {
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

			if !tt.wantErr {
				// Validate the session setup response if connection is successful
				message, err := session.Receive()

				if err != nil {
					t.Errorf("Receive() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if diff := cmp.Diff(message.SetupComplete, &LiveServerSetupComplete{}); diff != "" {
					t.Errorf("session setup mismatch (-want +got):\n%s", diff)
				}

			}
		})
	}
}

func TestLiveSendAndReceive(t *testing.T) {
	ctx := context.Background()
	ts := setupTestWebsocketServer(t, []string{`"setupComplete":{}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`})

	defer ts.Close()
	client := fakeLiveClient(ctx, t, ts)
	session, err := client.Live.Connect("test-model", &LiveConnectConfig{})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()
	// Discard the initial setup message.
	_, _ = session.Receive()

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
		t.Errorf("Receive failed: %v", err)
	}
	if diff := cmp.Diff(gotMessage, serverMessage); diff != "" {
		t.Errorf("Response message mismatch (-want +got):\n%s", diff)
	}
}

// Helper function to set up a test websocket server.
func setupTestWebsocketServer(t *testing.T, responseBodySlice []string) *httptest.Server {
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
			var serverMessage = &LiveServerMessage{}
			json.Unmarshal(message, serverMessage)

			err = conn.WriteMessage(mt, []byte(responseBodySlice[index]))
			index++
			if err != nil {
				t.Logf("write error: %v", err)
				break
			}
		}
	}))

	return ts
}

// Helper function to create a fake client for testing.
func fakeLiveClient(ctx context.Context, t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClient(ctx, &ClientConfig{
		baseURL:    strings.Replace(server.URL, "http", "ws", 1),
		HTTPClient: server.Client(),
		Backend:    BackendGoogleAI,
		APIKey:     "test-api-key",
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client
}
