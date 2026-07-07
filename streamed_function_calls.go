package genai

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type streamedFunctionCallAccumulator struct {
	calls map[string]map[string]any
}

type streamedFunctionCallPathSegment struct {
	key   string
	index int
	array bool
}

func newStreamedFunctionCallAccumulator() *streamedFunctionCallAccumulator {
	return &streamedFunctionCallAccumulator{calls: make(map[string]map[string]any)}
}

func (a *streamedFunctionCallAccumulator) accumulateResponse(response *GenerateContentResponse) error {
	if response == nil {
		return nil
	}
	for _, candidate := range response.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part == nil || part.FunctionCall == nil {
				continue
			}
			if err := a.accumulateFunctionCall(part.FunctionCall); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *streamedFunctionCallAccumulator) accumulateFunctionCalls(functionCalls []*FunctionCall) error {
	for _, functionCall := range functionCalls {
		if functionCall == nil {
			continue
		}
		if err := a.accumulateFunctionCall(functionCall); err != nil {
			return err
		}
	}
	return nil
}

func (a *streamedFunctionCallAccumulator) accumulateFunctionCall(functionCall *FunctionCall) error {
	if functionCall == nil {
		return nil
	}
	key := streamedFunctionCallKey(functionCall)
	accumulated := a.calls[key]
	if accumulated == nil {
		accumulated = cloneStringAnyMap(functionCall.Args)
	}

	for _, partialArg := range functionCall.PartialArgs {
		if partialArg == nil {
			continue
		}
		segments, err := parseStreamedFunctionCallJSONPath(partialArg.JsonPath)
		if err != nil {
			return err
		}
		value, err := streamedFunctionCallPartialArgValue(partialArg)
		if err != nil {
			return err
		}
		next, err := streamedFunctionCallSet(accumulated, segments, value)
		if err != nil {
			return err
		}
		accumulated = next
	}

	if len(accumulated) > 0 {
		functionCall.Args = cloneStringAnyMap(accumulated)
	}
	if functionCall.WillContinue == nil || !*functionCall.WillContinue {
		delete(a.calls, key)
	} else {
		a.calls[key] = accumulated
	}
	return nil
}

func streamedFunctionCallKey(functionCall *FunctionCall) string {
	if functionCall.ID != "" {
		return "id:" + functionCall.ID
	}
	if functionCall.Name != "" {
		return "name:" + functionCall.Name
	}
	return "anonymous"
}

func streamedFunctionCallPartialArgValue(partialArg *PartialArg) (any, error) {
	var value any
	seen := false
	set := func(next any) error {
		if seen {
			return fmt.Errorf("streamed function call partial arg has multiple value fields at %q", partialArg.JsonPath)
		}
		value = next
		seen = true
		return nil
	}
	if partialArg.BoolValue != nil {
		if err := set(*partialArg.BoolValue); err != nil {
			return nil, err
		}
	}
	if partialArg.NumberValue != nil {
		if err := set(*partialArg.NumberValue); err != nil {
			return nil, err
		}
	}
	if partialArg.StringValue != "" {
		if err := set(partialArg.StringValue); err != nil {
			return nil, err
		}
	}
	if partialArg.NULLValue != "" {
		if err := set(nil); err != nil {
			return nil, err
		}
	}
	return value, nil
}

