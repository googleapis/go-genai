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

// Probes for the "intersection" code-generation pattern:
//
//   type Outer struct {
//       A string `json:"a"`
//       InnerA                                       // anonymous embed
//       InnerB                                       // anonymous embed
//       apidata.DynamicFields `json:"-"`
//       meta              unmarshalinfo.Metadata
//   }
//
// Every layer (outer + each embed) has its own (Un)MarshalJSON via
// (Un)MarshalRoot, its own apidata.DynamicFields, and its own metadata.
//
// Intended semantics when marshaling/unmarshaling the outer:
//   - Fields from all embeds flatten into the outer's JSON. No
//     nested object, no key lookup by the embed's type name.
//   - The outer's MarshalRoot bypasses each embed's MarshalJSON —
//     embeds' custom (Un)MarshalJSON don't fire.
//   - Only the outer's DynamicFields overrides apply. Inner DynamicFields
//     are inert when the struct is marshaled as an embed.
//   - Unknown JSON fields land in the outer's DynamicFields.
//   - JSON null against a non-nullable field declared in an embed
//     lands as apidata.Mismatch in the outer's DynamicFields.

import (
	"testing"

	"google.golang.org/genai/interactions/internal/apijson"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
)

// ── Types ────────────────────────────────────────────────────────────

type interInnerA struct {
	B string `json:"b"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r interInnerA) MarshalJSON() ([]byte, error)  { return apijson.MarshalRoot(r) }
func (r *interInnerA) UnmarshalJSON(d []byte) error { return apijson.UnmarshalRoot(d, r, &r.meta) }

type interInnerB struct {
	C string `json:"c"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r interInnerB) MarshalJSON() ([]byte, error)  { return apijson.MarshalRoot(r) }
func (r *interInnerB) UnmarshalJSON(d []byte) error { return apijson.UnmarshalRoot(d, r, &r.meta) }

type interOuter struct {
	A string `json:"a"`
	interInnerA
	interInnerB

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r interOuter) MarshalJSON() ([]byte, error)  { return apijson.MarshalRoot(r) }
func (r *interOuter) UnmarshalJSON(d []byte) error { return apijson.UnmarshalRoot(d, r, &r.meta) }

// divergentInner emits a DIFFERENT key in its MarshalJSON than its
// struct tag declares. Used to prove that when the outer uses
// MarshalRoot, the embed's MarshalJSON is NOT called — field
// promotion happens instead (we'd see "custom_b" if the embed's
// MarshalJSON ran).
type divergentInner struct {
	B string `json:"b"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r divergentInner) MarshalJSON() ([]byte, error) {
	return []byte(`{"custom_b":"CUSTOM-` + r.B + `"}`), nil
}

// divergentInner also has a divergent UnmarshalJSON: it puts the
// whole input (not the "b" field) into B. If the outer's
// UnmarshalRoot calls it, we'd see B contain the raw JSON.
func (r *divergentInner) UnmarshalJSON(raw []byte) error {
	r.B = "CUSTOM-" + string(raw)
	return nil
}

type divergentOuter struct {
	A string `json:"a"`
	divergentInner

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r divergentOuter) MarshalJSON() ([]byte, error)  { return apijson.MarshalRoot(r) }
func (r *divergentOuter) UnmarshalJSON(d []byte) error { return apijson.UnmarshalRoot(d, r, &r.meta) }

// InterInnerPtr is embedded via *pointer rather than value — a
// distinct typeFields code path. Exported because stdlib refuses to
// alloc embedded pointers to unexported types (irrelevant to the
// generator, which produces exported types).
type InterInnerPtr struct {
	D string `json:"d"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r InterInnerPtr) MarshalJSON() ([]byte, error)  { return apijson.MarshalRoot(r) }
func (r *InterInnerPtr) UnmarshalJSON(d []byte) error { return apijson.UnmarshalRoot(d, r, &r.meta) }

