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
	"regexp"
	"strconv"
	"strings"
)

// InternalTResourceName is an internal function used for transforming resource names.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTResourceName(ac *InternalAPIClient, resourceName string, collectionIdentifier string, collectionHierarchyDepth int) string {
	return tResourceName(ac, resourceName, collectionIdentifier, collectionHierarchyDepth)
}

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

// InternalTCachedContentName is an internal function used for transforming cached content names.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTCachedContentName(ac *InternalAPIClient, name any) (string, error) {
	return tCachedContentName(ac, name)
}

func tCachedContentName(ac *apiClient, name any) (string, error) {
	return tResourceName(ac, name.(string), "cachedContents", 2), nil
}

// InternalTModel is an internal function used for transforming model names.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTModel(ac *InternalAPIClient, origin any) (string, error) {
	return tModel(ac, origin)
}

func tModel(ac *apiClient, origin any) (string, error) {
	switch model := origin.(type) {
	case string:
		if model == "" {
			return "", fmt.Errorf("tModel: model is empty")
		}
		if strings.Contains(model, "?") || strings.Contains(model, "&") || strings.Contains(model, "..") {
			return "", fmt.Errorf("tModel: invalid model parameter")
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

// InternalTModelFullName is an internal function used for transforming full model names.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTModelFullName(ac *InternalAPIClient, origin any) (string, error) {
	return tModelFullName(ac, origin)
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

// InternalTCachesModel is an internal function used for transforming cached model names.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTCachesModel(ac *InternalAPIClient, origin any) (string, error) {
	return tCachesModel(ac, origin)
}

func tCachesModel(ac *apiClient, origin any) (string, error) {
	return tModelFullName(ac, origin)
}

// InternalTContent is an internal function used for transforming content.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTContent(content any) (any, error) {
	return tContent(content)
}

func tContent(content any) (any, error) {
	return content, nil
}

// InternalTContents is an internal function used for transforming contents.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTContents(contents any) (any, error) {
	return tContents(contents)
}

func tContents(contents any) (any, error) {
	return contents, nil
}

// InternalTTool is an internal function used for transforming tools.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTTool(tool any) (any, error) {
	return tTool(tool)
}

func tTool(tool any) (any, error) {
	return tool, nil
}

// InternalTTools is an internal function used for transforming collections of tools.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTTools(tools any) (any, error) {
	return tTools(tools)
}

func tTools(tools any) (any, error) {
	return tools, nil
}

// InternalTSchema is an internal function used for transforming schemas.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTSchema(origin any) (any, error) {
	return tSchema(origin)
}

func tSchema(origin any) (any, error) {
	return origin, nil
}

// InternalTSpeechConfig is an internal function used for transforming speech configurations.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTSpeechConfig(speechConfig any) (any, error) {
	return tSpeechConfig(speechConfig)
}

func tSpeechConfig(speechConfig any) (any, error) {
	return speechConfig, nil
}

// InternalTLiveSpeechConfig is an internal function used for transforming live speech configurations.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTLiveSpeechConfig(speechConfig any) (any, error) {
	return tLiveSpeechConfig(speechConfig)
}

func tLiveSpeechConfig(speechConfig any) (any, error) {
	switch config := speechConfig.(type) {
	case map[string]any:
		if _, ok := config["multiSpeakerVoiceConfig"]; ok {
			return nil, fmt.Errorf("multiSpeakerVoiceConfig is not supported in the live API")
		}
		return config, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported speechConfig type: %T", speechConfig)
	}
}

// InternalTBytes is an internal function used for transforming bytes.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTBytes(fromImageBytes any) (any, error) {
	return tBytes(fromImageBytes)
}

func tBytes(fromImageBytes any) (any, error) {
	// TODO(b/389133914): Remove dummy bytes converter.
	return fromImageBytes, nil
}

// InternalTContentsForEmbed is an internal function used for transforming contents for embedding.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTContentsForEmbed(ac *InternalAPIClient, contents any) (any, error) {
	return tContentsForEmbed(ac, contents)
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

// InternalTModelsURL is an internal function used for generating models URL.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTModelsURL(ac *InternalAPIClient, baseModels any) (string, error) {
	return tModelsURL(ac, baseModels)
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

