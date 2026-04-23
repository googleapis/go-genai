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

package unmarshalinfo

import "slices"

// ExpectConstant penalizes the score if val does not equal expected.
func ExpectConstant[T comparable](data *Metadata, val T, expected T) {
	if val != expected {
		data.score.InvalidConstants += 1
	}
}

// PreferConstant is a weaker [ExpectConstant]: *val == zero(T) also skips.
func PreferConstant[T comparable](data *Metadata, val *T, expected T) {
	var zero T
	if val != nil && *val != zero && *val != expected {
		data.score.InvalidConstants += 1
	}
}

// ExpectEnum penalizes the score if val is not in allowed.
func ExpectEnum[T comparable](data *Metadata, val T, allowed ...T) {
	if !slices.Contains(allowed, val) {
		data.score.InvalidEnums += 1
	}
}

// PreferEnum is a weaker [ExpectEnum]: *val == zero(T) also skips.
func PreferEnum[T comparable](data *Metadata, val *T, allowed ...T) {
	var zero T
	if val != nil && *val != zero && !slices.Contains(allowed, *val) {
		data.score.InvalidEnums += 1
	}
}
