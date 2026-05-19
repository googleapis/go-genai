package genai

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)



type SecureRoundTripper struct {
	base         http.RoundTripper
	clientConfig *ClientConfig
	mu           sync.RWMutex
	sessions     map[string]*secureSessionState
}

func NewSecureRoundTripper(base http.RoundTripper, cc *ClientConfig) *SecureRoundTripper {
	return &SecureRoundTripper{
		base:         base,
		clientConfig: cc,
		sessions:     make(map[string]*secureSessionState),
	}
}

type pendingRequest struct {
	ch chan []byte
}

type secureSessionState struct {
	srt             *SecureRoundTripper
	model           string
	wsConn          *websocket.Conn
	tlsConn         *tls.Conn
	pendingRequests map[string]*pendingRequest
	mu              sync.Mutex
	closed          bool
	serverError     error
}

func (s *secureSessionState) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	if s.tlsConn != nil {
		s.tlsConn.Close()
	}
	if s.wsConn != nil {
		s.wsConn.Close()
	}
	for _, req := range s.pendingRequests {
		close(req.ch)
	}
	s.pendingRequests = make(map[string]*pendingRequest)
	s.mu.Unlock()

}

type wsTLSConn struct {
	wsConn  *websocket.Conn
	rBuf    bytes.Buffer
	session *secureSessionState
}

func (w *wsTLSConn) Read(b []byte) (int, error) {
	if w.rBuf.Len() > 0 {
		return w.rBuf.Read(b)
	}
	for {
		_, msg, err := w.wsConn.ReadMessage()
		if err != nil {
			return 0, err
		}
		var rawFrame map[string]any
		if err := json.Unmarshal(msg, &rawFrame); err != nil {
			return 0, err
		}
		for _, errKey := range []string{"error", "privateInferenceError", "private_inference_error"} {
			if errVal, hasErr := rawFrame[errKey]; hasErr {
				errStr := fmt.Sprintf("private inference server error: %v", errVal)
				if w.session != nil {
					w.session.mu.Lock()
					w.session.serverError = fmt.Errorf("%s", errStr)
					w.session.mu.Unlock()
				}
				return 0, fmt.Errorf("%s", errStr)
			}
		}
		if errVal, hasErr := rawFrame["status"]; hasErr {
			if statusMap, ok := errVal.(map[string]any); ok {
				if code, ok := statusMap["code"]; ok && fmt.Sprintf("%v", code) != "0" {
					errStr := fmt.Sprintf("private inference server error: %v", errVal)
					if w.session != nil {
						w.session.mu.Lock()
						w.session.serverError = fmt.Errorf("%s", errStr)
						w.session.mu.Unlock()
					}
					return 0, fmt.Errorf("%s", errStr)
				}
			}
		}
		var content string
		if tf, ok := rawFrame["tlsFrame"].(map[string]any); ok {
			if c, ok := tf["content"].(string); ok {
				content = c
			}
		} else if tf, ok := rawFrame["tls_frame"].(map[string]any); ok {
			if c, ok := tf["content"].(string); ok {
				content = c
			}
		}

		if content == "" {
			continue
		}
		decoded, err := base64.URLEncoding.DecodeString(content)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(content)
			if err != nil {
				return 0, err
			}
		}
		if len(decoded) > 0 {
			w.rBuf.Write(decoded)
			return w.rBuf.Read(b)
		}
	}
}

func (w *wsTLSConn) Write(b []byte) (int, error) {
	encoded := base64.URLEncoding.EncodeToString(b)
	frame := map[string]any{
		"tls_frame": map[string]string{
			"content": encoded,
		},
	}
	jsonBytes, err := json.Marshal(frame)
	if err != nil {
		return 0, err
	}
	err = w.wsConn.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *wsTLSConn) Close() error {
	return w.wsConn.Close()
}

func (w *wsTLSConn) LocalAddr() net.Addr {
	return w.wsConn.LocalAddr()
}

func (w *wsTLSConn) RemoteAddr() net.Addr {
	return w.wsConn.RemoteAddr()
}

func (w *wsTLSConn) SetDeadline(t time.Time) error {
	if err := w.wsConn.SetReadDeadline(t); err != nil {
		return err
	}
	return w.wsConn.SetWriteDeadline(t)
}

func (w *wsTLSConn) SetReadDeadline(t time.Time) error {
	return w.wsConn.SetReadDeadline(t)
}

func (w *wsTLSConn) SetWriteDeadline(t time.Time) error {
	return w.wsConn.SetWriteDeadline(t)
}

var muActivation sync.Mutex