// InternalTExtractModels is an internal function used for extracting models from a response.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTExtractModels(response any) (any, error) {
	return tExtractModels(response)
}

func tExtractModels(response any) (any, error) {
	switch response := response.(type) {
	case map[string]any:
		if models, ok := response["models"]; ok {
			return models, nil
		} else if tunedModels, ok := response["tunedModels"]; ok {
			return tunedModels, nil
		} else if publisherModels, ok := response["publisherModels"]; ok {
			return publisherModels, nil
		} else {
			log.Printf("Warning: Cannot find the models type(models, tunedModels, publisherModels) for response: %s", response)
			return []any{}, nil
		}
	default:
		return nil, fmt.Errorf("tExtractModels: response is not a map")
	}
}

// InternalTFileName is an internal function used for transforming filenames.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTFileName(name any) (string, error) {
	return tFileName(name)
}

func tFileName(name any) (string, error) {
	switch name := name.(type) {
	case string:
		{
			if strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "http://") {
				parts := strings.SplitN(name, "files/", 2)
				if len(parts) < 2 {
					return "", fmt.Errorf("could not find 'files/' in URI: %s", name)
				}
				suffix := parts[1]
				re := regexp.MustCompile("^[a-z0-9]+")
				match := re.FindStringSubmatch(suffix)
				if len(match) == 0 {
					return "", fmt.Errorf("could not extract file name from URI: %s", name)
				}
				name = match[0]
			} else if strings.HasPrefix(name, "files/") {
				name = strings.TrimPrefix(name, "files/")
			}
			return name, nil
		}
	default:
		return "", fmt.Errorf("tFileName: name is not a string")
	}
}

// InternalTBlobs is an internal function used for transforming collections of blobs.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTBlobs(blobs any) (any, error) {
	return tBlobs(blobs)
}

func tBlobs(blobs any) (any, error) {
	switch blobs := blobs.(type) {
	case []any:
		// The only use case of this tBlobs function is for LiveSendRealtimeInputParameters.Media field.
		// The Media field is a Blob type, not a list of Blob. So this branch will never be executed.
		// If tBlobs function is used for other purposes in the future, uncomment the following line to
		// enable this branch.
		// applyConverterToSlice(ac, blobs, tBlob)
		return nil, fmt.Errorf("unimplemented")
	default:
		blob, err := tBlob(blobs)
		if err != nil {
			return nil, err
		}
		return []any{blob}, nil
	}
}

// InternalTBlob is an internal function used for transforming a blob.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTBlob(blob any) (any, error) {
	return tBlob(blob)
}

func tBlob(blob any) (any, error) {
	return blob, nil
}

// InternalTImageBlob is an internal function used for transforming an image blob.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTImageBlob(blob any) (any, error) {
	return tImageBlob(blob)
}

func tImageBlob(blob any) (any, error) {
	switch blob := blob.(type) {
	case map[string]any:
		if strings.HasPrefix(blob["mimeType"].(string), "image/") {
			return blob, nil
		}
		return nil, fmt.Errorf("Unsupported mime type: %s", blob["mimeType"])
	default:
		return nil, fmt.Errorf("tImageBlob: blob is not a map")
	}
}

// InternalTAudioBlob is an internal function used for transforming an audio blob.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTAudioBlob(blob any) (any, error) {
	return tAudioBlob(blob)
}

func tAudioBlob(blob any) (any, error) {
	switch blob := blob.(type) {
	case map[string]any:
		if strings.HasPrefix(blob["mimeType"].(string), "audio/") {
			return blob, nil
		}
		return nil, fmt.Errorf("Unsupported mime type: %s", blob["mimeType"])
	default:
		return nil, fmt.Errorf("tAudioBlob: blob is not a map")
	}
}

// InternalTBatchJobSource is an internal function used for transforming batch job sources.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTBatchJobSource(src any) (any, error) {
	return tBatchJobSource(src)
}

func tBatchJobSource(src any) (any, error) {
	return src, nil
}

// InternalTBatchJobDestination is an internal function used for transforming batch job destinations.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTBatchJobDestination(dest any) (any, error) {
	return tBatchJobDestination(dest)
}

