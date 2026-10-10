// Copyright 2026 Google LLC
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
	"fmt"
	"reflect"
	"strings"
	"time"
)

func shouldEnableAutomaticContinuation(config *GenerateContentConfig) bool {
	if config == nil || config.AutomaticContinuation == nil {
		return true
	}
	return *config.AutomaticContinuation
}

func isResumableFinishReason(finishReason FinishReason) bool {
	return finishReason == FinishReasonContinuation
}

func shouldContinueGeneration(response *GenerateContentResponse) []byte {
	if response == nil || len(response.Candidates) == 0 {
		return nil
	}
	cand := response.Candidates[0]
	if cand == nil {
		return nil
	}
	if len(cand.ContinuationToken) > 0 && isResumableFinishReason(cand.FinishReason) {
		return cand.ContinuationToken
	}
	return nil
}

func prepareContinuationConfig(config *GenerateContentConfig, continuationToken []byte) *GenerateContentConfig {
	if config == nil {
		return &GenerateContentConfig{
			ContinuationToken: continuationToken,
		}
	}
	cloned := *config
	cloned.ContinuationToken = continuationToken
	return &cloned
}

func mergeContinuationResponses(responses []*GenerateContentResponse) (*GenerateContentResponse, error) {
	if len(responses) == 0 {
		return &GenerateContentResponse{}, nil
	}
	if len(responses) == 1 {
		return responses[0], nil
	}

	merged := responses[0]
	for i := 1; i < len(responses); i++ {
		res := mergeValuesReflect(reflect.ValueOf(merged), reflect.ValueOf(responses[i]), "", false)
		if !res.IsValid() {
			return nil, fmt.Errorf("failed to merge continuation responses at index %d", i)
		}
		merged = res.Interface().(*GenerateContentResponse)
	}

	// Always retain SDKHTTPResponse from the latest hop
	merged.SDKHTTPResponse = responses[len(responses)-1].SDKHTTPResponse
	return merged, nil
}

func isTerminalHopField(fieldName string) bool {
	f := strings.ToLower(fieldName)
	return f == "continuationtoken" ||
		f == "continuation_token" ||
		f == "finishreason" ||
		f == "finish_reason" ||
		f == "finishmessage" ||
		f == "finish_message"
}

func isTokensDetailsField(fieldName string) bool {
	f := strings.ToLower(fieldName)
	return strings.HasSuffix(f, "tokensdetails") ||
		strings.HasSuffix(f, "tokens_details")
}

func isSafetyRatingsField(fieldName string) bool {
	f := strings.ToLower(fieldName)
	return f == "safetyratings" || f == "safety_ratings"
}

func isCandidatesField(fieldName string) bool {
	f := strings.ToLower(fieldName)
	return f == "candidates"
}

func isCountOrSumField(fieldName string) bool {
	f := strings.ToLower(fieldName)
	return f == "tokencount" ||
		f == "token_count" ||
		strings.HasSuffix(f, "count") ||
		strings.HasSuffix(f, "_count") ||
		strings.HasSuffix(f, "sum") ||
		strings.HasSuffix(f, "_sum")
}

