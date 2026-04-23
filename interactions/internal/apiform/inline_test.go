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

package apiform_test

import (
	"testing"

	"google.golang.org/genai/interactions/packages/apidata"
)

func TestInline(t *testing.T) {
	type Inner struct {
		X string `form:"x"`
		Y int    `form:"y"`
	}
	type InlineStruct struct {
		A     string `form:"a"`
		Inner Inner  `form:",inline"`
	}
	type InlinePtr struct {
		A     string `form:"a"`
		Inner *Inner `form:",inline"`
	}
	type InlineOmitzero struct {
		A     string `form:"a"`
		Inner Inner  `form:",inline,omitzero"`
	}
	type InlineMapAndRaw struct {
		Name   string         `form:"name"`
		Extras map[string]any `form:",inline"`
		apidata.DynamicFields
	}
	type EmbedInner struct {
		A      bool           `form:"a"`
		Extras map[string]any `form:",inline"`
	}
	type EmbedOuter struct {
		B string `form:"b"`
		EmbedInner
	}

	tests := []struct {
		name   string
		val    any
		has    map[string]string
		lacks  []string
		sameAs string // if set, output must match the test with this name
	}{
		// ── Inline map ──────────────────────────────────────────────
		{
			name: "inline map spreads entries as sibling fields",
			val:  withInlineMap{Name: "alice", Extra: map[string]any{"foo": true, "bar": "value"}},
			has:  map[string]string{"name": "alice", "foo": "true", "bar": "value"},
		},
		{
			name: "typed inline map",
			val: struct {
				A      bool           `form:"a"`
				Extras map[string]int `form:",inline"`
			}{A: true, Extras: map[string]int{"count": 42}},
			has: map[string]string{"a": "true", "count": "42"},
		},
		{
			name: "empty inline map — baseline",
			val:  withInlineMap{Name: "a"},
		},
		{
			name:   "nil inline map matches baseline",
			val:    withInlineMap{Name: "a", Extra: nil},
			sameAs: "empty inline map — baseline",
		},
		{
			name:   "empty inline map matches baseline",
			val:    withInlineMap{Name: "a", Extra: map[string]any{}},
			sameAs: "empty inline map — baseline",
		},

		// ── Inline struct ───────────────────────────────────────────
		{
			name:  "inline struct spreads fields into parent",
			val:   InlineStruct{A: "hello", Inner: Inner{X: "val", Y: 42}},
			has:   map[string]string{"a": "hello", "x": "val", "y": "42"},
			lacks: []string{"inner"},
		},
		{
			name:  "inline struct spreads fields into parent",
			val:   InlineStruct{A: "hello", Inner: Inner{X: "", Y: 0}},
			has:   map[string]string{"a": "hello", "x": "", "y": "0"},
			lacks: []string{"inner"},
		},
		{
			name:  "inline pointer to struct",
			val:   InlinePtr{A: "hello", Inner: &Inner{X: "val", Y: 42}},
			has:   map[string]string{"a": "hello", "x": "val", "y": "42"},
			lacks: []string{"inner"},
		},
		{
			name:  "nil inline pointer produces no inner fields",
			val:   InlinePtr{A: "hello", Inner: nil},
			has:   map[string]string{"a": "hello"},
			lacks: []string{"x", "y"},
		},
		{
			name:  "inline struct with omitzero skips when zero",
			val:   InlineOmitzero{A: "hello"},
			has:   map[string]string{"a": "hello"},
			lacks: []string{"x", "y"},
		},

		// ── Inline map + embedded struct ────────────────────────────
		{
			name: "inner embedded inline map entries appear at top level",
			val: EmbedOuter{
				EmbedInner: EmbedInner{A: true, Extras: map[string]any{"inner_extra": "works"}},
				B:          "hello",
			},
			has: map[string]string{"a": "true", "b": "hello", "inner_extra": "works"},
		},

		// ── Inline map + DynamicFields coexistence ──────────────────────
		{
			name: "inline map and DynamicFields both contribute",
			val: InlineMapAndRaw{
				Name:          "alice",
				Extras:        map[string]any{"from_map": "yes"},
				DynamicFields: apidata.DynamicFields{"from_raw": "also_yes"},
			},
			has: map[string]string{"name": "alice", "from_map": "yes", "from_raw": "also_yes"},
		},
		{
			name: "DynamicFields Omit removes native field alongside inline map",
			val: InlineMapAndRaw{
				Name:          "alice",
				Extras:        map[string]any{"x": "1"},
				DynamicFields: apidata.DynamicFields{"name": apidata.Omit},
			},
			has:   map[string]string{"x": "1"},
			lacks: []string{"name"},
		},
		{
			name: "DynamicFields Unknown replaces native field alongside inline map",
			val: InlineMapAndRaw{
				Name:          "alice",
				Extras:        map[string]any{"x": "1"},
				DynamicFields: apidata.DynamicFields{"name": apidata.Unknown(`bob`)},
			},
			has: map[string]string{"name": "bob", "x": "1"},
		},
	}

	tf := tForm{t}
	results := map[string]formResult{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tf.Marshal(tt.val)
			results[tt.name] = r

			for field, value := range tt.has {
				r.Has(field, value)
			}
			for _, field := range tt.lacks {
				r.Lacks(field)
			}
			if tt.sameAs != "" {
				baseline, ok := results[tt.sameAs]
				if !ok {
					t.Fatalf("sameAs %q not found (must appear earlier in table)", tt.sameAs)
				}
				if r.raw != baseline.raw {
					t.Errorf("expected output identical to %q\ngot:  %q\nwant: %q", tt.sameAs, r.raw, baseline.raw)
				}
			}
		})
	}
}
