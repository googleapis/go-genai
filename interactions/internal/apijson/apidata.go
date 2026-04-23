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

// EDIT(begin): marshal APIData
package apijson

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"

	"github.com/tidwall/sjson"
)

// sjsonPathEscaper escapes the metacharacters that sjson/gjson paths
// interpret: `.` (nested path), `*` / `?` (wildcards), `#` (array
// op), `|` (modifier), and `\` (escape). A DynamicFields key that
// contains any of these would otherwise be silently reshaped or
// dropped when spliced via sjson.Set*/DeleteBytes.
var sjsonPathEscaper = strings.NewReplacer(
	`\`, `\\`,
	`.`, `\.`,
	`*`, `\*`,
	`?`, `\?`,
	`#`, `\#`,
	`|`, `\|`,
)

func escapeSjsonKey(k string) string { return sjsonPathEscaper.Replace(k) }

// MarshalRoot marshals an object to JSON, using the [apidata.DynamicFields] pattern.
//
// Unlike [Marshal], this skips the [Marshaler] interface check at the root level,
// so it can be called from within a type's MarshalJSON without infinite recursion.
func MarshalRoot(obj any) ([]byte, error) {
	result, err := marshalRoot(obj)
	if err != nil {
		return nil, err
	}

	DynamicFieldsIdx := unmarshalinfo.DynamicFieldsIndex(reflect.TypeOf(obj))
	if DynamicFieldsIdx != nil {
		if reflect.ValueOf(obj).FieldByIndex(DynamicFieldsIdx).IsZero() {
			return result, nil
		}

		extras, ok := reflect.ValueOf(obj).FieldByIndex(DynamicFieldsIdx).Interface().(apidata.DynamicFields)
		if !ok {
			return result, nil
		}

		// Handle [apidata.ExtraFields] overrides
		if len(extras) == 0 {
			return result, nil
		}

		keys := make([]string, 0, len(extras))
		for k := range extras {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := extras[k]

			// [apidata.Mismatch] shouldn't be marshaled for security reasons.
			if _, mismatch := v.(apidata.Mismatch); mismatch {
				continue
			}

			// sjson rejects empty paths ("path cannot be empty").
			// Just skip empty keys for now
			if k == "" {
				continue
			}

			path := escapeSjsonKey(k)
			if v == apidata.Omit {
				result, err = sjson.DeleteBytes(result, path)
			} else if raw, ok := v.(apidata.Unknown); ok {
				result, err = sjson.SetRawBytes(result, path, []byte(raw))
			} else {
				result, err = sjson.SetBytes(result, path, v)
			}
			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// MarshalUnionStruct marshals the first non-zero member of the struct
func MarshalUnionStruct(union any) ([]byte, error) {
	v := reflect.ValueOf(union)
	// De-ref pointers
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("union must be a struct")
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		if !fieldType.IsExported() || !field.IsValid() || field.IsZero() {
			continue
		}
		return Marshal(field.Interface())
	}
	return nil, fmt.Errorf("no union members set")
}

// UnmarshalRoot unmarshals raw JSON bytes into the target, skipping the
// UnmarshalJSON interface check at the root level so it can be called from
// within a type's UnmarshalJSON without infinite recursion.
//
// UnmarshalRoot respects the [apidata.DynamicFields] pattern and the [unmarshalinfo.Metadata] pattern.
func UnmarshalRoot(raw []byte, target any, data *unmarshalinfo.Metadata) error {
	score, err := unmarshalRootWithScore(raw, target)
	if err == nil {
		unmarshalinfo.SetUnmarshalState(raw, score, data)
	}
	return err
}

// EDIT(end): marshal APIData
