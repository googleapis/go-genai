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

package apijson

import (
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"reflect"
	"testing"
)

type metadata = unmarshalinfo.Metadata

// Superset trap types
type SupersetSmall struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	metadata
}

func (s *SupersetSmall) UnmarshalJSON(raw []byte) error {
	type shadow SupersetSmall
	return UnmarshalRoot(raw, (*shadow)(s), &s.metadata)
}

type SupersetLarge struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email" api:"required"`
	metadata
}

func (s *SupersetLarge) UnmarshalJSON(raw []byte) error {
	type shadow SupersetLarge
	return UnmarshalRoot(raw, (*shadow)(s), &s.metadata)
}

// Nested union types
type NestedInnerA struct {
	A string `json:"a"`
	B string `json:"b"`
	metadata
}

func (n *NestedInnerA) UnmarshalJSON(raw []byte) error {
	type shadow NestedInnerA
	return UnmarshalRoot(raw, (*shadow)(n), &n.metadata)
}

type NestedInnerB struct {
	A string `json:"a"`
	C int    `json:"c"`
	metadata
}

func (n *NestedInnerB) UnmarshalJSON(raw []byte) error {
	type shadow NestedInnerB
	return UnmarshalRoot(raw, (*shadow)(n), &n.metadata)
}

type NestedInnerUnion struct {
	OfA      *NestedInnerA `json:",inline,omitzero"`
	OfB      *NestedInnerB `json:",inline,omitzero"`
	metadata `api:"union"`
}

func (u *NestedInnerUnion) UnmarshalJSON(raw []byte) error {
	return UnmarshalUnion(raw, u, &u.metadata)
}

type NestedOuterObj struct {
	Kind string           `json:"kind"`
	Data NestedInnerUnion `json:"data"`
	metadata
}

func (n *NestedOuterObj) UnmarshalJSON(raw []byte) error {
	type shadow NestedOuterObj
	return UnmarshalRoot(raw, (*shadow)(n), &n.metadata)
}

type NestedOuterFlat struct {
	Kind  string `json:"kind"`
	Value int    `json:"value"`
	metadata
}

func (n *NestedOuterFlat) UnmarshalJSON(raw []byte) error {
	type shadow NestedOuterFlat
	return UnmarshalRoot(raw, (*shadow)(n), &n.metadata)
}

// U = T[] | V[]
// V = O[] | string
// T = { name }
// O = { name, extra } (superset of T)
type ObjT struct {
	Name string `json:"name"`
	metadata
}

func (o *ObjT) UnmarshalJSON(raw []byte) error {
	type shadow ObjT
	return UnmarshalRoot(raw, (*shadow)(o), &o.metadata)
}

type ObjO struct {
	Name  string `json:"name"`
	Extra string `json:"extra"`
	metadata
}

func (o *ObjO) UnmarshalJSON(raw []byte) error {
	type shadow ObjO
	return UnmarshalRoot(raw, (*shadow)(o), &o.metadata)
}

type UnionV struct {
	OfOArray []ObjO  `json:",inline,omitzero"`
	OfString *string `json:",inline,omitzero"`
	metadata `api:"union"`
}

func (u *UnionV) UnmarshalJSON(raw []byte) error {
	return UnmarshalUnion(raw, u, &u.metadata)
}

type UnionU struct {
	OfTArray []ObjT   `json:",inline,omitzero"`
	OfVArray []UnionV `json:",inline,omitzero"`
	metadata `api:"union"`
}

func (u *UnionU) UnmarshalJSON(raw []byte) error {
	return UnmarshalUnion(raw, u, &u.metadata)
}

