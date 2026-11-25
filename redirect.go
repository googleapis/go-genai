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
	"errors"
	"net/http"
	"strings"
)

// maxRedirects is net/http's default limit, which a custom CheckRedirect
// replaces rather than extends.
const maxRedirects = 10

// stripCredentialsOnCrossDomainRedirect is the redirect policy of the plain
// http.Client that NewClient builds when ClientConfig.HTTPClient is nil and
// the request carries an API key.
//
// It follows up to ten redirects, as net/http's default policy does, but
// removes the x-goog-api-key header from a redirect that leaves the domain
// of the original request (or a subdomain of it). net/http drops the
// Authorization header on such a redirect and forwards every other header as
// it is, so without this policy a redirect from the configured BaseURL would
// hand the API key to whatever host the Location header named. Once a chain
// has left the original domain the header stays off, even on a hop that
// returns to it, which is also how net/http treats Authorization.
//
// A client supplied through ClientConfig.HTTPClient keeps its own redirect
// policy; set this function as its CheckRedirect to get the same protection.
func stripCredentialsOnCrossDomainRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if leftDomain(req, via) {
		req.Header.Del("x-goog-api-key")
	}
	return nil
}

// stopAtCrossDomainRedirect is the redirect policy of the http.Client that
// NewClient builds through httptransport.NewClient for Vertex AI credentials.
//
// That client's transport sets the Authorization header from the credentials
// inside RoundTrip, on every request it sends, so a header deleted here would
// be put back on the redirected request and the access token would still
// reach the other domain. The policy therefore does not follow a redirect
// that leaves the domain of the original request (or a subdomain of it): it
// returns http.ErrUseLastResponse, and the caller gets the redirect response
// itself, which the SDK reports as an APIError. Redirects within the domain
// are followed, up to net/http's limit of ten.
func stopAtCrossDomainRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if leftDomain(req, via) {
		return http.ErrUseLastResponse
	}
	return nil
}

// leftDomain reports whether req, or any redirect before it, is outside the
// domain of the original request (or a subdomain of it). The original request
// is the reference, not the previous hop, so a chain that leaves the domain
// and comes back still counts as having left it.
func leftDomain(req *http.Request, via []*http.Request) bool {
	domain := strings.ToLower(via[0].URL.Hostname())
	if !isDomainOrSubdomain(strings.ToLower(req.URL.Hostname()), domain) {
		return true
	}
	for _, r := range via[1:] {
		if !isDomainOrSubdomain(strings.ToLower(r.URL.Hostname()), domain) {
			return true
		}
	}
	return false
}

// isDomainOrSubdomain reports whether sub is a subdomain (or exact match) of
// the parent domain. It is the rule net/http applies when deciding whether a
// redirect may keep the Authorization header, copied from net/http/client.go.
// Both domains must already be in canonical form.
func isDomainOrSubdomain(sub, parent string) bool {
	if sub == parent {
		return true
	}
	// If sub contains a :, it's probably an IPv6 address (and is definitely not a hostname).
	// Don't check the suffix in this case, to avoid matching the contents of a IPv6 zone.
	// For example, "::1%.www.example.com" is not a subdomain of "www.example.com".
	if strings.ContainsAny(sub, ":%") {
		return false
	}
	// If sub is "foo.example.com" and parent is "example.com",
	// that means sub must end in "."+parent.
	// Do it without allocating.
	if !strings.HasSuffix(sub, parent) {
		return false
	}
	return sub[len(sub)-len(parent)-1] == '.'
}
