// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package agentmessage surfaces server-sent Agent-Message response headers
// (RFC 9651 structured-field lists) as client-side warnings.
//
// It is shared by the GenerateContent client and the Interactions client so
// that each message is logged at most once per process across both.
package agentmessage

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

const headerName = "Agent-Message"

var (
	tokenRe = regexp.MustCompile(`^[A-Za-z*][A-Za-z0-9!#$%&'*+.^_` + "`" + `|~:/-]*$`)
	seen    sync.Map // map[string]struct{}
)

// Warn logs each message in the Agent-Message headers of h, at most once per
// process per distinct message. Malformed headers are ignored; Warn never
// panics.
func Warn(h http.Header) {
	values := h.Values(headerName)
	if len(values) == 0 {
		return
	}
	defer func() { _ = recover() }()
	for _, v := range values {
		for _, msg := range parseHeader(v) {
			if _, loaded := seen.LoadOrStore(msg, struct{}{}); !loaded {
				log.Printf("Warning: [Gemini API] %s", msg)
			}
		}
	}
}

func splitUnquoted(text string, delimiter byte) ([]string, bool) {
	var parts []string
	start := 0
	inQuotes := false
	isDisplayString := false
	escaped := false

	for i := 0; i < len(text); i++ {
		ch := text[i]
		if inQuotes {
			if !isDisplayString {
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
			}
			if ch == '"' {
				inQuotes = false
				isDisplayString = false
			}
		} else {
			if ch == '"' {
				inQuotes = true
				isDisplayString = i > 0 && text[i-1] == '%'
			} else if ch == delimiter {
				parts = append(parts, text[start:i])
				start = i + 1
			}
		}
	}

	if inQuotes {
		return nil, false
	}
	parts = append(parts, text[start:])
	return parts, true
}

func unescapeQuotedString(inner string) (string, bool) {
	var out strings.Builder
	escaped := false
	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if escaped {
			if ch != '"' && ch != '\\' {
				return "", false
			}
			out.WriteByte(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == '"' {
			return "", false
		} else {
			out.WriteByte(ch)
		}
	}
	if escaped {
		return "", false
	}
	return out.String(), true
}

func hexByte(b byte) (byte, bool) {
	switch {
	case '0' <= b && b <= '9':
		return b - '0', true
	case 'a' <= b && b <= 'f':
		return b - 'a' + 10, true
	case 'A' <= b && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}

func percentDecodeDisplayString(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			hi, ok1 := hexByte(s[i+1])
			lo, ok2 := hexByte(s[i+2])
			if ok1 && ok2 {
				out = append(out, (hi<<4)|lo)
				i += 2
				continue
			}
		}
		out = append(out, s[i])
	}
	return strings.ToValidUTF8(string(out), "\uFFFD")
}

func parseItem(rawItem string) (string, bool) {
	item := strings.TrimSpace(rawItem)
	if item == "" {
		return "", false
	}

	parts, ok := splitUnquoted(item, ';')
	if !ok || len(parts) < 2 {
		return "", false
	}

	code := strings.TrimSpace(parts[0])
	if !tokenRe.MatchString(code) {
		return "", false
	}

	var msgValue string
	hasMsg := false
	for _, rawParam := range parts[1:] {
		param := strings.TrimSpace(rawParam)
		if param == "" {
			return "", false
		}
		key, val, foundEq := strings.Cut(param, "=")
		if !foundEq {
			if !tokenRe.MatchString(param) {
				return "", false
			}
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if !tokenRe.MatchString(key) || val == "" {
			return "", false
		}

		if key == "msg" {
			if strings.HasPrefix(val, `%"`) && strings.HasSuffix(val, `"`) && len(val) >= 3 {
				inner := val[2 : len(val)-1]
				if strings.Contains(inner, `"`) {
					return "", false
				}
				msgValue = percentDecodeDisplayString(inner)
				hasMsg = true
			} else if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) && len(val) >= 2 {
				unescaped, ok := unescapeQuotedString(val[1 : len(val)-1])
				if !ok {
					return "", false
				}
				msgValue = unescaped
				hasMsg = true
			} else {
				return "", false
			}
		}
	}

	if !hasMsg || msgValue == "" {
		return "", false
	}
	return msgValue, true
}

func parseHeader(headerValue string) []string {
	stripped := strings.TrimSpace(headerValue)
	if stripped == "" {
		return nil
	}

	// Plain-text fallback for headers that aren't RFC 9651 structured items.
	if !strings.Contains(stripped, ";") && !strings.Contains(stripped, "=") && strings.Contains(stripped, " ") {
		if !strings.HasPrefix(stripped, "\"") && !strings.HasSuffix(stripped, "\"") {
			return []string{stripped}
		}
	}

	items, ok := splitUnquoted(stripped, ',')
	if !ok {
		return nil
	}

	var messages []string
	for _, rawItem := range items {
		if msg, ok := parseItem(rawItem); ok && msg != "" {
			messages = append(messages, msg)
		}
	}
	return messages
}