type pointerEmbedOuter struct {
	A string `json:"a"`
	*InterInnerPtr

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r pointerEmbedOuter) MarshalJSON() ([]byte, error) { return apijson.MarshalRoot(r) }
func (r *pointerEmbedOuter) UnmarshalJSON(d []byte) error {
	return apijson.UnmarshalRoot(d, r, &r.meta)
}

// populatedOuter returns an interOuter with every declared field set
// — used as the canonical "fully populated" fixture.
func populatedOuter() interOuter {
	return interOuter{
		A:           "a_val",
		interInnerA: interInnerA{B: "b_val"},
		interInnerB: interInnerB{C: "c_val"},
	}
}

// ── Tests ────────────────────────────────────────────────────────────

// TestIntersectionFieldPromotion — fields from every embed flatten
// as siblings at the outer level. No nesting, no type-name keys.
func TestIntersectionFieldPromotion(t *testing.T) {
	tj := tjson{t}
	tj.Marshal(populatedOuter()).Equals(`{"a":"a_val","b":"b_val","c":"c_val"}`)
}

// TestIntersectionEmbedMethodsAreBypassed — when the outer uses
// MarshalRoot/UnmarshalRoot, the embed's own (Un)MarshalJSON methods
// do NOT fire. Field promotion happens via the encoder's flattened
// field list; the decoder targets individual leaf fields.
//
// divergentInner is specifically designed so that if either of its
// custom methods ran, the output would contain "CUSTOM-" or
// "custom_b" — neither should appear.
func TestIntersectionEmbedMethodsAreBypassed(t *testing.T) {
	t.Run("marshal bypasses embed's MarshalJSON", func(t *testing.T) {
		tj := tjson{t}
		tj.Marshal(divergentOuter{
			A:              "a_val",
			divergentInner: divergentInner{B: "b_val"},
		}).Equals(`{"a":"a_val","b":"b_val"}`)
	})

	t.Run("unmarshal bypasses embed's UnmarshalJSON", func(t *testing.T) {
		tj := tjson{t}
		obj := unmarshal[divergentOuter](tj, `{"a":"a_val","b":"b_val"}`)
		if obj.A != "a_val" {
			t.Errorf("A: expected %q, got %q", "a_val", obj.A)
		}
		if obj.divergentInner.B != "b_val" {
			t.Errorf("B: expected %q from field-level decode, got %q "+
				"(divergent UnmarshalJSON would prepend CUSTOM-)",
				"b_val", obj.divergentInner.B)
		}
	})
}

// TestIntersectionRoundTrip — marshal then unmarshal (and vice versa)
// round-trip without drift. Exact field values land in the right
// embed, exact JSON bytes come back out.
func TestIntersectionRoundTrip(t *testing.T) {
	const canonical = `{"a":"a_val","b":"b_val","c":"c_val"}`
	tj := tjson{t}

	t.Run("JSON → struct → JSON preserves bytes", func(t *testing.T) {
		obj := unmarshal[interOuter](tj, canonical)
		tj.Marshal(obj).Equals(canonical)
	})

	t.Run("unmarshal populates each embed's field", func(t *testing.T) {
		obj := unmarshal[interOuter](tj, canonical)
		if obj.A != "a_val" {
			t.Errorf("A: expected %q, got %q", "a_val", obj.A)
		}
		if obj.interInnerA.B != "b_val" {
			t.Errorf("B (in interInnerA): expected %q, got %q", "b_val", obj.interInnerA.B)
		}
		if obj.interInnerB.C != "c_val" {
			t.Errorf("C (in interInnerB): expected %q, got %q", "c_val", obj.interInnerB.C)
		}
	})
}

