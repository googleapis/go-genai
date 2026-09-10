// Copyright 2026 Google LLC
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

package hooks

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

var legacyLyriaModels = map[string]bool{
	"lyria-3-pro-preview":  true,
	"lyria-3-clip-preview": true,
}

var legacyEventRenames = map[string]string{
	"interaction.start":    "interaction.created",
	"content.start":        "step.start",
	"content.delta":        "step.delta",
	"content.stop":         "step.stop",
	"interaction.complete": "interaction.completed",
}

type GoogleGenAILyriaHook struct{}

var _ afterSuccessHook = (*GoogleGenAILyriaHook)(nil)

func (h *GoogleGenAILyriaHook) AfterSuccess(hookCtx AfterSuccessContext, res *http.Response) (*http.Response, error) {
	if res == nil || res.Body == nil {
		return res, nil
	}

	contentType := res.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			return res, err
		}
		res.Body.Close()

		var data map[string]any
		if err := json.Unmarshal(rawBody, &data); err != nil {
			res.Body = io.NopCloser(bytes.NewReader(rawBody))
			return res, nil
		}

		model, _ := data["model"].(string)
		if legacyLyriaModels[model] {
			outputs, hasOutputs := data["outputs"]
			_, hasSteps := data["steps"]

			if hasOutputs && !hasSteps {
				steps := []any{
					map[string]any{
						"type":    "model_output",
						"content": outputs,
					},
				}
				data["steps"] = steps
				delete(data, "outputs")

				newBody, err := json.Marshal(data)
				if err != nil {
					res.Body = io.NopCloser(bytes.NewReader(rawBody))
					return res, nil
				}
				res.Body = io.NopCloser(bytes.NewReader(newBody))
				res.ContentLength = int64(len(newBody))
				return res, nil
			}
		}

		res.Body = io.NopCloser(bytes.NewReader(rawBody))
	} else if strings.Contains(contentType, "text/event-stream") {
		res.Body = newLegacyLyriaStreamReader(res.Body)
	}

	return res, nil
}

type legacyLyriaStreamReader struct {
	rc      io.ReadCloser
	r       *bufio.Reader
	remaped bytes.Buffer
}

func newLegacyLyriaStreamReader(rc io.ReadCloser) *legacyLyriaStreamReader {
	return &legacyLyriaStreamReader{
		rc: rc,
		r:  bufio.NewReader(rc),
	}
}

func (sr *legacyLyriaStreamReader) Read(p []byte) (int, error) {
	if sr.remaped.Len() > 0 {
		return sr.remaped.Read(p)
	}

	line, err := sr.r.ReadString('\n')
	if err != nil {
		if len(line) == 0 {
			return 0, err
		}
	}

	if strings.HasPrefix(line, "data: ") {
		dataStr := strings.TrimPrefix(line, "data: ")
		dataStr = strings.TrimSpace(dataStr)

		if dataStr != "[DONE]" && dataStr != "" {
			var event map[string]any
			if errJson := json.Unmarshal([]byte(dataStr), &event); errJson == nil {
				eventType, _ := event["event_type"].(string)
				remappedType := legacyEventRenames[eventType]
				if remappedType != "" {
					event["event_type"] = remappedType

					if eventType == "content.start" {
						content := event["content"]
						delete(event, "content")

						var contentList []any
						if content != nil {
							contentList = []any{content}
						} else {
							contentList = []any{}
						}

						event["step"] = map[string]any{
							"type":    "model_output",
							"content": contentList,
						}
					}

					newJson, errMarshal := json.Marshal(event)
					if errMarshal == nil {
						line = "data: " + string(newJson) + "\n"
					}
				}
			}
		}
	}

	sr.remaped.WriteString(line)
	return sr.remaped.Read(p)
}

func (sr *legacyLyriaStreamReader) Close() error {
	return sr.rc.Close()
}
