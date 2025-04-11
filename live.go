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
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// Preview. Live can be used to create a realtime connection to the API.
// It is initiated when creating a client. You don't need to create a new Live object.
// The live module is experimental.
//
//	client, _ := genai.NewClient(ctx, &genai.ClientConfig{})
//	session, _ := client.Live.Connect(model, &genai.LiveConnectConfig{}).
type Live struct {
	apiClient *apiClient
}

// Preview. Session is a realtime connection to the API.
// The live module is experimental.
type Session struct {
	conn      *websocket.Conn
	apiClient *apiClient
}

// Preview. Connect establishes a realtime connection to the specified model with given configuration.
// It returns a Session object representing the connection or an error if the connection fails.
// The live module is experimental.
func (r *Live) Connect(context context.Context, model string, config *LiveConnectConfig) (*Session, error) {
	httpOptions := r.apiClient.clientConfig.HTTPOptions
	if httpOptions.APIVersion == "" {
		return nil, fmt.Errorf("live module requires APIVersion to be set. You can set APIVersion to v1beta1 for BackendVertexAI or v1apha for BackendGeminiAPI")
	}
	baseURL, err := url.Parse(httpOptions.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	scheme := baseURL.Scheme
	// Avoid overwrite schema if websocket scheme is already specified.
	if scheme != "wss" && scheme != "ws" {
		scheme = "wss"
	}

	var u url.URL
	// TODO(b/406076143): Support function level httpOptions.
	var header http.Header = mergeHeaders(&httpOptions, nil)
	if r.apiClient.clientConfig.Backend == BackendVertexAI {
		token, err := r.apiClient.clientConfig.Credentials.Token(context)
		if err != nil {
			return nil, fmt.Errorf("failed to get token: %w", err)
		}
		header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Value))
		u = url.URL{
			Scheme: scheme,
			Host:   baseURL.Host,
			Path:   fmt.Sprintf("%s/ws/google.cloud.aiplatform.%s.LlmBidiService/BidiGenerateContent", baseURL.Path, httpOptions.APIVersion),
		}
	} else {
		u = url.URL{
			Scheme:   scheme,
			Host:     baseURL.Host,
			Path:     fmt.Sprintf("%s/ws/google.ai.generativelanguage.%s.GenerativeService.BidiGenerateContent", baseURL.Path, httpOptions.APIVersion),
			RawQuery: fmt.Sprintf("key=%s", r.apiClient.clientConfig.APIKey),
		}
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("Connect to %s failed: %w", u.String(), err)
	}
	s := &Session{
		conn:      conn,
		apiClient: r.apiClient,
	}
	modelFullName, err := tModelFullName(r.apiClient, model)
	if err != nil {
		return nil, err
	}
	kwargs := map[string]any{"model": modelFullName, "config": config}
	parameterMap := make(map[string]any)
	err = deepMarshal(kwargs, &parameterMap)
	if err != nil {
		return nil, err
	}

	var toConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	if r.apiClient.clientConfig.Backend == BackendVertexAI {
		toConverter = liveConnectParametersToVertex
	} else {
		toConverter = liveConnectParametersToMldev
	}
	body, err := toConverter(r.apiClient, parameterMap, nil)
	if err != nil {
		return nil, err
	}
	delete(body, "config")

	clientBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal LiveClientSetup failed: %w", err)
	}
	err = s.conn.WriteMessage(websocket.TextMessage, clientBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to write LiveClientSetup: %w", err)
	}
	return s, nil
}

// Preview. LiveClientContentInput is the input for [SendClientContent].
type LiveClientContentInput struct {
	// The content appended to the current conversation with the model.
	// For single-turn queries, this is a single instance. For multi-turn
	// queries, this is a repeated field that contains conversation history and
	// latest request.
	Turns []*Content `json:"turns,omitempty"`
	// TurnComplete is default to true, indicating that the server content generation should
	// start with the currently accumulated prompt. If set to false, the server will await
	// additional messages, accumulating the prompt, and start generation until received a
	// TurnComplete true message.
	TurnComplete *bool `json:"turnComplete,omitempty"`
}

