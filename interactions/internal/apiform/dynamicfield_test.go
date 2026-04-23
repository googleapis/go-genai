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

package apiform_test

import (
	"bytes"
	"google.golang.org/genai/interactions/internal/apiform"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
	"mime/multipart"
	"strings"
	"testing"
)

// ── Types ────────────────────────────────────────────────────────────

type simple struct {
	Name  string `form:"name"`
	Value int    `form:"value"`
	Tag   string `form:"tag"`

	apidata.DynamicFields `json:"-" form:"-"`
}

type nested struct {
	Parent string `form:"parent"`
	Child  simple `form:"child"`

	apidata.DynamicFields `json:"-" form:"-"`
}

type withInlineMap struct {
	Name  string         `form:"name"`
	Extra map[string]any `form:",inline"`

	apidata.DynamicFields `json:"-" form:"-"`
}

type unionWrapper struct {
	Union union `form:"union"`
}

type union struct {
	OfString *string                `form:",omitzero,inline"`
	OfInt    *int                   `form:",omitzero,inline"`
	meta     unmarshalinfo.Metadata `api:"union"`
}

func ptr[T any](v T) *T { return &v }

// ── Test helpers ─────────────────────────────────────────────────────

// tForm wraps testing.T with multipart form assertion helpers.
type tForm struct{ *testing.T }

func (tf tForm) Marshal(val any) formResult {
	tf.Helper()
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	if err := writer.SetBoundary("xxx"); err != nil {
		tf.Fatalf("SetBoundary: %v", err)
	}
	if err := apiform.MarshalWithSettings(val, writer, "indices:dots"); err != nil {
		tf.Fatalf("MarshalWithSettings: %v", err)
	}
	if err := writer.Close(); err != nil {
		tf.Fatalf("Close: %v", err)
	}
	return formResult{tf, buf.String()}
}

func (tf tForm) MarshalErr(val any) error {
	tf.Helper()
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	if err := writer.SetBoundary("xxx"); err != nil {
		tf.Fatalf("SetBoundary: %v", err)
	}
	return apiform.MarshalWithSettings(val, writer, "indices:dots")
}

// formResult holds parsed multipart output for assertions.
type formResult struct {
	tForm
	raw string
}

// fields parses Content-Disposition headers into name→value pairs.
func (r formResult) fields() map[string]string {
	result := make(map[string]string)
	parts := strings.Split(r.raw, "\r\n")
	for i, p := range parts {
		if strings.HasPrefix(p, "Content-Disposition:") {
			nameStart := strings.Index(p, `name="`) + 6
			nameEnd := strings.Index(p[nameStart:], `"`) + nameStart
			if i+2 < len(parts) {
				result[p[nameStart:nameEnd]] = parts[i+2]
			}
		}
	}
	return result
}

func (r formResult) Has(field, value string) {
	r.Helper()
	got, ok := r.fields()[field]
	if !ok {
		r.Errorf("expected field %q present, got:\n%s", field, r.raw)
	} else if got != value {
		r.Errorf("field %q: expected %q, got %q", field, value, got)
	}
}

func (r formResult) Lacks(field string) {
	r.Helper()
	if _, ok := r.fields()[field]; ok {
		r.Errorf("expected field %q absent, got:\n%s", field, r.raw)
	}
}

func (r formResult) Count(field string) int {
	return strings.Count(r.raw, `name="`+field+`"`)
}

// ── Tests ────────────────────────────────────────────────────────────