func (m *Models) StartSecureSession(ctx context.Context, model string, caPool string, rootCAPath string) error {
	muActivation.Lock()
	httpClient := m.apiClient.clientConfig.HTTPClient
	if httpClient == nil {
		muActivation.Unlock()
		return fmt.Errorf("HTTPClient is not initialized")
	}

	baseTransport := httpClient.Transport
	if srt, ok := baseTransport.(*SecureRoundTripper); ok {
		baseTransport = srt.base
	}
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	secureTransport := NewSecureRoundTripper(baseTransport, m.apiClient.clientConfig)

	clonedClient := *httpClient
	clonedClient.Transport = secureTransport
	m.apiClient.piClients.Store(model, &clonedClient)
	atomic.StoreInt32(&m.apiClient.hasPI, 1)

	muActivation.Unlock()

	return secureTransport.StartSecureSession(ctx, m.apiClient, model, caPool, rootCAPath)
}

func (s *SecureRoundTripper) StartSecureSession(ctx context.Context, apiClient *apiClient, model string, caPool string, rootCAPath string) error {
	s.mu.Lock()
	old, exists := s.sessions[model]
	if exists {
		delete(s.sessions, model)
	}
	s.mu.Unlock()

	if exists {
		old.close()
	}

	rootCA, err := os.ReadFile(rootCAPath)
	if err != nil {
		return fmt.Errorf("failed to read root CA file: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(rootCA) {
		return fmt.Errorf("failed to append root CA to pool")
	}

	httpOptions := s.clientConfig.HTTPOptions
	if httpOptions.APIVersion == "" {
		if s.clientConfig.Backend == BackendVertexAI {
			httpOptions.APIVersion = "v1beta1"
		} else {
			httpOptions.APIVersion = "v1beta"
		}
	}

	baseURL, err := url.Parse(httpOptions.BaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %w", err)
	}

	scheme := baseURL.Scheme
	if scheme != "wss" && scheme != "ws" {
		scheme = "wss"
	}

	var header http.Header = mergeHeaders(&httpOptions, nil)
	var wsPath string

	if s.clientConfig.Backend == BackendVertexAI {
		hasStandardAuth := s.clientConfig.Project != "" && s.clientConfig.Location != ""
		if s.clientConfig.Credentials != nil {
			token, err := s.clientConfig.Credentials.Token(ctx)
			if err != nil {
				return fmt.Errorf("failed to get token: %w", err)
			}
			header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Value))
		}

		wsPath = path.Join(baseURL.Path, fmt.Sprintf("ws/google.cloud.aiplatform.%s.PrivateInferenceService/StartSecureSession", httpOptions.APIVersion))
		if baseURL.String() != "" && !hasStandardAuth {
			wsPath = baseURL.Path
		}

		wsPath = path.Join(wsPath, "models", model)
	} else {
		return fmt.Errorf("StartSecureSession is only supported for Vertex AI backend")
	}

	u := url.URL{
		Scheme: scheme,
		Host:   baseURL.Host,
		Path:   wsPath,
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return fmt.Errorf("Connect to %s failed: %w", u.String(), err)
	}

	modelFullName, err := tModelFullName(apiClient, model)
	if err != nil {
		conn.Close()
		return err
	}

	setupReq := map[string]any{
		"setup_request": map[string]string{
			"model":   modelFullName,
			"ca_pool": caPool,
		},
	}
	setupBytes, err := json.Marshal(setupReq)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to marshal setup request: %w", err)
	}

	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, setupBytes); err != nil {
		conn.Close()
		return fmt.Errorf("failed to write setup request: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		conn.Close()
		return fmt.Errorf("failed to set read deadline: %w", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to read setup response: %w", err)
	}

	if err := conn.SetWriteDeadline(time.Time{}); err != nil {
		conn.Close()
		return fmt.Errorf("failed to reset write deadline: %w", err)
	}
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		conn.Close()
		return fmt.Errorf("failed to reset read deadline: %w", err)
	}

	var setupResp map[string]any
	dec := json.NewDecoder(bytes.NewReader(msg))
	dec.UseNumber()
	if err := dec.Decode(&setupResp); err != nil {
		conn.Close()
		return fmt.Errorf("failed to unmarshal setup response: %w", err)
	}

	wtConn := &wsTLSConn{wsConn: conn}
	tlsConfig := &tls.Config{
		RootCAs:    pool,
		ServerName: "aiplatform.googleapis.com",
	}

	tlsClientConn := tls.Client(wtConn, tlsConfig)
	if err := tlsClientConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	session := &secureSessionState{
		srt:             s,
		model:           model,
		wsConn:          conn,
		tlsConn:         tlsClientConn,
		pendingRequests: make(map[string]*pendingRequest),
	}

	s.mu.Lock()
	s.sessions[model] = session
	s.mu.Unlock()

	wtConn.session = session

	go session.readLoop()

	return nil
}

func extractModelFromPath(p string) string {
	parts := strings.Split(p, "models/")
	if len(parts) > 1 {
		return strings.Split(parts[1], ":")[0]
	}
	return ""
}

