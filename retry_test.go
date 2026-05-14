package genai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fastRetry(attempts int) *HTTPRetryOptions {
	return &HTTPRetryOptions{
		Attempts:     attempts,
		InitialDelay: time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		ExpBase:      2,
		Jitter:       time.Millisecond,
	}
}

func TestResolvedRetryOptions(t *testing.T) {
	tests := []struct {
		desc          string
		in            *HTTPRetryOptions
		wantNil       bool
		wantAttempts  int
		wantInitial   time.Duration
		wantMaxDelay  time.Duration
		wantExpBase   float64
		wantJitter    time.Duration
		wantCodesLen  int
	}{
		{
			desc:    "nil returns nil",
			in:      nil,
			wantNil: true,
		},
		{
			desc:    "attempts=1 disables retry",
			in:      &HTTPRetryOptions{Attempts: 1},
			wantNil: true,
		},
		{
			desc:         "defaults applied for empty options",
			in:           &HTTPRetryOptions{},
			wantAttempts: defaultRetryAttempts,
			wantInitial:  defaultRetryInitialDelay,
			wantMaxDelay: defaultRetryMaxDelay,
			wantExpBase:  defaultRetryExpBase,
			wantJitter:   defaultRetryJitter,
			wantCodesLen: len(defaultRetryHTTPStatusCodes),
		},
		{
			desc: "user values preserved",
			in: &HTTPRetryOptions{
				Attempts:        7,
				InitialDelay:    2 * time.Second,
				MaxDelay:        30 * time.Second,
				ExpBase:         3,
				Jitter:          500 * time.Millisecond,
				HTTPStatusCodes: []int{500, 503},
			},
			wantAttempts: 7,
			wantInitial:  2 * time.Second,
			wantMaxDelay: 30 * time.Second,
			wantExpBase:  3,
			wantJitter:   500 * time.Millisecond,
			wantCodesLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := resolvedRetryOptions(tt.in)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("resolvedRetryOptions() = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("resolvedRetryOptions() = nil, want non-nil")
			}
			if got.Attempts != tt.wantAttempts || got.InitialDelay != tt.wantInitial ||
				got.MaxDelay != tt.wantMaxDelay || got.ExpBase != tt.wantExpBase ||
				got.Jitter != tt.wantJitter || len(got.HTTPStatusCodes) != tt.wantCodesLen {
				t.Errorf("resolvedRetryOptions() = %#v", got)
			}
		})
	}
}