// TestIntersectionOuterDynamicFieldsOverrideEmbedField — outer's DynamicFields
// is the single source of truth for post-marshal overrides. It must
// reach fields declared in ANY embed, not just fields declared at the
// outer level.
func TestIntersectionOuterDynamicFieldsOverrideEmbedField(t *testing.T) {
	tj := tjson{t}

	t.Run("Omit removes a field declared in an embed", func(t *testing.T) {
		obj := populatedOuter()
		obj.DynamicFields = apidata.DynamicFields{"b": apidata.Omit}
		r := tj.Marshal(obj)
		r.Lacks(`"b"`)
		r.Has(`"a":"a_val"`)
		r.Has(`"c":"c_val"`)
	})

	t.Run("Omit can target fields across different embeds at once", func(t *testing.T) {
		obj := populatedOuter()
		obj.DynamicFields = apidata.DynamicFields{
			"b": apidata.Omit, // from interInnerA
			"c": apidata.Omit, // from interInnerB
		}
		tj.Marshal(obj).Equals(`{"a":"a_val"}`)
	})

	t.Run("Unknown replaces an embed field's value with raw JSON", func(t *testing.T) {
		obj := populatedOuter()
		obj.DynamicFields = apidata.DynamicFields{"b": apidata.Unknown(`42`)}
		tj.Marshal(obj).Equals(`{"a":"a_val","b":42,"c":"c_val"}`)
	})

	t.Run("DynamicFields can add a new sibling key", func(t *testing.T) {
		obj := populatedOuter()
		obj.DynamicFields = apidata.DynamicFields{"extra": "added"}
		r := tj.Marshal(obj)
		r.Has(`"a":"a_val"`)
		r.Has(`"b":"b_val"`)
		r.Has(`"c":"c_val"`)
		r.Has(`"extra":"added"`)
	})
}

// TestIntersectionInnerDynamicFieldsAreInert — any DynamicFields op set on
// an inner embed (add, Omit, Unknown) must have ZERO effect when the
// outer is marshaled. Inner DynamicFields matter only if that inner is
// marshaled as the root (not as an embed).
func TestIntersectionInnerDynamicFieldsAreInert(t *testing.T) {
	tests := []struct {
		name     string
		innerRaw apidata.DynamicFields
	}{
		{name: "inner adds new field", innerRaw: apidata.DynamicFields{"injected": "yes"}},
		{name: "inner omits sibling field", innerRaw: apidata.DynamicFields{"c": apidata.Omit}},
		{name: "inner omits own field", innerRaw: apidata.DynamicFields{"b": apidata.Omit}},
		{name: "inner replaces with Unknown", innerRaw: apidata.DynamicFields{"b": apidata.Unknown(`999`)}},
	}

	const wantInert = `{"a":"a_val","b":"b_val","c":"c_val"}`
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := populatedOuter()
			obj.interInnerA.DynamicFields = tt.innerRaw
			tjson{t}.Marshal(obj).Equals(wantInert)
		})
	}
}

// TestIntersectionUnknownFieldLandsInOuterDynamicFields — unknown JSON
// keys land in the outer's DynamicFields as apidata.Unknown, and
// DEFINITELY not in any inner's DynamicFields. Round-trip preserves the
// raw bytes verbatim (including numeric precision beyond float64).
func TestIntersectionUnknownFieldLandsInOuterDynamicFields(t *testing.T) {
	tj := tjson{t}
	const big = `10000000000000001` // > 2^53, precision-sensitive
	input := `{"a":"a_val","b":"b_val","c":"c_val","mystery":42,"big":` + big + `}`
	obj := unmarshal[interOuter](tj, input)

	t.Run("outer DynamicFields captures unknowns as apidata.Unknown", func(t *testing.T) {
		for key, wantRaw := range map[string]string{"mystery": "42", "big": big} {
			v, ok := obj.DynamicFields[key]
			if !ok {
				t.Errorf("outer DynamicFields missing %q: %v", key, obj.DynamicFields)
				continue
			}
			u, ok := v.(apidata.Unknown)
			if !ok {
				t.Errorf("%q: expected apidata.Unknown, got %T (%v)", key, v, v)
				continue
			}
			if string(u) != wantRaw {
				t.Errorf("%q: expected Unknown(%q), got Unknown(%q)", key, wantRaw, u)
			}
		}
	})

	t.Run("inner DynamicFields are untouched by unknowns", func(t *testing.T) {
		for _, inner := range []struct {
			name string
			raw  apidata.DynamicFields
		}{
			{"interInnerA", obj.interInnerA.DynamicFields},
			{"interInnerB", obj.interInnerB.DynamicFields},
		} {
			if len(inner.raw) != 0 {
				t.Errorf("%s.DynamicFields should be empty, got: %v", inner.name, inner.raw)
			}
		}
	})

	t.Run("round-trip preserves raw bytes of unknowns", func(t *testing.T) {
		r := tj.Marshal(obj)
		r.Has(`"mystery":42`)
		r.Has(`"big":` + big)
	})
}

