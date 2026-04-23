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

package apiform

import (
	"reflect"
	"strings"
)

const apiStructTag = "api"
const jsonStructTag = "json"
const formStructTag = "form"
const formatStructTag = "format"
const defaultStructTag = "default"

type parsedStructTag struct {
	name         string
	required     bool
	inline       bool
	omitzero     bool
	defaultValue any
}

func parseFormStructTag(field reflect.StructField) (tag parsedStructTag, ok bool) {
	raw, ok := field.Tag.Lookup(formStructTag)
	if !ok {
		raw, ok = field.Tag.Lookup(jsonStructTag)
	}
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
		case "inline":
			tag.inline = true
		case "omitzero":
			tag.omitzero = true
		}
	}

	parseApiStructTag(field, &tag)
	parseDefaultStructTag(field, &tag)
	return tag, ok
}

func parseDefaultStructTag(field reflect.StructField, tag *parsedStructTag) {
	if field.Type.Kind() != reflect.String {
		// Only strings are currently supported
		return
	}

	raw, ok := field.Tag.Lookup(defaultStructTag)
	if !ok {
		return
	}
	tag.defaultValue = raw
}

func parseApiStructTag(field reflect.StructField, tag *parsedStructTag) {
	raw, ok := field.Tag.Lookup(apiStructTag)
	if !ok {
		return
	}
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		switch part {
		case "required":
			tag.required = true
		}
	}
}

func parseFormatStructTag(field reflect.StructField) (format string, ok bool) {
	format, ok = field.Tag.Lookup(formatStructTag)
	return format, ok
}