func TestUnmarshalUnion(t *testing.T) {
	type WithRequired struct {
		Name  string `json:"name" api:"required"`
		Value string `json:"value"`
	}
	type WithoutRequired struct {
		Name  string `json:"name"`
		Other string `json:"other"`
	}
	type TwoFields struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	type OneField struct {
		A string `json:"a"`
	}
	type OneFieldAlias OneField
	type WithRequiredExtra struct {
		Name  string `json:"name" api:"required"`
		Extra string `json:"extra" api:"required"`
	}
	type WithManyFields struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type WithRequiredID struct {
		ID string `json:"id" api:"required"`
	}
	type InnerStruct struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	type NestedStruct struct {
		Outer string      `json:"outer"`
		Inner InnerStruct `json:"inner"`
	}
	type WithPointer struct {
		Name  *string `json:"name"`
		Value *int    `json:"value"`
	}
	type WithSlice struct {
		Items []string `json:"items"`
	}
	type WithMap struct {
		Data map[string]string `json:"data"`
	}
	type WithBool struct {
		Flag bool `json:"flag"`
	}
	type WithFloat struct {
		Value float64 `json:"value"`
	}
	type WithInt struct {
		Count int `json:"count"`
	}
	type EmptyStruct struct{}
	type ThreeFields struct {
		A string `json:"a"`
		B string `json:"b"`
		C string `json:"c"`
	}
	type AllRequired struct {
		A string `json:"a" api:"required"`
		B string `json:"b" api:"required"`
	}

	tests := []struct {
		name     string
		json     string
		union    any
		expected any
		wantErr  bool
	}{
		// Basic type matching
		{
			name: "matches_int",
			json: `42`,
			union: struct {
				A *int    `json:",inline"`
				B *string `json:",inline"`
			}{},
			expected: 42,
		},
		{
			name: "matches_string",
			json: `"hello"`,
			union: struct {
				A *int    `json:",inline"`
				B *string `json:",inline"`
			}{},
			expected: "hello",
		},
		{
			name: "matches_bool_true",
			json: `true`,
			union: struct {
				A *int  `json:",inline"`
				B *bool `json:",inline"`
			}{},
			expected: true,
		},
		{
			name: "matches_bool_false",
			json: `false`,
			union: struct {
				A *string `json:",inline"`
				B *bool   `json:",inline"`
			}{},
			expected: false,
		},
		{
			name: "matches_float",
			json: `3.14159`,
			union: struct {
				A *string  `json:",inline"`
				B *float64 `json:",inline"`
			}{},
			expected: 3.14159,
		},
		{
			name: "matches_negative_int",
			json: `-42`,
			union: struct {
				A *string `json:",inline"`
				B *int    `json:",inline"`
			}{},
			expected: -42,
		},
		{
			name: "matches_zero",
			json: `0`,
			union: struct {
				A *string `json:",inline"`
				B *int    `json:",inline"`
			}{},
			expected: 0,
		},
		{
			name: "matches_empty_string",
			json: `""`,
			union: struct {
				A *int    `json:",inline"`
				B *string `json:",inline"`
			}{},
			expected: "",
		},
		// Struct matching
		{
			name: "prefers_more_fields",
			json: `{"a":"x","b":"y"}`,
			union: struct {
				A *OneField  `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: TwoFields{A: "x", B: "y"},
		},
		{
			name: "prefer_first_match",
			json: `{"a":"x"}`,
			union: struct {
				A *OneField      `json:",inline"`
				B *OneFieldAlias `json:",inline"`
			}{},
			expected: OneField{A: "x"},
		},
		{
			name: "prefers_not_missing_fields",
			json: `{"a":"x"}`,
			union: struct {
				A *OneField  `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: OneField{A: "x"},
		},
		{
			name: "matches_struct",
			json: `{"a":"x","b":"y"}`,
			union: struct {
				A *string    `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: TwoFields{A: "x", B: "y"},
		},
		{
			name: "prefers_three_fields_over_two",
			json: `{"a":"1","b":"2","c":"3"}`,
			union: struct {
				A *TwoFields   `json:",inline"`
				B *ThreeFields `json:",inline"`
			}{},
			expected: ThreeFields{A: "1", B: "2", C: "3"},
		},
		{
			name: "prefers_exact_match_over_superset",
			json: `{"a":"1","b":"2"}`,
			union: struct {
				A *ThreeFields `json:",inline"`
				B *TwoFields   `json:",inline"`
			}{},
			expected: TwoFields{A: "1", B: "2"},
		},
		// Required field handling
		{
			name: "prefers_required_field_match",
			json: `{"name":"test","value":"v"}`,
			union: struct {
				A *WithoutRequired `json:",inline"`
				B *WithRequired    `json:",inline"`
			}{},
			expected: WithRequired{Name: "test", Value: "v"},
		},
		{
			name: "prefers_more_matching_fields",
			json: `{"a":"1","b":"2"}`,
			union: struct {
				A *OneField  `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: TwoFields{A: "1", B: "2"},
		},
		{
			name: "required_field_missing_loses",
			json: `{"name":"test"}`,
			union: struct {
				A *WithRequiredExtra `json:",inline"`
				B *WithoutRequired   `json:",inline"`
			}{},
			expected: WithoutRequired{Name: "test"},
		},
		{
			name: "more_matches_beats_required_fields",
			json: `{"id":"123","name":"n","value":"v"}`,
			union: struct {
				A *WithRequiredID `json:",inline"`
				B *WithManyFields `json:",inline"`
			}{},
			expected: WithManyFields{ID: "123", Name: "n", Value: "v"},
		},
		{
			name: "all_required_fields_present",
			json: `{"a":"1","b":"2"}`,
			union: struct {
				A *OneField    `json:",inline"`
				B *AllRequired `json:",inline"`
			}{},
			expected: AllRequired{A: "1", B: "2"},
		},
		{
			name: "all_required_fields_missing_one",
			json: `{"a":"1"}`,
			union: struct {
				A *AllRequired `json:",inline"`
				B *OneField    `json:",inline"`
			}{},
			expected: OneField{A: "1"},
		},
		// Arrays and slices
		{
			name: "matches_array_of_ints",
			json: `[1, 2, 3]`,
			union: struct {
				A *string `json:",inline"`
				B *[]int  `json:",inline"`
			}{},
			expected: []int{1, 2, 3},
		},
		{
			name: "matches_array_of_floats",
			json: `[1, 2.5, 3]`,
			union: struct {
				A *[]int     `json:",inline"`
				B *[]float64 `json:",inline"`
			}{},
			expected: []float64{1, 2.5, 3},
		},
		{
			name: "matches_array_of_strings",
			json: `["a", "b", "c"]`,
			union: struct {
				A *[]string `json:",inline"`
			}{},
			expected: []string{"a", "b", "c"},
		},
		{
			name: "matches_struct_with_slice",
			json: `{"items":["a","b"]}`,
			union: struct {
				A *OneField  `json:",inline"`
				B *WithSlice `json:",inline"`
			}{},
			expected: WithSlice{Items: []string{"a", "b"}},
		},
		// Maps
		{
			name: "matches_map",
			json: `{"key1":"value1","key2":"value2"}`,
			union: struct {
				A *int               `json:",inline"`
				B *map[string]string `json:",inline"`
			}{},
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name: "matches_struct_with_map",
			json: `{"data":{"k":"v"}}`,
			union: struct {
				A *OneField `json:",inline"`
				B *WithMap  `json:",inline"`
			}{},
			expected: WithMap{Data: map[string]string{"k": "v"}},
		},
		{
			name: "prefers_struct_to_map",
			json: `{"a":"x"}`,
			union: struct {
				A *map[string]string `json:",inline"`
				B *OneField          `json:",inline"`
			}{},
			expected: OneField{A: "x"},
		},
		{
			name: "prefers_map_to_struct_with_missing_fields",
			json: `{"a":"x","b":"y"}`,
			union: struct {
				A *OneField          `json:",inline"`
				B *map[string]string `json:",inline"`
			}{},
			expected: map[string]string{"a": "x", "b": "y"},
		},
		// Nested structs
		{
			name: "matches_nested_struct",
			json: `{"outer":"o","inner":{"a":"x","b":"y"}}`,
			union: struct {
				A *OneField     `json:",inline"`
				B *NestedStruct `json:",inline"`
			}{},
			expected: NestedStruct{Outer: "o", Inner: InnerStruct{A: "x", B: "y"}},
		},
		// Empty struct
		{
			name: "matches_empty_object_to_empty_struct",
			json: `{}`,
			union: struct {
				A *int         `json:",inline"`
				B *EmptyStruct `json:",inline"`
			}{},
			expected: EmptyStruct{},
		},
		// Type with bool field
		{
			name: "matches_struct_with_bool",
			json: `{"flag":true}`,
			union: struct {
				A *OneField `json:",inline"`
				B *WithBool `json:",inline"`
			}{},
			expected: WithBool{Flag: true},
		},
		// Type with float field
		{
			name: "matches_struct_with_float",
			json: `{"value":3.14}`,
			union: struct {
				A *WithInt   `json:",inline"`
				B *WithFloat `json:",inline"`
			}{},
			expected: WithFloat{Value: 3.14},
		},
		// Type with int field (integer value)
		{
			name: "matches_struct_with_int",
			json: `{"count":42}`,
			union: struct {
				A *WithFloat `json:",inline"`
				B *WithInt   `json:",inline"`
			}{},
			expected: WithInt{Count: 42},
		},
		// Extra fields in JSON
		{
			name: "ignores_extra_json_fields",
			json: `{"a":"x","b":"y","c":"z"}`,
			union: struct {
				A *OneField  `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: TwoFields{A: "x", B: "y"},
		},
		// Order independence
		{
			name: "order_independent_matching",
			json: `{"b":"y","a":"x"}`,
			union: struct {
				A *OneField  `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: TwoFields{A: "x", B: "y"},
		},
		// Unicode strings
		{
			name: "matches_unicode_string",
			json: `"héllo wörld 你好"`,
			union: struct {
				A *int    `json:",inline"`
				B *string `json:",inline"`
			}{},
			expected: "héllo wörld 你好",
		},
		// Escaped characters
		{
			name: "matches_escaped_string",
			json: `"line1\nline2\ttab"`,
			union: struct {
				A *int    `json:",inline"`
				B *string `json:",inline"`
			}{},
			expected: "line1\nline2\ttab",
		},
		// Large numbers
		{
			name: "matches_large_int",
			json: `9223372036854775807`,
			union: struct {
				A *string `json:",inline"`
				B *int64  `json:",inline"`
			}{},
			expected: int64(9223372036854775807),
		},
		// Scientific notation
		{
			name: "matches_scientific_notation",
			json: `1.23e10`,
			union: struct {
				A *string  `json:",inline"`
				B *float64 `json:",inline"`
			}{},
			expected: 1.23e10,
		},
		// Error cases
		{
			name: "no_match_returns_error",
			json: `{"complex":"object"}`,
			union: struct {
				A *int `json:",inline"`
			}{},
			wantErr: true,
		},
		{
			name: "invalid_json",
			json: `{invalid json}`,
			union: struct {
				A *string `json:",inline"`
				B *int    `json:",inline"`
			}{},
			wantErr: true,
		},
		{
			name: "truncated_json",
			json: `{"a":`,
			union: struct {
				A *OneField `json:",inline"`
			}{},
			wantErr: true,
		},
		{
			name: "type_mismatch_string_to_int_struct",
			json: `{"count":"not_an_int"}`,
			union: struct {
				A *WithInt `json:",inline"`
			}{},
			wantErr: true,
		},
		// Multiple valid matches - should pick best
		{
			name: "multiple_structs_picks_best_match",
			json: `{"a":"1","b":"2","c":"3"}`,
			union: struct {
				A *OneField    `json:",inline"`
				B *TwoFields   `json:",inline"`
				C *ThreeFields `json:",inline"`
			}{},
			expected: ThreeFields{A: "1", B: "2", C: "3"},
		},
		// Variant order should not matter
		{
			name: "variant_order_does_not_matter",
			json: `{"a":"x","b":"y"}`,
			union: struct {
				A *TwoFields `json:",inline"`
				B *OneField  `json:",inline"`
			}{},
			expected: TwoFields{A: "x", B: "y"},
		},
		// Complex nested array
		{
			name: "matches_array_of_structs",
			json: `[{"a":"1","b":"2"},{"a":"3","b":"4"}]`,
			union: struct {
				A *string      `json:",inline"`
				B *[]TwoFields `json:",inline"`
			}{},
			expected: []TwoFields{{A: "1", B: "2"}, {A: "3", B: "4"}},
		},
		// Pointer fields with values
		{
			name: "matches_struct_with_pointer_fields",
			json: `{"name":"test","value":42}`,
			union: struct {
				A *OneField    `json:",inline"`
				B *WithPointer `json:",inline"`
			}{},
			expected: WithPointer{Name: ptr("test"), Value: ptr(42)},
		},
		// any type
		{
			name: "matches_any_with_number",
			json: `42`,
			union: struct {
				A *any `json:",inline"`
			}{},
			expected: float64(42),
		},
		{
			name: "matches_any_array",
			json: `[1, "hello", 3]`,
			union: struct {
				A *[]any `json:",inline"`
			}{},
			expected: []any{float64(1), "hello", float64(3)},
		},
		{
			name: "prefers_specific_type_over_any",
			json: `"hello"`,
			union: struct {
				A *any    `json:",inline"`
				B *string `json:",inline"`
			}{},
			expected: "hello",
		},
		{
			name: "prefers_struct_over_any",
			json: `{"a":"x","b":"y"}`,
			union: struct {
				A *any       `json:",inline"`
				B *TwoFields `json:",inline"`
			}{},
			expected: TwoFields{A: "x", B: "y"},
		},
		// "Superset trap": Small vs Large where Large has a required field.
		// When the required field is absent, Small should win.
		{
			name: "superset_trap_missing_required_prefers_small",
			json: `{"id":1,"name":"Alice"}`,
			union: struct {
				Small *SupersetSmall `json:",inline"`
				Large *SupersetLarge `json:",inline"`
			}{},
			expected: SupersetSmall{ID: 1, Name: "Alice"},
		},
		// When the required field is present, Large should win (more fields matched).
		{
			name: "superset_trap_all_present_prefers_large",
			json: `{"id":1,"name":"Alice","email":"a@b.com"}`,
			union: struct {
				Small *SupersetSmall `json:",inline"`
				Large *SupersetLarge `json:",inline"`
			}{},
			expected: SupersetLarge{ID: 1, Name: "Alice", Email: "a@b.com"},
		},
		// "Nested union resolution": outer union where one variant contains an inner union.
		{
			name: "nested_union_recursive_scoring",
			json: `{"kind":"test","data":{"a":"hello","c":42}}`,
			union: struct {
				Obj  *NestedOuterObj  `json:",inline"`
				Flat *NestedOuterFlat `json:",inline"`
			}{},
			expected: NestedOuterObj{Kind: "test", Data: NestedInnerUnion{OfB: &NestedInnerB{A: "hello", C: 42}}},
		},
		// U = T[] | V[], V = O[] | string, O is superset of T.
		// Flat array of simple objects → []T wins (elements are objects, not arrays/strings).
		{
			name:     "nested_array_union_flat_objects_match_T_array",
			json:     `[{"name":"a"},{"name":"b"}]`,
			union:    UnionU{},
			expected: []ObjT{{Name: "a"}, {Name: "b"}},
		},
		// Array of arrays → []V wins, each inner array is O[].
		{
			name:  "nested_array_union_array_of_arrays_match_V_array",
			json:  `[[{"name":"a","extra":"x"}],[{"name":"b","extra":"y"}]]`,
			union: UnionU{},
			expected: []UnionV{
				{OfOArray: []ObjO{{Name: "a", Extra: "x"}}},
				{OfOArray: []ObjO{{Name: "b", Extra: "y"}}},
			},
		},
		// Mixed array: [O[], string] → []V wins.
		{
			name:  "nested_array_union_mixed_array_and_string_match_V_array",
			json:  `[[{"name":"a","extra":"x"}],"hello"]`,
			union: UnionU{},
			expected: []UnionV{
				{OfOArray: []ObjO{{Name: "a", Extra: "x"}}},
				{OfString: ptr("hello")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			union := reflect.New(reflect.TypeOf(tt.union)).Interface()
			err := UnmarshalUnion([]byte(tt.json), union, &unmarshalinfo.Metadata{})
			if tt.wantErr {
				if err != nil {
					return
				}
				t.Fatalf("expected error but got none")
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			value := getUnionValue(t, union)
			assertEqual(t, tt.expected, value)
		})
	}
}

func getUnionValue(t *testing.T, union any) any {
	t.Helper()
	v := reflect.ValueOf(union)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsZero() {
			if field.Kind() == reflect.Pointer {
				return field.Elem().Interface()
			}
			return field.Interface()
		}
	}
	t.Fatal("no non-nil field found in union")
	return nil
}

func TestUnmarshalDiscriminatedUnion(t *testing.T) {
	type TypeA struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type TypeB struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	type TypeC struct {
		Kind string `json:"kind"`
		Data string `json:"data"`
	}
	type TypeD struct {
		Kind   string `json:"kind"`
		Number int    `json:"number"`
	}
	type InnerStruct struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	type NestedType struct {
		Type  string      `json:"type"`
		Inner InnerStruct `json:"inner"`
	}
	type WithOptional struct {
		Type     string  `json:"type"`
		Required string  `json:"required"`
		Optional *string `json:"optional,omitempty"`
	}

	tests := []struct {
		name          string
		json          string
		discriminator string
		union         any
		expected      any
		wantErr       bool
	}{
		{
			name:          "matches_type_a",
			json:          `{"type":"a","value":"hello"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
				B *TypeB `json:",inline,omitzero" discriminator:"b"`
			}{},
			expected: TypeA{Type: "a", Value: "hello"},
		},
		{
			name:          "matches_type_b",
			json:          `{"type":"b","count":42}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
				B *TypeB `json:",inline,omitzero" discriminator:"b"`
			}{},
			expected: TypeB{Type: "b", Count: 42},
		},
		{
			name:          "missing_discriminator_field",
			json:          `{"value":"hello"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
			}{},
			wantErr: true,
		},
		{
			// Unknown discriminator value → forward-compat soft decode:
			// no error, union stays at its zero value. Raw bytes are
			// preserved via metadata.RawJSON() (see TestDiscriminatedUnionUnknownVariant).
			name:          "unknown_discriminator_value",
			json:          `{"type":"unknown","value":"hello"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
				B *TypeB `json:",inline,omitzero" discriminator:"b"`
			}{},
			// expected intentionally unset — union should be all-zero.
		},
		{
			name:          "different_discriminator_field_name",
			json:          `{"kind":"c","data":"test"}`,
			discriminator: "kind",
			union: &struct {
				C *TypeC `json:",inline,omitzero" discriminator:"c"`
				D *TypeD `json:",inline,omitzero" discriminator:"d"`
			}{},
			expected: TypeC{Kind: "c", Data: "test"},
		},
		{
			name:          "discriminator_with_numeric_value",
			json:          `{"kind":"d","number":100}`,
			discriminator: "kind",
			union: &struct {
				C *TypeC `json:",inline,omitzero" discriminator:"c"`
				D *TypeD `json:",inline,omitzero" discriminator:"d"`
			}{},
			expected: TypeD{Kind: "d", Number: 100},
		},
		{
			name:          "nested_struct_in_discriminated_type",
			json:          `{"type":"nested","inner":{"a":"x","b":"y"}}`,
			discriminator: "type",
			union: &struct {
				A      *TypeA      `json:",inline,omitzero" discriminator:"a"`
				Nested *NestedType `json:",inline,omitzero" discriminator:"nested"`
			}{},
			expected: NestedType{Type: "nested", Inner: InnerStruct{A: "x", B: "y"}},
		},
		{
			name:          "empty_string_discriminator_value",
			json:          `{"type":"","value":"empty"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:""`
			}{},
			expected: TypeA{Type: "", Value: "empty"},
		},
		{
			name:          "null_discriminator_value",
			json:          `{"type":null,"value":"test"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
			}{},
			wantErr: true,
		},
		{
			name:          "discriminator_value_is_number",
			json:          `{"type":123,"value":"test"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
			}{},
			wantErr: true,
		},
		{
			// Empty union struct → no variant field possibly matches.
			// Same forward-compat behavior as unknown_discriminator_value.
			name:          "empty_union_struct",
			json:          `{"type":"a","value":"test"}`,
			discriminator: "type",
			union:         &struct{}{},
		},
		{
			name:          "with_optional_field_present",
			json:          `{"type":"opt","required":"req","optional":"opt"}`,
			discriminator: "type",
			union: &struct {
				Opt *WithOptional `json:",inline,omitzero" discriminator:"opt"`
			}{},
			expected: WithOptional{Type: "opt", Required: "req", Optional: ptr("opt")},
		},
		{
			name:          "with_optional_field_absent",
			json:          `{"type":"opt","required":"req"}`,
			discriminator: "type",
			union: &struct {
				Opt *WithOptional `json:",inline,omitzero" discriminator:"opt"`
			}{},
			expected: WithOptional{Type: "opt", Required: "req", Optional: nil},
		},
		{
			name:          "extra_fields_ignored",
			json:          `{"type":"a","value":"test","extra":"ignored"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
			}{},
			expected: TypeA{Type: "a", Value: "test"},
		},
		{
			name:          "unicode_discriminator_value",
			json:          `{"type":"日本語","value":"test"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"日本語"`
			}{},
			expected: TypeA{Type: "日本語", Value: "test"},
		},
		{
			name:          "invalid_json",
			json:          `{invalid}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
			}{},
			wantErr: true,
		},
		{
			name:          "discriminator_at_end_of_object",
			json:          `{"value":"hello","type":"a"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
				B *TypeB `json:",inline,omitzero" discriminator:"b"`
			}{},
			expected: TypeA{Type: "a", Value: "hello"},
		},
		{
			// Discriminator values are case-sensitive. "A" doesn't match "a".
			// Forward-compat: no error, union stays zero.
			name:          "case_sensitive_discriminator_value",
			json:          `{"type":"A","value":"test"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
			}{},
		},
		{
			name:          "case_sensitive_discriminator_value_match",
			json:          `{"type":"A","value":"test"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"A"`
			}{},
			expected: TypeA{Type: "A", Value: "test"},
		},
		{
			name:          "whitespace_in_discriminator_value",
			json:          `{"type":" a ","value":"test"}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:" a "`
			}{},
			expected: TypeA{Type: " a ", Value: "test"},
		},
		{
			name:          "zero_count_in_type_b",
			json:          `{"type":"b","count":0}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
				B *TypeB `json:",inline,omitzero" discriminator:"b"`
			}{},
			expected: TypeB{Type: "b", Count: 0},
		},
		{
			name:          "negative_count_in_type_b",
			json:          `{"type":"b","count":-5}`,
			discriminator: "type",
			union: &struct {
				A *TypeA `json:",inline,omitzero" discriminator:"a"`
				B *TypeB `json:",inline,omitzero" discriminator:"b"`
			}{},
			expected: TypeB{Type: "b", Count: -5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalDiscriminatedUnion(
				[]byte(tt.json),
				tt.discriminator,
				tt.union,
				&unmarshalinfo.Metadata{},
			)
			if tt.wantErr {
				if err != nil {
					return
				}
				t.Fatal("expected error but got none")
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expected == nil {
				// Forward-compat no-variant case: skip the
				// getUnionValue check (which would fatal on an empty union).
				return
			}
			value := getUnionValue(t, tt.union)
			assertEqual(t, tt.expected, value)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

var internalType = reflect.TypeOf(unmarshalinfo.Metadata{})

// ── Union of discriminated unions ────────────────────────────────
// A non-discriminated outer union whose members are themselves
// discriminated unions. Each inner matches on a "type" field with
// a disjoint set of values:
//
//   ({type:foo}|{type:baz}) | ({type:xxx}|{type:yyy})
//
// Given input {type:xxx}, the outer must pick the second branch —
// the inner whose discriminator actually matched.

type fooPayload struct {
	Type string `json:"type"`
	Body string `json:"body"`
}
type bazPayload struct {
	Type string `json:"type"`
	Body string `json:"body"`
}
type xxxPayload struct {
	Type string `json:"type"`
	Body string `json:"body"`
}
type yyyPayload struct {
	Type string `json:"type"`
	Body string `json:"body"`
}

type innerFooBazUnion struct {
	Foo *fooPayload `json:",inline,omitzero" discriminator:"foo"`
	Baz *bazPayload `json:",inline,omitzero" discriminator:"baz"`

	metadata `api:"union"`
}

func (u *innerFooBazUnion) UnmarshalJSON(raw []byte) error {
	return UnmarshalDiscriminatedUnion(raw, "type", u, &u.metadata)
}

type innerXxxYyyUnion struct {
	Xxx *xxxPayload `json:",inline,omitzero" discriminator:"xxx"`
	Yyy *yyyPayload `json:",inline,omitzero" discriminator:"yyy"`

	metadata `api:"union"`
}

func (u *innerXxxYyyUnion) UnmarshalJSON(raw []byte) error {
	return UnmarshalDiscriminatedUnion(raw, "type", u, &u.metadata)
}

type outerUnionOfUnions struct {
	OfFooBaz *innerFooBazUnion `json:",inline,omitzero"`
	OfXxxYyy *innerXxxYyyUnion `json:",inline,omitzero"`

	metadata `api:"union"`
}

func (u *outerUnionOfUnions) UnmarshalJSON(raw []byte) error {
	return UnmarshalUnion(raw, u, &u.metadata)
}

func TestUnmarshalUnionOfDiscriminatedUnions(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantFoo bool // expect OfFooBaz.Foo populated, OfXxxYyy nil
		wantXxx bool // expect OfXxxYyy.Xxx populated, OfFooBaz nil
	}{
		{"xxx matches the second branch", `{"type":"xxx","body":"hi"}`, false, true},
		{"foo matches the first branch", `{"type":"foo","body":"hi"}`, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u outerUnionOfUnions
			if err := u.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if tt.wantFoo {
				if u.OfFooBaz == nil || u.OfFooBaz.Foo == nil {
					t.Fatalf("expected OfFooBaz.Foo populated; got %+v", u)
				}
				if u.OfXxxYyy != nil {
					t.Errorf("expected OfXxxYyy nil when foo matched; got %+v", u.OfXxxYyy)
				}
				if got := u.OfFooBaz.Foo.Body; got != "hi" {
					t.Errorf("OfFooBaz.Foo.Body: want %q, got %q", "hi", got)
				}
			}
			if tt.wantXxx {
				if u.OfXxxYyy == nil || u.OfXxxYyy.Xxx == nil {
					t.Fatalf("expected OfXxxYyy.Xxx populated; got %+v", u)
				}
				if u.OfFooBaz != nil {
					t.Errorf("expected OfFooBaz nil when xxx matched; got %+v", u.OfFooBaz)
				}
				if got := u.OfXxxYyy.Xxx.Body; got != "hi" {
					t.Errorf("OfXxxYyy.Xxx.Body: want %q, got %q", "hi", got)
				}
			}
		})
	}
}

// TestDiscriminatedUnionUnknownVariant pins the forward-compat
// contract for a discriminator value no variant matches:
//  1. Unmarshal succeeds; no variant field is populated.
//  2. The raw bytes remain accessible via RawJSON().
//  3. Re-marshaling errors with "no union members set".
func TestDiscriminatedUnionUnknownVariant(t *testing.T) {
	input := `{"type":"unknown","account_id":"12345"}`

	var u innerFooBazUnion
	if err := u.UnmarshalJSON([]byte(input)); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if u.Foo != nil || u.Baz != nil {
		t.Errorf("expected no variant set; got Foo=%v Baz=%v", u.Foo, u.Baz)
	}
	if got := string(u.RawJSON()); got != input {
		t.Errorf("RawJSON(): want %q, got %q", input, got)
	}
	if _, err := MarshalUnionStruct(u); err == nil {
		t.Errorf("expected re-marshal to error; got nil")
	}
}

// deepEqualIgnoringInternal compares two values, skipping apidata.Internal fields.
func deepEqualIgnoringInternal(a, b reflect.Value) bool {
	if !a.IsValid() && !b.IsValid() {
		return true
	}
	if !a.IsValid() || !b.IsValid() {
		return false
	}
	if a.Type() != b.Type() {
		return false
	}

	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() && b.IsNil() {
			return true
		}
		if a.IsNil() || b.IsNil() {
			return false
		}
		return deepEqualIgnoringInternal(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			if a.Type().Field(i).Type == internalType {
				continue
			}
			if !deepEqualIgnoringInternal(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !deepEqualIgnoringInternal(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Interface:
		if a.IsNil() && b.IsNil() {
			return true
		}
		if a.IsNil() || b.IsNil() {
			return false
		}
		return deepEqualIgnoringInternal(a.Elem(), b.Elem())
	default:
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

func assertEqual[T any](t *testing.T, expected, actual T) {
	t.Helper()
	if !deepEqualIgnoringInternal(reflect.ValueOf(expected), reflect.ValueOf(actual)) {
		t.Fatalf("expected:\n  %#v\nbut got:\n  %#v", expected, actual)
	}
}

// White-box tests verifying that the reflection-based decoder correctly
// populates score fields through the full UnmarshalRoot → custom UnmarshalJSON path.
func TestDecodeScoreThroughCustomUnmarshaler(t *testing.T) {
	t.Run("perfect_match_scores", func(t *testing.T) {
		var obj ObjO
		err := Unmarshal([]byte(`{"name":"a","extra":"b"}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := *obj.UnmarshalState()
		if !s.Succeeded {
			t.Fatal("expected Succeeded=true")
		}
		if s.FieldsMatched != 2 {
			t.Fatalf("expected FieldsMatched=2, got %d", s.FieldsMatched)
		}
		if s.UnknownFields != 0 {
			t.Fatalf("expected UnknownFields=0, got %d", s.UnknownFields)
		}
		if s.UnmatchedTargetFields != 0 {
			t.Fatalf("expected UnmatchedTargetFields=0, got %d", s.UnmatchedTargetFields)
		}
	})

	t.Run("unknown_fields_counted", func(t *testing.T) {
		var obj ObjT
		err := Unmarshal([]byte(`{"name":"a","surprise":"b"}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := *obj.UnmarshalState()
		if s.FieldsMatched != 1 {
			t.Fatalf("expected FieldsMatched=1, got %d", s.FieldsMatched)
		}
		if s.UnknownFields != 1 {
			t.Fatalf("expected UnknownFields=1, got %d", s.UnknownFields)
		}
	})

	t.Run("missing_required_fields_counted", func(t *testing.T) {
		var obj SupersetLarge // has email as required
		err := Unmarshal([]byte(`{"id":1,"name":"Alice"}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := *obj.UnmarshalState()
		if s.MissingRequiredFields != 1 {
			t.Fatalf("expected MissingRequiredFields=1, got %d", s.MissingRequiredFields)
		}
		if s.MatchedRequiredFields != 0 {
			t.Fatalf("expected MatchedRequiredFields=0, got %d", s.MatchedRequiredFields)
		}
	})

	t.Run("matched_required_fields_counted", func(t *testing.T) {
		var obj SupersetLarge
		err := Unmarshal([]byte(`{"id":1,"name":"Alice","email":"a@b.com"}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := *obj.UnmarshalState()
		if s.MissingRequiredFields != 0 {
			t.Fatalf("expected MissingRequiredFields=0, got %d", s.MissingRequiredFields)
		}
		if s.MatchedRequiredFields != 1 {
			t.Fatalf("expected MatchedRequiredFields=1, got %d", s.MatchedRequiredFields)
		}
	})

	t.Run("unmatched_target_fields_counted", func(t *testing.T) {
		var obj ObjO // has name + extra
		err := Unmarshal([]byte(`{"name":"a"}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := *obj.UnmarshalState()
		if s.UnmatchedTargetFields != 1 {
			t.Fatalf("expected UnmatchedTargetFields=1, got %d", s.UnmatchedTargetFields)
		}
	})

	t.Run("raw_bytes_stored", func(t *testing.T) {
		var obj ObjT
		input := []byte(`{"name":"test"}`)
		err := Unmarshal(input, &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(obj.RawJSON()) != string(input) {
			t.Fatalf("expected Raw=%s, got %s", input, obj.RawJSON())
		}
	})

	t.Run("nested_struct_scores_bubble_up", func(t *testing.T) {
		var obj NestedOuterObj
		err := Unmarshal([]byte(`{"kind":"x","data":{"a":"y","c":1}}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := *obj.UnmarshalState()
		if !s.Succeeded {
			t.Fatal("expected Succeeded=true")
		}
		// Outer struct: kind + data = 2, inner struct (via UnmarshalState): a + c = 2, total = 4
		if s.FieldsMatched != 4 {
			t.Fatalf("expected outer FieldsMatched=4, got %d", s.FieldsMatched)
		}
	})

	t.Run("union_winner_has_correct_raw", func(t *testing.T) {
		var u NestedInnerUnion
		input := []byte(`{"a":"hello","c":42}`)
		err := Unmarshal(input, &u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.OfB == nil {
			t.Fatal("expected OfB to be set")
		}
		if string(u.OfB.RawJSON()) != string(input) {
			t.Fatalf("expected inner Raw=%s, got %s", input, u.OfB.RawJSON())
		}
	})

	t.Run("unmarshal_state_after_unmarshal", func(t *testing.T) {
		var obj ObjO
		err := Unmarshal([]byte(`{"name":"a","extra":"b"}`), &obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		score := obj.UnmarshalState()
		if !score.Succeeded {
			t.Fatal("expected score Succeeded=true")
		}
		if score.FieldsMatched != 2 {
			t.Fatalf("expected FieldsMatched=2, got %d", score.FieldsMatched)
		}
	})
}

// Interface union types for registry tests
type Animal interface {
	isAnimal()
}

type Dog struct {
	Breed string `json:"breed"`
	metadata
}

func (d Dog) isAnimal() {}
func (d *Dog) UnmarshalJSON(raw []byte) error {
	type shadow Dog
	return UnmarshalRoot(raw, (*shadow)(d), &d.metadata)
}

type Cat struct {
	Indoor bool `json:"indoor"`
	metadata
}

func (c Cat) isAnimal() {}
func (c *Cat) UnmarshalJSON(raw []byte) error {
	type shadow Cat
	return UnmarshalRoot(raw, (*shadow)(c), &c.metadata)
}

type Fish struct {
	Species   string `json:"species"`
	Saltwater bool   `json:"saltwater"`
	metadata
}

func (f Fish) isAnimal() {}
func (f *Fish) UnmarshalJSON(raw []byte) error {
	type shadow Fish
	return UnmarshalRoot(raw, (*shadow)(f), &f.metadata)
}

type Zoo struct {
	Name   string `json:"name"`
	Mascot Animal `json:"mascot"`
	metadata
}

func (z *Zoo) UnmarshalJSON(raw []byte) error {
	type shadow Zoo
	return UnmarshalRoot(raw, (*shadow)(z), &z.metadata)
}
