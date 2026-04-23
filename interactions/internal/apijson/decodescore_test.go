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
	"fmt"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalscore"
	"reflect"
	"testing"
)

func TestUnmarshalPrecedence(t *testing.T) {
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
	type InnerA struct {
		X string `json:"x"`
		Y string `json:"y"`
	}
	type InnerB struct {
		X string `json:"x"`
	}
	type OuterA struct {
		Name  string `json:"name"`
		Inner InnerA `json:"inner"`
	}
	type OuterB struct {
		Name  string `json:"name"`
		Inner InnerB `json:"inner"`
	}

	tests := []struct {
		name     string
		json     string
		variants []any
		expected any
	}{
		// Basic type matching
		{
			name: "matches_int_over_string",
			json: `42`,
			variants: []any{
				ptrTo[int](),
				ptrTo[string](),
			},
			expected: 42,
		},
		{
			name: "matches_string_over_int",
			json: `"hello"`,
			variants: []any{
				ptrTo[int](),
				ptrTo[string](),
			},
			expected: "hello",
		},
		{
			name: "matches_bool",
			json: `true`,
			variants: []any{
				ptrTo[int](),
				ptrTo[bool](),
			},
			expected: true,
		},
		// Struct matching
		{
			name: "prefers_more_fields",
			json: `{"a":"x","b":"y"}`,
			variants: []any{
				ptrTo[OneField](),
				ptrTo[TwoFields](),
			},
			expected: TwoFields{A: "x", B: "y"},
		},
		{
			name: "prefers_not_missing_fields",
			json: `{"a":"x"}`,
			variants: []any{
				ptrTo[OneField](),
				ptrTo[TwoFields](),
			},
			expected: OneField{A: "x"},
		},
		{
			name: "prefers_exact_match_over_superset",
			json: `{"a":"1","b":"2"}`,
			variants: []any{
				ptrTo[ThreeFields](),
				ptrTo[TwoFields](),
			},
			expected: TwoFields{A: "1", B: "2"},
		},
		// Required field handling
		{
			name: "prefers_required_field_match",
			json: `{"name":"test"}`,
			variants: []any{
				ptrTo[WithoutRequired](),
				ptrTo[WithRequired](),
			},
			expected: WithRequired{Name: "test"},
		},
		{
			name: "required_field_missing_loses",
			json: `{"name":"test"}`,
			variants: []any{
				ptrTo[WithRequiredExtra](),
				ptrTo[WithoutRequired](),
			},
			expected: WithoutRequired{Name: "test"},
		},
		{
			name: "more_matches_beats_required_fields",
			json: `{"id":"123","name":"n","value":"v"}`,
			variants: []any{
				ptrTo[WithRequiredID](),
				ptrTo[WithManyFields](),
			},
			expected: WithManyFields{ID: "123", Name: "n", Value: "v"},
		},
		// Arrays and slices
		{
			name: "matches_array_of_floats_when_needed",
			json: `[1, 2.5, 3]`,
			variants: []any{
				ptrTo[[]int](),
				ptrTo[[]float64](),
			},
			expected: []float64{1, 2.5, 3},
		},
		// Maps
		{
			name: "prefers_struct_to_map",
			json: `{"a":"x"}`,
			variants: []any{
				ptrTo[map[string]string](),
				ptrTo[OneField](),
			},
			expected: OneField{A: "x"},
		},
		{
			name: "prefers_map_to_struct_with_missing_fields",
			json: `{"a":"x","b":"y"}`,
			variants: []any{
				ptrTo[OneField](),
				ptrTo[map[string]string](),
			},
			expected: map[string]string{"a": "x", "b": "y"},
		},
		// Nested structs
		{
			name: "matches_nested_struct",
			json: `{"outer":"o","inner":{"a":"x","b":"y"}}`,
			variants: []any{
				ptrTo[OneField](),
				ptrTo[NestedStruct](),
			},
			expected: NestedStruct{Outer: "o", Inner: InnerStruct{A: "x", B: "y"}},
		},
		{
			name: "nested_struct_prefers_better_inner_match_innerA",
			json: `{"name":"test","inner":{"x":"1","y":"2"}}`,
			variants: []any{
				ptrTo[OuterB](),
				ptrTo[OuterA](),
			},
			expected: OuterA{Name: "test", Inner: InnerA{X: "1", Y: "2"}},
		},
		{
			name: "nested_struct_prefers_better_inner_match_innerB",
			json: `{"name":"test","inner":{"x":"1"}}`,
			variants: []any{
				ptrTo[OuterA](),
				ptrTo[OuterB](),
			},
			expected: OuterB{Name: "test", Inner: InnerB{X: "1"}},
		},
		// Multiple valid matches - should pick best
		{
			name: "multiple_structs_picks_best_match",
			json: `{"a":"1","b":"2","c":"3"}`,
			variants: []any{
				ptrTo[OneField](),
				ptrTo[TwoFields](),
				ptrTo[ThreeFields](),
			},
			expected: ThreeFields{A: "1", B: "2", C: "3"},
		},
		// Variant order should not matter
		{
			name: "variant_order_does_not_matter",
			json: `{"a":"x","b":"y"}`,
			variants: []any{
				ptrTo[TwoFields](),
				ptrTo[OneField](),
			},
			expected: TwoFields{A: "x", B: "y"},
		},
		// any type
		{
			name: "prefers_specific_type_over_any",
			json: `"hello"`,
			variants: []any{
				ptrTo[any](),
				ptrTo[string](),
			},
			expected: "hello",
		},
		{
			name: "prefers_struct_over_any",
			json: `{"a":"x","b":"y"}`,
			variants: []any{
				ptrTo[any](),
				ptrTo[TwoFields](),
			},
			expected: TwoFields{A: "x", B: "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := make([]unmarshalscore.Score, 0, len(tt.variants))
			for _, v := range tt.variants {
				score, _ := UnmarshalWithScore([]byte(tt.json), v)
				scores = append(scores, score)
			}
			valuePtr, err := getBestScoring(tt.variants, scores)
			if err != nil {
				t.Logf("Failed to match with any variant: %s", tt.json)
				t.Fatal(err)
			}
			value := reflect.ValueOf(valuePtr).Elem().Interface()
			if !reflect.DeepEqual(tt.expected, value) {
				t.Errorf("expected %#v but got %#v", tt.expected, value)
				for i, variant := range tt.variants {
					score := scores[i]
					t.Errorf("%v: %+v", reflect.TypeOf(variant).Elem(), score)
				}
				t.FailNow()
			}
		})
	}
}

