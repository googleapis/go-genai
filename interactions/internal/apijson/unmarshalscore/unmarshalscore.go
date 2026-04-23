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

// Score represents a score for how well a value matched the target type.
// Higher scores indicate better matches.
type Score struct {
	// Succeeded is true for values that have successfully unmarshaled.
	Succeeded bool
	// InvalidConstants is the number of invalid constant values.
	InvalidConstants uint16
	// InvalidEnums is the number of invalid enum values.
	InvalidEnums uint16
	// FieldsMatched is the number of fields that were successfully decoded into the target.
	FieldsMatched uint16
	// MapValuesMatched is the number of map values that have been added.
	MapValuesMatched uint16
	// FieldsTotal is the total number of fields encountered.
	FieldsTotal uint16
	// UnknownFields is the number of fields that had no corresponding field in the target.
	UnknownFields uint16
	// MatchedRequiredFields is the number of required fields that were present.
	MatchedRequiredFields uint16
	// MissingRequiredFields is the number of required fields that were not present.
	MissingRequiredFields uint16
	// UnmatchedTargetFields is the number of target struct fields that were not present in the source.
	UnmatchedTargetFields uint16
}

// IsBetterThan returns true if this score represents a better match than the other score.
// The comparison prioritizes: fewer missing required fields, then more matched fields,
// then fewer type mismatches, then fewer unknown fields, then fewer unmatched target fields.
func (s Score) IsBetterThan(other Score) bool {
	if s.Succeeded && !other.Succeeded {
		return true
	}

	// Invalid constants are the worst
	if s.InvalidConstants != other.InvalidConstants {
		return s.InvalidConstants < other.InvalidConstants
	}

	// Next, compare missing required fields (fewer is better)
	if s.MissingRequiredFields != other.MissingRequiredFields {
		return s.MissingRequiredFields < other.MissingRequiredFields
	}

	// Invalid enums are bad, but sometimes happen
	if s.InvalidEnums != other.InvalidEnums {
		return s.InvalidEnums < other.InvalidEnums
	}

	// Then, compare unknown fields (fewer is better)
	if s.UnknownFields != other.UnknownFields {
		return s.UnknownFields < other.UnknownFields
	}

	// More matched required fields is better
	if s.MatchedRequiredFields != other.MatchedRequiredFields {
		return s.MatchedRequiredFields > other.MatchedRequiredFields
	}

	// Then, compare total matched fields (more is better)
	if s.FieldsMatched != other.FieldsMatched {
		return s.FieldsMatched > other.FieldsMatched
	}

	// Finally, compare unmatched target fields (fewer is better)
	if s.UnmatchedTargetFields != other.UnmatchedTargetFields {
		return s.UnmatchedTargetFields < other.UnmatchedTargetFields
	}

	// Then, compare total map values set (more is better)
	if s.MapValuesMatched != other.MapValuesMatched {
		return s.MapValuesMatched > other.MapValuesMatched
	}

	return false
}

func (d *Score) Add(other Score) {
	if !other.Succeeded {
		return
	}

	d.Succeeded = true
	d.FieldsMatched += other.FieldsMatched
	d.FieldsTotal += other.FieldsTotal
	d.UnknownFields += other.UnknownFields
	d.MatchedRequiredFields += other.MatchedRequiredFields
	d.MissingRequiredFields += other.MissingRequiredFields
	d.UnmatchedTargetFields += other.UnmatchedTargetFields
}

func (s Score) IsGreatMatch() bool {
	return s.MissingRequiredFields == 0 && s.UnknownFields == 0
}

// Prints either "exact", "partial", or "invalid" based on the score.
//
// Mostly to minimize debugging logging size
func (s Score) String() string {
	if !s.Succeeded {
		return "invalid"
	}
	if s.MissingRequiredFields+s.UnknownFields+s.InvalidConstants+s.InvalidEnums == 0 {
		return "exact"
	}
	return "partial"
}