func parseStreamedFunctionCallJSONPath(path string) ([]streamedFunctionCallPathSegment, error) {
	if path == "" {
		return nil, fmt.Errorf("streamed function call partial arg missing json path")
	}
	if path[0] != '$' {
		return nil, fmt.Errorf("unsupported streamed function call json path %q: must start with $", path)
	}
	var segments []streamedFunctionCallPathSegment
	for i := 1; i < len(path); {
		switch path[i] {
		case '.':
			i++
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("unsupported streamed function call json path %q: empty dot field", path)
			}
			segments = append(segments, streamedFunctionCallPathSegment{key: path[start:i]})
		case '[':
			i++
			if i >= len(path) {
				return nil, fmt.Errorf("unsupported streamed function call json path %q: unclosed bracket", path)
			}
			if path[i] == '\'' || path[i] == '"' {
				quote := path[i]
				i++
				start := i
				for i < len(path) && path[i] != quote {
					i++
				}
				if i >= len(path) {
					return nil, fmt.Errorf("unsupported streamed function call json path %q: unclosed quoted key", path)
				}
				key := path[start:i]
				i++
				if i >= len(path) || path[i] != ']' {
					return nil, fmt.Errorf("unsupported streamed function call json path %q: missing closing bracket", path)
				}
				i++
				segments = append(segments, streamedFunctionCallPathSegment{key: key})
				continue
			}
			start := i
			for i < len(path) && path[i] != ']' {
				i++
			}
			if i >= len(path) {
				return nil, fmt.Errorf("unsupported streamed function call json path %q: unclosed index", path)
			}
			indexText := strings.TrimSpace(path[start:i])
			i++
			index, err := strconv.Atoi(indexText)
			if err != nil || index < 0 {
				return nil, fmt.Errorf("unsupported streamed function call json path %q: array index must be a non-negative integer", path)
			}
			segments = append(segments, streamedFunctionCallPathSegment{array: true, index: index})
		default:
			return nil, fmt.Errorf("unsupported streamed function call json path %q: expected dot field or bracket segment", path)
		}
	}
	return segments, nil
}

func streamedFunctionCallSet(root map[string]any, segments []streamedFunctionCallPathSegment, value any) (map[string]any, error) {
	next := cloneStringAnyMap(root)
	if len(segments) == 0 {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("streamed function call root path requires object value")
		}
		for key, objectValue := range object {
			merged, err := streamedFunctionCallMergeLeaf(next[key], objectValue)
			if err != nil {
				return nil, err
			}
			next[key] = merged
		}
		return next, nil
	}
	updated, err := streamedFunctionCallSetAt(next, segments, value)
	if err != nil {
		return nil, err
	}
	object, ok := updated.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("streamed function call root must remain an object")
	}
	return object, nil
}

func streamedFunctionCallSetAt(current any, segments []streamedFunctionCallPathSegment, value any) (any, error) {
	if len(segments) == 0 {
		return streamedFunctionCallMergeLeaf(current, value)
	}
	segment := segments[0]
	if segment.array {
		array, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("streamed function call path conflict: expected array at index %d", segment.index)
		}
		next := cloneAnySlice(array)
		for len(next) <= segment.index {
			next = append(next, nil)
		}
		child := next[segment.index]
		if child == nil && len(segments) > 1 {
			child = emptyContainerForSegment(segments[1])
		}
		updated, err := streamedFunctionCallSetAt(child, segments[1:], value)
		if err != nil {
			return nil, err
		}
		next[segment.index] = updated
		return next, nil
	}

	object, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("streamed function call path conflict: expected object at key %q", segment.key)
	}
	next := cloneStringAnyMap(object)
	child := next[segment.key]
	if child == nil && len(segments) > 1 {
		child = emptyContainerForSegment(segments[1])
	}
	updated, err := streamedFunctionCallSetAt(child, segments[1:], value)
	if err != nil {
		return nil, err
	}
	next[segment.key] = updated
	return next, nil
}

func streamedFunctionCallMergeLeaf(current any, value any) (any, error) {
	if current == nil {
		return value, nil
	}
	currentString, currentIsString := current.(string)
	valueString, valueIsString := value.(string)
	if currentIsString && valueIsString {
		return currentString + valueString, nil
	}
	if reflect.DeepEqual(current, value) {
		return current, nil
	}
	return nil, fmt.Errorf("streamed function call path conflict: refusing to overwrite %T with %T", current, value)
}

func emptyContainerForSegment(segment streamedFunctionCallPathSegment) any {
	if segment.array {
		return []any{}
	}
	return map[string]any{}
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneJSONValue(value)
	}
	return out
}

func cloneAnySlice(in []any) []any {
	out := make([]any, len(in))
	for i, value := range in {
		out[i] = cloneJSONValue(value)
	}
	return out
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneStringAnyMap(typed)
	case []any:
		return cloneAnySlice(typed)
	default:
		return typed
	}
}
