// Copyright 2024 Google LLC
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

const (
	roleUser   = "user"
)

// Text returns a slice of Content with a single Part with the given text.
func Text(text string) []*Content {
	return []*Content{{
		Role:  roleUser,
		Parts: []*Part{{Text: text}},
	}}
}

func (c *GenerateContentConfig) setDefaults() {
	if c == nil {
		return
	}
	if c.CandidateCount == 0 {
		c.CandidateCount = 1
	}
	if c.SystemInstruction != nil && c.SystemInstruction.Role == "" {
		c.SystemInstruction.setDefaults()
	}
}

func (c *Content) setDefaults() {
	if c == nil {
		return
	}
	if c.Role == "" {
		c.Role = roleUser
	}
}