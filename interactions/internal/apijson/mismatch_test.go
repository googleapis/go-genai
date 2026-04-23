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

// Mismatch tests — when a known struct field can't accept the JSON
// value (e.g. null into a non-nullable string), the raw bytes are
// recorded in DynamicFields as apidata.Mismatch so the caller can detect
// what went wrong instead of silently seeing a zero value.
//
// Mismatch is intentionally *not* re-marshaled — the struct field's
// own value is emitted. This prevents a bad inbound value from
// silently flowing back out on the next request.

import (
	"strings"
	"testing"

	"google.golang.org/genai/interactions/internal/apijson"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
)

type mismatchSimple struct {
	MyString string `json:"mystring"`

	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

func (r mismatchSimple) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

func (r *mismatchSimple) UnmarshalJSON(raw []byte) error {
	return apijson.UnmarshalRoot(raw, r, &r.meta)
}

func TestMismatchNullIntoString(t *testing.T) {
	var obj mismatchSimple
	if err := apijson.Unmarshal([]byte(`{"mystring":null}`), &obj); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if obj.MyString != "" {
		t.Errorf("MyString should be unchanged by null, got %q", obj.MyString)
	}

	v, ok := obj.DynamicFields["mystring"]
	if !ok {
		t.Fatalf("expected DynamicFields[\"mystring\"] to be set, got: %v", obj.DynamicFields)
	}
	mm, ok := v.(apidata.Mismatch)
	if !ok {
		t.Fatalf("expected apidata.Mismatch, got %T (%v)", v, v)
	}
	if string(mm) != "null" {
		t.Errorf("expected Mismatch(%q), got Mismatch(%q)", "null", mm)
	}
}

// TestMismatchNotReMarshaled — a Mismatch entry in DynamicFields must
// never leak back out. Re-marshaling a struct whose DynamicFields
// contains Mismatch should emit only the struct field's own value,
// not the original (invalid) raw bytes.
func TestMismatchNotReMarshaled(t *testing.T) {
	t.Run("mismatch from unmarshal round-trips cleanly", func(t *testing.T) {
		var obj mismatchSimple
		if err := apijson.Unmarshal([]byte(`{"mystring":null}`), &obj); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		// Sanity check — Mismatch was recorded.
		if _, ok := obj.DynamicFields["mystring"].(apidata.Mismatch); !ok {
			t.Fatalf("expected Mismatch in DynamicFields, got: %v", obj.DynamicFields)
		}

		got, err := apijson.Marshal(obj)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		// Output should be the struct field's own value ("") — *not*
		// null, and *not* anything Mismatch-shaped.
		if strings.Contains(string(got), "null") {
			t.Errorf("Mismatch bytes leaked into output: %s", got)
		}
		if want := `{"mystring":""}`; string(got) != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	})

	t.Run("mismatch does not override a non-zero struct field", func(t *testing.T) {
		// User-constructed mismatch: the struct field has a real value,
		// but someone put a Mismatch in DynamicFields for the same key.
		// The struct field should win; Mismatch is inert.
		obj := mismatchSimple{
			MyString:      "real",
			DynamicFields: apidata.DynamicFields{"mystring": apidata.Mismatch("null")},
		}

		got, err := apijson.Marshal(obj)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}

		if want := `{"mystring":"real"}`; string(got) != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	})
}
