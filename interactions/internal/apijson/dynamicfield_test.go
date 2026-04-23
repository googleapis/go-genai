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

package apijson_test

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/genai/interactions/internal/apijson"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
)

func ptr[T any](v T) *T { return &v }

// ── Types ────────────────────────────────────────────────────────────

type diffSimple struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
	Tag   string `json:"tag"`

	apidata.DynamicFields `json:"-"`
}

func (r diffSimple) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

type diffRoundtrip struct {
	Name  string `json:"name"`
	Value int    `json:"value"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r diffRoundtrip) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

func (r *diffRoundtrip) UnmarshalJSON(raw []byte) error {
	return apijson.UnmarshalRoot(raw, r, &r.meta)
}

type diffUnion struct {
	OfString *string `json:",inline,omitzero"`
	OfInt    *int    `json:",inline,omitzero"`

	meta unmarshalinfo.Metadata `api:"union"`
}

// diffChild has a MarshalJSON that injects extra keys.
type diffChild struct {
	Tag string `json:"tag"`
}

func (c diffChild) MarshalJSON() ([]byte, error) {
	return []byte(`{"tag":"` + c.Tag + `","injected":true}`), nil
}

type diffParent struct {
	Label string    `json:"label"`
	Child diffChild `json:"child"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (p diffParent) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(p)
}

func (p *diffParent) UnmarshalJSON(raw []byte) error {
	return apijson.UnmarshalRoot(raw, p, &p.meta)
}

type diffWithInlineMap struct {
	Name  string         `json:"name"`
	Extra map[string]any `json:",inline"`

	apidata.DynamicFields `json:"-"`
}

func (r diffWithInlineMap) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

// ── Test helpers ─────────────────────────────────────────────────────

// tjson wraps testing.T with JSON assertion helpers.
type tjson struct{ *testing.T }

// Marshal marshals v and returns a jsonResult for chained assertions.
func (tj tjson) Marshal(v any) jsonResult {
	tj.Helper()
	got, err := apijson.Marshal(v)
	if err != nil {
		tj.Fatalf("Marshal: %v", err)
	}
	return jsonResult{tj, string(got)}
}

// unmarshal unmarshals input into a new T.
func unmarshal[T any](tj tjson, input string) T {
	tj.Helper()
	var v T
	if err := apijson.Unmarshal([]byte(input), &v); err != nil {
		tj.Fatalf("Unmarshal: %v", err)
	}
	return v
}

// jsonResult holds marshaled JSON for assertions.
type jsonResult struct {
	tjson
	raw string
}

// Has asserts the JSON output contains the substring.
func (r jsonResult) Has(substr string) {
	r.Helper()
	if !strings.Contains(r.raw, substr) {
		r.Errorf("expected %q in:\n%s", substr, r.raw)
	}
}

// Lacks asserts the JSON output does not contain the substring.
func (r jsonResult) Lacks(substr string) {
	r.Helper()
	if strings.Contains(r.raw, substr) {
		r.Errorf("expected %q absent from:\n%s", substr, r.raw)
	}
}

// Equals asserts exact JSON string equality.
func (r jsonResult) Equals(want string) {
	r.Helper()
	if r.raw != want {
		r.Errorf("expected:\n  %s\ngot:\n  %s", want, r.raw)
	}
}

