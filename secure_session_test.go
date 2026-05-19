package genai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWsTLSConnFraming(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			var req map[string]any
			if err := json.Unmarshal(message, &req); err == nil {
				// Echo back the same tls_frame
				_ = c.WriteMessage(mt, message)
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	wt := &wsTLSConn{wsConn: c}

	testData := []byte("hello tls tunnel")
	n, err := wt.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Fatalf("Expected %d bytes written, got %d", len(testData), n)
	}

	buf := make([]byte, 1024)
	n, err = wt.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(buf[:n]) != string(testData) {
		t.Fatalf("Expected %q, got %q", testData, buf[:n])
	}
}

func TestSecureRoundTripperBypass(t *testing.T) {
	cc := &ClientConfig{}
	base := http.DefaultTransport
	rt := NewSecureRoundTripper(base, cc)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("bypassed"))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "bypassed" {
		t.Fatalf("Expected 'bypassed', got %q", string(body))
	}
}

func TestStartSecureSessionInvalidRootCA(t *testing.T) {
	cc := &ClientConfig{
		Backend: BackendVertexAI,
		HTTPOptions: HTTPOptions{
			BaseURL: "https://example.com",
		},
	}
	ac := &apiClient{clientConfig: cc}
	rt := NewSecureRoundTripper(http.DefaultTransport, cc)

	err := rt.StartSecureSession(context.Background(), ac, "test-model", "ca-pool", "/path/to/nonexistent/rootCA.crt")
	if err == nil {
		t.Fatalf("Expected error for nonexistent root CA, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read root CA file") && !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("Expected error about missing file, got %v", err)
	}
}

func TestExtractModelFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/v1/models/gemini-pro:generateContent", "gemini-pro"},
		{"/v1beta1/projects/foo/locations/us-central1/publishers/google/models/gemini-2.5-pro-pie:streamGenerateContent", "gemini-2.5-pro-pie"},
		{"/no-model-here", ""},
	}

	for _, tt := range tests {
		actual := extractModelFromPath(tt.path)
		if actual != tt.expected {
			t.Errorf("extractModelFromPath(%q) = %q, want %q", tt.path, actual, tt.expected)
		}
	}
}