type HasAdditionalProperties struct {
	Name   string         `json:"name"`
	Extras map[string]any `json:",inline"`
}

func TestUnmarshalAdditionalProperties(t *testing.T) {
	json := `{"name":"test","extra1":"value1","extra2":"value2"}`
	var result HasAdditionalProperties
	score, err := UnmarshalWithScore([]byte(json), &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("expected Name to be 'test', got %q", result.Name)
	}
	if result.Extras == nil {
		t.Fatal("expected Extras to be populated")
	}
	if result.Extras["extra1"] != "value1" {
		t.Errorf("expected Extras['extra1'] to be 'value1', got %v", result.Extras["extra1"])
	}
	if result.Extras["extra2"] != "value2" {
		t.Errorf("expected Extras['extra2'] to be 'value2', got %v", result.Extras["extra2"])
	}
	if !score.Succeeded {
		t.Error("expected Succeeded to be true")
	}
	if score.FieldsMatched != 3 {
		t.Errorf("expected FieldsMatched to be 3, got %d", score.FieldsMatched)
	}
}

type WithCustomUnmarshaler struct {
	Name string `json:"name"`
	metadata
}

func (w *WithCustomUnmarshaler) UnmarshalJSON(raw []byte) error {
	type shadow WithCustomUnmarshaler
	return UnmarshalRoot(raw, (*shadow)(w), &w.metadata)
}

type HoldsCustomUnmarshaler struct {
	WithCustom WithCustomUnmarshaler `json:"with_custom"`
	OtherField string                `json:"other"`
}

func TestCustomUnmarshalScore(t *testing.T) {
	var h HoldsCustomUnmarshaler
	score, err := UnmarshalWithScore([]byte(`{"with_custom":{"name":"x"},"other":"y"}`), &h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The inner struct matches 1 field (name), outer matches 2 fields (with_custom + other).
	// Inner score is added to outer via the custom unmarshaler path.
	if score.FieldsMatched != 3 {
		t.Errorf("expected FieldsMatched to be 3, got %d", score.FieldsMatched)
	}
	if h.WithCustom.Name != "x" {
		t.Errorf("expected Name to be 'x', got %q", h.WithCustom.Name)
	}
}

func ptrTo[T any]() *T {
	return new(T)
}

func getBestScoring(variants []any, scores []unmarshalscore.Score) (any, error) {
	bestScore := unmarshalscore.Score{}
	bestIndex := -1
	for i, score := range scores {
		if score.IsBetterThan(bestScore) {
			bestScore = score
			bestIndex = i
		}
	}
	if bestIndex == -1 {
		return nil, fmt.Errorf("no valid value found")
	}
	return variants[bestIndex], nil
}
