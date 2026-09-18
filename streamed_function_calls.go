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
	"strconv"
	"strings"
)

type streamedFunctionCallAccumulator struct {
	argsByCall map[string]map[string]any
}

func newStreamedFunctionCallAccumulator() *streamedFunctionCallAccumulator {
	return &streamedFunctionCallAccumulator{argsByCall: map[string]map[string]any{}}
}

func (a *streamedFunctionCallAccumulator) applyGenerateContentResponse(response *GenerateContentResponse) error {
	if response == nil {
		return nil
	}
	for _, candidate := range response.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part != nil && part.FunctionCall != nil {
				if err := a.applyFunctionCall(part.FunctionCall); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mergeStreamedFunctionCallContents(contents []*Content) []*Content {
	if len(contents) == 0 {
		return contents
	}
	var merged []*Content
	var callParts []*Part
	callIndex := map[string]int{}
	role := ""
	for _, content := range contents {
		if content == nil {
			continue
		}
		if role == "" && content.Role != "" {
			role = content.Role
		}
		var nonCallParts []*Part
		for _, part := range content.Parts {
			if part == nil || part.FunctionCall == nil {
				nonCallParts = append(nonCallParts, part)
				continue
			}
			cloned := cloneStreamedFunctionCallPart(part)
			finalizeStreamedFunctionCall(cloned.FunctionCall)
			key := streamedFunctionCallKey(cloned.FunctionCall)
			if idx, ok := callIndex[key]; ok {
				callParts[idx] = cloned
				continue
			}
			callIndex[key] = len(callParts)
			callParts = append(callParts, cloned)
		}
		if len(nonCallParts) > 0 {
			copyContent := *content
			copyContent.Parts = nonCallParts
			merged = append(merged, &copyContent)
		}
	}
	if len(callParts) > 0 {
		if role == "" {
			role = "model"
		}
		merged = append(merged, &Content{Role: role, Parts: callParts})
	}
	if len(merged) == 0 {
		return contents
	}
	return merged
}

func cloneStreamedFunctionCallPart(part *Part) *Part {
	if part == nil {
		return nil
	}
	out := *part
	if part.FunctionCall != nil {
		call := *part.FunctionCall
		call.Args = cloneJSONMap(call.Args)
		call.PartialArgs = clonePartialArgs(call.PartialArgs)
		out.FunctionCall = &call
	}
	return &out
}

func clonePartialArgs(in []*PartialArg) []*PartialArg {
	if len(in) == 0 {
		return nil
	}
	out := make([]*PartialArg, len(in))
	for i, partial := range in {
		if partial == nil {
			continue
		}
		copyPartial := *partial
		out[i] = &copyPartial
	}
	return out
}

func finalizeStreamedFunctionCall(call *FunctionCall) {
	if call == nil {
		return
	}
	if call.WillContinue != nil && *call.WillContinue {
		return
	}
	call.PartialArgs = nil
	call.WillContinue = nil
}

func contentContainsFunctionCall(content *Content) bool {
	for _, part := range content.Parts {
		if part != nil && part.FunctionCall != nil {
			return true
		}
	}
	return false
}

func (a *streamedFunctionCallAccumulator) applyLiveServerMessage(message *LiveServerMessage) error {
	if message == nil || message.ToolCall == nil {
		return nil
	}
	for _, call := range message.ToolCall.FunctionCalls {
		if err := a.applyFunctionCall(call); err != nil {
			return err
		}
	}
	return nil
}

func (a *streamedFunctionCallAccumulator) applyFunctionCall(call *FunctionCall) error {
	if call == nil || len(call.PartialArgs) == 0 {
		return nil
	}
	key := streamedFunctionCallKey(call)
	args := cloneJSONMap(call.Args)
	if prior, ok := a.argsByCall[key]; ok {
		args = cloneJSONMap(prior)
	}
	for _, partial := range call.PartialArgs {
		if partial == nil {
			continue
		}
		value, err := streamedPartialArgValue(partial)
		if err != nil {
			return err
		}
		if err := setStreamedJSONPath(args, partial.JsonPath, value); err != nil {
			return err
		}
	}
	call.Args = cloneJSONMap(args)
	if call.WillContinue != nil && *call.WillContinue {
		a.argsByCall[key] = cloneJSONMap(args)
	} else {
		delete(a.argsByCall, key)
		call.PartialArgs = nil
		call.WillContinue = nil
	}
	return nil
}

func streamedFunctionCallKey(call *FunctionCall) string {
	if call.ID != "" {
		return "id:" + call.ID
	}
	if call.Name != "" {
		return "name:" + call.Name
	}
	return "anonymous"
}

func streamedPartialArgValue(partial *PartialArg) (any, error) {
	set := 0
	var value any
	if partial.BoolValue != nil {
		set++
		value = *partial.BoolValue
	}
	if partial.NumberValue != nil {
		set++
		value = *partial.NumberValue
	}
	if partial.StringValue != "" {
		set++
		value = partial.StringValue
	}
	if partial.NULLValue != "" {
		set++
		value = nil
	}
	if set != 1 {
		return nil, fmt.Errorf("streamed function call partial arg must contain exactly one value, got %d", set)
	}
	return value, nil
}

func cloneJSONMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = cloneJSONValue(value)
	}
	return out
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneJSONValue(item)
		}
		return out
	default:
		return typed
	}
}

