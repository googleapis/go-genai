package genai

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"testing"
)

const (
	apiMode     = "api"
	replayMode  = "replay"
	requestMode = "request"
)

// TODO(b/382773687): Enable the TestModelsGenerateContentStream tests.
var (
	disabledTestsForAll = []string{
		// TODO(b/380108306): bytes related tests are not supported in replay tests.
		"models/generate_content_part/test_image_base64",
	}
	disabledTestsByMode = map[string][]string{
		apiMode: []string{
			"TestTable/",
			"TestModelsGenerateContentStream/mldev/v1beta/models/generate_content/test_simple_shared_generation_config",
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
			"TestModelsGenerateContentStream/",
			// TODO(b/383351834): Enable the test after the bug is fixed.
			"TestTable/mldev/models/generate_content_config_zero_value",
		},
		requestMode: []string{
			"TestTable/",
			"TestModelsGenerateContentStream/",
		},
	}
	mode     = flag.String("mode", replayMode, "Test mode")
	backends = []struct {
		name       string
		apiVersion string
		backend    Backend
	}{
		{
			name:       "mldev",
			apiVersion: "v1beta",
			backend:    BackendGoogleAI,
		},
		{
			name:       "vertex",
			apiVersion: "v1beta1",
			backend:    BackendVertexAI,
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