func mergeValuesReflect(prev, curr reflect.Value, fieldName string, isInsideUsageMetadata bool) reflect.Value {
	// Rule 1: Terminal hop fields always overwrite with curr (even if zero / nil)
	if isTerminalHopField(fieldName) {
		if !curr.IsValid() {
			if prev.IsValid() {
				return reflect.Zero(prev.Type())
			}
			return curr
		}
		return deepCopyValue(curr)
	}

	// Rule 2: Null / Absent handling: keep non-zero side
	if !curr.IsValid() || curr.IsZero() {
		return deepCopyValue(prev)
	}
	if !prev.IsValid() || prev.IsZero() {
		return deepCopyValue(curr)
	}

	// SdkHttpResponse: retain from curr
	if strings.EqualFold(fieldName, "sdkhttpresponse") || strings.EqualFold(fieldName, "sdk_http_response") {
		return deepCopyValue(curr)
	}

	// Pointers
	if prev.Kind() == reflect.Pointer && curr.Kind() == reflect.Pointer {
		mergedElem := mergeValuesReflect(prev.Elem(), curr.Elem(), fieldName, isInsideUsageMetadata)
		newPtr := reflect.New(prev.Type().Elem())
		newPtr.Elem().Set(mergedElem)
		return newPtr
	}

	// Interfaces
	if prev.Kind() == reflect.Interface && curr.Kind() == reflect.Interface {
		mergedElem := mergeValuesReflect(prev.Elem(), curr.Elem(), fieldName, isInsideUsageMetadata)
		newIface := reflect.New(prev.Type()).Elem()
		newIface.Set(mergedElem)
		return newIface
	}

	// Structs
	if prev.Kind() == reflect.Struct && curr.Kind() == reflect.Struct {
		// Special handling for time.Time: scalar, take curr
		if prev.Type() == reflect.TypeOf(time.Time{}) {
			return curr
		}

		inUsage := isInsideUsageMetadata ||
			strings.EqualFold(fieldName, "usagemetadata") ||
			strings.EqualFold(fieldName, "usage_metadata")

		newStruct := reflect.New(prev.Type()).Elem()
		for i := 0; i < prev.NumField(); i++ {
			field := prev.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			fName := field.Name
			prevF := prev.Field(i)
			currF := curr.Field(i)
			mergedF := mergeValuesReflect(prevF, currF, fName, inUsage)
			if mergedF.IsValid() && newStruct.Field(i).CanSet() {
				newStruct.Field(i).Set(mergedF)
			}
		}
		return newStruct
	}

	// Slices
	if prev.Kind() == reflect.Slice && curr.Kind() == reflect.Slice {
		// Keyed-list exception 1: candidates: merge element-wise
		if isCandidatesField(fieldName) {
			maxLen := prev.Len()
			if curr.Len() > maxLen {
				maxLen = curr.Len()
			}
			newSlice := reflect.MakeSlice(prev.Type(), maxLen, maxLen)
			for i := 0; i < maxLen; i++ {
				var p, c reflect.Value
				if i < prev.Len() {
					p = prev.Index(i)
				}
				if i < curr.Len() {
					c = curr.Index(i)
				}
				mergedElem := mergeValuesReflect(p, c, "candidate", isInsideUsageMetadata)
				newSlice.Index(i).Set(mergedElem)
			}
			return newSlice
		}

		// Keyed-list exception 2: ModalityTokenCount lists (*TokensDetails): group by modality and sum tokenCount
		if isTokensDetailsField(fieldName) {
			return mergeModalityTokenCounts(prev, curr)
		}

		// Keyed-list exception 3: safetyRatings: group by category and keep latest rating
		if isSafetyRatingsField(fieldName) {
			return mergeSafetyRatings(prev, curr)
		}

		// Default: Concatenate prev + curr preserving all items in order
		newSlice := reflect.MakeSlice(prev.Type(), prev.Len()+curr.Len(), prev.Len()+curr.Len())
		for i := 0; i < prev.Len(); i++ {
			newSlice.Index(i).Set(deepCopyValue(prev.Index(i)))
		}
		for i := 0; i < curr.Len(); i++ {
			newSlice.Index(prev.Len()+i).Set(deepCopyValue(curr.Index(i)))
		}
		return newSlice
	}

	// Numbers (sum inside usageMetadata or tokenCount / *Count / *Sum)
	shouldSum := isInsideUsageMetadata || isCountOrSumField(fieldName)
	if shouldSum {
		switch prev.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			sum := prev.Int() + curr.Int()
			return reflect.ValueOf(sum).Convert(prev.Type())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			sum := prev.Uint() + curr.Uint()
			return reflect.ValueOf(sum).Convert(prev.Type())
		case reflect.Float32, reflect.Float64:
			sum := prev.Float() + curr.Float()
			return reflect.ValueOf(sum).Convert(prev.Type())
		}
	}

	// Other Scalars (numbers not summed, strings, booleans, enums) - latest hop wins
	return deepCopyValue(curr)
}