func TestBackoffDelay(t *testing.T) {
	tests := []struct {
		desc string
		opts *HTTPRetryOptions
		n    int
		want time.Duration
	}{
		{
			desc: "first retry uses initial delay",
			opts: &HTTPRetryOptions{InitialDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second, ExpBase: 2},
			n:    1,
			want: 100 * time.Millisecond,
		},
		{
			desc: "second retry doubles",
			opts: &HTTPRetryOptions{InitialDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second, ExpBase: 2},
			n:    2,
			want: 200 * time.Millisecond,
		},
		{
			desc: "third retry doubles again",
			opts: &HTTPRetryOptions{InitialDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second, ExpBase: 2},
			n:    3,
			want: 400 * time.Millisecond,
		},
		{
			desc: "capped by max delay",
			opts: &HTTPRetryOptions{InitialDelay: time.Second, MaxDelay: 2 * time.Second, ExpBase: 10},
			n:    5,
			want: 2 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := backoffDelay(tt.opts, tt.n); got != tt.want {
				t.Errorf("backoffDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoRequestWithRetry(t *testing.T) {
	tests := []struct {
		desc          string
		retryOptions  *HTTPRetryOptions
		serverHandler func(calls *int32) http.HandlerFunc
		wantCalls     int32
		wantErr       bool
		wantStatus    int // expected APIError.Code when wantErr is true
	}{
		{
			desc:         "no retry options, single attempt on 5xx",
			retryOptions: nil,
			serverHandler: func(calls *int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(calls, 1)
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			},
			wantCalls:  1,
			wantErr:    true,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			desc:         "retries until success",
			retryOptions: fastRetry(5),
			serverHandler: func(calls *int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if atomic.AddInt32(calls, 1) < 3 {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"ok":true}`))
				}
			},
			wantCalls: 3,
			wantErr:   false,
		},
		{
			desc:         "exhausts attempts on persistent retriable status",
			retryOptions: fastRetry(3),
			serverHandler: func(calls *int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(calls, 1)
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte(`{"error":{"code":502,"message":"bad gateway"}}`))
				}
			},
			wantCalls:  3,
			wantErr:    true,
			wantStatus: http.StatusBadGateway,
		},
		{
			desc:         "non-retriable status returns immediately",
			retryOptions: fastRetry(5),
			serverHandler: func(calls *int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(calls, 1)
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"error":{"code":400,"message":"bad"}}`))
				}
			},
			wantCalls:  1,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			desc: "custom status codes trigger retry",
			retryOptions: func() *HTTPRetryOptions {
				o := fastRetry(4)
				o.HTTPStatusCodes = []int{http.StatusTeapot}
				return o
			}(),
			serverHandler: func(calls *int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if atomic.AddInt32(calls, 1) < 2 {
						w.WriteHeader(http.StatusTeapot)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"ok":true}`))
				}
			},
			wantCalls: 2,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var calls int32
			ts := httptest.NewServer(tt.serverHandler(&calls))
			defer ts.Close()

			ac := &apiClient{
				clientConfig: &ClientConfig{
					HTTPOptions: HTTPOptions{BaseURL: ts.URL},
					HTTPClient:  ts.Client(),
				},
			}

			_, err := sendRequest(context.Background(), ac, "foo", http.MethodPost,
				map[string]any{"k": "v"},
				&HTTPOptions{BaseURL: ts.URL, RetryOptions: tt.retryOptions})

			if (err != nil) != tt.wantErr {
				t.Fatalf("sendRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				apiErr, ok := err.(APIError)
				if !ok {
					t.Fatalf("want APIError, got %T: %v", err, err)
				}
				if apiErr.Code != tt.wantStatus {
					t.Errorf("APIError.Code = %d, want %d", apiErr.Code, tt.wantStatus)
				}
			}
			if got := atomic.LoadInt32(&calls); got != tt.wantCalls {
				t.Errorf("calls = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestDoRequestWithRetry_ContextCancellation(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	ac := &apiClient{
		clientConfig: &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: ts.URL},
			HTTPClient:  ts.Client(),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	opts := &HTTPRetryOptions{
		Attempts:     10,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     time.Second,
		ExpBase:      2,
	}
	_, err := sendRequest(ctx, ac, "foo", http.MethodPost,
		map[string]any{"k": "v"},
		&HTTPOptions{BaseURL: ts.URL, RetryOptions: opts})
	if err == nil {
		t.Fatal("sendRequest() returned nil error, want cancellation error")
	}
	if got := atomic.LoadInt32(&calls); got > 2 {
		t.Errorf("calls = %d, want at most 2 before cancel", got)
	}
}

func TestDoRequestWithRetry_TransportError(t *testing.T) {
	// Stand up a server then close it so connections fail.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL
	ts.Close()

	ac := &apiClient{
		clientConfig: &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: url},
			HTTPClient:  &http.Client{Timeout: 50 * time.Millisecond},
		},
	}
	_, err := sendRequest(context.Background(), ac, "foo", http.MethodPost,
		map[string]any{"k": "v"},
		&HTTPOptions{BaseURL: url, RetryOptions: fastRetry(3)})
	if err == nil {
		t.Fatal("sendRequest() returned nil error, want transport error")
	}
	if !strings.Contains(err.Error(), "doRequest") {
		t.Errorf("error = %q, want doRequest-wrapped error", err.Error())
	}
}

func TestBuildRequest_BodyIsRewindable(t *testing.T) {
	ac := &apiClient{
		clientConfig: &ClientConfig{
			HTTPOptions: HTTPOptions{BaseURL: "http://example.com"},
			HTTPClient:  &http.Client{},
		},
	}
	req, _, err := buildRequest(context.Background(), ac, "foo",
		map[string]any{"k": "v"}, http.MethodPost,
		&HTTPOptions{BaseURL: "http://example.com"})
	if err != nil {
		t.Fatalf("buildRequest() error = %v", err)
	}
	if req.GetBody == nil {
		t.Fatal("req.GetBody = nil, want non-nil so retry can rewind body")
	}
	body, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody() error = %v", err)
	}
	defer body.Close()
	buf := make([]byte, 64)
	n, _ := body.Read(buf)
	if !strings.Contains(string(buf[:n]), `"k"`) {
		t.Errorf("rewound body = %q, want it to contain key", string(buf[:n]))
	}
}
