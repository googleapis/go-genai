// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"net/http"
	"os"
	"strconv"
	"testing"
)

func CheckTestServer(t *testing.T, url string) bool {
	if _, err := http.Get(url); err != nil {
		const SKIP_MOCK_TESTS = "SKIP_MOCK_TESTS"
		if str, ok := os.LookupEnv(SKIP_MOCK_TESTS); ok {
			skip, err := strconv.ParseBool(str)
			if err != nil {
				t.Fatalf("strconv.ParseBool(os.LookupEnv(%s)) failed: %s", SKIP_MOCK_TESTS, err)
			}
			if skip {
				t.Skip("The test will not run without a mock server running against your OpenAPI spec")
				return false
			}
			t.Errorf("The test will not run without a mock server running against your OpenAPI spec. You can set the environment variable %s to true to skip running any tests that require the mock server", SKIP_MOCK_TESTS)
			return false
		}
	}
	return true
}
