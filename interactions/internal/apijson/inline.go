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

// EDIT(begin): inline field helpers for encoding and decoding
package apijson

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
)

// inlineField returns the first field marked with the "inline" tag option,
// or false if no such field exists. Used by both the encoder (to spread
// inline map entries) and the decoder (to capture unknown fields).
func (s structFields) inlineField() (field, bool) {
	for _, f := range s.list {
		if f.inline {
			return f, true
		}
	}
	return field{}, false
}

// encodeInlineMapEntries spreads the entries of an inline map field as
// sibling JSON keys. next is the separator byte ('{' if no fields have
// been written yet, ',' otherwise) and is returned updated.
func encodeInlineMapEntries(e *encodeState, se structEncoder, v reflect.Value, opts encOpts, next byte) byte {
	f, ok := se.fields.inlineField()
	if !ok {
		return next
	}

	fv := v
	for _, i := range f.index {
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				return next
			}
			fv = fv.Elem()
		}
		fv = fv.Field(i)
	}

	if fv.Kind() != reflect.Map || fv.IsNil() || fv.Len() == 0 {
		return next
	}

	// Sort keys for deterministic output.
	sv := make([]reflectWithString, fv.Len())
	mi := fv.MapRange()
	for i := 0; mi.Next(); i++ {
		ks, err := resolveKeyName(mi.Key())
		if err != nil {
			e.error(fmt.Errorf("json: encoding error for inline map key: %w", err))
		}
		sv[i] = reflectWithString{ks: ks, v: mi.Value()}
	}
	slices.SortFunc(sv, func(a, b reflectWithString) int {
		return strings.Compare(a.ks, b.ks)
	})

	elemEnc := typeEncoder(fv.Type().Elem())
	for _, kv := range sv {
		e.WriteByte(next)
		next = ','
		e.Write(appendString(e.AvailableBuffer(), kv.ks, opts.escapeHTML))
		e.WriteByte(':')
		elemEnc(e, kv.v, opts)
	}
	return next
}

// isTypedMap reports whether the map has a concrete element type (not any/interface{}).
func isTypedMap(m reflect.Value) bool {
	elemType := m.Type().Elem()
	return elemType.Kind() != reflect.Interface || elemType.NumMethod() > 0
}

// routeDeferredInlineValue routes a value that was decoded as any into
// either the typed inline map (if the value fits) or DynamicFields as
// apidata.Unknown (if not). Returns true if the value was stored in
// the inline map.
func routeDeferredInlineValue(key []byte, decoded reflect.Value, additionalProps reflect.Value, extraFields reflect.Value) bool {
	elemType := additionalProps.Type().Elem()
	if decoded.Elem().IsValid() && decoded.Elem().Type().AssignableTo(elemType) {
		kt := additionalProps.Type().Key()
		kv := reflect.New(kt).Elem()
		kv.SetString(string(key))
		additionalProps.SetMapIndex(kv, decoded.Elem().Convert(elemType))
		return true
	}
	storeAsUnknown(extraFields, string(key), decoded)
	return false
}

// storeAsUnknown re-marshals a decoded any value to JSON bytes and stores
// it in the DynamicFields map as apidata.Unknown, preserving the original JSON
// representation through round-trips.
func storeAsUnknown(extraFields reflect.Value, key string, decoded reflect.Value) {
	raw, err := Marshal(decoded.Interface())
	if err != nil {
		return
	}
	extraFields.SetMapIndex(
		reflect.ValueOf(key),
		reflect.ValueOf(apidata.Unknown(raw)),
	)
}

// ── extraFieldRouter ────────────────────────────────────────────────
//
// extraFieldRouter handles routing of JSON fields that don't match any
// struct field. It decides whether each value goes into the inline map
// (additional properties) or DynamicFields, based on type compatibility.
//
// Each JSON field falls into one of these categories:
//
//  1. Known field, correct type  — handled by the vendored encoding/json
//  2. Known field, wrong type    — handled by the vendored encoding/json
//  3. Additional property, fits  — stored in the inline map
//  4. Additional property, wrong type — falls back to DynamicFields
//  5. Unknown field (no inline map) — stored in DynamicFields as apidata.Unknown
//
// The router handles cases 3-5. The decoder skips decoding for unknown
// fields entirely — it just captures the raw JSON bytes and passes them
// to Route():
//
//	router := newExtraFieldRouter(fields, v, t)
//
//	// Per unknown field in the decode loop:
//	valStart := d.readIndex()
//	d.value(subv)  // subv is invalid, scanner advances without decoding
//	result := router.Route(string(key), d.data[valStart:d.readIndex()])
type extraFieldRouter struct {
	additionalProps reflect.Value // the inline map, e.g. map[string]string
	extraFields     reflect.Value // the DynamicFields map (map[string]any)
}

// extraFieldResult tells the caller what happened so it can update scoring.
type extraFieldResult int

