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
			"TestModelsGenerateContentAudio/",
		},
		replayMode: []string{
			// TODO(b/372730941): httpOptions related tests are not covered in replay mode.
			"models/delete/test_delete_model_with_http_options_in_method",
			"models/generate_content/test_http_options_in_method",
			"models/get/test_get_vertex_tuned_model_with_http_options_in_method",
			"models/get/test_get_mldev_base_model_with_http_options_in_method",
			"models/list/test_list_models_with_http_options_in_method",
			"models/update/test_mldev_tuned_models_update_with_http_options_in_method",
			"models/update/test_vertex_tuned_models_update_with_http_options_in_method",
			"caches/create_custom_url/test_caches_create_with_googleai_file",
			"caches/delete_custom_url/test_caches_delete_with_mldev_cache_name",
			"caches/get_custom_url/test_caches_get_with_mldev_cache_name",
			"caches/update_custom_url/test_caches_update_with_mldev_cache_name",
			"models/count_tokens/test_count_tokens_mldev_custom_url",
			"caches/create_custom_url/test_caches_create_with_gcs_uri",
			"caches/delete_custom_url/test_caches_delete_with_vertex_cache_name",
			"caches/get_custom_url/test_caches_get_with_vertex_cache_name",
			"caches/update_custom_url/test_caches_update_with_vertex_cache_name",
			"models/compute_tokens/test_compute_tokens_vertex_custom_url",
			"models/count_tokens/test_count_tokens_vertex_custom_url",
			"models/compute_tokens/test_compute_tokens_mldev_custom_url",

			// TODO(b/424824119): generateVideos tests disabled due to backwards compatibility
			"/models/generate_videos",
			// filter=display_name%3A%22genai_%2A%22&pageSize=5 are reordered and mismatched
			"batches/list/test_list_batch_jobs_with_config",
		},
		unitMode: []string{
			// We don't run table tests in unit mode.
			"TestTable/",
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
