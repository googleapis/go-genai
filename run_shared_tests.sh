#!/bin/bash

export GOOGLE_GENAI_REPLAYS_DIRECTORY="`blaze info workspace 2>/dev/null`/google/cloud/aiplatform/sdk/genai/replays"
export GOOGLE_GENAI_TESTS_SUBDIR=shared

echo "Replays directory: $GOOGLE_GENAI_REPLAYS_DIRECTORY"
echo "Tests subdirectory: $GOOGLE_GENAI_TESTS_SUBDIR"
echo "Running shared table tests in API mode..."
go test -v -run "^TestTable$" -mode api