func mergeModalityTokenCounts(prev, curr reflect.Value) reflect.Value {
	elemType := prev.Type().Elem()
	isPtr := elemType.Kind() == reflect.Pointer
	var structType reflect.Type
	if isPtr {
		structType = elemType.Elem()
	} else {
		structType = elemType
	}

	countsByModality := make(map[any]int64)
	var order []any
	modalityValues := make(map[any]reflect.Value)

	accumulate := func(slice reflect.Value) {
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i)
			if item.IsZero() {
				continue
			}
			structVal := item
			if isPtr {
				if item.IsNil() {
					continue
				}
				structVal = item.Elem()
			}
			modField := structVal.FieldByName("Modality")
			countField := structVal.FieldByName("TokenCount")
			if !modField.IsValid() || !countField.IsValid() {
				continue
			}
			modKey := modField.Interface()
			var count int64
			switch countField.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				count = countField.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				count = int64(countField.Uint())
			}

			if _, exists := countsByModality[modKey]; !exists {
				countsByModality[modKey] = 0
				order = append(order, modKey)
				modalityValues[modKey] = modField
			}
			countsByModality[modKey] += count
		}
	}

	accumulate(prev)
	accumulate(curr)

	resultSlice := reflect.MakeSlice(prev.Type(), len(order), len(order))
	for i, modKey := range order {
		newStruct := reflect.New(structType).Elem()
		modField := newStruct.FieldByName("Modality")
		if modField.IsValid() && modField.CanSet() {
			modField.Set(modalityValues[modKey])
		}
		countField := newStruct.FieldByName("TokenCount")
		if countField.IsValid() && countField.CanSet() {
			countField.Set(reflect.ValueOf(countsByModality[modKey]).Convert(countField.Type()))
		}
		if isPtr {
			ptr := reflect.New(structType)
			ptr.Elem().Set(newStruct)
			resultSlice.Index(i).Set(ptr)
		} else {
			resultSlice.Index(i).Set(newStruct)
		}
	}
	return resultSlice
}

func mergeSafetyRatings(prev, curr reflect.Value) reflect.Value {
	elemType := prev.Type().Elem()
	isPtr := elemType.Kind() == reflect.Pointer
	byCategory := make(map[any]reflect.Value)
	var order []any

	accumulate := func(slice reflect.Value) {
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i)
			if item.IsZero() {
				continue
			}
			structVal := item
			if isPtr {
				if item.IsNil() {
					continue
				}
				structVal = item.Elem()
			}
			catField := structVal.FieldByName("Category")
			if !catField.IsValid() {
				continue
			}
			catKey := catField.Interface()
			if _, exists := byCategory[catKey]; !exists {
				order = append(order, catKey)
			}
			byCategory[catKey] = deepCopyValue(item)
		}
	}

	accumulate(prev)
	accumulate(curr)

	resultSlice := reflect.MakeSlice(prev.Type(), len(order), len(order))
	for i, catKey := range order {
		resultSlice.Index(i).Set(byCategory[catKey])
	}
	return resultSlice
}

func deepCopyValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newPtr := reflect.New(v.Type().Elem())
		newPtr.Elem().Set(deepCopyValue(v.Elem()))
		return newPtr
	case reflect.Interface:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newIface := reflect.New(v.Type()).Elem()
		newIface.Set(deepCopyValue(v.Elem()))
		return newIface
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			newSlice.Index(i).Set(deepCopyValue(v.Index(i)))
		}
		return newSlice
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v
		}
		newStruct := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && newStruct.Field(i).CanSet() {
				newStruct.Field(i).Set(deepCopyValue(v.Field(i)))
			}
		}
		return newStruct
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newMap := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			newMap.SetMapIndex(iter.Key(), deepCopyValue(iter.Value()))
		}
		return newMap
	default:
		return v
	}
}