// Keys returns the top-level keys of the JSON object.
func (r jsonResult) Keys() []string {
	r.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(r.raw), &m); err != nil {
		r.Fatalf("invalid JSON: %s", r.raw)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ── Tests ────────────────────────────────────────────────────────────

func TestMarshalDynamicFields(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{
			name: "extra field added",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"extra": "v"}},
			want: `{"name":"a","value":1,"tag":"","extra":"v"}`,
		},
		{
			name: "omit native field",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"name": apidata.Omit}},
			want: `{"value":1,"tag":""}`,
		},
		{
			name: "replace native with raw JSON",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"name": apidata.Unknown(`42`)}},
			want: `{"name":42,"value":1,"tag":""}`,
		},
		{
			name: "replace native with null",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"name": apidata.Unknown(`null`)}},
			want: `{"name":null,"value":1,"tag":""}`,
		},
		{
			name: "replace native with array",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"value": apidata.Unknown(`[1,2,3]`)}},
			want: `{"name":"a","value":[1,2,3],"tag":""}`,
		},
		{
			name: "add complex nested JSON",
			val:  diffSimple{Name: "a", DynamicFields: apidata.DynamicFields{"complex": apidata.Unknown(`{"inner":{"deep":[1,2]},"flag":true}`)}},
			want: `{"name":"a","value":0,"tag":"","complex":{"inner":{"deep":[1,2]},"flag":true}}`,
		},
		{
			name: "omit nonexistent field is no-op",
			val:  diffSimple{Name: "a", DynamicFields: apidata.DynamicFields{"ghost": apidata.Omit}},
			want: `{"name":"a","value":0,"tag":""}`,
		},
		{
			name: "add via Unknown on nonexistent field",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"new_field": apidata.Unknown(`99`)}},
			want: `{"name":"a","value":1,"tag":"","new_field":99}`,
		},
		{
			name: "nil DynamicFields — baseline",
			val:  diffSimple{Name: "a", Value: 1},
			want: `{"name":"a","value":1,"tag":""}`,
		},
		{
			name: "empty DynamicFields matches nil",
			val:  diffSimple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{}},
			want: `{"name":"a","value":1,"tag":""}`,
		},
		{
			name: "inline map and DynamicFields both contribute",
			val: diffWithInlineMap{
				Name:          "a",
				Extra:         map[string]any{"from_map": "m"},
				DynamicFields: apidata.DynamicFields{"from_extras": "e"},
			},
			want: `{"name":"a","from_map":"m","from_extras":"e"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tj := tjson{t}
			tj.Marshal(tt.val).Equals(tt.want)
		})
	}
}

func TestMarshalInlineUnion(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		want    string
		wantErr bool
	}{
		{name: "string member", val: diffUnion{OfString: ptr("hello")}, want: `"hello"`},
		{name: "int member", val: diffUnion{OfInt: ptr(42)}, want: `42`},
		{name: "first non-zero wins", val: diffUnion{OfString: ptr("first"), OfInt: ptr(999)}, want: `"first"`},
		{name: "no member set errors", val: diffUnion{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := apijson.MarshalUnionStruct(tt.val)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("MarshalUnionStruct: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("expected %s, got %s", tt.want, string(got))
			}
		})
	}
}

func TestRoundtrip(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		mutate func(*diffRoundtrip)
		want   string
	}{
		{
			name:  "preserves single unknown field",
			input: `{"name":"alice","value":1,"extra":"surprise"}`,
			want:  `{"name":"alice","value":1,"extra":"surprise"}`,
		},
		{
			name:  "preserves multiple unknown fields",
			input: `{"name":"bob","value":2,"extra1":"a","extra2":42}`,
			want:  `{"name":"bob","value":2,"extra1":"a","extra2":42}`,
		},
		{
			name:  "preserves unknown null",
			input: `{"name":"a","value":1,"extra":null}`,
			want:  `{"name":"a","value":1,"extra":null}`,
		},
		{
			name:  "preserves unknown nested object",
			input: `{"name":"a","value":1,"nested":{"x":1,"y":"two"}}`,
			want:  `{"name":"a","value":1,"nested":{"x":1,"y":"two"}}`,
		},
		{
			name:  "preserves unknown array",
			input: `{"name":"a","value":1,"items":[1,"two",true]}`,
			want:  `{"name":"a","value":1,"items":[1,"two",true]}`,
		},
		{
			name:  "preserves unknown boolean",
			input: `{"name":"a","value":1,"flag":true}`,
			want:  `{"name":"a","value":1,"flag":true}`,
		},
		{
			name:  "no unknowns round-trips cleanly",
			input: `{"name":"a","value":1}`,
			want:  `{"name":"a","value":1}`,
		},
		{
			name:  "empty object round-trips to zero values",
			input: `{}`,
			want:  `{"name":"","value":0}`,
		},
		{
			name:   "mutated known field uses new value",
			input:  `{"name":"original","value":1}`,
			mutate: func(o *diffRoundtrip) { o.Name = "modified" },
			want:   `{"name":"modified","value":1}`,
		},
		{
			name:  "Omit overrides captured unknown",
			input: `{"name":"a","value":1,"extra":"captured"}`,
			mutate: func(o *diffRoundtrip) {
				o.DynamicFields["extra"] = apidata.Omit
			},
			want: `{"name":"a","value":1}`,
		},
		{
			name:  "Unknown overrides captured unknown",
			input: `{"name":"a","value":1,"extra":"original"}`,
			mutate: func(o *diffRoundtrip) {
				o.DynamicFields["extra"] = apidata.Unknown(`{"replaced":true}`)
			},
			want: `{"name":"a","value":1,"extra":{"replaced":true}}`,
		},
		{
			name:   "all JSON types survive round-trip with mutation",
			input:  `{"name":"alice","value":1,"str":"s","num":3.14,"obj":{"k":"v"},"arr":[1,true],"nil":null,"bool":true}`,
			mutate: func(o *diffRoundtrip) { o.Name = "modified" },
			want:   `{"name":"modified","value":1,"arr":[1,true],"bool":true,"nil":null,"num":3.14,"obj":{"k":"v"},"str":"s"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tj := tjson{t}
			obj := unmarshal[diffRoundtrip](tj, tt.input)
			if tt.mutate != nil {
				tt.mutate(&obj)
			}
			tj.Marshal(obj).Equals(tt.want)
		})
	}
}

