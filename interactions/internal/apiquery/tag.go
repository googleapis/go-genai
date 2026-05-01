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
	"reflect"
	"strings"
)

const queryStructTag = "query"
const formatStructTag = "format"

type parsedStructTag struct {
	name     string
	omitzero bool
	inline   bool
}

func parseQueryStructTag(field reflect.StructField) (parsedStructTag, bool) {
	var tag parsedStructTag
	raw, ok := field.Tag.Lookup(queryStructTag)
	if !ok {
		return tag, ok
	}
	parts := strings.Split(raw, ",")
	if len(parts) == 0 {
		return tag, false
	}
	tag.name = parts[0]
	for _, part := range parts[1:] {
		switch part {
		case "omitzero":
			tag.omitzero = true
		case "inline":
			tag.inline = true
		}
	}
	return tag, ok
}

func parseFormatStructTag(field reflect.StructField) (format string, ok bool) {
	return field.Tag.Lookup(formatStructTag)
}
