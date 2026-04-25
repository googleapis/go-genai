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

package unmarshalinfo

import (
	"google.golang.org/genai/interactions/packages/apidata"
	"reflect"
	"slices"
	"strings"
	"sync"
)

var extraFieldsIdxCache sync.Map // map[reflect.Type][]int
var isUnionCache sync.Map        // map[reflect.Type]bool

var extraFieldsType = reflect.TypeFor[apidata.DynamicFields]()
var metadataType = reflect.TypeFor[Metadata]()

// DynamicFieldsIndex returns the struct field index for the embedded
// [apidata.DynamicFields] field, or nil if the type doesn't have one.
// Results are cached per type.
//
// Panics if the type is not a struct.
func DynamicFieldsIndex(t reflect.Type) []int {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	if cached, ok := extraFieldsIdxCache.Load(t); ok {
		idx, _ := cached.([]int)
		return idx
	}

	var idx []int
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == extraFieldsType {
			idx = field.Index
			break
		}
	}
	extraFieldsIdxCache.Store(t, idx)
	return idx
}

// IsUnion reports whether the given type is a union struct, indicated by
// an Internal field tagged with api:"union".
func IsUnion(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	if cached, ok := isUnionCache.Load(t); ok {
		return cached.(bool)
	}

	var isUnion bool

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type != metadataType {
			continue
		}

		if raw, ok := field.Tag.Lookup("api"); ok {
			isUnion = slices.Contains(strings.Split(raw, ","), "union")
		}
	}

	isUnionCache.Store(t, isUnion)
	return isUnion
}
