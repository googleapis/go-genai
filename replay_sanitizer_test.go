// Copyright 2024 Google LLC
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

package genai

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type nestedStruct struct {
	ByteField      []byte            `json:"byteField,omitempty"`
	ByteSliceField [][]byte          `json:"byteSliceField,omitempty"`
	ByteMapField   map[string][]byte `json:"byteMapField,omitempty"`
}

type outerStruct struct {
	PointerField      *nestedStruct   `json:"pointerField,omitempty"`
	StructField       nestedStruct    `json:"structField"`
	SliceField        []nestedStruct  `json:"sliceField,omitempty"`
	SlicePointerField []*nestedStruct `json:"slicePointerField,omitempty"`
	// Recursive types.
	PointerOuterField *outerStruct `json:"pointerOuterField,omitempty"`
}

func TestGetFieldPath(t *testing.T) {
	testCases := []struct {
		targetType    reflect.Type
		expectedPaths []string
	}{
		{
			targetType:    reflect.TypeOf([]byte{}),
			expectedPaths: []string{"pointerField.byteField", "structField.byteField", "[]sliceField.byteField", "[]slicePointerField.byteField", "pointerOuterField.pointerField.byteField", "pointerOuterField.structField.byteField", "pointerOuterField.[]sliceField.byteField", "pointerOuterField.[]slicePointerField.byteField"},
		},
		{
			targetType:    reflect.TypeOf([][]byte{}),
			expectedPaths: []string{"pointerField.byteSliceField", "structField.byteSliceField", "[]sliceField.byteSliceField", "[]slicePointerField.byteSliceField", "pointerOuterField.pointerField.byteSliceField", "pointerOuterField.structField.byteSliceField", "pointerOuterField.[]sliceField.byteSliceField", "pointerOuterField.[]slicePointerField.byteSliceField"},
		},
		{
			targetType:    reflect.TypeOf(map[string][]byte{}),
			expectedPaths: []string{"pointerField.byteMapField", "structField.byteMapField", "[]sliceField.byteMapField", "[]slicePointerField.byteMapField", "pointerOuterField.pointerField.byteMapField", "pointerOuterField.structField.byteMapField", "pointerOuterField.[]sliceField.byteMapField", "pointerOuterField.[]slicePointerField.byteMapField"},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.targetType), func(t *testing.T) {
			paths := make([]string, 0)
			visitedTypes := make(map[string]bool)

			_ = getFieldPath(reflect.TypeOf(outerStruct{}), tc.targetType, &paths, "", visitedTypes, false)
			if diff := cmp.Diff(paths, tc.expectedPaths); diff != "" {
				t.Errorf("path mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSanitizeMapWithSourceType(t *testing.T) {
	testCases := []struct {
		input     map[string]any
		sanitized map[string]any
	}{
		// URL Base64. Convert to Std Base64.
		{
			input:     map[string]any{"structField": map[string]any{"byteField": "6L-Z5piv5LiA"}},
			sanitized: map[string]any{"structField": map[string]any{"byteField": "6L+Z5piv5LiA"}},
		},
		// Std Base64. No conversion.
		{
			input:     map[string]any{"structField": map[string]any{"byteField": "6L+Z5piv5LiA"}},
			sanitized: map[string]any{"structField": map[string]any{"byteField": "6L+Z5piv5LiA"}},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
			sanitizeMapWithSourceType(t, reflect.TypeOf(outerStruct{}), tc.input)
			if diff := cmp.Diff(tc.input, tc.sanitized); diff != "" {
				t.Errorf("path mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSanitizeMapByPath(t *testing.T) {
	sanitizer := func(data any, path string) any {
		return "sanitized"
	}
	testCases := []struct {
		input     map[string]any
		path      string
		sanitized map[string]any
	}{
		// Sanitize success
		{
			input:     map[string]any{"k1": "v1"},
			path:      "k1",
			sanitized: map[string]any{"k1": "sanitized"},
		},
		{
			input:     map[string]any{"k1": map[string]any{"k2": "v2"}},
			path:      "k1.k2",
			sanitized: map[string]any{"k1": map[string]any{"k2": "sanitized"}},
		},
		{
			input:     map[string]any{"k1": []any{"v1", "v1"}},
			path:      "[]k1",
			sanitized: map[string]any{"k1": []any{"sanitized", "sanitized"}},
		},
		{
			input:     map[string]any{"k1": []map[string]any{{"k2": "v2"}, {"k2": "v2"}}},
			path:      "[]k1.k2",
			sanitized: map[string]any{"k1": []map[string]any{{"k2": "sanitized"}, {"k2": "sanitized"}}},
		},
		{
			input:     map[string]any{"k1": map[string]any{"k2": []any{"v2", "v2"}}},
			path:      "k1.[]k2",
			sanitized: map[string]any{"k1": map[string]any{"k2": []any{"sanitized", "sanitized"}}},
		},
		// Path name mismatch, no sanitize
		{
			input:     map[string]any{"k1": "v1"},
			path:      "wrongPath",
			sanitized: map[string]any{"k1": "v1"},
		},
		{
			input:     map[string]any{"k1": map[string]any{"k2": "v2"}},
			path:      "k1.wrongPath",
			sanitized: map[string]any{"k1": map[string]any{"k2": "v2"}},
		},
		{
			input:     map[string]any{"k1": []any{"v1", "v1"}},
			path:      "[]wrongPath",
			sanitized: map[string]any{"k1": []any{"v1", "v1"}},
		},
		{
			input:     map[string]any{"k1": []map[string]any{{"k2": "v2"}, {"k2": "v2"}}},
			path:      "[]wrongPath.k2",
			sanitized: map[string]any{"k1": []map[string]any{{"k2": "v2"}, {"k2": "v2"}}},
		},
		{
			input:     map[string]any{"k1": map[string]any{"k2": []string{"v2", "v2"}}},
			path:      "k1.[]wrongPath",
			sanitized: map[string]any{"k1": map[string]any{"k2": []string{"v2", "v2"}}},
		},
		// Path type misatch, no sanitize
		{
			input:     map[string]any{"k1": []any{"v1", "v1"}},
			path:      "k1.wrongPath",
			sanitized: map[string]any{"k1": []any{"v1", "v1"}},
		},
		{
			input:     map[string]any{"k1": map[string]any{"k2": "v2"}},
			path:      "k1.[]k2",
			sanitized: map[string]any{"k1": map[string]any{"k2": "v2"}},
		},
		{
			input:     map[string]any{"k1": map[string]any{"k2": map[string]any{"k3": "v3"}}},
			path:      "k1.[]k2.k3",
			sanitized: map[string]any{"k1": map[string]any{"k2": map[string]any{"k3": "v3"}}},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
			sanitizeMapByPath(tc.input, tc.path, sanitizer, false)
			if diff := cmp.Diff(tc.input, tc.sanitized); diff != "" {
				t.Errorf("path mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