func TestRoundtripCustomMarshalJSON(t *testing.T) {
	t.Run("child custom MarshalJSON keys appear in parent output", func(t *testing.T) {
		tj := tjson{t}
		r := tj.Marshal(diffParent{
			Label: "parent",
			Child: diffChild{Tag: "hello"},
			DynamicFields: apidata.DynamicFields{
				"label": apidata.Unknown(`"overridden"`),
			},
		})
		r.Has(`"label":"overridden"`)
		r.Has(`"injected":true`)
		r.Has(`"tag":"hello"`)
	})

	t.Run("child UnmarshalJSON respected and unknowns preserved", func(t *testing.T) {
		tj := tjson{t}
		obj := unmarshal[diffParent](tj, `{"label":"p","child":{"tag":"world"},"unknown":"kept"}`)
		if obj.Label != "p" {
			t.Fatalf("expected label=p, got %s", obj.Label)
		}
		if obj.Child.Tag != "world" {
			t.Fatalf("expected child.tag=world, got %s", obj.Child.Tag)
		}
		raw, ok := obj.DynamicFields["unknown"].(apidata.Unknown)
		if !ok {
			t.Fatalf("expected apidata.Unknown, got %T", obj.DynamicFields["unknown"])
		}
		if string(raw) != `"kept"` {
			t.Fatalf("expected raw \"kept\", got %s", string(raw))
		}
	})
}

// TestDynamicFieldsKeyWithSjsonMetacharacters pins the escaping behavior
// for keys that contain sjson path metacharacters (`.`, `*`, `?`,
// `#`, `|`, `\`). Without escaping, sjson would reshape/drop these
// keys silently — see MarshalRoot in apidata.go.
//
// Covers both directions:
//   - Set via DynamicFields by the caller   (marshal path)
//   - Captured from inbound JSON        (unmarshal → re-marshal round-trip)
func TestDynamicFieldsKeyWithSjsonMetacharacters(t *testing.T) {
	metas := []string{"a.b", "a*b", "a?b", "a#b", "a|b", `a\b`}

	t.Run("marshal preserves the key verbatim", func(t *testing.T) {
		for _, k := range metas {
			obj := diffSimple{Name: "x", Value: 1, DynamicFields: apidata.DynamicFields{k: "v"}}
			got, err := apijson.Marshal(obj)
			if err != nil {
				t.Errorf("%q: Marshal: %v", k, err)
				continue
			}
			// Re-parse and verify the key lands at the top level.
			var m map[string]any
			if err := json.Unmarshal(got, &m); err != nil {
				t.Errorf("%q: output isn't valid JSON: %v\n  raw: %s", k, err, got)
				continue
			}
			if _, ok := m[k]; !ok {
				t.Errorf("%q: missing from top-level output; got: %s", k, got)
			}
		}
	})

	t.Run("round-trip through unmarshal preserves the key verbatim", func(t *testing.T) {
		for _, k := range metas {
			// Hand-roll the JSON so we control the exact key bytes.
			kEsc, _ := json.Marshal(k)
			input := `{"name":"x","value":1,"tag":"","` + string(kEsc[1:len(kEsc)-1]) + `":"v"}`

			var obj diffRoundtrip
			if err := apijson.Unmarshal([]byte(input), &obj); err != nil {
				t.Errorf("%q: Unmarshal: %v", k, err)
				continue
			}
			if _, ok := obj.DynamicFields[k]; !ok {
				t.Errorf("%q: decoder didn't capture into DynamicFields: %v", k, obj.DynamicFields)
				continue
			}
			got, err := apijson.Marshal(obj)
			if err != nil {
				t.Errorf("%q: Marshal: %v", k, err)
				continue
			}
			var m map[string]any
			if err := json.Unmarshal(got, &m); err != nil {
				t.Errorf("%q: re-marshaled output isn't valid JSON: %v\n  raw: %s", k, err, got)
				continue
			}
			if _, ok := m[k]; !ok {
				t.Errorf("%q: round-trip dropped the key; got: %s", k, got)
			}
		}
	})

	t.Run("Omit with metacharacter key deletes only that key", func(t *testing.T) {
		// Seed both `a.b` and a sibling `a` via DynamicFields, then Omit `a.b`.
		// Without escaping, Omit("a.b") would path-navigate and corrupt the
		// top-level `a` instead.
		obj := diffSimple{
			Name:  "n",
			Value: 1,
			DynamicFields: apidata.DynamicFields{
				"a":   "keep",
				"a.b": apidata.Omit, // no-op: key doesn't exist in the marshaled output
			},
		}
		got, err := apijson.Marshal(obj)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(got, &m); err != nil {
			t.Fatalf("invalid JSON: %v\n  raw: %s", err, got)
		}
		if m["a"] != "keep" {
			t.Errorf("Omit on %q corrupted sibling %q; got: %s", "a.b", "a", got)
		}
	})
}

// TestDynamicFieldsEmptyKeyDropped — empty-string keys are silently
// dropped on marshal (sjson rejects empty paths; handling them
// specially isn't worth the complexity).
func TestDynamicFieldsEmptyKeyDropped(t *testing.T) {
	obj := diffSimple{Name: "n", DynamicFields: apidata.DynamicFields{"": "ignored"}}
	got, err := apijson.Marshal(obj)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(got), `"":`) {
		t.Errorf("empty-key entry leaked into output: %s", got)
	}
}
