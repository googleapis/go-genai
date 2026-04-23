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

package apijson

import (
	"fmt"
	"reflect"

	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalscore"

	"github.com/tidwall/gjson"
)

func UnmarshalUnion(raw []byte, union any, meta *unmarshalinfo.Metadata) error {
	score, err := UnmarshalUnionWithScore(raw, union)
	if err != nil {
		return err
	}
	unmarshalinfo.SetUnmarshalState(raw, score, meta)
	return nil
}

func UnmarshalDiscriminatedUnion(raw []byte, discriminatorName string, union any, meta *unmarshalinfo.Metadata) error {
	score, err := UnmarshalDiscriminatedUnionWithScore(raw, discriminatorName, union)
	if err != nil {
		return err
	}
	unmarshalinfo.SetUnmarshalState(raw, score, meta)
	return err
}

// UnmarshalUnion unmarshals raw JSON bytes into the union member that populates
// the most fields. It tries each member in order and selects the one with the
// highest number of populated fields. The union should be a pointer to a struct
// with pointer fields for each possible union member, like this:
//
//	type FooUnion struct {
//		A *AType   `json:",inline,omitzero"`
//		B *BType   `json:",inline,omitzero"`
//		C []string `json:",inline,omitzero"`
//		D *string  `json:",inline,omitzero"`
//	}
func UnmarshalUnionWithScore(raw []byte, union any) (unmarshalscore.Score, error) {
	v := reflect.ValueOf(union)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return unmarshalscore.Score{}, fmt.Errorf("UnmarshalUnion requires a non-nil pointer to a struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return unmarshalscore.Score{}, fmt.Errorf("UnmarshalUnion requires a pointer to a struct")
	}

	t := v.Type()
	bestIdx := -1
	bestScore := unmarshalscore.Score{}

	// During this process, we want to avoid leaking the raw response back,
	// since we're trying to observe which parts of the response end up making
	// it into the object itself.
	metadataType := reflect.TypeOf(unmarshalinfo.Metadata{})
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip the metadata field and unexported fields
		if fieldType.Type == metadataType || !fieldType.IsExported() {
			continue
		}

		// Skip discriminated fields, this function's logic is only for
		// undiscriminated union members. Discriminated union members should be
		// handled by UnmarshalDiscriminatedUnionWithScore(), which falls back
		// to this code if there are any undiscriminated members in the union.
		// This check ensures we don't bother wasting time on discriminated
		// fields when we know none will work out.
		if fieldType.Tag.Get("discriminator") != "" {
			continue
		}

		// Create a new instance to unmarshal into
		var target reflect.Value
		if field.Kind() == reflect.Pointer {
			target = reflect.New(fieldType.Type.Elem())
		} else {
			target = reflect.New(fieldType.Type)
		}

		// Try to unmarshal into this member
		score, err := UnmarshalWithScore(raw, target.Interface())
		if err != nil {
			continue
		}

		if score.IsBetterThan(bestScore) {
			bestIdx = i
			bestScore = score
			if field.Kind() == reflect.Pointer {
				field.Set(target)
			} else {
				field.Set(target.Elem())
			}
		}
	}

	// If we found a valid member, clear other members:
	if bestIdx >= 0 {
		for i := 0; i < v.NumField(); i++ {
			fieldType := t.Field(i)

			// Skip unexported fields
			if !fieldType.IsExported() {
				continue
			}

			if i != bestIdx {
				field := v.Field(i)
				field.Set(reflect.Zero(fieldType.Type))
			}
		}
		return bestScore, nil
	}

	return unmarshalscore.Score{}, fmt.Errorf("no union member matched successfully")
}

// UnmarshalDiscriminatedUnion unmarshals raw JSON bytes into a union member
// based on a discriminator field. It looks up the discriminator value in the
// raw JSON and uses the `discriminator` struct tag to select the appropriate
// member. The union should look like this:
//
//	type FooUnion struct {
//		A *AType   `json:",inline,omitzero" discriminator:"a"`
//		B *BType   `json:",inline,omitzero" discriminator:"b"`
//	}
func UnmarshalDiscriminatedUnionWithScore(raw []byte, discriminatorName string, union any) (unmarshalscore.Score, error) {
	v := reflect.ValueOf(union)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return unmarshalscore.Score{}, fmt.Errorf("UnmarshalDiscriminatedUnion requires a non-nil pointer to a struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return unmarshalscore.Score{}, fmt.Errorf("UnmarshalDiscriminatedUnion requires a pointer to a struct")
	}

	t := v.Type()

	// Get the discriminator value from the raw JSON
	discriminatorResult := gjson.GetBytes(raw, discriminatorName)
	if !discriminatorResult.Exists() || discriminatorResult.Type != gjson.String {
		// If there's no discriminator and some of the union members are
		// undiscriminated, then we can fall back to the undiscriminated union
		// unmarshaling logic:
		for i := 0; i < v.NumField(); i++ {
			fieldType := t.Field(i)
			if fieldType.IsExported() && fieldType.Tag.Get("discriminator") == "" {
				return UnmarshalUnionWithScore(raw, union)
			}
		}

		return unmarshalscore.Score{}, fmt.Errorf("no %q discriminator found in JSON", discriminatorName)
	}

	discriminator := discriminatorResult.String()

	// Find the field that matches the discriminator value
	foundDiscriminator := false
	score := unmarshalscore.Score{}
	emptyDiscriminators := false
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Parse the discriminator tag
		fieldDiscriminator := fieldType.Tag.Get("discriminator")
		if fieldDiscriminator == discriminator {
			if foundDiscriminator {
				return unmarshalscore.Score{}, fmt.Errorf("duplicate discriminator: %s", discriminator)
			}
			foundDiscriminator = true
			// Unmarshal into the matching field
			var target reflect.Value
			if fieldType.Type.Kind() == reflect.Pointer {
				target = reflect.New(fieldType.Type.Elem())
			} else {
				target = reflect.New(fieldType.Type)
			}
			var err error
			score, err = UnmarshalWithScore(raw, target.Interface())
			if err != nil {
				return unmarshalscore.Score{}, err
			}
			if fieldType.Type.Kind() == reflect.Pointer {
				fieldValue.Set(target)
			} else {
				fieldValue.Set(target.Elem())
			}
		} else {
			if fieldDiscriminator == "" {
				emptyDiscriminators = true
			}
			// Clear other members
			fieldValue.Set(reflect.Zero(fieldType.Type))
		}
	}

	if foundDiscriminator {
		return score, nil
	}

	if emptyDiscriminators {
		// Fall back to regular union unmarshaling logic if we failed to find a discriminator
		return UnmarshalUnionWithScore(raw, union)
	}

	// Unknown discriminator value: leave the union at its zero value.
	// The raw bytes remain accessible via the metadata's RawJSON().
	return unmarshalscore.Score{}, nil
}
