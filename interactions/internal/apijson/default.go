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
	"reflect"
	"strconv"
)

// continueWithDefault wraps an encoder so that zero values are replaced with
// the default before encoding. The inner encoder handles quoting and escaping.
func continueWithDefault(field *field, k encoderFunc) encoderFunc {
	dv, err := parseDefaultTag(field)
	return func(e *encodeState, v reflect.Value, opts encOpts) {
		if v.IsZero() {
			if err != nil {
				e.error(err)
				return
			}
			k(e, dv, opts)
			return
		}
		k(e, v, opts)
	}
}

// parseDefaultTag gets called during construction of the encoders
func parseDefaultTag(field *field) (reflect.Value, error) {
	typ, defaultValue := field.typ, field.defaultValue
	dv := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.String:
		dv.SetString(defaultValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(defaultValue, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("json: invalid default %q for int field: %w", defaultValue, err)
		}
		dv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(defaultValue, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("json: invalid default %q for uint field: %w", defaultValue, err)
		}
		dv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("json: invalid default %q for float field: %w", defaultValue, err)
		}
		dv.SetFloat(n)
	case reflect.Bool:
		dv.SetBool(defaultValue == "true")
	}
	return dv, nil
}
