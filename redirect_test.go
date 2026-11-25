// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/httptransport"
)

// TestAPIKeyNotForwardedOnCrossDomainRedirect checks where the API key goes
// when the SDK's default http.Client follows a redirect.
//
// Both test servers listen on the loopback interface, but server A is
// addressed as "localhost" and server B as "127.0.0.1". net/http decides
// whether a redirect leaves the original domain by comparing hostnames, so
// that hop is cross-domain, whereas two httptest servers addressed by their
// URLs are both 127.0.0.1 and count as the same host. The control request,
// sent through a client with net/http's default policy, shows that net/http
// forwards x-goog-api-key on that hop by itself.
func TestAPIKeyNotForwardedOnCrossDomainRedirect(t *testing.T) {
	const key = "secret"

	var mu sync.Mutex
	got := map[string]string{} // final path -> x-goog-api-key received there
	record := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got[r.URL.Path] = r.Header.Get("x-goog-api-key")
	}
	received := func(path string) string {
		mu.Lock()
		defer mu.Unlock()
		return got[path]
	}

	var addrA string // server A, addressed as localhost
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bounce" {
			http.Redirect(w, r, addrA+"/back", http.StatusFound)
			return
		}
		record(w, r)
	}))
	defer serverB.Close()
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/same":
			http.Redirect(w, r, addrA+"/stayed", http.StatusFound)
		case "/cross":
			http.Redirect(w, r, serverB.URL+"/landed", http.StatusFound)
		case "/chain":
			http.Redirect(w, r, serverB.URL+"/bounce", http.StatusFound)
		default:
			record(w, r)
		}
	}))
	defer serverA.Close()
	u, err := url.Parse(serverA.URL)
	if err != nil {
		t.Fatal(err)
	}
	addrA = "http://localhost:" + u.Port()

	// get sends the request the way api_client.go does: the key is a header
	// on the request handed to the client.
	get := func(client *http.Client, path string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, addrA+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("x-goog-api-key", key)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
	}

	// Control: with net/http's default policy the key crosses the domain,
	// which is the behaviour the SDK's policy exists to prevent.
	get(&http.Client{}, "/cross")
	if k := received("/landed"); k != key {
		t.Fatalf("control: net/http forwarded x-goog-api-key %q, want %q; the hop is not cross-domain on this machine", k, key)
	}
	mu.Lock()
	got = map[string]string{}
	mu.Unlock()

	client := &http.Client{CheckRedirect: stripCredentialsOnCrossDomainRedirect}
	tests := []struct {
		path, final, want string
	}{
		// A redirect within the domain keeps the key.
		{"/same", "/stayed", key},
		// A redirect to another domain gets no key.
		{"/cross", "/landed", ""},
		// A chain that left the domain gets no key even where it comes back.
		{"/chain", "/back", ""},
	}
	for _, tt := range tests {
		get(client, tt.path)
		if k := received(tt.final); k != tt.want {
			t.Errorf("GET %s: the redirect target received x-goog-api-key %q, want %q", tt.path, k, tt.want)
		}
	}
}

func TestNewClientInstallsRedirectPolicy(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, APIKey: "secret"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.clientConfig.HTTPClient.CheckRedirect == nil {
		t.Error("the default HTTP client has no CheckRedirect; a cross-domain redirect would forward the API key")
	}

	// A client the caller supplies is used as given.
	supplied := &http.Client{}
	client, err = NewClient(ctx, &ClientConfig{Backend: BackendGeminiAPI, APIKey: "secret", HTTPClient: supplied})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.clientConfig.HTTPClient != supplied || supplied.CheckRedirect != nil {
		t.Error("a supplied HTTP client must be used as given")
	}
}

func TestStripCredentialsOnCrossDomainRedirectLimit(t *testing.T) {
	first, _ := url.Parse("https://example.com/")
	via := make([]*http.Request, maxRedirects)
	for i := range via {
		via[i] = &http.Request{URL: first}
	}
	req := &http.Request{URL: first, Header: http.Header{}}
	if err := stripCredentialsOnCrossDomainRedirect(req, via); err == nil {
		t.Error("the policy followed more redirects than net/http's default would")
	}
}

// TestVertexClientStopsAtCrossDomainRedirect covers the client that NewClient
// builds through httptransport.NewClient for Vertex AI credentials. Its
// transport sets Authorization from the credentials on every request it
// sends, including a redirected one, so deleting the header in CheckRedirect
// does not keep the access token off another domain; the control request
// shows the token reaching server B through such a client with net/http's
// default policy. The SDK's client does not follow that redirect at all.
func TestVertexClientStopsAtCrossDomainRedirect(t *testing.T) {
	const token = "ACCESS_TOKEN_MUST_NOT_LEAK"
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	var mu sync.Mutex
	got := map[string]string{} // path -> Authorization received there
	record := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got[r.URL.Path] = r.Header.Get("Authorization")
	}
	received := func(path string) (string, bool) {
		mu.Lock()
		defer mu.Unlock()
		v, ok := got[path]
		return v, ok
	}

	var addrA string // server A, addressed as localhost
	serverB := httptest.NewServer(http.HandlerFunc(record))
	defer serverB.Close()
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/same":
			http.Redirect(w, r, addrA+"/stayed", http.StatusFound)
		case "/cross", "/control":
			http.Redirect(w, r, serverB.URL+"/landed"+r.URL.Path, http.StatusFound)
		default:
			record(w, r)
		}
	}))
	defer serverA.Close()
	u, err := url.Parse(serverA.URL)
	if err != nil {
		t.Fatal(err)
	}
	addrA = "http://localhost:" + u.Port()

	creds := auth.NewCredentials(&auth.CredentialsOptions{
		TokenProvider: &staticTokenProvider{token: &auth.Token{Value: token}},
	})
	get := func(client *http.Client, path string) *http.Response {
		t.Helper()
		resp, err := client.Get(addrA + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		return resp
	}

	// Control: the transport signs the redirected request too, so with
	// net/http's default policy the token crosses the domain.
	control, err := httptransport.NewClient(&httptransport.Options{Credentials: creds})
	if err != nil {
		t.Fatal(err)
	}
	get(control, "/control")
	if a, _ := received("/landed/control"); a != "Bearer "+token {
		t.Fatalf("control: the redirect target received Authorization %q, want the bearer token; the hop is not cross-domain on this machine", a)
	}

	client, err := NewClient(context.Background(), &ClientConfig{
		Backend:     BackendVertexAI,
		Project:     "project",
		Location:    "us-central1",
		Credentials: creds,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	hc := client.clientConfig.HTTPClient

	// A redirect within the domain is followed, token and all.
	if resp := get(hc, "/same"); resp.StatusCode != http.StatusOK {
		t.Errorf("GET /same: status %d, want 200; a redirect within the domain must be followed", resp.StatusCode)
	}
	if a, _ := received("/stayed"); a != "Bearer "+token {
		t.Errorf("GET /same: the redirect target received Authorization %q, want the bearer token", a)
	}

	// A redirect to another domain is not followed: the caller gets the
	// redirect response and server B gets no request.
	if resp := get(hc, "/cross"); resp.StatusCode != http.StatusFound {
		t.Errorf("GET /cross: status %d, want 302; a redirect to another domain must not be followed", resp.StatusCode)
	}
	if a, ok := received("/landed/cross"); ok {
		t.Errorf("GET /cross: the other domain was reached, with Authorization %q", a)
	}
}
