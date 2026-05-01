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
	"testing"

	"google.golang.org/genai/interactions/internal/apijson"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
)

func TestInline(t *testing.T) {
	type Inner struct {
		X string `json:"x"`
		Y int    `json:"y"`
	}
	type InlineStruct struct {
		A     string `json:"a"`
		Inner Inner  `json:",inline"`
	}
	type InlinePtr struct {
		A     string `json:"a"`
		Inner *Inner `json:",inline"`
	}
	type InlineMap struct {
		Name   string         `json:"name"`
		Extras map[string]any `json:",inline"`
	}
	type InlineMapAndRaw struct {
		Name   string         `json:"name"`
		Extras map[string]any `json:",inline"`
		apidata.DynamicFields
	}
	type EmbedInner struct {
		A      bool           `json:"a"`
		Extras map[string]any `json:",inline"`
	}
	type EmbedOuter struct {
		B string `json:"b"`
		EmbedInner
	}

	tests := []struct {
		name string
		val  any
		want string
	}{
		// ── Inline struct ───────────────────────────────────────────
		{
			name: "inline struct spreads fields into parent",
			val:  InlineStruct{A: "hello", Inner: Inner{X: "val", Y: 42}},
			want: `{"a":"hello","x":"val","y":42}`,
		},
		{
			name: "inline struct with zero-valued inner fields",
			val:  InlineStruct{A: "hello", Inner: Inner{X: "", Y: 0}},
			want: `{"a":"hello","x":"","y":0}`,
		},
		{
			name: "inline pointer to struct",
			val:  InlinePtr{A: "hello", Inner: &Inner{X: "val", Y: 42}},
			want: `{"a":"hello","x":"val","y":42}`,
		},
		{
			name: "nil inline pointer produces no inner fields",
			val:  InlinePtr{A: "hello", Inner: nil},
			want: `{"a":"hello"}`,
		},

		// ── Inline map ──────────────────────────────────────────────
		{
			name: "inline map spreads entries as sibling fields",
			val:  InlineMap{Name: "alice", Extras: map[string]any{"foo": "bar"}},
			want: `{"name":"alice","foo":"bar"}`,
		},
		{
			name: "nil inline map produces no extra fields",
			val:  InlineMap{Name: "alice", Extras: nil},
			want: `{"name":"alice"}`,
		},
		{
			name: "empty inline map produces no extra fields",
			val:  InlineMap{Name: "alice", Extras: map[string]any{}},
			want: `{"name":"alice"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tj := tjson{t}
			tj.Marshal(tt.val).Equals(tt.want)
		})
	}
}

// ── Types for unmarshal tests (need methods, so package-level) ───────

// inlineMapRoundtrip has both an inline map and DynamicFields, plus
// MarshalJSON/UnmarshalJSON via MarshalRoot/UnmarshalRoot.
type inlineMapRoundtrip struct {
	Name   string         `json:"name"`
	Extras map[string]any `json:",inline"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r inlineMapRoundtrip) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

func (r *inlineMapRoundtrip) UnmarshalJSON(raw []byte) error {
	return apijson.UnmarshalRoot(raw, r, &r.meta)
}

// TestInlineUnmarshal verifies the unmarshal side of inline fields.
// The key invariant: when both an inline map and DynamicFields are present,
// known fields decode into struct fields, unknown fields with matching
// inline map keys go into the inline map, and truly unknown fields go
// into DynamicFields.
func TestInlineUnmarshal(t *testing.T) {
	t.Run("unknown fields land in inline map", func(t *testing.T) {
		var obj inlineMapRoundtrip
		if err := apijson.Unmarshal([]byte(`{"name":"alice","color":"blue","count":3}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if obj.Name != "alice" {
			t.Errorf("Name: expected \"alice\", got %q", obj.Name)
		}
		// Unknown fields should be captured in the inline map (Extras),
		// not in DynamicFields, because the inline map is the catch-all.
		if obj.Extras["color"] != "blue" {
			t.Errorf("Extras[\"color\"]: expected \"blue\", got %v", obj.Extras["color"])
		}
		if obj.Extras["count"] != float64(3) {
			t.Errorf("Extras[\"count\"]: expected 3, got %v", obj.Extras["count"])
		}
	})

	t.Run("round-trip preserves inline map entries", func(t *testing.T) {
		input := `{"name":"alice","color":"blue","count":3}`
		var obj inlineMapRoundtrip
		if err := apijson.Unmarshal([]byte(input), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		got, err := apijson.Marshal(obj)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		tj := tjson{t}
		r := jsonResult{tj, string(got)}
		r.Has(`"name":"alice"`)
		r.Has(`"color":"blue"`)
		r.Has(`"count":3`)
	})

	t.Run("DynamicFields Omit can suppress an inline map entry after unmarshal", func(t *testing.T) {
		var obj inlineMapRoundtrip
		if err := apijson.Unmarshal([]byte(`{"name":"a","extra":"kill_me"}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		obj.DynamicFields = apidata.DynamicFields{"extra": apidata.Omit}

		got, err := apijson.Marshal(obj)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		tj := tjson{t}
		r := jsonResult{tj, string(got)}
		r.Has(`"name":"a"`)
		r.Lacks(`"extra"`)
	})
}

// inlineTypedMap has a typed inline map — only string values fit.
// When JSON has a non-string value for an unknown key, it can't decode
// into the map. The question is: does it error, silently drop, or land
// in DynamicFields?
type inlineTypedMap struct {
	Name   string            `json:"name"`
	Extras map[string]string `json:",inline"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r inlineTypedMap) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

func (r *inlineTypedMap) UnmarshalJSON(raw []byte) error {
	return apijson.UnmarshalRoot(raw, r, &r.meta)
}

func TestInlineTypedMapMismatch(t *testing.T) {
	t.Run("string value fits in typed map", func(t *testing.T) {
		var obj inlineTypedMap
		if err := apijson.Unmarshal([]byte(`{"name":"a","color":"blue"}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if obj.Extras["color"] != "blue" {
			t.Errorf("Extras[\"color\"]: expected \"blue\", got %v", obj.Extras["color"])
		}
	})

	t.Run("number value falls through to DynamicFields", func(t *testing.T) {
		var obj inlineTypedMap
		if err := apijson.Unmarshal([]byte(`{"name":"a","count":3}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		// "count" can't decode into string — it should land in DynamicFields,
		// not error or silently become a zero string in Extras.
		if _, ok := obj.Extras["count"]; ok {
			t.Errorf("Extras[\"count\"] should not be set, got %v", obj.Extras["count"])
		}
		if obj.DynamicFields["count"] == nil {
			t.Error("expected count in DynamicFields, got nil")
		}
	})

	t.Run("null value falls through to DynamicFields", func(t *testing.T) {
		// stdlib Unmarshal of null into a non-pointer string target is a
		// silent no-op (returns nil, leaves ""). Without an explicit null
		// check, the value would silently become "" in Extras. It should
		// land in DynamicFields instead, like a type-mismatched number.
		var obj inlineTypedMap
		if err := apijson.Unmarshal([]byte(`{"name":"a","missing":null}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if v, ok := obj.Extras["missing"]; ok {
			t.Errorf("Extras[\"missing\"] should not be set for JSON null, got %q", v)
		}
		if obj.DynamicFields["missing"] == nil {
			t.Error("expected null to land in DynamicFields, got nil")
		}
	})

	t.Run("mixed types: string fits in map, number falls through to DynamicFields", func(t *testing.T) {
		var obj inlineTypedMap
		if err := apijson.Unmarshal([]byte(`{"name":"a","color":"blue","count":3,"more":"red"}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if obj.Extras["color"] != "blue" {
			t.Errorf("Extras[\"color\"]: expected \"blue\", got %v", obj.Extras["color"])
		}
		if obj.Extras["more"] != "red" {
			t.Errorf("Extras[\"red\"]: expected \"red\", got %v", obj.Extras["color"])
		}
		if _, ok := obj.Extras["count"]; ok {
			t.Errorf("Extras[\"count\"] should not be set, got %v", obj.Extras["count"])
		}
		if obj.DynamicFields["count"] == nil {
			t.Error("expected count in DynamicFields, got nil")
		}
	})

	t.Run("round-trip with typed map preserves string entries", func(t *testing.T) {
		var obj inlineTypedMap
		if err := apijson.Unmarshal([]byte(`{"name":"a","color":"blue"}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		got, err := apijson.Marshal(obj)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		tj := tjson{t}
		r := jsonResult{tj, string(got)}
		r.Has(`"name":"a"`)
		r.Has(`"color":"blue"`)
	})
}

// ── Types for custom MarshalJSON tests ──────────────────────────────

// customInner has a MarshalJSON that injects an extra key.
type customInner struct {
	X string `json:"x"`
}

func (c customInner) MarshalJSON() ([]byte, error) {
	return []byte(`{"x":"` + c.X + `","injected":true}`), nil
}

// parentAnonymous embeds customInner (which has MarshalJSON) and defines
// its own MarshalJSON via MarshalRoot, like generated types do.
type parentAnonymous struct {
	A string `json:"a"`
	customInner
}

func (p parentAnonymous) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(p)
}

// TestInlineCustomMarshalJSON verifies that when a struct with its own
// MarshalJSON is used as an inline field, the custom marshaler is
// bypassed — inline means "spread these fields", not "delegate to
// this type's marshaler". The MarshalJSON-injected keys should NOT
// appear because the encoder recurses into the struct's fields directly.
func TestInlineCustomMarshalJSON(t *testing.T) {
	// Non-anonymous inline field with custom MarshalJSON.
	type ParentExplicit struct {
		A     string      `json:"a"`
		Inner customInner `json:",inline"`
	}

	tests := []struct {
		name string
		val  any
		want string
	}{
		{
			// customInner.MarshalJSON injects "injected":true, but inline
			// bypasses it — only the declared struct fields are spread.
			name: "non-anonymous inline ignores inner MarshalJSON",
			val:  ParentExplicit{A: "hello", Inner: customInner{X: "val"}},
			want: `{"a":"hello","x":"val"}`,
		},
		{
			// parentAnonymous has its own MarshalJSON (via MarshalRoot),
			// so the embedded customInner's MarshalJSON should NOT take
			// over. The inner fields should be promoted into the parent.
			name: "anonymous embed with parent MarshalRoot ignores inner MarshalJSON",
			val:  parentAnonymous{A: "hello", customInner: customInner{X: "val"}},
			want: `{"a":"hello","x":"val"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := apijson.Marshal(tt.val)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("expected:\n  %s\ngot:\n  %s", tt.want, string(got))
			}
		})
	}
}
