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

package apiquery_test

import (
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/internal/apiquery"
	"net/url"
	"testing"
	"time"
)

func P[T any](v T) *T { return &v }

type Primitives struct {
	A bool    `query:"a"`
	B int     `query:"b"`
	C uint    `query:"c"`
	D float64 `query:"d"`
	E float32 `query:"e"`
	F []int   `query:"f"`
}

type PrimitivePointers struct {
	A *bool    `query:"a"`
	B *int     `query:"b"`
	C *uint    `query:"c"`
	D *float64 `query:"d"`
	E *float32 `query:"e"`
	F *[]int   `query:"f"`
}

type Slices struct {
	Slice []Primitives `query:"slices"`
	Mixed []any        `query:"mixed"`
}

type DateTime struct {
	Date     time.Time `query:"date" format:"date"`
	DateTime time.Time `query:"date-time" format:"date-time"`
}

type AdditionalProperties struct {
	A      bool           `query:"a"`
	Extras map[string]any `query:"-,inline"`
}

type Recursive struct {
	Name  string     `query:"name"`
	Child *Recursive `query:"child"`
}

type UnknownStruct struct {
	Unknown any `query:"unknown"`
}

type UnionStruct struct {
	Union Union `query:"union" format:"date"`
}

type Union interface {
	union()
}

type UnionInteger int64

func (UnionInteger) union() {}

type UnionString string

func (UnionString) union() {}

type UnionStructA struct {
	Type string `query:"type"`
	A    string `query:"a"`
	B    string `query:"b"`
}

func (UnionStructA) union() {}

type UnionStructB struct {
	Type string `query:"type"`
	A    string `query:"a"`
}

func (UnionStructB) union() {}

type UnionTime time.Time

func (UnionTime) union() {}

type DeeplyNested struct {
	A DeeplyNested1 `query:"a"`
}

type DeeplyNested1 struct {
	B DeeplyNested2 `query:"b"`
}

type DeeplyNested2 struct {
	C DeeplyNested3 `query:"c"`
}

type DeeplyNested3 struct {
	D *string `query:"d"`
}

type QueryOmitTest struct {
	A *string `query:"a,omitzero"`
	B string  `query:"b,omitzero"`
}

type NamedEnum string

const NamedEnumFoo NamedEnum = "foo"

type StructUnionWrapper struct {
	Union StructUnion `query:"union"`
}

type StructUnion struct {
	OfInt    *int64                 `query:",omitzero,inline"`
	OfString *string                `query:",omitzero,inline"`
	OfEnum   *NamedEnum             `query:",omitzero,inline"`
	OfA      UnionStructA           `query:",omitzero,inline"`
	OfB      UnionStructB           `query:",omitzero,inline"`
	meta     unmarshalinfo.Metadata `api:"union"`
}

