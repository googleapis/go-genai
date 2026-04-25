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

// EDIT(begin): custom time marshaler
package apijson

import (
	"google.golang.org/genai/interactions/internal/apijson/shims"
	"reflect"
	"time"
)

type TimeMarshaler interface {
	MarshalJSONWithTimeLayout(string) []byte
}

func TimeLayout(fmt string) string {
	switch fmt {
	case "", "date-time":
		return time.RFC3339
	case "date":
		return time.DateOnly
	default:
		return fmt
	}
}

var timeType = shims.TypeFor[time.Time]()

func newTimeEncoder() encoderFunc {
	return func(e *encodeState, v reflect.Value, opts encOpts) {
		t := v.Interface().(time.Time)
		fmtted := t.Format(TimeLayout(opts.format))
		stringEncoder(e, reflect.ValueOf(fmtted), opts)
	}
}

// Uses continuation passing style, to add the format option to k
func continueWithFormat(format string, k encoderFunc) encoderFunc {
	return func(e *encodeState, v reflect.Value, opts encOpts) {
		opts.format = format
		k(e, v, opts)
	}
}

func timeMarshalEncoder(e *encodeState, v reflect.Value, opts encOpts) bool {
	tm, ok := v.Interface().(TimeMarshaler)
	if !ok {
		return false
	}

	b := tm.MarshalJSONWithTimeLayout(opts.format)
	if b != nil {
		e.Grow(len(b))
		out := e.AvailableBuffer()
		out, _ = appendCompact(out, b, opts.escapeHTML)
		e.Buffer.Write(out)
		return true
	}

	return false
}

// EDIT(end)