func (s *SecureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !strings.Contains(req.URL.Path, ":generateContent") {
		return s.base.RoundTrip(req)
	}

	modelName := extractModelFromPath(req.URL.Path)
	s.mu.RLock()
	session, active := s.sessions[modelName]
	s.mu.RUnlock()

	if !active || modelName == "" {
		return s.base.RoundTrip(req)
	}

	session.mu.Lock()
	closed := session.closed
	session.mu.Unlock()
	if closed {
		return nil, fmt.Errorf("private inference secure session dropped or failed, please re-initialize via StartSecureSession")
	}

	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var reqBody map[string]any
	if len(bodyBytes) > 0 {
		dec := json.NewDecoder(bytes.NewReader(bodyBytes))
		dec.UseNumber()
		if err := dec.Decode(&reqBody); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON body: %w", err)
		}
	} else {
		reqBody = make(map[string]any)
	}

	modelFullName, err := tModelFullName(&apiClient{clientConfig: s.clientConfig}, modelName)
	if err != nil {
		return nil, err
	}
	reqBody["model"] = modelFullName

	var uuidBytes [16]byte
	if _, err := crand.Read(uuidBytes[:]); err != nil {
		return nil, fmt.Errorf("failed to generate secure request id: %w", err)
	}
	reqID := fmt.Sprintf("id-%x", uuidBytes[:])

	var requestTTL any
	if val, ok := reqBody["requestTtl"]; ok {
		requestTTL = val
		delete(reqBody, "requestTtl")
	} else if val, ok := reqBody["request_ttl"]; ok {
		requestTTL = val
		delete(reqBody, "request_ttl")
	}

	payload := map[string]any{
		"generate_content_request": reqBody,
		"request_id":               reqID,
	}
	if requestTTL != nil {
		payload["request_ttl"] = requestTTL
	}

	modifiedJSONBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified JSON body: %w", err)
	}

	respChan := make(chan []byte, 100)
	pending := &pendingRequest{
		ch: respChan,
	}

	session.mu.Lock()
	session.pendingRequests[reqID] = pending
	session.mu.Unlock()

	_, err = session.tlsConn.Write(modifiedJSONBytes)
	if err != nil {
		session.mu.Lock()
		delete(session.pendingRequests, reqID)
		session.mu.Unlock()
		return nil, err
	}

	var timeoutChan <-chan time.Time
	var ttlStr string
	if requestTTL != nil {
		if s, ok := requestTTL.(string); ok && s != "" {
			ttlStr = s
			if d, err := time.ParseDuration(s); err == nil {
				timer := time.NewTimer(d)
				defer timer.Stop()
				timeoutChan = timer.C
			}
		}
	}

	select {
	case respBytes, ok := <-respChan:
		if !ok {
			session.mu.Lock()
			srvErr := session.serverError
			session.mu.Unlock()
			if srvErr != nil {
				return nil, srvErr
			}
			return nil, fmt.Errorf("secure session closed unexpectedly")
		}

		statusCode := 200
		var m map[string]any
		if err := json.Unmarshal(respBytes, &m); err == nil {
			if errVal, hasErr := m["error"]; hasErr {
				statusCode = 400
				if errMap, ok := errVal.(map[string]any); ok {
					if codeVal, ok := errMap["code"]; ok {
						var c int
						if _, err := fmt.Sscanf(fmt.Sprintf("%v", codeVal), "%d", &c); err == nil && c > 0 {
							statusCode = c
						}
					}
				}
			} else {
				var gcResp any
				if val, ok := m["generate_content_response"]; ok {
					gcResp = val
				} else if val, ok := m["generateContentResponse"]; ok {
					gcResp = val
				}
				if gcResp != nil {
					if unwrappedBytes, err := json.Marshal(gcResp); err == nil {
						respBytes = unwrappedBytes
					}
				}
			}
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewReader(respBytes)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    req,
		}, nil
	case <-req.Context().Done():
		session.mu.Lock()
		delete(session.pendingRequests, reqID)
		session.mu.Unlock()
		return nil, req.Context().Err()
	case <-timeoutChan:
		session.mu.Lock()
		delete(session.pendingRequests, reqID)
		session.mu.Unlock()
		if ttlStr != "" {
			return nil, fmt.Errorf("request timed out locally after %s", ttlStr)
		}
		return nil, fmt.Errorf("request timed out locally")
	}
}

func (s *secureSessionState) readLoop() {
	defer s.close()
	decoder := json.NewDecoder(s.tlsConn)
	decoder.UseNumber()
	for {
		var m map[string]any
		if err := decoder.Decode(&m); err != nil {
			return
		}
		reqIDAny, ok := m["request_id"]
		if !ok {
			continue
		}
		reqID, ok := reqIDAny.(string)
		if !ok {
			continue
		}

		s.mu.Lock()
		req, exists := s.pendingRequests[reqID]
		if exists {
			delete(s.pendingRequests, reqID)
			if !s.closed {
				respBytes, _ := json.Marshal(m)
				req.ch <- respBytes
			}
		}
		s.mu.Unlock()
	}
}
