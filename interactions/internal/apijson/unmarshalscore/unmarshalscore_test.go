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

package unmarshalscore

import (
	"testing"
)

func TestDecodeScoreIsBetterThan(t *testing.T) {
	tests := []struct {
		name     string
		score    Score
		other    Score
		expected bool
	}{
		{
			name:     "fewer missing required fields is better",
			score:    Score{MissingRequiredFields: 0, FieldsMatched: 1},
			other:    Score{MissingRequiredFields: 1, FieldsMatched: 1},
			expected: true,
		},
		{
			name:     "more matched fields is better",
			score:    Score{FieldsMatched: 3},
			other:    Score{FieldsMatched: 2},
			expected: true,
		},
		{
			name:     "fewer unknown fields is better",
			score:    Score{FieldsMatched: 2, UnknownFields: 0},
			other:    Score{FieldsMatched: 2, UnknownFields: 1},
			expected: true,
		},
		{
			name:     "fewer unmatched target fields is better",
			score:    Score{FieldsMatched: 2, UnmatchedTargetFields: 0},
			other:    Score{FieldsMatched: 2, UnmatchedTargetFields: 1},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.score.IsBetterThan(tt.other)
			if result != tt.expected {
				t.Errorf("IsBetterThan() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
