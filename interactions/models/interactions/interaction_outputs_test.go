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

package interactions

import (
	"encoding/json"
	"testing"
)

const (
	userInput     = `{"type":"user_input","content":[{"type":"text","text":"question"}]}`
	thought       = `{"type":"thought","signature":"sig"}`
	functionCall  = `{"type":"function_call","id":"c1","name":"f","arguments":{}}`
	functionReply = `{"type":"function_result","call_id":"c1","name":"f","result":{}}`
	emptyOutput   = `{"type":"model_output"}`
	imageA        = `{"type":"image","data":"QQ==","mime_type":"image/png"}`
	imageB        = `{"type":"image","data":"Qg==","mime_type":"image/png"}`
	audioA        = `{"type":"audio","data":"QQ==","mime_type":"audio/wav"}`
	videoA        = `{"type":"video","uri":"gs://bucket/a.mp4","mime_type":"video/mp4"}`
)

func text(s string) string {
	b, _ := json.Marshal(s)
	return `{"type":"text","text":` + string(b) + `}`
}

func modelOutput(content ...string) string {
	out := `{"type":"model_output","content":[`
	for i, c := range content {
		if i > 0 {
			out += ","
		}
		out += c
	}
	return out + `]}`
}

func unmarshalInteraction(t *testing.T, steps ...string) Interaction {
	t.Helper()
	body := `{"id":"interaction-1","status":"completed","steps":[`
	for i, s := range steps {
		if i > 0 {
			body += ","
		}
		body += s
	}
	body += `]}`
	var got Interaction
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("json.Unmarshal(%s) failed: %v", body, err)
	}
	return got
}

func TestInteractionOutputText(t *testing.T) {
	tests := []struct {
		name  string
		steps []string
		want  *string
	}{
		{
			name:  "single model output",
			steps: []string{userInput, modelOutput(text("Hello"))},
			want:  ptr("Hello"),
		},
		{
			name:  "multi-part text is concatenated",
			steps: []string{userInput, modelOutput(text("Hello, "), text("world"))},
			want:  ptr("Hello, world"),
		},
		{
			name: "thought steps between model outputs are skipped",
			steps: []string{
				userInput,
				thought, modelOutput(text("first ")),
				thought, modelOutput(text("second")),
			},
			want: ptr("first second"),
		},
		{
			name: "tool call boundary stops collection",
			steps: []string{
				userInput,
				modelOutput(text("before tool")),
				functionCall, functionReply,
				modelOutput(text("after tool")),
			},
			want: ptr("after tool"),
		},
		{
			name: "trailing tool call is skipped before collection starts",
			steps: []string{
				userInput,
				modelOutput(text("answer")),
				functionCall,
			},
			want: ptr("answer"),
		},
		{
			name: "user input boundary stops collection",
			steps: []string{
				userInput, modelOutput(text("previous turn")),
				userInput, modelOutput(text("current turn")),
			},
			want: ptr("current turn"),
		},
		{
			name:  "user input as last step yields no text",
			steps: []string{modelOutput(text("previous turn")), userInput},
			want:  nil,
		},
		{
			name: "empty model output is skipped",
			steps: []string{
				userInput,
				modelOutput(text("answer")),
				emptyOutput,
			},
			want: ptr("answer"),
		},
		{
			name: "non-text content stops text collection",
			steps: []string{
				userInput,
				modelOutput(text("before image "), imageA, text("after image")),
			},
			want: ptr("after image"),
		},
		{
			name: "non-text content in an earlier step stops text collection",
			steps: []string{
				userInput,
				modelOutput(text("before image"), imageA),
				modelOutput(text("after image")),
			},
			want: ptr("after image"),
		},
		{
			name:  "trailing non-text content is skipped before collection starts",
			steps: []string{userInput, modelOutput(text("caption"), imageA)},
			want:  ptr("caption"),
		},
		{
			name:  "no text leaves field unset",
			steps: []string{userInput, modelOutput(imageA)},
			want:  nil,
		},
		{
			name:  "no steps leaves field unset",
			steps: nil,
			want:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := unmarshalInteraction(t, tc.steps...)
			switch {
			case tc.want == nil && got.OutputText != nil:
				t.Errorf("OutputText = %q, want nil", *got.OutputText)
			case tc.want != nil && got.OutputText == nil:
				t.Errorf("OutputText = nil, want %q", *tc.want)
			case tc.want != nil && *got.OutputText != *tc.want:
				t.Errorf("OutputText = %q, want %q", *got.OutputText, *tc.want)
			}
		})
	}
}

func TestInteractionOutputMedia(t *testing.T) {
	got := unmarshalInteraction(t,
		userInput,
		modelOutput(imageA, audioA),
		thought,
		modelOutput(videoA, imageB, text("done")),
	)
	if got.OutputImage == nil || got.OutputImage.Data == nil || *got.OutputImage.Data != "Qg==" {
		t.Errorf("OutputImage = %+v, want the last image (data Qg==)", got.OutputImage)
	}
	if got.OutputAudio == nil || got.OutputAudio.Data == nil || *got.OutputAudio.Data != "QQ==" {
		t.Errorf("OutputAudio = %+v, want audio with data QQ==", got.OutputAudio)
	}
	if got.OutputVideo == nil || got.OutputVideo.URI == nil || *got.OutputVideo.URI != "gs://bucket/a.mp4" {
		t.Errorf("OutputVideo = %+v, want video with uri gs://bucket/a.mp4", got.OutputVideo)
	}
	if got.OutputText == nil || *got.OutputText != "done" {
		t.Errorf("OutputText = %v, want %q", got.OutputText, "done")
	}
}

func TestInteractionOutputMediaStopsAtUserInput(t *testing.T) {
	got := unmarshalInteraction(t,
		userInput,
		modelOutput(imageA, audioA, videoA),
		userInput,
		modelOutput(text("text only")),
	)
	if got.OutputImage != nil {
		t.Errorf("OutputImage = %+v, want nil", got.OutputImage)
	}
	if got.OutputAudio != nil {
		t.Errorf("OutputAudio = %+v, want nil", got.OutputAudio)
	}
	if got.OutputVideo != nil {
		t.Errorf("OutputVideo = %+v, want nil", got.OutputVideo)
	}
}

func ptr(s string) *string { return &s }
