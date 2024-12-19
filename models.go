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
	"context"
	"fmt"
	"iter"
	"net/http"
)

func partToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)
	if getValueByPath(fromObject, []string{"videoMetadata"}) != nil {
		return nil, fmt.Errorf("video_metadata parameter is not supported in Google AI")
	}

	fromThought := getValueByPath(fromObject, []string{"thought"})
	if fromThought != nil {
		setValueByPath(toObject, []string{"thought"}, fromThought)
	}

	fromCodeExecutionResult := getValueByPath(fromObject, []string{"codeExecutionResult"})
	if fromCodeExecutionResult != nil {
		setValueByPath(toObject, []string{"codeExecutionResult"}, fromCodeExecutionResult)
	}

	fromExecutableCode := getValueByPath(fromObject, []string{"executableCode"})
	if fromExecutableCode != nil {
		setValueByPath(toObject, []string{"executableCode"}, fromExecutableCode)
	}

	fromFileData := getValueByPath(fromObject, []string{"fileData"})
	if fromFileData != nil {
		setValueByPath(toObject, []string{"fileData"}, fromFileData)
	}

	fromFunctionCall := getValueByPath(fromObject, []string{"functionCall"})
	if fromFunctionCall != nil {
		setValueByPath(toObject, []string{"functionCall"}, fromFunctionCall)
	}

	fromFunctionResponse := getValueByPath(fromObject, []string{"functionResponse"})
	if fromFunctionResponse != nil {
		setValueByPath(toObject, []string{"functionResponse"}, fromFunctionResponse)
	}

	fromInlineData := getValueByPath(fromObject, []string{"inlineData"})
	if fromInlineData != nil {
		setValueByPath(toObject, []string{"inlineData"}, fromInlineData)
	}

	fromText := getValueByPath(fromObject, []string{"text"})
	if fromText != nil {
		setValueByPath(toObject, []string{"text"}, fromText)
	}

	return toObject, nil
}

func partToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromVideoMetadata := getValueByPath(fromObject, []string{"videoMetadata"})
	if fromVideoMetadata != nil {
		setValueByPath(toObject, []string{"videoMetadata"}, fromVideoMetadata)
	}

	fromThought := getValueByPath(fromObject, []string{"thought"})
	if fromThought != nil {
		setValueByPath(toObject, []string{"thought"}, fromThought)
	}

	fromCodeExecutionResult := getValueByPath(fromObject, []string{"codeExecutionResult"})
	if fromCodeExecutionResult != nil {
		setValueByPath(toObject, []string{"codeExecutionResult"}, fromCodeExecutionResult)
	}

	fromExecutableCode := getValueByPath(fromObject, []string{"executableCode"})
	if fromExecutableCode != nil {
		setValueByPath(toObject, []string{"executableCode"}, fromExecutableCode)
	}

	fromFileData := getValueByPath(fromObject, []string{"fileData"})
	if fromFileData != nil {
		setValueByPath(toObject, []string{"fileData"}, fromFileData)
	}

	fromFunctionCall := getValueByPath(fromObject, []string{"functionCall"})
	if fromFunctionCall != nil {
		setValueByPath(toObject, []string{"functionCall"}, fromFunctionCall)
	}

	fromFunctionResponse := getValueByPath(fromObject, []string{"functionResponse"})
	if fromFunctionResponse != nil {
		setValueByPath(toObject, []string{"functionResponse"}, fromFunctionResponse)
	}

	fromInlineData := getValueByPath(fromObject, []string{"inlineData"})
	if fromInlineData != nil {
		setValueByPath(toObject, []string{"inlineData"}, fromInlineData)
	}

	fromText := getValueByPath(fromObject, []string{"text"})
	if fromText != nil {
		setValueByPath(toObject, []string{"text"}, fromText)
	}

	return toObject, nil
}

func contentToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromParts := getValueByPath(fromObject, []string{"parts"})
	if fromParts != nil {
		fromParts, err = applyConverterToSlice(ac, fromParts.([]any), partToMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"parts"}, fromParts)
	}

	fromRole := getValueByPath(fromObject, []string{"role"})
	if fromRole != nil {
		setValueByPath(toObject, []string{"role"}, fromRole)
	}

	return toObject, nil
}

func contentToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromParts := getValueByPath(fromObject, []string{"parts"})
	if fromParts != nil {
		fromParts, err = applyConverterToSlice(ac, fromParts.([]any), partToVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"parts"}, fromParts)
	}

	fromRole := getValueByPath(fromObject, []string{"role"})
	if fromRole != nil {
		setValueByPath(toObject, []string{"role"}, fromRole)
	}

	return toObject, nil
}

func schemaToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)
	if getValueByPath(fromObject, []string{"minItems"}) != nil {
		return nil, fmt.Errorf("min_items parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"example"}) != nil {
		return nil, fmt.Errorf("example parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"propertyOrdering"}) != nil {
		return nil, fmt.Errorf("property_ordering parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"pattern"}) != nil {
		return nil, fmt.Errorf("pattern parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"minimum"}) != nil {
		return nil, fmt.Errorf("minimum parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"default"}) != nil {
		return nil, fmt.Errorf("default parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"anyOf"}) != nil {
		return nil, fmt.Errorf("any_of parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"maxLength"}) != nil {
		return nil, fmt.Errorf("max_length parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"title"}) != nil {
		return nil, fmt.Errorf("title parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"minLength"}) != nil {
		return nil, fmt.Errorf("min_length parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"minProperties"}) != nil {
		return nil, fmt.Errorf("min_properties parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"maxItems"}) != nil {
		return nil, fmt.Errorf("max_items parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"maximum"}) != nil {
		return nil, fmt.Errorf("maximum parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"nullable"}) != nil {
		return nil, fmt.Errorf("nullable parameter is not supported in Google AI")
	}

	if getValueByPath(fromObject, []string{"maxProperties"}) != nil {
		return nil, fmt.Errorf("max_properties parameter is not supported in Google AI")
	}

	fromType := getValueByPath(fromObject, []string{"type"})
	if fromType != nil {
		setValueByPath(toObject, []string{"type"}, fromType)
	}

	fromDescription := getValueByPath(fromObject, []string{"description"})
	if fromDescription != nil {
		setValueByPath(toObject, []string{"description"}, fromDescription)
	}

	fromEnum := getValueByPath(fromObject, []string{"enum"})
	if fromEnum != nil {
		setValueByPath(toObject, []string{"enum"}, fromEnum)
	}

	fromFormat := getValueByPath(fromObject, []string{"format"})
	if fromFormat != nil {
		setValueByPath(toObject, []string{"format"}, fromFormat)
	}

	fromItems := getValueByPath(fromObject, []string{"items"})
	if fromItems != nil {
		setValueByPath(toObject, []string{"items"}, fromItems)
	}

	fromProperties := getValueByPath(fromObject, []string{"properties"})
	if fromProperties != nil {
		setValueByPath(toObject, []string{"properties"}, fromProperties)
	}

	fromRequired := getValueByPath(fromObject, []string{"required"})
	if fromRequired != nil {
		setValueByPath(toObject, []string{"required"}, fromRequired)
	}

	return toObject, nil
}

func schemaToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromMinItems := getValueByPath(fromObject, []string{"minItems"})
	if fromMinItems != nil {
		setValueByPath(toObject, []string{"minItems"}, fromMinItems)
	}

	fromExample := getValueByPath(fromObject, []string{"example"})
	if fromExample != nil {
		setValueByPath(toObject, []string{"example"}, fromExample)
	}

	fromPropertyOrdering := getValueByPath(fromObject, []string{"propertyOrdering"})
	if fromPropertyOrdering != nil {
		setValueByPath(toObject, []string{"propertyOrdering"}, fromPropertyOrdering)
	}

	fromPattern := getValueByPath(fromObject, []string{"pattern"})
	if fromPattern != nil {
		setValueByPath(toObject, []string{"pattern"}, fromPattern)
	}

	fromMinimum := getValueByPath(fromObject, []string{"minimum"})
	if fromMinimum != nil {
		setValueByPath(toObject, []string{"minimum"}, fromMinimum)
	}

	fromDefault := getValueByPath(fromObject, []string{"default"})
	if fromDefault != nil {
		setValueByPath(toObject, []string{"default"}, fromDefault)
	}

	fromAnyOf := getValueByPath(fromObject, []string{"anyOf"})
	if fromAnyOf != nil {
		setValueByPath(toObject, []string{"anyOf"}, fromAnyOf)
	}

	fromMaxLength := getValueByPath(fromObject, []string{"maxLength"})
	if fromMaxLength != nil {
		setValueByPath(toObject, []string{"maxLength"}, fromMaxLength)
	}

	fromTitle := getValueByPath(fromObject, []string{"title"})
	if fromTitle != nil {
		setValueByPath(toObject, []string{"title"}, fromTitle)
	}

	fromMinLength := getValueByPath(fromObject, []string{"minLength"})
	if fromMinLength != nil {
		setValueByPath(toObject, []string{"minLength"}, fromMinLength)
	}

	fromMinProperties := getValueByPath(fromObject, []string{"minProperties"})
	if fromMinProperties != nil {
		setValueByPath(toObject, []string{"minProperties"}, fromMinProperties)
	}

	fromMaxItems := getValueByPath(fromObject, []string{"maxItems"})
	if fromMaxItems != nil {
		setValueByPath(toObject, []string{"maxItems"}, fromMaxItems)
	}

	fromMaximum := getValueByPath(fromObject, []string{"maximum"})
	if fromMaximum != nil {
		setValueByPath(toObject, []string{"maximum"}, fromMaximum)
	}

	fromNullable := getValueByPath(fromObject, []string{"nullable"})
	if fromNullable != nil {
		setValueByPath(toObject, []string{"nullable"}, fromNullable)
	}

	fromMaxProperties := getValueByPath(fromObject, []string{"maxProperties"})
	if fromMaxProperties != nil {
		setValueByPath(toObject, []string{"maxProperties"}, fromMaxProperties)
	}

	fromType := getValueByPath(fromObject, []string{"type"})
	if fromType != nil {
		setValueByPath(toObject, []string{"type"}, fromType)
	}

	fromDescription := getValueByPath(fromObject, []string{"description"})
	if fromDescription != nil {
		setValueByPath(toObject, []string{"description"}, fromDescription)
	}

	fromEnum := getValueByPath(fromObject, []string{"enum"})
	if fromEnum != nil {
		setValueByPath(toObject, []string{"enum"}, fromEnum)
	}

	fromFormat := getValueByPath(fromObject, []string{"format"})
	if fromFormat != nil {
		setValueByPath(toObject, []string{"format"}, fromFormat)
	}

	fromItems := getValueByPath(fromObject, []string{"items"})
	if fromItems != nil {
		setValueByPath(toObject, []string{"items"}, fromItems)
	}

	fromProperties := getValueByPath(fromObject, []string{"properties"})
	if fromProperties != nil {
		setValueByPath(toObject, []string{"properties"}, fromProperties)
	}

	fromRequired := getValueByPath(fromObject, []string{"required"})
	if fromRequired != nil {
		setValueByPath(toObject, []string{"required"}, fromRequired)
	}

	return toObject, nil
}

func safetySettingToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)
	if getValueByPath(fromObject, []string{"method"}) != nil {
		return nil, fmt.Errorf("method parameter is not supported in Google AI")
	}

	fromCategory := getValueByPath(fromObject, []string{"category"})
	if fromCategory != nil {
		setValueByPath(toObject, []string{"category"}, fromCategory)
	}

	fromThreshold := getValueByPath(fromObject, []string{"threshold"})
	if fromThreshold != nil {
		setValueByPath(toObject, []string{"threshold"}, fromThreshold)
	}

	return toObject, nil
}

func safetySettingToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromMethod := getValueByPath(fromObject, []string{"method"})
	if fromMethod != nil {
		setValueByPath(toObject, []string{"method"}, fromMethod)
	}

	fromCategory := getValueByPath(fromObject, []string{"category"})
	if fromCategory != nil {
		setValueByPath(toObject, []string{"category"}, fromCategory)
	}

	fromThreshold := getValueByPath(fromObject, []string{"threshold"})
	if fromThreshold != nil {
		setValueByPath(toObject, []string{"threshold"}, fromThreshold)
	}

	return toObject, nil
}

func functionDeclarationToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)
	if getValueByPath(fromObject, []string{"response"}) != nil {
		return nil, fmt.Errorf("response parameter is not supported in Google AI")
	}

	fromDescription := getValueByPath(fromObject, []string{"description"})
	if fromDescription != nil {
		setValueByPath(toObject, []string{"description"}, fromDescription)
	}

	fromName := getValueByPath(fromObject, []string{"name"})
	if fromName != nil {
		setValueByPath(toObject, []string{"name"}, fromName)
	}

	fromParameters := getValueByPath(fromObject, []string{"parameters"})
	if fromParameters != nil {
		setValueByPath(toObject, []string{"parameters"}, fromParameters)
	}

	return toObject, nil
}

func functionDeclarationToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromResponse := getValueByPath(fromObject, []string{"response"})
	if fromResponse != nil {
		fromResponse, err = schemaToVertex(ac, fromResponse.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"response"}, fromResponse)
	}

	fromDescription := getValueByPath(fromObject, []string{"description"})
	if fromDescription != nil {
		setValueByPath(toObject, []string{"description"}, fromDescription)
	}

	fromName := getValueByPath(fromObject, []string{"name"})
	if fromName != nil {
		setValueByPath(toObject, []string{"name"}, fromName)
	}

	fromParameters := getValueByPath(fromObject, []string{"parameters"})
	if fromParameters != nil {
		setValueByPath(toObject, []string{"parameters"}, fromParameters)
	}

	return toObject, nil
}

func googleSearchToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	return toObject, nil
}

func googleSearchToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	return toObject, nil
}

func dynamicRetrievalConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromMode := getValueByPath(fromObject, []string{"mode"})
	if fromMode != nil {
		setValueByPath(toObject, []string{"mode"}, fromMode)
	}

	fromDynamicThreshold := getValueByPath(fromObject, []string{"dynamicThreshold"})
	if fromDynamicThreshold != nil {
		setValueByPath(toObject, []string{"dynamicThreshold"}, fromDynamicThreshold)
	}

	return toObject, nil
}

func dynamicRetrievalConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromMode := getValueByPath(fromObject, []string{"mode"})
	if fromMode != nil {
		setValueByPath(toObject, []string{"mode"}, fromMode)
	}

	fromDynamicThreshold := getValueByPath(fromObject, []string{"dynamicThreshold"})
	if fromDynamicThreshold != nil {
		setValueByPath(toObject, []string{"dynamicThreshold"}, fromDynamicThreshold)
	}

	return toObject, nil
}

func googleSearchRetrievalToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromDynamicRetrievalConfig := getValueByPath(fromObject, []string{"dynamicRetrievalConfig"})
	if fromDynamicRetrievalConfig != nil {
		fromDynamicRetrievalConfig, err = dynamicRetrievalConfigToMldev(ac, fromDynamicRetrievalConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"dynamicRetrievalConfig"}, fromDynamicRetrievalConfig)
	}

	return toObject, nil
}

func googleSearchRetrievalToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromDynamicRetrievalConfig := getValueByPath(fromObject, []string{"dynamicRetrievalConfig"})
	if fromDynamicRetrievalConfig != nil {
		fromDynamicRetrievalConfig, err = dynamicRetrievalConfigToVertex(ac, fromDynamicRetrievalConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"dynamicRetrievalConfig"}, fromDynamicRetrievalConfig)
	}

	return toObject, nil
}

func toolToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromFunctionDeclarations := getValueByPath(fromObject, []string{"functionDeclarations"})
	if fromFunctionDeclarations != nil {
		fromFunctionDeclarations, err = applyConverterToSlice(ac, fromFunctionDeclarations.([]any), functionDeclarationToMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"functionDeclarations"}, fromFunctionDeclarations)
	}

	if getValueByPath(fromObject, []string{"retrieval"}) != nil {
		return nil, fmt.Errorf("retrieval parameter is not supported in Google AI")
	}

	fromGoogleSearch := getValueByPath(fromObject, []string{"googleSearch"})
	if fromGoogleSearch != nil {
		fromGoogleSearch, err = googleSearchToMldev(ac, fromGoogleSearch.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"googleSearch"}, fromGoogleSearch)
	}

	fromGoogleSearchRetrieval := getValueByPath(fromObject, []string{"googleSearchRetrieval"})
	if fromGoogleSearchRetrieval != nil {
		fromGoogleSearchRetrieval, err = googleSearchRetrievalToMldev(ac, fromGoogleSearchRetrieval.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"googleSearchRetrieval"}, fromGoogleSearchRetrieval)
	}

	fromCodeExecution := getValueByPath(fromObject, []string{"codeExecution"})
	if fromCodeExecution != nil {
		setValueByPath(toObject, []string{"codeExecution"}, fromCodeExecution)
	}

	return toObject, nil
}

func toolToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromFunctionDeclarations := getValueByPath(fromObject, []string{"functionDeclarations"})
	if fromFunctionDeclarations != nil {
		fromFunctionDeclarations, err = applyConverterToSlice(ac, fromFunctionDeclarations.([]any), functionDeclarationToVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"functionDeclarations"}, fromFunctionDeclarations)
	}

	fromRetrieval := getValueByPath(fromObject, []string{"retrieval"})
	if fromRetrieval != nil {
		setValueByPath(toObject, []string{"retrieval"}, fromRetrieval)
	}

	fromGoogleSearch := getValueByPath(fromObject, []string{"googleSearch"})
	if fromGoogleSearch != nil {
		fromGoogleSearch, err = googleSearchToVertex(ac, fromGoogleSearch.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"googleSearch"}, fromGoogleSearch)
	}

	fromGoogleSearchRetrieval := getValueByPath(fromObject, []string{"googleSearchRetrieval"})
	if fromGoogleSearchRetrieval != nil {
		fromGoogleSearchRetrieval, err = googleSearchRetrievalToVertex(ac, fromGoogleSearchRetrieval.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"googleSearchRetrieval"}, fromGoogleSearchRetrieval)
	}

	fromCodeExecution := getValueByPath(fromObject, []string{"codeExecution"})
	if fromCodeExecution != nil {
		setValueByPath(toObject, []string{"codeExecution"}, fromCodeExecution)
	}

	return toObject, nil
}

func functionCallingConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromMode := getValueByPath(fromObject, []string{"mode"})
	if fromMode != nil {
		setValueByPath(toObject, []string{"mode"}, fromMode)
	}

	fromAllowedFunctionNames := getValueByPath(fromObject, []string{"allowedFunctionNames"})
	if fromAllowedFunctionNames != nil {
		setValueByPath(toObject, []string{"allowedFunctionNames"}, fromAllowedFunctionNames)
	}

	return toObject, nil
}

func functionCallingConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromMode := getValueByPath(fromObject, []string{"mode"})
	if fromMode != nil {
		setValueByPath(toObject, []string{"mode"}, fromMode)
	}

	fromAllowedFunctionNames := getValueByPath(fromObject, []string{"allowedFunctionNames"})
	if fromAllowedFunctionNames != nil {
		setValueByPath(toObject, []string{"allowedFunctionNames"}, fromAllowedFunctionNames)
	}

	return toObject, nil
}

func toolConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromFunctionCallingConfig := getValueByPath(fromObject, []string{"functionCallingConfig"})
	if fromFunctionCallingConfig != nil {
		fromFunctionCallingConfig, err = functionCallingConfigToMldev(ac, fromFunctionCallingConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"functionCallingConfig"}, fromFunctionCallingConfig)
	}

	return toObject, nil
}

func toolConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromFunctionCallingConfig := getValueByPath(fromObject, []string{"functionCallingConfig"})
	if fromFunctionCallingConfig != nil {
		fromFunctionCallingConfig, err = functionCallingConfigToVertex(ac, fromFunctionCallingConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"functionCallingConfig"}, fromFunctionCallingConfig)
	}

	return toObject, nil
}

func prebuiltVoiceConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromVoiceName := getValueByPath(fromObject, []string{"voiceName"})
	if fromVoiceName != nil {
		setValueByPath(toObject, []string{"voiceName"}, fromVoiceName)
	}

	return toObject, nil
}

func prebuiltVoiceConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromVoiceName := getValueByPath(fromObject, []string{"voiceName"})
	if fromVoiceName != nil {
		setValueByPath(toObject, []string{"voiceName"}, fromVoiceName)
	}

	return toObject, nil
}

func voiceConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromPrebuiltVoiceConfig := getValueByPath(fromObject, []string{"prebuiltVoiceConfig"})
	if fromPrebuiltVoiceConfig != nil {
		fromPrebuiltVoiceConfig, err = prebuiltVoiceConfigToMldev(ac, fromPrebuiltVoiceConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"prebuiltVoiceConfig"}, fromPrebuiltVoiceConfig)
	}

	return toObject, nil
}

func voiceConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromPrebuiltVoiceConfig := getValueByPath(fromObject, []string{"prebuiltVoiceConfig"})
	if fromPrebuiltVoiceConfig != nil {
		fromPrebuiltVoiceConfig, err = prebuiltVoiceConfigToVertex(ac, fromPrebuiltVoiceConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"prebuiltVoiceConfig"}, fromPrebuiltVoiceConfig)
	}

	return toObject, nil
}

func speechConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromVoiceConfig := getValueByPath(fromObject, []string{"voiceConfig"})
	if fromVoiceConfig != nil {
		fromVoiceConfig, err = voiceConfigToMldev(ac, fromVoiceConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"voiceConfig"}, fromVoiceConfig)
	}

	return toObject, nil
}

func speechConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromVoiceConfig := getValueByPath(fromObject, []string{"voiceConfig"})
	if fromVoiceConfig != nil {
		fromVoiceConfig, err = voiceConfigToVertex(ac, fromVoiceConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"voiceConfig"}, fromVoiceConfig)
	}

	return toObject, nil
}

func generateContentConfigToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromSystemInstruction := getValueByPath(fromObject, []string{"systemInstruction"})
	if fromSystemInstruction != nil {
		fromSystemInstruction, err = tContent(ac, fromSystemInstruction)
		if err != nil {
			return nil, err
		}

		fromSystemInstruction, err = contentToMldev(ac, fromSystemInstruction.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"systemInstruction"}, fromSystemInstruction)
	}

	fromTemperature := getValueByPath(fromObject, []string{"temperature"})
	if fromTemperature != nil {
		setValueByPath(toObject, []string{"temperature"}, fromTemperature)
	}

	fromTopP := getValueByPath(fromObject, []string{"topP"})
	if fromTopP != nil {
		setValueByPath(toObject, []string{"topP"}, fromTopP)
	}

	fromTopK := getValueByPath(fromObject, []string{"topK"})
	if fromTopK != nil {
		setValueByPath(toObject, []string{"topK"}, fromTopK)
	}

	fromCandidateCount := getValueByPath(fromObject, []string{"candidateCount"})
	if fromCandidateCount != nil {
		setValueByPath(toObject, []string{"candidateCount"}, fromCandidateCount)
	}

	fromMaxOutputTokens := getValueByPath(fromObject, []string{"maxOutputTokens"})
	if fromMaxOutputTokens != nil {
		setValueByPath(toObject, []string{"maxOutputTokens"}, fromMaxOutputTokens)
	}

	fromStopSequences := getValueByPath(fromObject, []string{"stopSequences"})
	if fromStopSequences != nil {
		setValueByPath(toObject, []string{"stopSequences"}, fromStopSequences)
	}

	fromResponseLogprobs := getValueByPath(fromObject, []string{"responseLogprobs"})
	if fromResponseLogprobs != nil {
		setValueByPath(toObject, []string{"responseLogprobs"}, fromResponseLogprobs)
	}

	fromLogprobs := getValueByPath(fromObject, []string{"logprobs"})
	if fromLogprobs != nil {
		setValueByPath(toObject, []string{"logprobs"}, fromLogprobs)
	}

	fromPresencePenalty := getValueByPath(fromObject, []string{"presencePenalty"})
	if fromPresencePenalty != nil {
		setValueByPath(toObject, []string{"presencePenalty"}, fromPresencePenalty)
	}

	fromFrequencyPenalty := getValueByPath(fromObject, []string{"frequencyPenalty"})
	if fromFrequencyPenalty != nil {
		setValueByPath(toObject, []string{"frequencyPenalty"}, fromFrequencyPenalty)
	}

	fromSeed := getValueByPath(fromObject, []string{"seed"})
	if fromSeed != nil {
		setValueByPath(toObject, []string{"seed"}, fromSeed)
	}

	fromResponseMimeType := getValueByPath(fromObject, []string{"responseMimeType"})
	if fromResponseMimeType != nil {
		setValueByPath(toObject, []string{"responseMimeType"}, fromResponseMimeType)
	}

	fromResponseSchema := getValueByPath(fromObject, []string{"responseSchema"})
	if fromResponseSchema != nil {
		fromResponseSchema, err = tSchema(ac, fromResponseSchema)
		if err != nil {
			return nil, err
		}

		fromResponseSchema, err = schemaToMldev(ac, fromResponseSchema.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"responseSchema"}, fromResponseSchema)
	}

	if getValueByPath(fromObject, []string{"routingConfig"}) != nil {
		return nil, fmt.Errorf("routing_config parameter is not supported in Google AI")
	}

	fromSafetySettings := getValueByPath(fromObject, []string{"safetySettings"})
	if fromSafetySettings != nil {
		fromSafetySettings, err = applyConverterToSlice(ac, fromSafetySettings.([]any), safetySettingToMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"safetySettings"}, fromSafetySettings)
	}

	fromTools := getValueByPath(fromObject, []string{"tools"})
	if fromTools != nil {
		fromTools, err = applyItemTransformerToSlice(ac, fromTools.([]any), tTool)

		fromTools, err = tTools(ac, fromTools)
		if err != nil {
			return nil, err
		}

		fromTools, err = applyConverterToSlice(ac, fromTools.([]any), toolToMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"tools"}, fromTools)
	}

	fromToolConfig := getValueByPath(fromObject, []string{"toolConfig"})
	if fromToolConfig != nil {
		fromToolConfig, err = toolConfigToMldev(ac, fromToolConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"toolConfig"}, fromToolConfig)
	}

	fromCachedContent := getValueByPath(fromObject, []string{"cachedContent"})
	if fromCachedContent != nil {
		fromCachedContent, err = tCachedContentName(ac, fromCachedContent)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"cachedContent"}, fromCachedContent)
	}

	fromResponseModalities := getValueByPath(fromObject, []string{"responseModalities"})
	if fromResponseModalities != nil {
		setValueByPath(toObject, []string{"responseModalities"}, fromResponseModalities)
	}

	if getValueByPath(fromObject, []string{"mediaResolution"}) != nil {
		return nil, fmt.Errorf("media_resolution parameter is not supported in Google AI")
	}

	fromSpeechConfig := getValueByPath(fromObject, []string{"speechConfig"})
	if fromSpeechConfig != nil {
		fromSpeechConfig, err = tSpeechConfig(ac, fromSpeechConfig)
		if err != nil {
			return nil, err
		}

		fromSpeechConfig, err = speechConfigToMldev(ac, fromSpeechConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"speechConfig"}, fromSpeechConfig)
	}

	return toObject, nil
}

func generateContentConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromSystemInstruction := getValueByPath(fromObject, []string{"systemInstruction"})
	if fromSystemInstruction != nil {
		fromSystemInstruction, err = tContent(ac, fromSystemInstruction)
		if err != nil {
			return nil, err
		}

		fromSystemInstruction, err = contentToVertex(ac, fromSystemInstruction.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"systemInstruction"}, fromSystemInstruction)
	}

	fromTemperature := getValueByPath(fromObject, []string{"temperature"})
	if fromTemperature != nil {
		setValueByPath(toObject, []string{"temperature"}, fromTemperature)
	}

	fromTopP := getValueByPath(fromObject, []string{"topP"})
	if fromTopP != nil {
		setValueByPath(toObject, []string{"topP"}, fromTopP)
	}

	fromTopK := getValueByPath(fromObject, []string{"topK"})
	if fromTopK != nil {
		setValueByPath(toObject, []string{"topK"}, fromTopK)
	}

	fromCandidateCount := getValueByPath(fromObject, []string{"candidateCount"})
	if fromCandidateCount != nil {
		setValueByPath(toObject, []string{"candidateCount"}, fromCandidateCount)
	}

	fromMaxOutputTokens := getValueByPath(fromObject, []string{"maxOutputTokens"})
	if fromMaxOutputTokens != nil {
		setValueByPath(toObject, []string{"maxOutputTokens"}, fromMaxOutputTokens)
	}

	fromStopSequences := getValueByPath(fromObject, []string{"stopSequences"})
	if fromStopSequences != nil {
		setValueByPath(toObject, []string{"stopSequences"}, fromStopSequences)
	}

	fromResponseLogprobs := getValueByPath(fromObject, []string{"responseLogprobs"})
	if fromResponseLogprobs != nil {
		setValueByPath(toObject, []string{"responseLogprobs"}, fromResponseLogprobs)
	}

	fromLogprobs := getValueByPath(fromObject, []string{"logprobs"})
	if fromLogprobs != nil {
		setValueByPath(toObject, []string{"logprobs"}, fromLogprobs)
	}

	fromPresencePenalty := getValueByPath(fromObject, []string{"presencePenalty"})
	if fromPresencePenalty != nil {
		setValueByPath(toObject, []string{"presencePenalty"}, fromPresencePenalty)
	}

	fromFrequencyPenalty := getValueByPath(fromObject, []string{"frequencyPenalty"})
	if fromFrequencyPenalty != nil {
		setValueByPath(toObject, []string{"frequencyPenalty"}, fromFrequencyPenalty)
	}

	fromSeed := getValueByPath(fromObject, []string{"seed"})
	if fromSeed != nil {
		setValueByPath(toObject, []string{"seed"}, fromSeed)
	}

	fromResponseMimeType := getValueByPath(fromObject, []string{"responseMimeType"})
	if fromResponseMimeType != nil {
		setValueByPath(toObject, []string{"responseMimeType"}, fromResponseMimeType)
	}

	fromResponseSchema := getValueByPath(fromObject, []string{"responseSchema"})
	if fromResponseSchema != nil {
		fromResponseSchema, err = tSchema(ac, fromResponseSchema)
		if err != nil {
			return nil, err
		}

		fromResponseSchema, err = schemaToVertex(ac, fromResponseSchema.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"responseSchema"}, fromResponseSchema)
	}

	fromRoutingConfig := getValueByPath(fromObject, []string{"routingConfig"})
	if fromRoutingConfig != nil {
		setValueByPath(toObject, []string{"routingConfig"}, fromRoutingConfig)
	}

	fromSafetySettings := getValueByPath(fromObject, []string{"safetySettings"})
	if fromSafetySettings != nil {
		fromSafetySettings, err = applyConverterToSlice(ac, fromSafetySettings.([]any), safetySettingToVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"safetySettings"}, fromSafetySettings)
	}

	fromTools := getValueByPath(fromObject, []string{"tools"})
	if fromTools != nil {
		fromTools, err = applyItemTransformerToSlice(ac, fromTools.([]any), tTool)

		fromTools, err = tTools(ac, fromTools)
		if err != nil {
			return nil, err
		}

		fromTools, err = applyConverterToSlice(ac, fromTools.([]any), toolToVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"tools"}, fromTools)
	}

	fromToolConfig := getValueByPath(fromObject, []string{"toolConfig"})
	if fromToolConfig != nil {
		fromToolConfig, err = toolConfigToVertex(ac, fromToolConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"toolConfig"}, fromToolConfig)
	}

	fromCachedContent := getValueByPath(fromObject, []string{"cachedContent"})
	if fromCachedContent != nil {
		fromCachedContent, err = tCachedContentName(ac, fromCachedContent)
		if err != nil {
			return nil, err
		}

		setValueByPath(parentObject, []string{"cachedContent"}, fromCachedContent)
	}

	fromResponseModalities := getValueByPath(fromObject, []string{"responseModalities"})
	if fromResponseModalities != nil {
		setValueByPath(toObject, []string{"responseModalities"}, fromResponseModalities)
	}

	fromMediaResolution := getValueByPath(fromObject, []string{"mediaResolution"})
	if fromMediaResolution != nil {
		setValueByPath(toObject, []string{"mediaResolution"}, fromMediaResolution)
	}

	fromSpeechConfig := getValueByPath(fromObject, []string{"speechConfig"})
	if fromSpeechConfig != nil {
		fromSpeechConfig, err = tSpeechConfig(ac, fromSpeechConfig)
		if err != nil {
			return nil, err
		}

		fromSpeechConfig, err = speechConfigToVertex(ac, fromSpeechConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"speechConfig"}, fromSpeechConfig)
	}

	return toObject, nil
}

func generateContentParametersToMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromModel := getValueByPath(fromObject, []string{"model"})
	if fromModel != nil {
		fromModel, err = tModel(ac, fromModel)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"_url", "model"}, fromModel)
	}

	fromContents := getValueByPath(fromObject, []string{"contents"})
	if fromContents != nil {
		fromContents, err = tContents(ac, fromContents)
		if err != nil {
			return nil, err
		}

		fromContents, err = applyConverterToSlice(ac, fromContents.([]any), contentToMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"contents"}, fromContents)
	}

	fromConfig := getValueByPath(fromObject, []string{"config"})
	if fromConfig != nil {
		fromConfig, err = generateContentConfigToMldev(ac, fromConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"generationConfig"}, fromConfig)
	}

	return toObject, nil
}

func generateContentParametersToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromModel := getValueByPath(fromObject, []string{"model"})
	if fromModel != nil {
		fromModel, err = tModel(ac, fromModel)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"_url", "model"}, fromModel)
	}

	fromContents := getValueByPath(fromObject, []string{"contents"})
	if fromContents != nil {
		fromContents, err = tContents(ac, fromContents)
		if err != nil {
			return nil, err
		}

		fromContents, err = applyConverterToSlice(ac, fromContents.([]any), contentToVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"contents"}, fromContents)
	}

	fromConfig := getValueByPath(fromObject, []string{"config"})
	if fromConfig != nil {
		fromConfig, err = generateContentConfigToVertex(ac, fromConfig.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"generationConfig"}, fromConfig)
	}

	return toObject, nil
}

func partFromMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromThought := getValueByPath(fromObject, []string{"thought"})
	if fromThought != nil {
		setValueByPath(toObject, []string{"thought"}, fromThought)
	}

	fromCodeExecutionResult := getValueByPath(fromObject, []string{"codeExecutionResult"})
	if fromCodeExecutionResult != nil {
		setValueByPath(toObject, []string{"codeExecutionResult"}, fromCodeExecutionResult)
	}

	fromExecutableCode := getValueByPath(fromObject, []string{"executableCode"})
	if fromExecutableCode != nil {
		setValueByPath(toObject, []string{"executableCode"}, fromExecutableCode)
	}

	fromFileData := getValueByPath(fromObject, []string{"fileData"})
	if fromFileData != nil {
		setValueByPath(toObject, []string{"fileData"}, fromFileData)
	}

	fromFunctionCall := getValueByPath(fromObject, []string{"functionCall"})
	if fromFunctionCall != nil {
		setValueByPath(toObject, []string{"functionCall"}, fromFunctionCall)
	}

	fromFunctionResponse := getValueByPath(fromObject, []string{"functionResponse"})
	if fromFunctionResponse != nil {
		setValueByPath(toObject, []string{"functionResponse"}, fromFunctionResponse)
	}

	fromInlineData := getValueByPath(fromObject, []string{"inlineData"})
	if fromInlineData != nil {
		setValueByPath(toObject, []string{"inlineData"}, fromInlineData)
	}

	fromText := getValueByPath(fromObject, []string{"text"})
	if fromText != nil {
		setValueByPath(toObject, []string{"text"}, fromText)
	}

	return toObject, nil
}

func partFromVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromVideoMetadata := getValueByPath(fromObject, []string{"videoMetadata"})
	if fromVideoMetadata != nil {
		setValueByPath(toObject, []string{"videoMetadata"}, fromVideoMetadata)
	}

	fromThought := getValueByPath(fromObject, []string{"thought"})
	if fromThought != nil {
		setValueByPath(toObject, []string{"thought"}, fromThought)
	}

	fromCodeExecutionResult := getValueByPath(fromObject, []string{"codeExecutionResult"})
	if fromCodeExecutionResult != nil {
		setValueByPath(toObject, []string{"codeExecutionResult"}, fromCodeExecutionResult)
	}

	fromExecutableCode := getValueByPath(fromObject, []string{"executableCode"})
	if fromExecutableCode != nil {
		setValueByPath(toObject, []string{"executableCode"}, fromExecutableCode)
	}

	fromFileData := getValueByPath(fromObject, []string{"fileData"})
	if fromFileData != nil {
		setValueByPath(toObject, []string{"fileData"}, fromFileData)
	}

	fromFunctionCall := getValueByPath(fromObject, []string{"functionCall"})
	if fromFunctionCall != nil {
		setValueByPath(toObject, []string{"functionCall"}, fromFunctionCall)
	}

	fromFunctionResponse := getValueByPath(fromObject, []string{"functionResponse"})
	if fromFunctionResponse != nil {
		setValueByPath(toObject, []string{"functionResponse"}, fromFunctionResponse)
	}

	fromInlineData := getValueByPath(fromObject, []string{"inlineData"})
	if fromInlineData != nil {
		setValueByPath(toObject, []string{"inlineData"}, fromInlineData)
	}

	fromText := getValueByPath(fromObject, []string{"text"})
	if fromText != nil {
		setValueByPath(toObject, []string{"text"}, fromText)
	}

	return toObject, nil
}

func contentFromMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromParts := getValueByPath(fromObject, []string{"parts"})
	if fromParts != nil {
		fromParts, err = applyConverterToSlice(ac, fromParts.([]any), partFromMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"parts"}, fromParts)
	}

	fromRole := getValueByPath(fromObject, []string{"role"})
	if fromRole != nil {
		setValueByPath(toObject, []string{"role"}, fromRole)
	}

	return toObject, nil
}

func contentFromVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromParts := getValueByPath(fromObject, []string{"parts"})
	if fromParts != nil {
		fromParts, err = applyConverterToSlice(ac, fromParts.([]any), partFromVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"parts"}, fromParts)
	}

	fromRole := getValueByPath(fromObject, []string{"role"})
	if fromRole != nil {
		setValueByPath(toObject, []string{"role"}, fromRole)
	}

	return toObject, nil
}

func citationMetadataFromMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromCitations := getValueByPath(fromObject, []string{"citationSources"})
	if fromCitations != nil {
		setValueByPath(toObject, []string{"citations"}, fromCitations)
	}

	return toObject, nil
}

func citationMetadataFromVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromCitations := getValueByPath(fromObject, []string{"citations"})
	if fromCitations != nil {
		setValueByPath(toObject, []string{"citations"}, fromCitations)
	}

	return toObject, nil
}

func candidateFromMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromContent := getValueByPath(fromObject, []string{"content"})
	if fromContent != nil {
		fromContent, err = contentFromMldev(ac, fromContent.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"content"}, fromContent)
	}

	fromCitationMetadata := getValueByPath(fromObject, []string{"citationMetadata"})
	if fromCitationMetadata != nil {
		fromCitationMetadata, err = citationMetadataFromMldev(ac, fromCitationMetadata.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"citationMetadata"}, fromCitationMetadata)
	}

	fromTokenCount := getValueByPath(fromObject, []string{"tokenCount"})
	if fromTokenCount != nil {
		setValueByPath(toObject, []string{"tokenCount"}, fromTokenCount)
	}

	fromAvgLogprobs := getValueByPath(fromObject, []string{"avgLogprobs"})
	if fromAvgLogprobs != nil {
		setValueByPath(toObject, []string{"avgLogprobs"}, fromAvgLogprobs)
	}

	fromFinishReason := getValueByPath(fromObject, []string{"finishReason"})
	if fromFinishReason != nil {
		setValueByPath(toObject, []string{"finishReason"}, fromFinishReason)
	}

	fromGroundingMetadata := getValueByPath(fromObject, []string{"groundingMetadata"})
	if fromGroundingMetadata != nil {
		setValueByPath(toObject, []string{"groundingMetadata"}, fromGroundingMetadata)
	}

	fromIndex := getValueByPath(fromObject, []string{"index"})
	if fromIndex != nil {
		setValueByPath(toObject, []string{"index"}, fromIndex)
	}

	fromLogprobsResult := getValueByPath(fromObject, []string{"logprobsResult"})
	if fromLogprobsResult != nil {
		setValueByPath(toObject, []string{"logprobsResult"}, fromLogprobsResult)
	}

	fromSafetyRatings := getValueByPath(fromObject, []string{"safetyRatings"})
	if fromSafetyRatings != nil {
		setValueByPath(toObject, []string{"safetyRatings"}, fromSafetyRatings)
	}

	return toObject, nil
}

func candidateFromVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromContent := getValueByPath(fromObject, []string{"content"})
	if fromContent != nil {
		fromContent, err = contentFromVertex(ac, fromContent.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"content"}, fromContent)
	}

	fromCitationMetadata := getValueByPath(fromObject, []string{"citationMetadata"})
	if fromCitationMetadata != nil {
		fromCitationMetadata, err = citationMetadataFromVertex(ac, fromCitationMetadata.(map[string]any), toObject)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"citationMetadata"}, fromCitationMetadata)
	}

	fromFinishMessage := getValueByPath(fromObject, []string{"finishMessage"})
	if fromFinishMessage != nil {
		setValueByPath(toObject, []string{"finishMessage"}, fromFinishMessage)
	}

	fromAvgLogprobs := getValueByPath(fromObject, []string{"avgLogprobs"})
	if fromAvgLogprobs != nil {
		setValueByPath(toObject, []string{"avgLogprobs"}, fromAvgLogprobs)
	}

	fromFinishReason := getValueByPath(fromObject, []string{"finishReason"})
	if fromFinishReason != nil {
		setValueByPath(toObject, []string{"finishReason"}, fromFinishReason)
	}

	fromGroundingMetadata := getValueByPath(fromObject, []string{"groundingMetadata"})
	if fromGroundingMetadata != nil {
		setValueByPath(toObject, []string{"groundingMetadata"}, fromGroundingMetadata)
	}

	fromIndex := getValueByPath(fromObject, []string{"index"})
	if fromIndex != nil {
		setValueByPath(toObject, []string{"index"}, fromIndex)
	}

	fromLogprobsResult := getValueByPath(fromObject, []string{"logprobsResult"})
	if fromLogprobsResult != nil {
		setValueByPath(toObject, []string{"logprobsResult"}, fromLogprobsResult)
	}

	fromSafetyRatings := getValueByPath(fromObject, []string{"safetyRatings"})
	if fromSafetyRatings != nil {
		setValueByPath(toObject, []string{"safetyRatings"}, fromSafetyRatings)
	}

	return toObject, nil
}

