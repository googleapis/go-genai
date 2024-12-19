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
	"strings"
)

// TODO(b/376323000): align with python resource name implementation.
func tResourceName(ac *apiClient, resourceName string, resourcePrefix string) string {
	if ac.clientConfig.Backend == BackendVertexAI {
		if strings.HasPrefix(resourceName, "projects/") {
			return resourceName
		} else if strings.HasPrefix(resourceName, "locations/") {
			return fmt.Sprintf("projects/%s/%s", ac.clientConfig.Project, resourceName)
		} else if strings.HasPrefix(resourceName, fmt.Sprintf("%s/", resourcePrefix)) {
			return fmt.Sprintf("projects/%s/locations/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, resourceName)
		} else {
			return fmt.Sprintf("projects/%s/locations/%s/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, resourcePrefix, resourceName)
		}
	} else {
		if strings.HasPrefix(resourceName, fmt.Sprintf("%s/", resourcePrefix)) {
			return resourceName
		} else {
			return fmt.Sprintf("%s/%s", resourcePrefix, resourceName)
		}
	}
}

func tCachedContentName(ac *apiClient, name any) (string, error) {
	return tResourceName(ac, name.(string), "cachedContents"), nil
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
