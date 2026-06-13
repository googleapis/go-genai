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

import (
	"fmt"
	"google.golang.org/genai/interactions/internal/apijson"
	"google.golang.org/genai/interactions/packages/apidata"
	"testing"
)

type defaultSimple struct {
	Name  string `json:"name" default:"HOUR"`
	Value int    `json:"value" default:"42"`

	apidata.DynamicFields `json:"-"`
}

func (r defaultSimple) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

// customString implements MarshalJSON to wrap in angle brackets.
type customString struct {
	Val string
}

func (c customString) MarshalJSON() ([]byte, error) {
	return []byte(`"(` + c.Val + `)"`), nil
}

func (c customString) IsZero() bool {
	return c.Val == ""
}

type defaultCustom struct {
	Tag customString `json:"tag" default:"fallback"`

	apidata.DynamicFields `json:"-"`
}

func (r defaultCustom) MarshalJSON() ([]byte, error) {
	return apijson.MarshalRoot(r)
}

func TestDefaultStringZeroUsesDefault(t *testing.T) {
	got, err := apijson.Marshal(defaultSimple{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"name":"HOUR","value":42}`
	if string(got) != want {
		t.Fatalf("expected %s, got %s", want, string(got))
	}
}

func TestDefaultNonZeroIgnoresDefault(t *testing.T) {
	got, err := apijson.Marshal(defaultSimple{Name: "custom", Value: 99})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"name":"custom","value":99}`
	if string(got) != want {
		t.Fatalf("expected %s, got %s", want, string(got))
	}
}

func TestDefaultWithCustomMarshalJSON(t *testing.T) {
	// Zero value — should the default kick in or the custom marshaler?
	got, err := apijson.Marshal(defaultCustom{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	fmt.Printf("=== custom MarshalJSON + default, zero value ===\n")
	fmt.Printf("got: %s\n", got)

	// Non-zero — custom marshaler should be used, default ignored.
	got2, err := apijson.Marshal(defaultCustom{Tag: customString{Val: "hello"}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	fmt.Printf("=== custom MarshalJSON + default, non-zero ===\n")
	fmt.Printf("got: %s\n", got2)
	if string(got2) != `{"tag":"(hello)"}` {
		t.Fatalf("expected custom marshaler output, got %s", got2)
	}
}