func setStreamedJSONPath(root map[string]any, path string, value any) error {
	tokens, err := parseStreamedJSONPath(path)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return fmt.Errorf("streamed function call json path has no target")
	}
	updated, err := setStreamedJSONPathValue(root, tokens, value)
	if err != nil {
		return err
	}
	if _, ok := updated.(map[string]any); !ok {
		return fmt.Errorf("streamed function call root must remain an object")
	}
	return nil
}

func setStreamedJSONPathValue(current any, tokens []any, value any) (any, error) {
	if len(tokens) == 0 {
		return mergeStreamedJSONLeaf(current, value)
	}
	token := tokens[0]
	last := len(tokens) == 1
	switch typed := token.(type) {
	case string:
		container, ok := current.(map[string]any)
		if current == nil {
			container = map[string]any{}
		} else if !ok {
			return nil, fmt.Errorf("streamed function call path conflict: expected object key, got %T", current)
		}
		if last {
			merged, err := mergeStreamedJSONLeaf(container[typed], value)
			if err != nil {
				return nil, err
			}
			container[typed] = merged
			return container, nil
		}
		next := container[typed]
		if next == nil {
			next = newStreamedContainer(tokens[1])
		}
		updated, err := setStreamedJSONPathValue(next, tokens[1:], value)
		if err != nil {
			return nil, err
		}
		container[typed] = updated
		return container, nil
	case int:
		if typed < 0 {
			return nil, fmt.Errorf("streamed function call json path has invalid array index %d", typed)
		}
		container, ok := current.([]any)
		if current == nil {
			container = []any{}
		} else if !ok {
			return nil, fmt.Errorf("streamed function call path conflict: expected array index, got %T", current)
		}
		container = growStreamedArray(container, typed+1)
		if last {
			merged, err := mergeStreamedJSONLeaf(container[typed], value)
			if err != nil {
				return nil, err
			}
			container[typed] = merged
			return container, nil
		}
		if container[typed] == nil {
			container[typed] = newStreamedContainer(tokens[1])
		}
		updated, err := setStreamedJSONPathValue(container[typed], tokens[1:], value)
		if err != nil {
			return nil, err
		}
		container[typed] = updated
		return container, nil
	default:
		return nil, fmt.Errorf("streamed function call json path has unsupported token type %T", token)
	}
}

func growStreamedArray(in []any, size int) []any {
	if len(in) >= size {
		return in
	}
	out := make([]any, size)
	copy(out, in)
	return out
}

func newStreamedContainer(next any) any {
	if index, ok := next.(int); ok {
		return make([]any, index+1)
	}
	return map[string]any{}
}

func mergeStreamedJSONLeaf(current any, value any) (any, error) {
	if current == nil {
		return value, nil
	}
	currentString, currentIsString := current.(string)
	valueString, valueIsString := value.(string)
	if currentIsString && valueIsString {
		return currentString + valueString, nil
	}
	if current == value {
		return current, nil
	}
	return nil, fmt.Errorf("streamed function call path conflict: refusing to overwrite %T with %T", current, value)
}

func parseStreamedJSONPath(path string) ([]any, error) {
	if path == "" || path[0] != '$' {
		return nil, fmt.Errorf("streamed function call json path must start with $")
	}
	var tokens []any
	for i := 1; i < len(path); {
		switch path[i] {
		case '.':
			i++
			start := i
			for i < len(path) && (path[i] == '_' || path[i] == '-' || path[i] >= '0' && path[i] <= '9' || path[i] >= 'A' && path[i] <= 'Z' || path[i] >= 'a' && path[i] <= 'z') {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("streamed function call json path has empty field")
			}
			tokens = append(tokens, path[start:i])
		case '[':
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("streamed function call json path has unterminated bracket")
			}
			body := path[i+1 : i+end]
			if body == "" {
				return nil, fmt.Errorf("streamed function call json path has empty bracket")
			}
			if body[0] == '\'' || body[0] == '"' {
				if len(body) < 2 || body[len(body)-1] != body[0] {
					return nil, fmt.Errorf("streamed function call json path has unterminated quoted key")
				}
				tokens = append(tokens, body[1:len(body)-1])
			} else {
				index, err := strconv.Atoi(body)
				if err != nil || index < 0 {
					return nil, fmt.Errorf("streamed function call json path has invalid array index %q", body)
				}
				tokens = append(tokens, index)
			}
			i += end + 1
		default:
			return nil, fmt.Errorf("streamed function call json path has unsupported token at byte %d", i)
		}
	}
	return tokens, nil
}
