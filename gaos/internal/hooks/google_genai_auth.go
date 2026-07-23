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

package hooks

import (
	"net/http"
	"strings"

	"google.golang.org/genai/gaos/models/components"
)

const (
	googleGenAIApiRevision = "2026-05-20"
)

type GoogleGenAIAuthHook struct{}

var _ beforeRequestHook = (*GoogleGenAIAuthHook)(nil)

func (h *GoogleGenAIAuthHook) BeforeRequest(hookCtx BeforeRequestContext, req *http.Request) (*http.Request, error) {
	if hookCtx.SecuritySource == nil {
		return req, nil
	}

	securityObj, err := hookCtx.SecuritySource(hookCtx.Context)
	if err != nil {
		return req, err
	}

	security, ok := securityObj.(*components.Security)
	if !ok {
		// Try without pointer just in case
		if s, ok := securityObj.(components.Security); ok {
			security = &s
		}
	}

	if security == nil {
		return req, nil
	}

	// Apply default headers
	for key, value := range security.DefaultHeaders {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}

	if req.Header.Get("Api-Revision") == "" {
		req.Header.Set("Api-Revision", googleGenAIApiRevision)
	}

	// Apply auth if not already present
	if req.Header.Get("Authorization") == "" && req.Header.Get("x-goog-api-key") == "" {
		if security.APIKey != nil && *security.APIKey != "" {
			req.Header.Set("x-goog-api-key", *security.APIKey)
		} else if security.AccessToken != nil && *security.AccessToken != "" {
			req.Header.Set("Authorization", bearer(*security.AccessToken))
		}
	}

	return req, nil
}

func bearer(token string) string {
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}