// TestIntersectionMismatchInEmbedField — JSON null for a non-nullable
// field declared in an embed lands in the outer's DynamicFields as
// apidata.Mismatch (keyed by the leaf field name). The embed's own
// field stays at its zero value. On re-marshal, the Mismatch is
// inert — the struct field's zero value is emitted.
func TestIntersectionMismatchInEmbedField(t *testing.T) {
	tj := tjson{t}
	obj := unmarshal[interOuter](tj, `{"a":"a_val","b":null,"c":"c_val"}`)

	t.Run("embed's field stays at its zero value", func(t *testing.T) {
		if obj.interInnerA.B != "" {
			t.Errorf("B should be zero-valued after null, got %q", obj.interInnerA.B)
		}
	})

	t.Run("outer DynamicFields records apidata.Mismatch with the raw bytes", func(t *testing.T) {
		v, ok := obj.DynamicFields["b"]
		if !ok {
			t.Fatalf("expected %q in outer DynamicFields, got: %v", "b", obj.DynamicFields)
		}
		mm, ok := v.(apidata.Mismatch)
		if !ok {
			t.Fatalf("expected apidata.Mismatch, got %T (%v)", v, v)
		}
		if string(mm) != "null" {
			t.Errorf("expected Mismatch(%q), got Mismatch(%q)", "null", mm)
		}
	})

	t.Run("inner DynamicFields are not used for Mismatch", func(t *testing.T) {
		if len(obj.interInnerA.DynamicFields) != 0 {
			t.Errorf("interInnerA.DynamicFields should be empty, got: %v", obj.interInnerA.DynamicFields)
		}
	})

	t.Run("Mismatch is inert on re-marshal", func(t *testing.T) {
		tj.Marshal(obj).Equals(`{"a":"a_val","b":"","c":"c_val"}`)
	})
}

// TestIntersectionPointerEmbed — anonymous pointer embed behaves
// symmetrically to a value embed: fields promote when non-nil, the
// pointer is allocated as needed on unmarshal, and the outer's
// DynamicFields reaches through to the pointed-to type's fields.
func TestIntersectionPointerEmbed(t *testing.T) {
	tj := tjson{t}

	t.Run("non-nil pointer promotes fields (flat output)", func(t *testing.T) {
		tj.Marshal(pointerEmbedOuter{
			A:             "a_val",
			InterInnerPtr: &InterInnerPtr{D: "d_val"},
		}).Equals(`{"a":"a_val","d":"d_val"}`)
	})

	t.Run("nil pointer omits the embed's fields entirely", func(t *testing.T) {
		r := tj.Marshal(pointerEmbedOuter{A: "a_val"})
		r.Equals(`{"a":"a_val"}`)
	})

	t.Run("unmarshal allocates the pointer and populates it", func(t *testing.T) {
		obj := unmarshal[pointerEmbedOuter](tj, `{"a":"a_val","d":"d_val"}`)
		if obj.InterInnerPtr == nil {
			t.Fatal("expected embedded pointer allocated by unmarshal")
		}
		if obj.InterInnerPtr.D != "d_val" {
			t.Errorf("D: expected %q, got %q", "d_val", obj.InterInnerPtr.D)
		}
	})

	t.Run("outer DynamicFields Omit reaches a pointer-embed field", func(t *testing.T) {
		tj.Marshal(pointerEmbedOuter{
			A:             "a_val",
			InterInnerPtr: &InterInnerPtr{D: "d_val"},
			DynamicFields: apidata.DynamicFields{"d": apidata.Omit},
		}).Equals(`{"a":"a_val"}`)
	})
}
