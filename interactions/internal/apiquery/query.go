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

package apiquery

import (
	"net/url"
	"reflect"
	"time"
)

func MarshalWithSettings(value any, settings QuerySettings) (url.Values, error) {
	e := encoder{time.RFC3339, true, settings}
	kv := url.Values{}
	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return nil, nil
	}
	typ := val.Type()

	pairs, err := e.typeEncoder(typ)("", val)
	if err != nil {
		return nil, err
	}
	for _, pair := range pairs {
		kv.Add(pair.key, pair.value)
	}
	return kv, nil
}

func Marshal(value any) (url.Values, error) {
	return MarshalWithSettings(value, QuerySettings{})
}

type Queryer interface {
	URLQuery() (url.Values, error)
}

type QuerySettings struct {
	NestedFormat NestedQueryFormat
	ArrayFormat  ArrayQueryFormat
}

type NestedQueryFormat int

const (
	NestedQueryFormatBrackets NestedQueryFormat = iota
	NestedQueryFormatDots
)

type ArrayQueryFormat int

const (
	ArrayQueryFormatComma ArrayQueryFormat = iota
	ArrayQueryFormatRepeat
	ArrayQueryFormatIndices
	ArrayQueryFormatBrackets
)
