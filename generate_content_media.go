// Copyright 2026 WG ModelHub contributors
// Licensed under the Apache License, Version 2.0. See LICENSE.

package genai

// Keep inline media out of the JSON round trip used for parameter conversion.
// All other fields retain the SDK's existing JSON representation.
func marshalGenerateContentParameters(model string, contents []*Content, config *GenerateContentConfig) (map[string]any, error) {
	// The callback historically sees Base64 strings and a snapshot of media bytes.
	if config != nil && config.HTTPOptions != nil && config.HTTPOptions.ExtrasRequestProvider != nil {
		var parameters map[string]any
		err := deepMarshal(map[string]any{"model": model, "contents": contents, "config": config}, &parameters)
		return parameters, err
	}
	lightContents := contents
	if contents != nil {
		lightContents = make([]*Content, len(contents))
		for i, content := range contents {
			lightContents[i] = contentWithoutInlineData(content)
		}
	}
	lightConfig := config
	if config != nil {
		lightSystem := contentWithoutInlineData(config.SystemInstruction)
		if lightSystem != config.SystemInstruction {
			cloned := *config
			cloned.SystemInstruction = lightSystem
			lightConfig = &cloned
		}
	}
	var parameters map[string]any
	if err := deepMarshal(map[string]any{"model": model, "contents": lightContents, "config": lightConfig}, &parameters); err != nil {
		return nil, err
	}
	for i, content := range contents {
		if lightContents[i] != content {
			restoreInlineData(parameters["contents"].([]any)[i].(map[string]any), content)
		}
	}
	if lightConfig != config {
		restoreInlineData(parameters["config"].(map[string]any)["systemInstruction"].(map[string]any), config.SystemInstruction)
	}
	return parameters, nil
}

func contentWithoutInlineData(content *Content) *Content {
	if content == nil {
		return nil
	}
	result := content
	for i, part := range content.Parts {
		if part == nil || part.InlineData == nil || len(part.InlineData.Data) == 0 {
			continue
		}
		if result == content {
			cloned := *content
			cloned.Parts = append([]*Part(nil), content.Parts...)
			result = &cloned
		}
		clonedPart := *part
		clonedBlob := *part.InlineData
		clonedBlob.Data = nil
		clonedPart.InlineData = &clonedBlob
		result.Parts[i] = &clonedPart
	}
	return result
}

func restoreInlineData(target map[string]any, source *Content) {
	parts := target["parts"].([]any)
	for i, part := range source.Parts {
		if part != nil && part.InlineData != nil && len(part.InlineData.Data) > 0 {
			parts[i].(map[string]any)["inlineData"].(map[string]any)["data"] = part.InlineData.Data
		}
	}
}
