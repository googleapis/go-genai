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
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/packages/apidata"
	"reflect"
	"testing"
)

type object struct {
	Name                  string `json:"name"`
	apidata.DynamicFields `json:"-"`
	meta                  unmarshalinfo.Metadata
}

type noExtras struct {
	Name string `json:"name"`
}

func TestExtraFieldsIndex(t *testing.T) {
	idx := unmarshalinfo.DynamicFieldsIndex(reflect.TypeOf(object{}))
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
}

func TestExtraFieldsIndexNoEmbedding(t *testing.T) {
	idx := unmarshalinfo.DynamicFieldsIndex(reflect.TypeOf(noExtras{}))
	if idx != nil {
		t.Fatalf("expected nil, got %v", idx)
	}
}

func TestExtraFieldsIndexPointer(t *testing.T) {
	idx := unmarshalinfo.DynamicFieldsIndex(reflect.TypeOf(&object{}))
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
}