func tBatchJobDestination(dest any) (any, error) {
	return dest, nil
}

// InternalTRecvBatchJobDestination is an internal function used for transforming received batch job destinations.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTRecvBatchJobDestination(dest any) (any, error) {
	return tRecvBatchJobDestination(dest)
}

func tRecvBatchJobDestination(dest any) (any, error) {
	return dest, nil
}

// InternalTBatchJobName is an internal function used for transforming batch job names.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTBatchJobName(ac *InternalAPIClient, name any) (any, error) {
	return tBatchJobName(ac, name)
}

func tBatchJobName(ac *apiClient, name any) (any, error) {
	var (
		mldevBatchPattern  = regexp.MustCompile("batches/[^/]+$")
		vertexBatchPattern = regexp.MustCompile("^projects/[^/]+/locations/[^/]+/batchPredictionJobs/[^/]+$")
	)
	// Convert any to string.
	nameStr := name.(string)
	if ac.clientConfig.Backend == BackendVertexAI {
		if vertexBatchPattern.MatchString(nameStr) {
			parts := strings.Split(nameStr, "/")
			return parts[len(parts)-1], nil
		}
		if _, err := strconv.Atoi(nameStr); err == nil {
			return nameStr, nil
		}
		return nil, fmt.Errorf("Invalid batch job name: %s. Expected format like 'projects/id/locations/id/batchPredictionJobs/id' or 'id'", nameStr)
	}
	if mldevBatchPattern.MatchString(nameStr) {
		parts := strings.Split(nameStr, "/")
		return parts[len(parts)-1], nil
	}
	return nil, fmt.Errorf("Invalid batch job name: %s. Expected format like 'batches/id'", nameStr)
}

// InternalTJobState is an internal function used for transforming job states.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTJobState(state any) (any, error) {
	return tJobState(state)
}

func tJobState(state any) (any, error) {
	switch state {
	case "BATCH_STATE_UNSPECIFIED":
		return "JOB_STATE_UNSPECIFIED", nil
	case "BATCH_STATE_PENDING":
		return "JOB_STATE_PENDING", nil
	case "BATCH_STATE_RUNNING":
		return "JOB_STATE_RUNNING", nil
	case "BATCH_STATE_SUCCEEDED":
		return "JOB_STATE_SUCCEEDED", nil
	case "BATCH_STATE_FAILED":
		return "JOB_STATE_FAILED", nil
	case "BATCH_STATE_CANCELLED":
		return "JOB_STATE_CANCELLED", nil
	case "BATCH_STATE_EXPIRED":
		return "JOB_STATE_EXPIRED", nil
	default:
		return state, nil
	}
}

// InternalTIsVertexEmbedContentModel is an internal function that checks if a model is a Vertex AI embed content model.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTIsVertexEmbedContentModel(model string) bool {
	return tIsVertexEmbedContentModel(model)
}

// tIsVertexEmbedContentModel checks if a model is a Vertex AI embed content model.
// This is the equivalent of t_is_vertex_embed_content_model in the Python SDK.
func tIsVertexEmbedContentModel(model string) bool {
	// Gemini Embeddings except gemini-embedding-001.
	isGeminiEmbed := strings.Contains(model, "gemini") && model != "gemini-embedding-001"
	// Open-source MaaS embedding models.
	isMaaS := strings.Contains(model, "maas")
	return isGeminiEmbed || isMaaS
}

// InternalTTuningJobStatus is an internal function used for transforming tuning job status.
// This function is public only for internal purposes and its support is not guaranteed in future
// versions. External consumers must not use it.
func InternalTTuningJobStatus(state any) (any, error) {
	return tTuningJobStatus(state)
}

func tTuningJobStatus(state any) (any, error) {
	switch state {
	case "STATE_UNSPECIFIED":
		return "JOB_STATE_UNSPECIFIED", nil
	case "CREATING":
		return "JOB_STATE_RUNNING", nil
	case "ACTIVE":
		return "JOB_STATE_SUCCEEDED", nil
	case "FAILED":
		return "JOB_STATE_FAILED", nil
	default:
		return state, nil
	}
}
