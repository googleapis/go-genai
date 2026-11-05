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
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"cloud.google.com/go/auth"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
)

type mockCredentials struct {
	MockToken *auth.Token
}

func (mts mockCredentials) Token(context context.Context) (*auth.Token, error) {
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
	mldevClientWithToken, err := NewClient(ctx, &ClientConfig{
		Backend: BackendGeminiAPI,
		APIKey:  "auth_tokens/1234567890",
	})
	if err != nil {
		t.Fatal(err)
	}
	mockToken := &auth.Token{
		Value: "fake_access_token",
	}
	mockCred := mockCredentials{
		MockToken: mockToken,
	}

	vertexClient, err := NewClient(ctx, &ClientConfig{
		Backend:  BackendVertexAI,
		Project:  "test-project",
		Location: "test-location",
		Credentials: auth.NewCredentials(&auth.CredentialsOptions{
			TokenProvider: mockCred,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	connectTests := []struct {
		desc             string
		client           *Client
		clientHTTPOpts   *HTTPOptions
		config           *LiveConnectConfig
		fakeResponseBody string
		wantRequestBody  string
		wantHeaders      map[string]string
		wantPath         string
		wantErr          bool
		wantErrMessage   string
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
				Temperature:       Ptr[float32](0.5),
				SystemInstruction: &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:             []*Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			wantRequestBody: `{"setup":{"generationConfig":{"temperature":0.5},"model":"models/test-model","systemInstruction":{"parts":[{"text":"test instruction"}]},"tools":[{"googleSearch":{}}]}}`,
		},
		{
			desc:   "Fail if multispeaker config.",
			client: mldevClient,
			config: &LiveConnectConfig{
				SpeechConfig: &SpeechConfig{
					MultiSpeakerVoiceConfig: &MultiSpeakerVoiceConfig{
						SpeakerVoiceConfigs: []*SpeakerVoiceConfig{
							{
								Speaker: "Alice",
								VoiceConfig: &VoiceConfig{
									PrebuiltVoiceConfig: &PrebuiltVoiceConfig{VoiceName: "kore"},
								},
							},
							{
								Speaker: "Bob",
								VoiceConfig: &VoiceConfig{
									PrebuiltVoiceConfig: &PrebuiltVoiceConfig{VoiceName: "puck"},
								},
							},
						},
					},
				},
				Temperature:       Ptr[float32](0.5),
				SystemInstruction: &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:             []*Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			wantErr:        true,
			wantErrMessage: "multiSpeakerVoiceConfig is not supported",
		},
		{
			desc:            "successful connection with http options mldev",
			client:          mldevClient,
			clientHTTPOpts:  &HTTPOptions{Headers: map[string][]string{"test-header": {"test-value"}}, APIVersion: "test-api-version"},
			wantRequestBody: `{"setup":{"model":"models/test-model"}}`,
			wantHeaders:     map[string]string{"test-header": "test-value", "x-goog-api-key": "test-api-key"},
			wantPath:        "/ws/google.ai.generativelanguage.test-api-version.GenerativeService.BidiGenerateContent",
			wantErr:         false,
		},
		{
			desc:            "successful connection with http options mldev with ephemeral token",
			client:          mldevClientWithToken,
			clientHTTPOpts:  &HTTPOptions{APIVersion: "v1alpha"},
			wantRequestBody: `{"setup":{"model":"models/test-model"}}`,
			wantHeaders:     map[string]string{"Authorization": "Token auth_tokens/1234567890"},
			wantPath:        "/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained",
			wantErr:         false,
		},
		{
			desc:            "failed connection with http options mldev with ephemeral token",
			client:          mldevClientWithToken,
			clientHTTPOpts:  &HTTPOptions{APIVersion: "v1beta"},
			wantRequestBody: `{"setup":{"model":"models/test-model"}}`,
			wantErr:         true,
			wantErrMessage:  "Warning: Ephemeral token support is only supported in v1alpha API version. Please use clientConfig: ClientConfig{HTTPOptions: HTTPOptions{APIVersion: \"v1alpha\"}}",
		},
		{
			desc:            "failed connection with http options mldev",
			client:          mldevClient,
			clientHTTPOpts:  &HTTPOptions{BaseURL: "http://not-the-testing-server-url/path", APIVersion: "v1apha"},
			wantRequestBody: `{"setup":{"model":"models/test-model"}}`,
			wantErrMessage:  "Connect to ws://not-the-testing-server-url/path/ws/",
			wantErr:         true,
		},
		{
			desc:            "successful connection vertex",
			client:          vertexClient,
			wantRequestBody: `{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`,
		},
		{
			desc: "successful connection vertex express mode",
			client: func() *Client {
				c, _ := NewClient(ctx, &ClientConfig{
					Backend: BackendVertexAI,
					APIKey:  "test-api-key",
				})
				return c
			}(),
			wantRequestBody: `{"setup":{"model":"publishers/google/models/test-model"}}`,
		},
		{
			desc: "successful connection vertex custom proxy",
			client: func() *Client {
				c, _ := NewClient(ctx, &ClientConfig{
					Backend: BackendVertexAI,
					HTTPOptions: HTTPOptions{
						BaseURL: "ws://custom-proxy-server.com/my-custom-path",
					},
				})
				return c
			}(),
			wantRequestBody: `{"setup":{"model":"publishers/google/models/test-model"}}`,
			wantPath:        "/my-custom-path",
		},
		{
			desc:   "successful connection with config vertex",
			client: vertexClient,
			config: &LiveConnectConfig{
				Temperature:              Ptr[float32](0.5),
				SystemInstruction:        &Content{Parts: []*Part{{Text: "test instruction"}}},
				Tools:                    []*Tool{{GoogleSearch: &GoogleSearch{}}},
				OutputAudioTranscription: &AudioTranscriptionConfig{},
				ContextWindowCompression: &ContextWindowCompressionConfig{
					TriggerTokens: Ptr[int64](1024),
					SlidingWindow: &SlidingWindow{TargetTokens: Ptr[int64](1024)},
				},
				RealtimeInputConfig: &RealtimeInputConfig{
					AutomaticActivityDetection: &AutomaticActivityDetection{
						Disabled:                 true,
						StartOfSpeechSensitivity: StartSensitivityLow,
						EndOfSpeechSensitivity:   EndSensitivityLow,
						PrefixPaddingMs:          Ptr[int32](1000),
						SilenceDurationMs:        Ptr[int32](2000),
					},
				},
			},
			wantRequestBody: `{"setup":{"contextWindowCompression":{"slidingWindow":{"targetTokens":"1024"},"triggerTokens":"1024"},"generationConfig":{"temperature":0.5},"model":"projects/test-project/locations/test-location/publishers/google/models/test-model","outputAudioTranscription":{},"realtimeInputConfig":{"automaticActivityDetection":{"disabled":true,"endOfSpeechSensitivity":"END_SENSITIVITY_LOW","prefixPaddingMs":1000,"silenceDurationMs":2000,"startOfSpeechSensitivity":"START_SENSITIVITY_LOW"}},"systemInstruction":{"parts":[{"text":"test instruction"}]},"tools":[{"googleSearch":{}}]}}`,
		},
		{
			desc:   "failed connection when set transparent using mldev client",
			client: mldevClient,
			config: &LiveConnectConfig{
				SessionResumption: &SessionResumptionConfig{
					Handle:      "test_handle",
					Transparent: true,
				},
			},
			wantErr:        true,
			wantErrMessage: "transparent parameter is only supported in Gemini Enterprise Agent Platform mode, not in Gemini Developer API mode.",
		},
		{
			desc:   "successful connection when set transparent using vertex client",
			client: vertexClient,
			config: &LiveConnectConfig{
				SessionResumption: &SessionResumptionConfig{
					Handle:      "test_handle",
					Transparent: true,
				},
			},
			fakeResponseBody: `{"sessionResumptionUpdate":{"newHandle":"test_handle","resumable":true,"lastConsumedClientMessageIndex":"123456789"}}`,
			wantRequestBody:  `{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model","sessionResumption":{"handle":"test_handle","transparent":true}}}`,
		},
	}

	for _, tt := range connectTests {
		t.Run(tt.desc, func(t *testing.T) {
			var upgrader = websocket.Upgrader{}
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, _ := upgrader.Upgrade(w, r, nil)
				defer conn.Close()

				if tt.config != nil && tt.clientHTTPOpts != nil {
					if tt.wantHeaders != nil {
						if diff := cmp.Diff(r.Header.Get("test-header"), tt.wantHeaders["test-header"]); diff != "" {
							t.Errorf("Request header mismatch (-want +got):\n%s", diff)
						}
						if diff := cmp.Diff(r.Header.Get("x-goog-api-key"), tt.wantHeaders["x-goog-api-key"]); diff != "" {
							t.Errorf("Request header mismatch (-want +got):\n%s", diff)
						}
						if diff := cmp.Diff(r.Header.Get("Authorization"), tt.wantHeaders["Authorization"]); diff != "" {
							t.Errorf("Request header mismatch (-want +got):\n%s", diff)
						}
					}
					if tt.wantPath != "" {
						if diff := cmp.Diff(r.URL.String(), tt.wantPath); diff != "" {
							t.Errorf("Request URL mismatch (-want +got):\n%s", diff)
						}
					}
				}

				mt, message, err := conn.ReadMessage()

				if err != nil {
					if tt.wantErr {
						return
					}
					t.Fatalf("ReadMessage: %v", err)
				}

				if string(message) != tt.wantRequestBody {
					t.Errorf("Request message mismatch got %s, want %s", string(message), tt.wantRequestBody)
				}
				if tt.wantErr {
					conn.Close()
					return
				}

				if tt.fakeResponseBody == "" {
					tt.fakeResponseBody = `{"setupComplete":{}}`
				}

				err = conn.WriteMessage(mt, []byte(tt.fakeResponseBody))
				if err != nil {
					t.Fatalf("WriteMessage: %v", err)
				}
			}))
			defer ts.Close()

			url := ts.URL
			if tt.clientHTTPOpts != nil {
				tt.client.Live.apiClient.clientConfig.HTTPOptions = *tt.clientHTTPOpts
				url = tt.clientHTTPOpts.BaseURL
			}
			tt.client.Live.apiClient.clientConfig.HTTPOptions.BaseURL = strings.Replace(url, "http", "ws", 1)
			tt.client.Live.apiClient.clientConfig.HTTPClient = ts.Client()
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}
			session, err := tt.client.Live.Connect(ctx, model, tt.config)

			if tt.wantErr && !strings.Contains(err.Error(), tt.wantErrMessage) {
				t.Errorf("Connect() error message = %v, wantErrMessage %v", err.Error(), tt.wantErrMessage)
				return
			}

			defer session.Close()
		})
	}

	t.Run("SendClientContent and Receive", func(t *testing.T) {
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
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"clientContent":{"turnComplete":true,"turns":[{"parts":[{"text":"client test message"}],"role":"user"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "send clientContent to Vertex AI",
				client:                vertexClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`, `{"clientContent":{"turnComplete":true,"turns":[{"parts":[{"text":"client test message"}],"role":"user"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "received error in response",
				client:                mldevClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"clientContent":{"turnComplete":true,"turns":[{"parts":[{"text":"client test message"}],"role":"user"}]}}`},
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

				session, err := tt.client.Live.Connect(ctx, "test-model", &LiveConnectConfig{})
				if err != nil {
					t.Fatalf("Connect failed: %v", err)
				}
				defer session.Close()

				if session.SetupComplete == nil {
					t.Errorf("SetupComplete is nil after Connect")
				}

				// Test sending the message
				err = session.SendClientContent(LiveClientContentInput{Turns: Text("client test message")})
				if err != nil {
					t.Errorf("Send failed : %v", err)
				}

				// Construct the expected response
				serverMessage := &LiveServerMessage{ServerContent: &LiveServerContent{ModelTurn: Text("server test message")[0]}}
				// Test receiving the response
				_, err = session.Receive()
				if err != nil {
					t.Errorf("Receive failed: %v", err)
				}
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

	t.Run("SendRealtimeInput and Receive", func(t *testing.T) {
		sendReceiveTests := []struct {
			desc                  string
			client                *Client
			realtimeInput         LiveRealtimeInput
			wantRequestBodySlice  []string
			fakeResponseBodySlice []string
			wantErr               bool
		}{
			{
				desc:                  "send realtimeInput to Google AI",
				client:                mldevClient,
				realtimeInput:         LiveRealtimeInput{Media: &Blob{Data: []byte("test data"), MIMEType: "image/png"}},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"realtimeInput":{"mediaChunks":[{"data":"dGVzdCBkYXRh","mimeType":"image/png"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "send realtimeInput to Vertex AI",
				client:                vertexClient,
				realtimeInput:         LiveRealtimeInput{Media: &Blob{Data: []byte("test data"), MIMEType: "image/png"}},
				wantRequestBodySlice:  []string{`{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`, `{"realtimeInput":{"mediaChunks":[{"data":"dGVzdCBkYXRh","mimeType":"image/png"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "received error in response",
				client:                mldevClient,
				realtimeInput:         LiveRealtimeInput{Media: &Blob{Data: []byte("test data"), MIMEType: "image/png"}},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"realtimeInput":{"mediaChunks":[{"data":"dGVzdCBkYXRh","mimeType":"image/png"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"error":{"code":400,"message":"test error message","status":"INVALID_ARGUMENT"}}`},
				wantErr:               true,
			},
			{
				desc:                  "send audio realtimeInput to Google AI",
				client:                mldevClient,
				realtimeInput:         LiveRealtimeInput{Audio: &Blob{Data: []byte("test data"), MIMEType: "audio/pcm"}},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"realtimeInput":{"audio":{"data":"dGVzdCBkYXRh","mimeType":"audio/pcm"}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "send video realtimeInput to Google AI",
				client:                mldevClient,
				realtimeInput:         LiveRealtimeInput{Video: &Blob{Data: []byte("test data"), MIMEType: "image/jpeg"}},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"realtimeInput":{"video":{"data":"dGVzdCBkYXRh","mimeType":"image/jpeg"}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "send text realtimeInput to Google AI",
				client:                mldevClient,
				realtimeInput:         LiveRealtimeInput{Text: "test data"},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"realtimeInput":{"text":"test data"}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
		}

		for _, tt := range sendReceiveTests {
			t.Run(tt.desc, func(t *testing.T) {
				ts := setupTestWebsocketServer(t, tt.wantRequestBodySlice, tt.fakeResponseBodySlice)
				defer ts.Close()

				tt.client.Live.apiClient.clientConfig.HTTPOptions.BaseURL = strings.Replace(ts.URL, "http", "ws", 1)
				tt.client.Live.apiClient.clientConfig.HTTPClient = ts.Client()

				session, err := tt.client.Live.Connect(ctx, "test-model", &LiveConnectConfig{})
				if err != nil {
					t.Fatalf("Connect failed: %v", err)
				}
				defer session.Close()

				if session.SetupComplete == nil {
					t.Errorf("SetupComplete is nil after Connect")
				}

				// Test sending the message
				err = session.SendRealtimeInput(tt.realtimeInput)
				if err != nil {
					t.Errorf("Send failed : %v", err)
				}

				// Construct the expected response
				serverMessage := &LiveServerMessage{ServerContent: &LiveServerContent{ModelTurn: Text("server test message")[0]}}
				// Test receiving the response
				_, err = session.Receive()
				if err != nil {
					t.Errorf("Receive failed: %v", err)
				}
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

	t.Run("SendToolResponse and Receive", func(t *testing.T) {
		sendReceiveTests := []struct {
			desc                  string
			client                *Client
			wantRequestBodySlice  []string
			fakeResponseBodySlice []string
			wantErr               bool
		}{
			{
				desc:                  "send realtimeInput to Google AI",
				client:                mldevClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"toolResponse":{"functionResponses":[{"name":"test-function"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "send realtimeInput to Vertex AI",
				client:                vertexClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`, `{"toolResponse":{"functionResponses":[{"name":"test-function"}]}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:                  "received error in response",
				client:                mldevClient,
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"toolResponse":{"functionResponses":[{"name":"test-function"}]}}`},
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

				session, err := tt.client.Live.Connect(ctx, "test-model", &LiveConnectConfig{})
				if err != nil {
					t.Fatalf("Connect failed: %v", err)
				}
				defer session.Close()

				if session.SetupComplete == nil {
					t.Errorf("SetupComplete is nil after Connect")
				}

				// Test sending the message
				err = session.SendToolResponse(LiveToolResponseInput{FunctionResponses: []*FunctionResponse{{Name: "test-function"}}})
				if err != nil {
					t.Errorf("Send failed : %v", err)
				}

				// Construct the expected response
				serverMessage := &LiveServerMessage{ServerContent: &LiveServerContent{ModelTurn: Text("server test message")[0]}}
				// Test receiving the response
				_, err = session.Receive()
				if err != nil {
					t.Errorf("Receive failed: %v", err)
				}
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

	t.Run("SendContextUpdate and Receive", func(t *testing.T) {
		sendReceiveTests := []struct {
			desc                  string
			client                *Client
			contextUpdate         LiveClientContextUpdate
			wantRequestBodySlice  []string
			fakeResponseBodySlice []string
			wantErr               bool
		}{
			{
				desc:   "send systemInstruction contextUpdate to Google AI",
				client: mldevClient,
				contextUpdate: LiveClientContextUpdate{
					SystemInstruction: &Content{Parts: []*Part{{Text: "You are a helpful assistant."}}},
				},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"contextUpdate":{"systemInstruction":{"parts":[{"text":"You are a helpful assistant."}]}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:   "send systemInstruction contextUpdate to Vertex AI",
				client: vertexClient,
				contextUpdate: LiveClientContextUpdate{
					SystemInstruction: &Content{Parts: []*Part{{Text: "You are a helpful assistant."}}},
				},
				wantRequestBodySlice:  []string{`{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`, `{"contextUpdate":{"systemInstruction":{"parts":[{"text":"You are a helpful assistant."}]}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:   "send clear tools contextUpdate to Google AI",
				client: mldevClient,
				contextUpdate: LiveClientContextUpdate{
					Tools: &LiveClientContextUpdateTools{Tools: []*Tool{}},
				},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"contextUpdate":{"tools":{}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:   "send function declaration tools contextUpdate to Google AI",
				client: mldevClient,
				contextUpdate: LiveClientContextUpdate{
					Tools: &LiveClientContextUpdateTools{
						Tools: []*Tool{
							{
								FunctionDeclarations: []*FunctionDeclaration{
									{Name: "get_current_weather"},
								},
							},
						},
					},
				},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"contextUpdate":{"tools":{"tools":[{"functionDeclarations":[{"name":"get_current_weather"}]}]}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:   "send function declaration tools contextUpdate to Vertex AI",
				client: vertexClient,
				contextUpdate: LiveClientContextUpdate{
					Tools: &LiveClientContextUpdateTools{
						Tools: []*Tool{
							{
								FunctionDeclarations: []*FunctionDeclaration{
									{Name: "get_current_weather"},
								},
							},
						},
					},
				},
				wantRequestBodySlice:  []string{`{"setup":{"model":"projects/test-project/locations/test-location/publishers/google/models/test-model"}}`, `{"contextUpdate":{"tools":{"tools":[{"functionDeclarations":[{"name":"get_current_weather"}]}]}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
			{
				desc:   "send both systemInstruction and tools contextUpdate to Google AI",
				client: mldevClient,
				contextUpdate: LiveClientContextUpdate{
					SystemInstruction: &Content{Parts: []*Part{{Text: "Updated instruction."}}},
					Tools: &LiveClientContextUpdateTools{
						Tools: []*Tool{
							{
								FunctionDeclarations: []*FunctionDeclaration{
									{Name: "get_current_weather"},
								},
							},
						},
					},
				},
				wantRequestBodySlice:  []string{`{"setup":{"model":"models/test-model"}}`, `{"contextUpdate":{"systemInstruction":{"parts":[{"text":"Updated instruction."}]},"tools":{"tools":[{"functionDeclarations":[{"name":"get_current_weather"}]}]}}}`},
				fakeResponseBodySlice: []string{`{"setupComplete":{}}`, `{"serverContent":{"modelTurn":{"parts":[{"text":"server test message"}],"role":"user"}}}`},
			},
		}

		for _, tt := range sendReceiveTests {
			t.Run(tt.desc, func(t *testing.T) {
				ts := setupTestWebsocketServer(t, tt.wantRequestBodySlice, tt.fakeResponseBodySlice)
				defer ts.Close()

				tt.client.Live.apiClient.clientConfig.HTTPOptions.BaseURL = strings.Replace(ts.URL, "http", "ws", 1)
				tt.client.Live.apiClient.clientConfig.HTTPClient = ts.Client()

				session, err := tt.client.Live.Connect(ctx, "test-model", &LiveConnectConfig{})
				if err != nil {
					t.Fatalf("Connect failed: %v", err)
				}
				defer session.Close()

				if session.SetupComplete == nil {
					t.Errorf("SetupComplete is nil after Connect")
				}

				err = session.SendContextUpdate(tt.contextUpdate)
				if err != nil {
					t.Errorf("SendContextUpdate failed: %v", err)
				}

				serverMessage := &LiveServerMessage{ServerContent: &LiveServerContent{ModelTurn: Text("server test message")[0]}}
				_, err = session.Receive()
				if err != nil {
					t.Errorf("Receive failed: %v", err)
				}
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

// TestLiveConnectSDKHeaders pins that the Live WebSocket handshake carries the
// same SDK User-Agent and x-goog-api-client values as HTTP requests.
func TestLiveConnectSDKHeaders(t *testing.T) {
	ctx := context.Background()
	sdkHeader := fmt.Sprintf("google-genai-sdk/%s gl-go/%s", version, runtime.Version())
	vertexModulesHeader := fmt.Sprintf("google-genai-sdk/%s+vertex-genai-modules/0.20.0 gl-go/%s", version, runtime.Version())
	creds := auth.NewCredentials(&auth.CredentialsOptions{
		TokenProvider: mockCredentials{MockToken: &auth.Token{Value: "fake_access_token"}},
	})

	tests := []struct {
		desc          string
		config        ClientConfig
		wantUserAgent string
		wantAPIClient []string
	}{
		{
			desc:          "gemini api",
			config:        ClientConfig{Backend: BackendGeminiAPI, APIKey: "test-api-key"},
			wantUserAgent: sdkHeader,
			wantAPIClient: []string{sdkHeader},
		},
		{
			desc: "vertex ai",
			config: ClientConfig{
				Backend:     BackendVertexAI,
				Project:     "test-project",
				Location:    "test-location",
				Credentials: creds,
			},
			wantUserAgent: sdkHeader,
			wantAPIClient: []string{sdkHeader},
		},
		{
			// net/http writes only the first User-Agent value to the wire.
			desc: "custom user agent",
			config: ClientConfig{
				Backend:     BackendGeminiAPI,
				APIKey:      "test-api-key",
				HTTPOptions: HTTPOptions{Headers: http.Header{"User-Agent": {"my-custom-agent/1.0"}}},
			},
			wantUserAgent: "my-custom-agent/1.0",
			wantAPIClient: []string{sdkHeader},
		},
		{
			desc: "vertex-genai-modules user agent",
			config: ClientConfig{
				Backend:     BackendGeminiAPI,
				APIKey:      "test-api-key",
				HTTPOptions: HTTPOptions{Headers: http.Header{"User-Agent": {"vertex-genai-modules/0.20.0"}}},
			},
			wantUserAgent: vertexModulesHeader,
			wantAPIClient: []string{vertexModulesHeader},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			headers := make(chan http.Header, 1)
			upgrader := websocket.Upgrader{}
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				headers <- r.Header.Clone()
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				mt, _, err := conn.ReadMessage()
				if err != nil {
					return
				}
				conn.WriteMessage(mt, []byte(`{"setupComplete":{}}`))
			}))
			defer ts.Close()

			cfg := tt.config
			cfg.HTTPOptions.BaseURL = "ws" + strings.TrimPrefix(ts.URL, "http")
			client, err := NewClient(ctx, &cfg)
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}
			clientHeaders := client.Live.apiClient.clientConfig.HTTPOptions.Headers.Clone()

			session, err := client.Live.Connect(ctx, "test-model", nil)
			if err != nil {
				t.Fatalf("Connect failed: %v", err)
			}
			session.Close()

			got := <-headers
			if diff := cmp.Diff(tt.wantUserAgent, got.Get("User-Agent")); diff != "" {
				t.Errorf("User-Agent mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAPIClient, got.Values("X-Goog-Api-Client")); diff != "" {
				t.Errorf("X-Goog-Api-Client mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(clientHeaders, client.Live.apiClient.clientConfig.HTTPOptions.Headers); diff != "" {
				t.Errorf("Connect mutated client headers (-before +after):\n%s", diff)
			}
		})
	}
}
