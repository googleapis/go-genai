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

package unmarshalinfo_test

import (
	"testing"

	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
)

func TestExpectAndPreferConstant(t *testing.T) {
	var m unmarshalinfo.Metadata

	a, empty := "a", ""

	unmarshalinfo.ExpectConstant(&m, a, "a") // match → no penalty
	if m.UnmarshalState().InvalidConstants != 0 {
		t.Error("ExpectConstant match should not penalize")
	}

	unmarshalinfo.ExpectConstant(&m, a, "b") // mismatch → penalty
	if m.UnmarshalState().InvalidConstants != 1 {
		t.Errorf("expected 1, got %d", m.UnmarshalState().InvalidConstants)
	}

	unmarshalinfo.PreferConstant(&m, &empty, "b") // zero value → skip
	if m.UnmarshalState().InvalidConstants != 1 {
		t.Error("PreferConstant zero value should not penalize")
	}

	unmarshalinfo.PreferConstant(&m, nil, "b") // nil → skip
	if m.UnmarshalState().InvalidConstants != 1 {
		t.Error("PreferConstant nil should not penalize")
	}

	unmarshalinfo.PreferConstant(&m, &a, "b") // non-zero mismatch → penalty
	if m.UnmarshalState().InvalidConstants != 2 {
		t.Errorf("expected 2, got %d", m.UnmarshalState().InvalidConstants)
	}
}

func TestExpectAndPreferEnum(t *testing.T) {
	var m unmarshalinfo.Metadata

	x, z, empty := "x", "z", ""

	unmarshalinfo.ExpectEnum(&m, x, "x", "y") // member → no penalty
	if m.UnmarshalState().InvalidEnums != 0 {
		t.Error("ExpectEnum match should not penalize")
	}

	unmarshalinfo.ExpectEnum(&m, z, "x", "y") // non-member → penalty
	if m.UnmarshalState().InvalidEnums != 1 {
		t.Errorf("expected 1, got %d", m.UnmarshalState().InvalidEnums)
	}

	unmarshalinfo.PreferEnum[string](&m, nil, "x", "y") // nil → skip
	if m.UnmarshalState().InvalidEnums != 1 {
		t.Error("PreferEnum nil should not penalize")
	}

	unmarshalinfo.PreferEnum(&m, &empty, "x", "y") // zero value → skip
	if m.UnmarshalState().InvalidEnums != 1 {
		t.Error("PreferEnum zero value should not penalize")
	}

	unmarshalinfo.PreferEnum(&m, &z, "x", "y") // non-zero non-member → penalty
	if m.UnmarshalState().InvalidEnums != 2 {
		t.Errorf("expected 2, got %d", m.UnmarshalState().InvalidEnums)
	}
}
