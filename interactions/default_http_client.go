// Copyright 2025 Google LLC
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

// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package interactions

import (
	"net/http"
	"time"
)

// defaultResponseHeaderTimeout bounds the time between a fully written request
// and the server's response headers. It does not apply to the response body,
// so long-running streams are unaffected. Without this, a server that accepts
// the connection but never responds would hang the request indefinitely.
const defaultResponseHeaderTimeout = 10 * time.Minute

// defaultHTTPClient returns an [*http.Client] used when the caller does not
// supply one via [option.WithHTTPClient]. When [http.DefaultTransport] is the
// stdlib [*http.Transport], it is cloned and a [http.Transport.ResponseHeaderTimeout]
// is set so stuck connections fail fast instead of compounding across retries.
// If [http.DefaultTransport] has been wrapped (for example by otelhttp for
// distributed tracing), the wrapping is preserved and the header timeout is
// skipped.
func defaultHTTPClient() *http.Client {
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t = t.Clone()
		t.ResponseHeaderTimeout = defaultResponseHeaderTimeout
		return &http.Client{Transport: t}
	}
	return &http.Client{Transport: http.DefaultTransport}
}
