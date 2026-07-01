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
	if len(contents) <= 1 {
		return sanitizeCompletedFunctionCallContents(contents)
	}
	var merged []*Content
	var order []string
	seen := map[string]bool{}
	completedByKey := map[string]*Part{}
	role := ""
	for _, content := range contents {
		if content == nil {
			continue
		}
		if !contentContainsFunctionCall(content) {
			merged = append(merged, content)
			continue
		}
		role = content.Role
		for _, part := range content.Parts {
			if part == nil || part.FunctionCall == nil {
				continue
			}
			key := streamedFunctionCallKey(part.FunctionCall)
			if !seen[key] {
				seen[key] = true
				order = append(order, key)
			}
			if !functionCallWillContinue(part.FunctionCall) {
				completedByKey[key] = cloneCompletedFunctionCallPart(part.FunctionCall)
			}
		}
	}
	var completed []*Part
	for _, key := range order {
		if part, ok := completedByKey[key]; ok {
			completed = append(completed, part)
		}
	}
	if len(completed) > 0 {
		merged = append(merged, &Content{Role: role, Parts: completed})
	}
	return merged
}

func contentContainsFunctionCall(content *Content) bool {
	for _, part := range content.Parts {
		if part != nil && part.FunctionCall != nil {
			return true
		}
	}
	return false
}

func sanitizeCompletedFunctionCallContents(contents []*Content) []*Content {
	for _, content := range contents {
		if content == nil || !contentContainsFunctionCall(content) {
			continue
		}
		var parts []*Part
		for _, part := range content.Parts {
			if part == nil || part.FunctionCall == nil {
				parts = append(parts, part)
				continue
			}
			if !functionCallWillContinue(part.FunctionCall) {
				parts = append(parts, cloneCompletedFunctionCallPart(part.FunctionCall))
			}
		}
		content.Parts = parts
	}
	return contents
}

func functionCallWillContinue(call *FunctionCall) bool {
	return call != nil && call.WillContinue != nil && *call.WillContinue
}

func cloneCompletedFunctionCallPart(call *FunctionCall) *Part {
	return &Part{FunctionCall: cloneCompletedFunctionCall(call)}
}

func cloneCompletedFunctionCall(call *FunctionCall) *FunctionCall {
	if call == nil {
		return nil
	}
	return &FunctionCall{
		ID:   call.ID,
		Name: call.Name,
		Args: cloneJSONMap(call.Args),
	}
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
	current := any(root)
	for i, token := range tokens {
		last := i == len(tokens)-1
		switch container := current.(type) {
		case map[string]any:
			key, ok := token.(string)
			if !ok {
				return fmt.Errorf("streamed function call path conflict: expected object key, got array index")
			}
			if last {
				merged, err := mergeStreamedJSONLeaf(container[key], value)
				if err != nil {
					return err
				}
				container[key] = merged
				continue
			}
			next, ok := container[key]
			if !ok || next == nil {
				next = newStreamedContainer(tokens[i+1])
				container[key] = next
			}
			current = next
		case []any:
			index, ok := token.(int)
			if !ok {
				return fmt.Errorf("streamed function call path conflict: expected array index, got object key")
			}
			if index < 0 {
				return fmt.Errorf("streamed function call array index %d out of bounds", index)
			}
			if index >= len(container) {
				grown := make([]any, index+1)
				copy(grown, container)
				container = grown
				if err := replaceStreamedJSONArray(root, tokens[:i], container); err != nil {
					return err
				}
			}
			if last {
				merged, err := mergeStreamedJSONLeaf(container[index], value)
				if err != nil {
					return err
				}
				container[index] = merged
				continue
			}
			if container[index] == nil {
				container[index] = newStreamedContainer(tokens[i+1])
			}
			current = container[index]
		default:
			return fmt.Errorf("streamed function call path conflict: cannot descend into %T", current)
		}
	}
	return nil
}

func replaceStreamedJSONArray(root map[string]any, tokens []any, replacement []any) error {
	if len(tokens) == 0 {
		return fmt.Errorf("streamed function call json path cannot replace root array")
	}
	current := any(root)
	for i, token := range tokens {
		last := i == len(tokens)-1
		switch container := current.(type) {
		case map[string]any:
			key, ok := token.(string)
			if !ok {
				return fmt.Errorf("streamed function call path conflict: expected object key while growing array")
			}
			if last {
				container[key] = replacement
				return nil
			}
			current = container[key]
		case []any:
			index, ok := token.(int)
			if !ok || index < 0 || index >= len(container) {
				return fmt.Errorf("streamed function call path conflict: invalid array index while growing array")
			}
			if last {
				container[index] = replacement
				return nil
			}
			current = container[index]
		default:
			return fmt.Errorf("streamed function call path conflict: cannot grow array under %T", current)
		}
	}
	return nil
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
