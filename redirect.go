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

// stripCredentialsOnCrossDomainRedirect is the redirect policy of the
// http.Client that NewClient builds when ClientConfig.HTTPClient is nil.
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
	domain := strings.ToLower(via[0].URL.Hostname())
	cross := !isDomainOrSubdomain(strings.ToLower(req.URL.Hostname()), domain)
	for _, r := range via[1:] {
		if cross {
			break
		}
		cross = !isDomainOrSubdomain(strings.ToLower(r.URL.Hostname()), domain)
	}
	if cross {
		req.Header.Del("x-goog-api-key")
	}
	return nil
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
