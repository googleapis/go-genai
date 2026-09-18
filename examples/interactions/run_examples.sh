#!/bin/bash

# Resolve the SDK root directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
SDK_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"


cd "$SDK_ROOT"

echo "SDK Root: $SDK_ROOT"

# Find all .go files in examples, excluding tests and backups
examples=$(find examples -name "*.go" -not -name "*_test.go" -not -name "*.orig")

# Set environment variable to skip live calls if using Vertex AI path
# This helps them pass without real keys if the example supports it.
export GOOGLE_GENAI_USE_VERTEXAI=true

echo "Running all examples using 'go run'..."

declare -A results

for e in $examples; do
    echo "----------------------------------------"
    echo "Running $e..."
    echo "----------------------------------------"
    # Run from the SDK root
    if go run "$e"; then
        results["$e"]="PASS"
    else
        results["$e"]="FAIL"
    fi
done

echo ""
echo "========================================"
echo "Report Card"
echo "========================================"
for e in $examples; do
    if [ "${results["$e"]}" == "PASS" ]; then
        # Green check
        echo -e "$e: \033[32m✔\033[0m"
    else
        # Red x
        echo -e "$e: \033[31m✘\033[0m"
    fi
done
echo "========================================"
