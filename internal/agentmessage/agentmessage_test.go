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

package agentmessage

import (
	"bytes"
	"log"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   []string
	}{
		{
			name:   "single structured item",
			header: `field_ignored;field="temperature";msg=%"temperature is ignored by Gemini 3.6+ models and will be removed in a future release."`,
			want:   []string{"temperature is ignored by Gemini 3.6+ models and will be removed in a future release."},
		},
		{
			name:   "multiple comma separated items",
			header: `field_ignored;field="temperature";msg=%"temperature is ignored, using default.", field_ignored;field="top_k";msg=%"top_k is ignored; see docs."`,
			want:   []string{"temperature is ignored, using default.", "top_k is ignored; see docs."},
		},
		{
			name:   "item without field and with unknown params",
			header: `legacy_usage;severity="warn";experimental;msg=%"generateContent is a legacy endpoint. Use the Interactions API for new development."`,
			want:   []string{"generateContent is a legacy endpoint. Use the Interactions API for new development."},
		},
		{
			name:   "percent encoded UTF-8 and quoted string",
			header: `field_ignored;msg=%"use %22max_output_tokens%22 %E2%80%94 100%25", other_code;msg="plain \"quoted\" \\ string"`,
			want:   []string{"use \"max_output_tokens\" \u2014 100%", `plain "quoted" \ string`},
		},
		{
			name:   "skips malformed sibling item",
			header: `field_ignored;field="temperature", field_ignored;field="top_k";msg=%"top_k is ignored."`,
			want:   []string{"top_k is ignored."},
		},
		{
			name:   "plain string fallback",
			header: "generation_config.temperature is ignored, using default value.",
			want:   []string{"generation_config.temperature is ignored, using default value."},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseHeader(tc.header); !slices.Equal(got, tc.want) {
				t.Errorf("parseHeader(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestParseHeaderMalformedIgnored(t *testing.T) {
	cases := []string{
		"",
		"   ",
		";;;",
		"field_ignored",
		`field_ignored;field="temperature"`,
		`field_ignored;msg=unquoted`,
		`field_ignored;msg=%"unterminated`,
		`bad code;msg=%"not a token"`,
	}
	for _, tc := range cases {
		if got := parseHeader(tc); len(got) != 0 {
			t.Errorf("parseHeader(%q) = %q, want empty", tc, got)
		}
	}
}

func TestWarnLogsOncePerMessage(t *testing.T) {
	seen.Clear()
	t.Cleanup(seen.Clear)

	var buf bytes.Buffer
	origOut, origFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(origOut)
		log.SetFlags(origFlags)
	})

	header1 := `field_ignored;field="temperature";msg=%"temperature is ignored by Gemini 3.6+ models."`
	header2 := `field_ignored;field="top_k";msg=%"top_k is ignored by Gemini 3.6+ models."`

	h := http.Header{}
	h.Add("agent-message", header1) // Add canonicalizes the key.
	Warn(h)
	Warn(http.Header{"Agent-Message": {header1, header2}})
	Warn(nil)

	got := strings.Split(strings.TrimSpace(buf.String()), "\n")
	want := []string{
		"Warning: [Gemini API] temperature is ignored by Gemini 3.6+ models.",
		"Warning: [Gemini API] top_k is ignored by Gemini 3.6+ models.",
	}
	if !slices.Equal(got, want) {
		t.Errorf("logged warnings = %q, want %q", got, want)
	}
}
