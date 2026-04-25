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

// EDIT(begin): Logic for scoring decodings
package apijson

// This file defines some logic for evaluating the score of different
// unmarshalings, to let you judge between which unmarshalings are better than
// others. This is an addition on top of the Go standard decoding logic.

import (
	"google.golang.org/genai/interactions/internal/apijson/unmarshalscore"
)

// UnmarshalWithScore parses the JSON-encoded data and stores the result
// in the value pointed to by v, returning a DecodeScore that indicates
// how well the JSON matched the target type.
func UnmarshalWithScore(data []byte, v any) (unmarshalscore.Score, error) {
	var d decodeState
	err := checkValid(data, &d.scan)
	if err != nil {
		return unmarshalscore.Score{}, err
	}

	d.init(data)
	score := unmarshalscore.Score{}
	d.score = &score
	err = d.unmarshal(v)
	if err != nil {
		return unmarshalscore.Score{Succeeded: false}, err
	}
	score.Succeeded = true
	return score, err
}

// unmarshalRootWithScore is like UnmarshalWithScore but skips the
// UnmarshalJSON interface check at the root level.
func unmarshalRootWithScore(data []byte, v any) (unmarshalscore.Score, error) {
	var d decodeState
	err := checkValid(data, &d.scan)
	if err != nil {
		return unmarshalscore.Score{}, err
	}

	d.init(data)
	score := unmarshalscore.Score{}
	d.score = &score
	err = d.unmarshalRoot(v)
	if err != nil {
		return unmarshalscore.Score{Succeeded: false}, err
	}
	score.Succeeded = true
	return score, err
}

// EDIT(end)