const (
	extraFieldNone    extraFieldResult = iota // no routing happened
	extraFieldMatched                         // stored in inline map (case 3)
	extraFieldUnknown                         // stored in DynamicFields (cases 4/5)
	extraFieldSkipped                         // no container, value dropped
)

// newExtraFieldRouter creates a router by resolving the inline map and
// DynamicFields from the struct value. Returns a zero-value router (Active()
// returns false) if neither container exists.
func newExtraFieldRouter(fields structFields, v reflect.Value, t reflect.Type) extraFieldRouter {
	var r extraFieldRouter

	if v.Kind() != reflect.Struct {
		return r
	}

	// Resolve inline map.
	if f, ok := fields.inlineField(); ok {
		ap := v
		for _, i := range f.index {
			if ap.Kind() == reflect.Pointer {
				if ap.IsNil() {
					if !ap.CanSet() {
						ap = reflect.Value{}
						break
					}
					ap.Set(reflect.New(ap.Type().Elem()))
				}
				ap = ap.Elem()
			}
			ap = ap.Field(i)
		}
		if ap.IsValid() {
			if ap.IsNil() {
				ap.Set(reflect.MakeMap(ap.Type()))
			}
			r.additionalProps = ap
		}
	}

	// Resolve DynamicFields.
	if idx := unmarshalinfo.DynamicFieldsIndex(t); idx != nil {
		ef := v.FieldByIndex(idx)
		if ef.IsNil() {
			ef.Set(reflect.MakeMap(ef.Type()))
		}
		r.extraFields = ef
	}

	return r
}

// Active reports whether this router has any containers to route into.
func (r *extraFieldRouter) Active() bool {
	return r.additionalProps.IsValid() || r.extraFields.IsValid()
}

// Route takes the raw JSON bytes of an unknown field's value and stores
// it in the right container. If an inline map exists, it tries to
// unmarshal into the map's element type (case 3). If that fails or no
// inline map exists, it stores the raw bytes as apidata.Unknown in
// DynamicFields (cases 4/5).
func (r *extraFieldRouter) Route(key string, rawValue []byte) extraFieldResult {
	// Try the inline map first (cases 3/4).
	if r.additionalProps.IsValid() {
		if r.tryStoreInMap(key, rawValue) {
			return extraFieldMatched
		}
		// Didn't fit — fall through to DynamicFields (case 4).
	}

	// Store in DynamicFields as raw JSON bytes (cases 4/5).
	if r.extraFields.IsValid() {
		raw := make(apidata.Unknown, len(rawValue))
		copy(raw, rawValue)
		r.extraFields.SetMapIndex(
			reflect.ValueOf(key),
			reflect.ValueOf(raw),
		)
		return extraFieldUnknown
	}

	return extraFieldSkipped
}

// tryStoreInMap attempts to unmarshal rawValue into the inline map's
// element type. Returns true if the unmarshal succeeds and the value is
// stored. For map[string]any this always succeeds. For typed maps like
// map[string]string, it fails for non-string JSON values.
func (r *extraFieldRouter) tryStoreInMap(key string, rawValue []byte) bool {
	elemType := r.additionalProps.Type().Elem()

	// JSON null into a non-nullable type (string, int, struct, ...) is a
	// silent no-op in stdlib Unmarshal — it returns nil and leaves the
	// target at its zero value. Treat that as "doesn't fit" so the value
	// falls through to DynamicFields instead of silently becoming "" or 0.
	if bytes.Equal(bytes.TrimSpace(rawValue), nullLiteral) && !elemAcceptsNull(elemType) {
		return false
	}

	target := reflect.New(elemType)
	if err := Unmarshal(rawValue, target.Interface()); err != nil {
		return false
	}
	r.additionalProps.SetMapIndex(
		reflect.ValueOf(key),
		target.Elem(),
	)
	return true
}

// elemAcceptsNull reports whether a JSON null can be meaningfully
// represented in the given Go type (as nil).
func elemAcceptsNull(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return true
	}
	return false
}

// RecordMismatch stores rawValue in DynamicFields under key as
// apidata.Mismatch — used by the decoder when a known struct field
// can't accept the JSON value it received (e.g. null into a string).
// The caller can detect the mismatch instead of seeing only a zero
// value. Mismatches are intentionally not re-marshaled.
//
// No-op if the struct has no DynamicFields embed.
func (r *extraFieldRouter) RecordMismatch(key string, rawValue []byte) {
	if !r.extraFields.IsValid() {
		return
	}
	raw := make(apidata.Mismatch, len(rawValue))
	copy(raw, rawValue)
	r.extraFields.SetMapIndex(
		reflect.ValueOf(key),
		reflect.ValueOf(raw),
	)
}

// IsNullForNonNullable reports whether rawValue is JSON null and the
// given type cannot meaningfully hold null — i.e. stdlib Unmarshal
// would silently leave the target at its zero value rather than
// expressing the null.
func IsNullForNonNullable(rawValue []byte, t reflect.Type) bool {
	return bytes.Equal(bytes.TrimSpace(rawValue), nullLiteral) && !elemAcceptsNull(t)
}

// EDIT(end)
