package genai

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"testing"
)

const (
	apiMode    = "api"
	replayMode = "replay"
	unitMode   = "unit" // Unit tests runs in the github actions.
)

// TODO(b/382773687): Enable the TestModelsGenerateContentStream tests.
var (
	disabledTestsForAll = []string{
		// TODO(b/380108306): bytes related tests are not supported in replay tests.
		"vertex/models/generate_content_part/test_image_base64",
		"mldev/models/generate_content_part/test_image_base64",

		// TODO(b/392156165): Support enum value converter/validator
		"mldev/models/generate_images/test_all_vertexai_config_safety_filter_level_enum_parameters",
		"mldev/models/generate_images/test_all_vertexai_config_safety_filter_level_enum_parameters_2",
		"mldev/models/generate_images/test_all_vertexai_config_safety_filter_level_enum_parameters_3",
		"mldev/models/generate_images/test_all_vertexai_config_person_generation_enum_parameters",
		"mldev/models/generate_images/test_all_vertexai_config_person_generation_enum_parameters_2",
		"mldev/models/generate_images/test_all_vertexai_config_person_generation_enum_parameters_3",
	}
	disabledTestsByMode = map[string][]string{
		apiMode: []string{
			"TestModelsGenerateContentStream/mldev/models/generate_content/test_simple_shared_generation_config",
			"TestModelsGenerateContentStream/mldev/models/generate_content_cached_content/",
			"TestModelsGenerateContentStream/mldev/models/generate_content_part/",
			"TestModelsGenerateContentStream/vertex/models/generate_content/test_2_candidates_gemini_1_5_flash",
			"TestModelsGenerateContentStream/vertex/models/generate_content/test_llama",
			"TestModelsGenerateContentStream/vertex/models/generate_content/test_simple_shared_generation_config",
			"TestModelsGenerateContentStream/vertex/models/generate_content_cached_content",
			"TestModelsGenerateContentStream/vertex/models/generate_content_part/test_video_gcs_file_uri",
			"TestModelsGenerateContentStream/vertex/models/generate_content_tools/test_code_execution",
			"TestModelsGenerateContentStream/mldev/models/generate_content_model",
			"TestModelsGenerateContentAudio/",
		},
		replayMode: []string{
			// TODO(b/372730941): httpOptions related tests are not covered in replay mode.
			"TestTable/vertex/models/delete/test_delete_model_with_http_options_in_method",
			"TestTable/mldev/models/delete/test_delete_model_with_http_options_in_method",
			"TestTable/vertex/models/generate_content/test_http_options_in_method",
			"TestTable/mldev/models/generate_content/test_http_options_in_method",
			"TestTable/vertex/models/get/test_get_vertex_tuned_model_with_http_options_in_method",
			"TestTable/mldev/models/get/test_get_vertex_tuned_model_with_http_options_in_method",
			"TestTable/vertex/models/get/test_get_mldev_base_model_with_http_options_in_method",
			"TestTable/mldev/models/get/test_get_mldev_base_model_with_http_options_in_method",
			"TestTable/vertex/models/list/test_list_models_with_http_options_in_method",
			"TestTable/mldev/models/list/test_list_models_with_http_options_in_method",
			"TestTable/vertex/models/update/test_mldev_tuned_models_update_with_http_options_in_method",
			"TestTable/mldev/models/update/test_mldev_tuned_models_update_with_http_options_in_method",
			"TestTable/vertex/models/update/test_vertex_tuned_models_update_with_http_options_in_method",
			"TestTable/mldev/models/update/test_vertex_tuned_models_update_with_http_options_in_method",
			"TestTable/mldev/caches/create_custom_url/test_caches_create_with_googleai_file",
			"TestTable/mldev/caches/delete_custom_url/test_caches_delete_with_mldev_cache_name",
			"TestTable/mldev/caches/get_custom_url/test_caches_get_with_mldev_cache_name",
			"TestTable/mldev/caches/update_custom_url/test_caches_update_with_mldev_cache_name",
			"TestTable/mldev/models/count_tokens/test_count_tokens_mldev_custom_url",
			"TestTable/vertex/caches/create_custom_url/test_caches_create_with_gcs_uri",
			"TestTable/vertex/caches/delete_custom_url/test_caches_delete_with_vertex_cache_name",
			"TestTable/vertex/caches/get_custom_url/test_caches_get_with_vertex_cache_name",
			"TestTable/vertex/caches/update_custom_url/test_caches_update_with_vertex_cache_name",
			"TestTable/vertex/models/compute_tokens/test_compute_tokens_vertex_custom_url",
			"TestTable/vertex/models/count_tokens/test_count_tokens_vertex_custom_url",
		},
	}
	mode     = flag.String("mode", replayMode, "Test mode")
	backends = []struct {
		name    string
		Backend Backend
	}{
		{
			name:    "mldev",
			Backend: BackendGeminiAPI,
		},
		{
			name:    "vertex",
			Backend: BackendVertexAI,
		},
	}
)

func isDisabledTest(t *testing.T) bool {
	disabledTestPatterns := append(disabledTestsForAll, disabledTestsByMode[*mode]...)
	for _, p := range disabledTestPatterns {
		r := regexp.MustCompile(p)
		if r.MatchString(t.Name()) {
			return true
		}
	}
	return false
}

func TestMain(m *testing.M) {
	flag.Parse()
	fmt.Println("Running tests in", *mode)
	exitCode := m.Run()
	os.Exit(exitCode)
}
