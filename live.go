package genai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// Live struct encapsulates the configuration for realtime interaction with the Generative Language API.
type Live struct {
	apiClient *apiClient
}

// Session struct represents a realtime connection to the API.
type Session struct {
	conn      *websocket.Conn
	apiClient *apiClient
}

// Connect establishes a realtime connection to the specified model with given configuration.
// It returns a Session object representing the connection or an error if the connection fails.
func (r *Live) Connect(model string, config *LiveConnectConfig) (*Session, error) {
	baseURL, err := url.Parse(r.apiClient.clientConfig.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	scheme := baseURL.Scheme
	// Avoid overwrite schema if websocket scheme is already specified.
	if scheme != "wss" && scheme != "ws" {
		scheme = "wss"
	}

	var u url.URL
	var header http.Header
	if r.apiClient.clientConfig.Backend == BackendVertexAI {
		token, err := r.apiClient.clientConfig.Credentials.TokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to get token: %w", err)
		}
		header = http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{fmt.Sprintf("Bearer %s", token.AccessToken)},
		}
		u = url.URL{
			Scheme: scheme,
			Host:   baseURL.Host,
			// Host:     "generativelanguage.googleapis.com",
			// TODO(b/372231289): support custom api version.
			Path: "/ws/google.cloud.aiplatform.v1beta1.LlmBidiService/BidiGenerateContent",
		}
	} else {
		u = url.URL{
			Scheme: scheme,
			Host:   baseURL.Host,
			// Host:     "generativelanguage.googleapis.com",
			// TODO(b/372231289): support custom api version.
			Path:     "/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent",
			RawQuery: fmt.Sprintf("key=%s", r.apiClient.clientConfig.APIKey),
		}
		// TODO(b/372730941): support custom header
		header = http.Header{}
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("Connect to %s failed: %w", u.String(), err)
	}
	s := &Session{
		conn:      conn,
		apiClient: r.apiClient,
	}
	m, err := tModelFullName(r.apiClient, model)
	if err != nil {
		return nil, err
	}
	setup := &LiveClientSetup{
		Model: m,
	}

	if config != nil {
		setup.GenerationConfig = config.GenerationConfig
		setup.SystemInstruction = config.SystemInstruction
		setup.Tools = config.Tools
	}
	clientMessage := &LiveClientMessage{
		Setup: setup,
	}

	clientBytes, err := json.Marshal(clientMessage)
	if err != nil {
		return nil, fmt.Errorf("marshal LiveClientSetup failed: %w", err)
	}
	s.conn.WriteMessage(websocket.TextMessage, clientBytes)
	return s, nil
}

// Send transmits a BidiClientMessage over the established websocket connection.
// It returns an error if sending the message fails.
func (s *Session) Send(input *LiveClientMessage) error {
	if input.Setup != nil {
		return fmt.Errorf("message SetUp is not supported in Send(). Use Connect() instead")
	}
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal client message error: %w", err)
	}
	return s.conn.WriteMessage(websocket.TextMessage, []byte(data))
}

// Receive reads a BidiServerMessage from the websocket connection.
// It returns the received message or an error if reading or unmarshalling fails.
func (s *Session) Receive() (*LiveServerMessage, error) {
	messageType, msgBytes, err := s.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	// TODO(b/365983028): Implement response error handling.
	var message = new(LiveServerMessage)
	err = json.Unmarshal(msgBytes, message)
	if err != nil {
		return nil, fmt.Errorf("invalid message format. messageType: %d, message: %s", messageType, msgBytes)
	}
	return message, err
}

// Close terminates the websocket connection.
func (s *Session) Close() {
	s.conn.Close()
}