// Preview. SendClientContent transmits a [LiveClientContent] over the established connection.
// It returns an error if sending the message fails.
// The live module is experimental.
func (s *Session) SendClientContent(input LiveClientContentInput) error {
	if input.TurnComplete == nil {
		input.TurnComplete = Ptr(true)
	}
	clientMessage := &LiveClientMessage{
		ClientContent: &LiveClientContent{Turns: input.Turns, TurnComplete: *input.TurnComplete},
	}
	return s.send(clientMessage)
}

// Preview. LiveRealtimeInput is the input for [SendRealtimeInput].
type LiveRealtimeInput struct {
	Media *Blob `json:"media,omitempty"`
}

// Preview. SendRealtimeInput transmits a [LiveClientRealtimeInput] over the established connection.
// It returns an error if sending the message fails.
// The live module is experimental.
func (s *Session) SendRealtimeInput(input LiveRealtimeInput) error {
	clientMessage := &LiveClientMessage{
		RealtimeInput: &LiveClientRealtimeInput{MediaChunks: []*Blob{input.Media}},
	}
	return s.send(clientMessage)
}

// Preview. LiveToolResponseInput is the input for [SendToolResponse].
type LiveToolResponseInput struct {
	// The response to the function calls.
	FunctionResponses []*FunctionResponse `json:"functionResponses,omitempty"`
}

// Preview. SendToolResponse transmits a [LiveClientToolResponse] over the established connection.
// It returns an error if sending the message fails.
// The live module is experimental.
func (s *Session) SendToolResponse(input LiveToolResponseInput) error {
	clientMessage := &LiveClientMessage{
		ToolResponse: &LiveClientToolResponse{FunctionResponses: input.FunctionResponses},
	}
	return s.send(clientMessage)
}

// Send transmits a LiveClientMessage over the established connection.
// It returns an error if sending the message fails.
// The live module is experimental.
func (s *Session) send(input *LiveClientMessage) error {
	if input.Setup != nil {
		return fmt.Errorf("message SetUp is not supported in Send(). Use Connect() instead")
	}

	parameterMap := make(map[string]any)
	err := deepMarshal(input, &parameterMap)
	if err != nil {
		return err
	}

	var toConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	if s.apiClient.clientConfig.Backend == BackendVertexAI {
		toConverter = liveClientMessageToVertex
	} else {
		toConverter = liveClientMessageToMldev
	}
	body, err := toConverter(s.apiClient, parameterMap, nil)
	if err != nil {
		return err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal client message error: %w", err)
	}
	return s.conn.WriteMessage(websocket.TextMessage, []byte(data))
}

// Preview. Receive reads a LiveServerMessage from the connection.
// It returns the received message or an error if reading or unmarshalling fails.
// The live module is experimental.
func (s *Session) Receive() (*LiveServerMessage, error) {
	messageType, msgBytes, err := s.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	responseMap := make(map[string]any)
	err = json.Unmarshal(msgBytes, &responseMap)
	if err != nil {
		return nil, fmt.Errorf("invalid message format. Error %w. messageType: %d, message: %s", err, messageType, msgBytes)
	}
	if responseMap["error"] != nil {
		return nil, fmt.Errorf("received error in response: %v", string(msgBytes))
	}

	var fromConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	if s.apiClient.clientConfig.Backend == BackendVertexAI {
		fromConverter = liveServerMessageFromVertex
	} else {
		fromConverter = liveServerMessageFromMldev
	}
	responseMap, err = fromConverter(s.apiClient, responseMap, nil)
	if err != nil {
		return nil, err
	}

	var message = new(LiveServerMessage)
	err = mapToStruct(responseMap, message)
	if err != nil {
		return nil, err
	}
	return message, err
}

// Preview. Close terminates the connection.
// The live module is experimental.
func (s *Session) Close() error {
	if s != nil && s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