var tests = map[string]struct {
	enc      string
	val      any
	settings apiquery.QuerySettings
}{
	"primitives": {
		"a=false&b=237628372683&c=654&d=9999.43&e=43.7599983215332&f=1,2,3,4",
		Primitives{A: false, B: 237628372683, C: uint(654), D: 9999.43, E: 43.76, F: []int{1, 2, 3, 4}},
		apiquery.QuerySettings{},
	},

	"slices_brackets": {
		`mixed[]=1&mixed[]=2.3&mixed[]=hello&slices[][a]=false&slices[][a]=false&slices[][b]=237628372683&slices[][b]=237628372683&slices[][c]=654&slices[][c]=654&slices[][d]=9999.43&slices[][d]=9999.43&slices[][e]=43.7599983215332&slices[][e]=43.7599983215332&slices[][f][]=1&slices[][f][]=2&slices[][f][]=3&slices[][f][]=4&slices[][f][]=1&slices[][f][]=2&slices[][f][]=3&slices[][f][]=4`,
		Slices{
			Slice: []Primitives{
				{A: false, B: 237628372683, C: uint(654), D: 9999.43, E: 43.76, F: []int{1, 2, 3, 4}},
				{A: false, B: 237628372683, C: uint(654), D: 9999.43, E: 43.76, F: []int{1, 2, 3, 4}},
			},
			Mixed: []any{1, 2.3, "hello"},
		},
		apiquery.QuerySettings{ArrayFormat: apiquery.ArrayQueryFormatBrackets},
	},

	"slices_comma": {
		`mixed=1,2.3,hello`,
		Slices{
			Mixed: []any{1, 2.3, "hello"},
		},
		apiquery.QuerySettings{ArrayFormat: apiquery.ArrayQueryFormatComma},
	},

	"slices_repeat": {
		`mixed=1&mixed=2.3&mixed=hello&slices[a]=false&slices[a]=false&slices[b]=237628372683&slices[b]=237628372683&slices[c]=654&slices[c]=654&slices[d]=9999.43&slices[d]=9999.43&slices[e]=43.7599983215332&slices[e]=43.7599983215332&slices[f]=1&slices[f]=2&slices[f]=3&slices[f]=4&slices[f]=1&slices[f]=2&slices[f]=3&slices[f]=4`,
		Slices{
			Slice: []Primitives{
				{A: false, B: 237628372683, C: uint(654), D: 9999.43, E: 43.76, F: []int{1, 2, 3, 4}},
				{A: false, B: 237628372683, C: uint(654), D: 9999.43, E: 43.76, F: []int{1, 2, 3, 4}},
			},
			Mixed: []any{1, 2.3, "hello"},
		},
		apiquery.QuerySettings{ArrayFormat: apiquery.ArrayQueryFormatRepeat},
	},

	"primitive_pointer_struct": {
		"a=false&b=237628372683&c=654&d=9999.43&e=43.7599983215332&f=1,2,3,4,5",
		PrimitivePointers{
			A: P(false),
			B: P(237628372683),
			C: P(uint(654)),
			D: P(9999.43),
			E: P(float32(43.76)),
			F: &[]int{1, 2, 3, 4, 5},
		},
		apiquery.QuerySettings{},
	},

	"datetime_struct": {
		`date=2006-01-02&date-time=2006-01-02T15:04:05Z`,
		DateTime{
			Date:     time.Date(2006, time.January, 2, 0, 0, 0, 0, time.UTC),
			DateTime: time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
		},
		apiquery.QuerySettings{},
	},

	"additional_properties": {
		`a=true&bar=value&foo=true`,
		AdditionalProperties{
			A: true,
			Extras: map[string]any{
				"bar": "value",
				"foo": true,
			},
		},
		apiquery.QuerySettings{},
	},

	"recursive_struct_brackets": {
		`child[name]=Alex&name=Robert`,
		Recursive{Name: "Robert", Child: &Recursive{Name: "Alex"}},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatBrackets},
	},

	"recursive_struct_dots": {
		`child.name=Alex&name=Robert`,
		Recursive{Name: "Robert", Child: &Recursive{Name: "Alex"}},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatDots},
	},

	"unknown_struct_number": {
		`unknown=12`,
		UnknownStruct{
			Unknown: 12.,
		},
		apiquery.QuerySettings{},
	},

	"unknown_struct_map_brackets": {
		`unknown[foo]=bar`,
		UnknownStruct{
			Unknown: map[string]any{
				"foo": "bar",
			},
		},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatBrackets},
	},

	"unknown_struct_map_dots": {
		`unknown.foo=bar`,
		UnknownStruct{
			Unknown: map[string]any{
				"foo": "bar",
			},
		},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatDots},
	},

	"struct_union_string": {
		`union=hello`,
		StructUnionWrapper{
			Union: StructUnion{OfString: P("hello")},
		},
		apiquery.QuerySettings{},
	},

	"union_string": {
		`union=hello`,
		UnionStruct{
			Union: UnionString("hello"),
		},
		apiquery.QuerySettings{},
	},

	"struct_union_integer": {
		`union=12`,
		StructUnionWrapper{
			Union: StructUnion{OfInt: P[int64](12)},
		},
		apiquery.QuerySettings{},
	},

	"union_integer": {
		`union=12`,
		UnionStruct{
			Union: UnionInteger(12),
		},
		apiquery.QuerySettings{},
	},

	"struct_union_enum": {
		`union=foo`,
		StructUnionWrapper{
			Union: StructUnion{OfEnum: P[NamedEnum](NamedEnumFoo)},
		},
		apiquery.QuerySettings{},
	},

	"struct_union_struct_discriminated_a": {
		`union[a]=foo&union[b]=bar&union[type]=typeA`,
		StructUnionWrapper{
			Union: StructUnion{OfA: UnionStructA{
				Type: "typeA",
				A:    "foo",
				B:    "bar",
			}},
		},
		apiquery.QuerySettings{},
	},

	"union_struct_discriminated_a": {
		`union[a]=foo&union[b]=bar&union[type]=typeA`,
		UnionStruct{
			Union: UnionStructA{
				Type: "typeA",
				A:    "foo",
				B:    "bar",
			},
		},
		apiquery.QuerySettings{},
	},

	"struct_union_struct_discriminated_b": {
		`union[a]=foo&union[type]=typeB`,
		StructUnionWrapper{
			Union: StructUnion{OfB: UnionStructB{
				Type: "typeB",
				A:    "foo",
			}},
		},
		apiquery.QuerySettings{},
	},

	"union_struct_discriminated_b": {
		`union[a]=foo&union[type]=typeB`,
		UnionStruct{
			Union: UnionStructB{
				Type: "typeB",
				A:    "foo",
			},
		},
		apiquery.QuerySettings{},
	},

	"union_struct_time": {
		`union=2010-05-23`,
		UnionStruct{
			Union: UnionTime(time.Date(2010, 05, 23, 0, 0, 0, 0, time.UTC)),
		},
		apiquery.QuerySettings{},
	},

	"deeply_nested_brackets": {
		`a[b][c][d]=hello`,
		DeeplyNested{
			A: DeeplyNested1{
				B: DeeplyNested2{
					C: DeeplyNested3{
						D: P("hello"),
					},
				},
			},
		},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatBrackets},
	},

	"deeply_nested_dots": {
		`a.b.c.d=hello`,
		DeeplyNested{
			A: DeeplyNested1{
				B: DeeplyNested2{
					C: DeeplyNested3{
						D: P("hello"),
					},
				},
			},
		},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatDots},
	},

	"deeply_nested_brackets_empty": {
		``,
		DeeplyNested{
			A: DeeplyNested1{
				B: DeeplyNested2{
					C: DeeplyNested3{
						D: nil,
					},
				},
			},
		},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatBrackets},
	},

	"deeply_nested_dots_empty": {
		``,
		DeeplyNested{
			A: DeeplyNested1{
				B: DeeplyNested2{
					C: DeeplyNested3{
						D: nil,
					},
				},
			},
		},
		apiquery.QuerySettings{NestedFormat: apiquery.NestedQueryFormatDots},
	},

	"query_omit_nil": {
		``,
		QueryOmitTest{
			A: nil,
		},
		apiquery.QuerySettings{},
	},
	"query_omit_set": {
		`a=hello`,
		QueryOmitTest{
			A: P("hello"),
		},
		apiquery.QuerySettings{},
	},
}

func TestEncode(t *testing.T) {
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			values, err := apiquery.MarshalWithSettings(test.val, test.settings)
			if err != nil {
				t.Fatalf("failed to marshal url %s", err)
			}
			str, _ := url.QueryUnescape(values.Encode())
			if str != test.enc {
				t.Fatalf("expected %+#v to serialize to %s but got %s", test.val, test.enc, str)
			}
		})
	}
}
