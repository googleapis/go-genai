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

package apidata

// DynamicFields is embedded in a struct to override the marshaling
// behavior of fields.
//
//	type MyStruct struct {
//		Field1 string `json:"field1"`
//		Field2 string `json:"field2"`
//		Field3 int 	  `json:"field3"`
//
//		apidata.DynamicFields `json:"-"`
//	}
//
// When marshaling, [DynamicFields] can customize the marshaled data.
//
//	myStruct := MyStruct{
//		Field1: "value1",
//		DynamicFields: apidata.DynamicFields{
//			"field2": map[string]any{"over":"ride"},
//			"extraField": "extraValue",
//			"field3": apidata.Omit,
//		},
//	}
//
//	{"field1":"value1","field2":{"over":"ride"},"extraField":"extraValue"}
//
// When unmarshaling, unknown fields are stored as [Unknown] values.
// When unmarshaling, fields that incorrectly collide with struct fields are stored as [Mismatch] values.
// For example, if a string was expected, but 'null' was received. Then you'd have a Mismatch('null') for that field.
//
// For security reasons, [Mismatch] are skipped when marshaling.
type DynamicFields map[string]any

// Omit as an [DynamicFields] value will remove the field from the marshaled data.
// See [DynamicFields] for usage examples.
var Omit = omit{}

type omit struct{}

// Unknown holds the raw JSON bytes of an extra field produced by unmarshaling into [DynamicFields].
type Unknown []byte

// Mismatch holds the raw JSON bytes of a field whose value didn't match its
// struct field's declared type during unmarshaling (e.g. a string field that
// received `null`). For security reasons, [Mismatch] are skipped when marshaling.
type Mismatch []byte