func generateContentResponseFromMldev(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromCandidates := getValueByPath(fromObject, []string{"candidates"})
	if fromCandidates != nil {
		fromCandidates, err = applyConverterToSlice(ac, fromCandidates.([]any), candidateFromMldev)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"candidates"}, fromCandidates)
	}

	fromModelVersion := getValueByPath(fromObject, []string{"modelVersion"})
	if fromModelVersion != nil {
		setValueByPath(toObject, []string{"modelVersion"}, fromModelVersion)
	}

	fromPromptFeedback := getValueByPath(fromObject, []string{"promptFeedback"})
	if fromPromptFeedback != nil {
		setValueByPath(toObject, []string{"promptFeedback"}, fromPromptFeedback)
	}

	fromUsageMetadata := getValueByPath(fromObject, []string{"usageMetadata"})
	if fromUsageMetadata != nil {
		setValueByPath(toObject, []string{"usageMetadata"}, fromUsageMetadata)
	}

	return toObject, nil
}

func generateContentResponseFromVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
	toObject = make(map[string]any)

	fromCandidates := getValueByPath(fromObject, []string{"candidates"})
	if fromCandidates != nil {
		fromCandidates, err = applyConverterToSlice(ac, fromCandidates.([]any), candidateFromVertex)
		if err != nil {
			return nil, err
		}

		setValueByPath(toObject, []string{"candidates"}, fromCandidates)
	}

	fromModelVersion := getValueByPath(fromObject, []string{"modelVersion"})
	if fromModelVersion != nil {
		setValueByPath(toObject, []string{"modelVersion"}, fromModelVersion)
	}

	fromPromptFeedback := getValueByPath(fromObject, []string{"promptFeedback"})
	if fromPromptFeedback != nil {
		setValueByPath(toObject, []string{"promptFeedback"}, fromPromptFeedback)
	}

	fromUsageMetadata := getValueByPath(fromObject, []string{"usageMetadata"})
	if fromUsageMetadata != nil {
		setValueByPath(toObject, []string{"usageMetadata"}, fromUsageMetadata)
	}

	return toObject, nil
}

type Models struct {
	apiClient *apiClient
}

func (m Models) generateContent(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error) {
	parameterMap := make(map[string]any)

	kwargs := map[string]any{"model": model, "contents": contents, "config": config}
	deepMarshal(kwargs, &parameterMap)

	var response = new(GenerateContentResponse)
	var responseMap map[string]any
	var fromConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	var toConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	if m.apiClient.ClientConfig.Backend == BackendVertexAI {
		toConverter = generateContentParametersToVertex
		fromConverter = generateContentResponseFromVertex
	} else {
		toConverter = generateContentParametersToMldev
		fromConverter = generateContentResponseFromMldev
	}

	body, err := toConverter(m.apiClient, parameterMap, nil)
	if err != nil {
		return nil, err
	}
	urlParams := body["_url"].(map[string]any)
	path, err := formatMap("{model}:generateContent", urlParams)
	if err != nil {
		return nil, fmt.Errorf("invalid url params: %#v.\n%w", urlParams, err)
	}
	delete(body, "_url")
	responseMap, err = sendRequest(ctx, m.apiClient, path, http.MethodPost, &body)
	if err != nil {
		return nil, err
	}
	responseMap, err = fromConverter(m.apiClient, responseMap, nil)
	if err != nil {
		return nil, err
	}
	err = mapToStruct(responseMap, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// generateContentStream ...
func (m Models) generateContentStream(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) iter.Seq2[*GenerateContentResponse, error] {
	parameterMap := make(map[string]any)

	kwargs := map[string]any{"model": model, "contents": contents, "config": config}
	deepMarshal(kwargs, &parameterMap)

	var rs responseStream[GenerateContentResponse]
	var fromConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	var toConverter func(*apiClient, map[string]any, map[string]any) (map[string]any, error)
	yieldErrorAndEndIterator := func(err error) iter.Seq2[*GenerateContentResponse, error] {
		return func(yield func(*GenerateContentResponse, error) bool) {
			if !yield(nil, err) {
				return
			}
		}
	}
	if m.apiClient.ClientConfig.Backend == BackendVertexAI {
		toConverter = generateContentParametersToVertex
		fromConverter = generateContentResponseFromVertex
	} else {
		toConverter = generateContentParametersToMldev
		fromConverter = generateContentResponseFromMldev
	}

	body, err := toConverter(m.apiClient, parameterMap, nil)
	if err != nil {
		return yieldErrorAndEndIterator(err)
	}
	urlParams := body["_url"].(map[string]any)
	path, err := formatMap("{model}:streamGenerateContent?alt=sse", urlParams)
	if err != nil {
		return yieldErrorAndEndIterator(fmt.Errorf("invalid url params: %#v.\n%w", urlParams, err))
	}
	delete(body, "_url")
	err = sendStreamRequest(ctx, m.apiClient, path, http.MethodPost, &body, &rs)
	if err != nil {
		return yieldErrorAndEndIterator(err)
	}
	return iterateResponseStream(&rs, func(responseMap map[string]any) (*GenerateContentResponse, error) {
		responseMap, err := fromConverter(m.apiClient, responseMap, nil)
		if err != nil {
			return nil, err
		}
		var response = new(GenerateContentResponse)
		err = mapToStruct(responseMap, response)
		if err != nil {
			return nil, err
		}
		return response, nil
	})
}

// GenerateContent calls the GenerateContent method on the model.
func (m Models) GenerateContent(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error) {
	if config != nil {
		config.setDefaults()
	}
	for _, c := range contents {
		c.setDefaults()
	}
	return m.generateContent(ctx, model, contents, config)
}

// GenerateContentStream calls the GenerateContentStream method on the model.
func (m Models) GenerateContentStream(ctx context.Context, model string, contents []*Content, config *GenerateContentConfig) iter.Seq2[*GenerateContentResponse, error] {
	if config != nil {
		config.setDefaults()
	}
	for _, c := range contents {
		c.setDefaults()
	}
	return m.generateContentStream(ctx, model, contents, config)
}
