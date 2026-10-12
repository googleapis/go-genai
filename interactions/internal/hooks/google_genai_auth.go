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
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"google.golang.org/genai/interactions/models/components"
)

const (
	googleGenAIApiRevision = "2026-05-20"
)

var (
	loggedTemperatureWarning sync.Once
	loggedTopPWarning        sync.Once
	loggedTopKWarning        sync.Once
)

type GoogleGenAIAuthHook struct{}

var _ beforeRequestHook = (*GoogleGenAIAuthHook)(nil)

func (h *GoogleGenAIAuthHook) BeforeRequest(hookCtx BeforeRequestContext, req *http.Request) (*http.Request, error) {
	warnUnsupportedInteractionParameters(hookCtx, req)

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

func isGemini36OrNewer(model string) bool {
	if model == "" {
		return false
	}
	modelID := strings.ToLower(model)
	if idx := strings.LastIndexByte(modelID, '/'); idx >= 0 {
		modelID = modelID[idx+1:]
	}
	if !strings.HasPrefix(modelID, "gemini-") {
		return false
	}
	rest := modelID[len("gemini-"):]
	major, n, ok := parseLeadingUint(rest)
	if !ok {
		return false
	}
	rest = rest[n:]
	minor := 0
	if strings.HasPrefix(rest, ".") {
		var okMinor bool
		minor, n, okMinor = parseLeadingUint(rest[1:])
		if !okMinor {
			return false
		}
		rest = rest[1+n:]
	}
	if rest != "" && !strings.HasPrefix(rest, "-") {
		return false
	}
	return major > 3 || (major == 3 && minor >= 6)
}

func parseLeadingUint(s string) (int, int, bool) {
	i := 0
	val := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		val = val*10 + int(s[i]-'0')
		i++
	}
	return val, i, i > 0
}

func warnUnsupportedInteractionParameters(hookCtx BeforeRequestContext, req *http.Request) {
	if hookCtx.OperationID != "CreateInteraction" || req == nil || req.GetBody == nil {
		return
	}
	rc, err := req.GetBody()
	if err != nil {
		return
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return
	}
	var payload struct {
		Model            string          `json:"model"`
		GenerationConfig json.RawMessage `json:"generation_config"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || !isGemini36OrNewer(payload.Model) || len(payload.GenerationConfig) == 0 {
		return
	}
	var genConfig map[string]any
	if err := json.Unmarshal(payload.GenerationConfig, &genConfig); err != nil {
		return
	}
	if v, ok := genConfig["temperature"]; ok && v != nil {
		loggedTemperatureWarning.Do(func() {
			log.Println("WARNING: temperature is not supported on Gemini 3.6 and newer models and is ignored.")
		})
	}
	if v, ok := genConfig["top_p"]; ok && v != nil {
		loggedTopPWarning.Do(func() {
			log.Println("WARNING: top_p is not supported on Gemini 3.6 and newer models and is ignored.")
		})
	}
	if v, ok := genConfig["top_k"]; ok && v != nil {
		loggedTopKWarning.Do(func() {
			log.Println("WARNING: top_k is not supported on Gemini 3.6 and newer models and is ignored.")
		})
	}
}

func bearer(token string) string {
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}
