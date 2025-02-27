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
	"fmt"
	"log"
	"strings"
)

func tResourceName(ac *apiClient, resourceName string, collectionIdentifier string, collectionHierarchyDepth int) string {
	shouldPrependCollectionIdentifier := !strings.HasPrefix(resourceName, collectionIdentifier+"/") &&
		strings.Count(collectionIdentifier+"/"+resourceName, "/")+1 == collectionHierarchyDepth

	switch ac.clientConfig.Backend {
	case BackendVertexAI:
		if strings.HasPrefix(resourceName, "projects/") {
			return resourceName
		} else if strings.HasPrefix(resourceName, "locations/") {
			return fmt.Sprintf("projects/%s/%s", ac.clientConfig.Project, resourceName)
		} else if strings.HasPrefix(resourceName, collectionIdentifier+"/") {
			return fmt.Sprintf("projects/%s/locations/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, resourceName)
		} else if shouldPrependCollectionIdentifier {
			return fmt.Sprintf("projects/%s/locations/%s/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, collectionIdentifier, resourceName)
		} else {
			return resourceName
		}
	default:
		if shouldPrependCollectionIdentifier {
			return fmt.Sprintf("%s/%s", collectionIdentifier, resourceName)
		} else {
			return resourceName
		}
	}
}

func tCachedContentName(ac *apiClient, name any) (string, error) {
	return tResourceName(ac, name.(string), "cachedContents", 2), nil
}

func tModel(ac *apiClient, origin any) (string, error) {
	switch model := origin.(type) {
	case string:
		if model == "" {
			return "", fmt.Errorf("tModel: model is empty")
		}
		if ac.clientConfig.Backend == BackendVertexAI {
			if strings.HasPrefix(model, "projects/") || strings.HasPrefix(model, "models/") || strings.HasPrefix(model, "publishers/") {
				return model, nil
			} else if strings.Contains(model, "/") {
				parts := strings.SplitN(model, "/", 2)
				return fmt.Sprintf("publishers/%s/models/%s", parts[0], parts[1]), nil
			} else {
				return fmt.Sprintf("publishers/google/models/%s", model), nil
			}
		} else {
			if strings.HasPrefix(model, "models/") || strings.HasPrefix(model, "tunedModels/") {
				return model, nil
			} else {
				return fmt.Sprintf("models/%s", model), nil
			}
		}
	default:
		return "", fmt.Errorf("tModel: model is not a string")
	}
}

func tModelFullName(ac *apiClient, origin any) (string, error) {
	switch model := origin.(type) {
	case string:
		name, err := tModel(ac, model)
		if err != nil {
			return "", fmt.Errorf("tModelFullName: %w", err)
		}
		if strings.HasPrefix(name, "publishers/") && ac.clientConfig.Backend == BackendVertexAI {
			return fmt.Sprintf("projects/%s/locations/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, name), nil
		} else if strings.HasPrefix(name, "models/") && ac.clientConfig.Backend == BackendVertexAI {
			return fmt.Sprintf("projects/%s/locations/%s/publishers/google/%s", ac.clientConfig.Project, ac.clientConfig.Location, name), nil
		} else {
			return name, nil
		}
	default:
		return "", fmt.Errorf("tModelFullName: model is not a string")
	}
}

func tCachesModel(ac *apiClient, origin any) (string, error) {
	return tModelFullName(ac, origin)
}

func tContent(_ *apiClient, content any) (any, error) {
	return content, nil
}

func tContents(_ *apiClient, contents any) (any, error) {
	return contents, nil
}

func tTool(_ *apiClient, tool any) (any, error) {
	return tool, nil
}

func tTools(_ *apiClient, tools any) (any, error) {
	return tools, nil
}

func tSchema(_ *apiClient, origin any) (any, error) {
	return origin, nil
}

func tSpeechConfig(_ *apiClient, speechConfig any) (any, error) {
	return speechConfig, nil
}

func tBytes(_ *apiClient, fromImageBytes any) (any, error) {
	// TODO(b/389133914): Remove dummy bytes converter.
	return fromImageBytes, nil
}

func tContentsForEmbed(ac *apiClient, contents any) (any, error) {
	if ac.clientConfig.Backend == BackendVertexAI {
		switch v := contents.(type) {
		case []any:
			texts := []string{}
			for _, content := range v {
				parts, ok := content.(map[string]any)["parts"].([]any)
				if !ok || len(parts) == 0 {
					return nil, fmt.Errorf("tContentsForEmbed: content parts is not a non-empty list")
				}
				text, ok := parts[0].(map[string]any)["text"].(string)
				if !ok {
					return nil, fmt.Errorf("tContentsForEmbed: content part text is not a string")
				}
				texts = append(texts, text)
			}
			return texts, nil
		default:
			return nil, fmt.Errorf("tContentsForEmbed: contents is not a list")
		}
	} else {
		return contents, nil
	}
}

func tModelsURL(ac *apiClient, baseModels any) (string, error) {
	if ac.clientConfig.Backend == BackendVertexAI {
		if baseModels.(bool) {
			return "publishers/google/models", nil
		} else {
			return "models", nil
		}
	} else {
		if baseModels.(bool) {
			return "models", nil
		} else {
			return "tunedModels", nil
		}
	}
}

func tExtractModels(ac *apiClient, response any) (any, error) {
	switch response.(type) {
	case map[string]any:
		if models, ok := response.(map[string]any)["models"]; ok {
			return models, nil
		} else if tunedModels, ok := response.(map[string]any)["tunedModels"]; ok {
			return tunedModels, nil
		} else if publisherModels, ok := response.(map[string]any)["publisherModels"]; ok {
			return publisherModels, nil
		} else {
			log.Printf("Warning: Cannot find the models type(models, tunedModels, publisherModels) for response: %s", response)
			return []any{}, nil
		}
	default:
		return nil, fmt.Errorf("tExtractModels: response is not a map")
	}
}