func TestDynamicFieldsMarshal(t *testing.T) {
	tests := []struct {
		name  string
		val   any
		has   map[string]string
		lacks []string
	}{
		{
			name: "extra field added",
			val:  simple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"extra": "v"}},
			has:  map[string]string{"name": "a", "value": "1", "extra": "v"},
		},
		{
			name:  "Omit removes native field",
			val:   simple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"name": apidata.Omit}},
			has:   map[string]string{"value": "1"},
			lacks: []string{"name"},
		},
		{
			name: "Unknown replaces native field with raw value",
			val:  simple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"name": apidata.Unknown(`42`)}},
			has:  map[string]string{"name": "42", "value": "1"},
		},
		{
			name: "Unknown replaces with empty string",
			val:  simple{Name: "will_disappear", Value: 1, DynamicFields: apidata.DynamicFields{"name": apidata.Unknown(``)}},
			has:  map[string]string{"name": ""},
		},
		{
			name:  "Omit on nonexistent field is no-op",
			val:   simple{Name: "a", DynamicFields: apidata.DynamicFields{"ghost": apidata.Omit}},
			has:   map[string]string{"name": "a"},
			lacks: []string{"ghost"},
		},
		{
			name: "plain value adds a new field",
			val:  simple{Name: "a", Value: 1, DynamicFields: apidata.DynamicFields{"new_field": 99}},
			has:  map[string]string{"name": "a", "value": "1", "new_field": "99"},
		},
		{
			name: "all operations at once",
			val: simple{
				Name: "alice", Value: 42, Tag: "original",
				DynamicFields: apidata.DynamicFields{
					"name":  apidata.Omit,
					"value": apidata.Unknown(`9999`),
					"added": "new_value",
					"ghost": apidata.Omit,
				},
			},
			has:   map[string]string{"value": "9999", "tag": "original", "added": "new_value"},
			lacks: []string{"name", "ghost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := tForm{t}
			r := tf.Marshal(tt.val)
			for field, value := range tt.has {
				r.Has(field, value)
			}
			for _, field := range tt.lacks {
				r.Lacks(field)
			}
		})
	}
}

func TestDynamicFieldsNilVsEmpty(t *testing.T) {
	tf := tForm{t}
	base := simple{Name: "a", Value: 1, Tag: "t"}

	nilR := tf.Marshal(base)
	base.DynamicFields = apidata.DynamicFields{}
	emptyR := tf.Marshal(base)

	if nilR.raw != emptyR.raw {
		t.Fatalf("nil vs empty differ:\nnil:   %q\nempty: %q", nilR.raw, emptyR.raw)
	}
}

func TestDynamicFieldsNestedIsolation(t *testing.T) {
	tf := tForm{t}
	r := tf.Marshal(nested{
		Parent: "p",
		Child: simple{
			Name:          "child_name",
			Value:         10,
			DynamicFields: apidata.DynamicFields{"name": apidata.Omit},
		},
		DynamicFields: apidata.DynamicFields{"injected": "top_level"},
	})

	r.Has("parent", "p")
	r.Has("injected", "top_level")
	r.Has("child.value", "10")
	r.Lacks("child.name")
}

func TestDynamicFieldsWithInlineMap(t *testing.T) {
	t.Run("both contribute fields", func(t *testing.T) {
		tf := tForm{t}
		r := tf.Marshal(withInlineMap{
			Name:          "native",
			Extra:         map[string]any{"from_map": "m"},
			DynamicFields: apidata.DynamicFields{"from_raw": "r"},
		})
		r.Has("name", "native")
		r.Has("from_map", "m")
		r.Has("from_raw", "r")
	})

	t.Run("same key in both produces duplicate", func(t *testing.T) {
		tf := tForm{t}
		r := tf.Marshal(withInlineMap{
			Name:  "native",
			Extra: map[string]any{"overlap": "from_map"},
			DynamicFields: apidata.DynamicFields{
				"overlap": "from_raw",
				"name":    apidata.Omit,
			},
		})
		r.Lacks("name")
		if n := r.Count("overlap"); n < 2 {
			t.Errorf("expected both inline-map and DynamicFields 'overlap', got %d", n)
		}
	})
}

func TestDynamicFieldsUnionMarshal(t *testing.T) {
	tests := []struct {
		name    string
		val     any
		has     map[string]string
		wantErr bool
	}{
		{name: "string member", val: unionWrapper{Union: union{OfString: ptr("hello")}}, has: map[string]string{"union": "hello"}},
		{name: "int member", val: unionWrapper{Union: union{OfInt: ptr(42)}}, has: map[string]string{"union": "42"}},
		{name: "first non-zero wins", val: unionWrapper{Union: union{OfString: ptr("first"), OfInt: ptr(999)}}, has: map[string]string{"union": "first"}},
		{name: "no member set errors", val: unionWrapper{Union: union{}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := tForm{t}
			if tt.wantErr {
				if err := tf.MarshalErr(tt.val); err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			r := tf.Marshal(tt.val)
			for field, value := range tt.has {
				r.Has(field, value)
			}
		})
	}
}
