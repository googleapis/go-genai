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

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"reflect"
	"strconv"
)

// Ptr returns a pointer to its argument.
// It can be used to initialize pointer fields:
//
//	genai.GenerateContentConfig{Temperature: genai.Ptr(0.5)}
func Ptr[T any](t T) *T { return &t }

type converterFunc func(*apiClient, map[string]any, map[string]any) (map[string]any, error)

type transformerFunc[T any] func(*apiClient, T) (T, error)

func setValueByPath(data map[string]any, keys []string, value any) {
	if value == nil {
		return
	}
	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		if _, ok := data[key]; !ok {
			data[key] = make(map[string]any)
		}
		if _, ok := data[key].(map[string]any); !ok {
			data[key] = make(map[string]any)
		}
		data = data[key].(map[string]any)
	}
	if !reflect.ValueOf(value).IsZero() {
		data[keys[len(keys)-1]] = value
	}
}

func getValueByPath(data map[string]any, keys []string) any {
	if len(keys) == 1 && keys[0] == "_self" {
		return data
	}
	var current any = data
	for _, key := range keys {
		switch v := current.(type) {
		case map[string]any:
			current = v[key]
		default:
			return nil // Key not found or invalid type
		}
	}
	return current
}

func formatMap(template string, variables map[string]any) (string, error) {
	var buffer bytes.Buffer
	for i := 0; i < len(template); i++ {
		if template[i] == '{' {
			j := i + 1
			for j < len(template) && template[j] != '}' {
				j++
			}
			if j < len(template) {
				key := template[i+1 : j]
				if value, ok := variables[key]; ok {
					switch val := value.(type) {
					case string:
						buffer.WriteString(val)
					default:
						return "", errors.New("formatMap: nested interface or unsupported type found")
					}
				}
				i = j
			}
		} else {
			buffer.WriteByte(template[i])
		}
	}
	return buffer.String(), nil
}

// applyConverterToSlice calls converter function to each element of the slice.
func applyConverterToSlice(ac *apiClient, inputs []any, converter converterFunc) ([]map[string]any, error) {
	var outputs []map[string]any
	for _, object := range inputs {
		object, err := converter(ac, object.(map[string]any), nil)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, object)
	}
	return outputs, nil
}

// applyItemTransformerToSlice calls item transformer function to each element of the slice.
func applyItemTransformerToSlice[T any](ac *apiClient, inputs []T, itemTransformer transformerFunc[T]) ([]T, error) {
	var outputs []T
	for _, input := range inputs {
		object, err := itemTransformer(ac, input)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, object)
	}
	return outputs, nil
}

func deepMarshal(input any, output *map[string]any) error {
	if inputBytes, err := json.Marshal(input); err != nil {
		return fmt.Errorf("deepMarshal: unable to marshal input: %w", err)
	} else if err := json.Unmarshal(inputBytes, output); err != nil {
		return fmt.Errorf("deepMarshal: unable to unmarshal input: %w", err)
	}
	return nil
}

func createURLQuery(query map[string]any) (string, error) {
	v := url.Values{}
	for key, value := range query {
		switch value := value.(type) {
		case string:
			v.Add(key, value)
		case int:
			v.Add(key, strconv.Itoa(value))
		case float64:
			v.Add(key, strconv.FormatFloat(value, 'f', -1, 64))
		case bool:
			v.Add(key, strconv.FormatBool(value))
		case []any:
			for _, item := range value {
				v.Add(key, item.(string))
			}
		default:
			return "", fmt.Errorf("unsupported type: %T", value)
		}
	}
	return v.Encode(), nil
}

func yieldErrorAndEndIterator[T any](err error) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		if !yield(nil, err) {
			return
		}
	}
}
