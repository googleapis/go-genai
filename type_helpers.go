// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genai

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type int64sFromStringSlice []int64

func (s *int64sFromStringSlice) UnmarshalJSON(data []byte) error {
	var stringSlice []string
	if err := json.Unmarshal(data, &stringSlice); err != nil {
		// If both attempts fail, return a more informative error
		return fmt.Errorf("failed to unmarshal as []int64 or []string: %w", err)
	}

	// If successful as a []string, convert each element to int64
	result := make([]int64, 0, len(stringSlice))
	for _, str := range stringSlice {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err // Error during string-to-int conversion
		}
		result = append(result, val)
	}

	*s = result
	return nil
}

func (s int64sFromStringSlice) MarshalJSON() ([]byte, error) {
	stringSlice := make([]string, 0, len(s))
	for _, val := range s {
		stringSlice = append(stringSlice, strconv.FormatInt(val, 10))
	}
	return json.Marshal(stringSlice)
}
