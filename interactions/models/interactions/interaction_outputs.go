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

import "strings"

// PopulateOutputProperties derives the SDK-only convenience fields
// (OutputText, OutputImage, OutputAudio and OutputVideo) from Steps, as done
// after deserialization. It is for SDK code that assembles an Interaction
// itself (e.g. by merging automatic continuation hops); fields are only set
// when a matching output is found, so clear them first to recompute.
func (i *Interaction) PopulateOutputProperties() {
	i.populateOutputProperties()
}

// populateOutputProperties derives the SDK-only convenience fields
// (OutputText, OutputImage, OutputAudio and OutputVideo) from Steps.
//
// It is called after every Interaction is deserialized, so every operation
// that returns an Interaction (create, get, cancel, ...) gets these fields
// populated. Fields are only set when a matching output is found; otherwise
// they are left unchanged.
func (i *Interaction) populateOutputProperties() {
	if i == nil {
		return
	}
	if text := outputTextFromSteps(i.Steps); text != "" {
		i.OutputText = &text
	}
	image, audio, video := outputMediaFromSteps(i.Steps)
	if image != nil {
		i.OutputImage = image
	}
	if audio != nil {
		i.OutputAudio = audio
	}
	if video != nil {
		i.OutputVideo = video
	}
}

// outputTextFromSteps returns the concatenated trailing model-output text of
// the current turn.
//
// Steps are walked backwards until a user_input step. Thought steps and
// model_output steps without content are skipped. Any other step (e.g. tool
// calls or results) ends collection once text has been found. Within a
// model_output step, a non-text content item ends collection once text has
// been found.
func outputTextFromSteps(steps []Step) string {
	var parts []string
	collecting := false
outer:
	for idx := len(steps) - 1; idx >= 0; idx-- {
		step := steps[idx]
		switch step.Type {
		case StepTypeUserInput:
			break outer
		case StepTypeThought:
			continue
		case StepTypeModelOutput:
		default:
			if collecting {
				break outer
			}
			continue
		}
		if step.ModelOutputStep == nil {
			continue
		}
		content := step.ModelOutputStep.Content
		for j := len(content) - 1; j >= 0; j-- {
			item := content[j]
			if item.Type == ContentTypeText {
				collecting = true
				text := ""
				if item.TextContent != nil {
					text = item.TextContent.Text
				}
				parts = append(parts, text)
			} else if collecting {
				break outer
			}
		}
	}

	var sb strings.Builder
	for j := len(parts) - 1; j >= 0; j-- {
		sb.WriteString(parts[j])
	}
	return sb.String()
}

// outputMediaFromSteps returns the last image, audio and video content
// produced by model_output steps in the current turn (i.e. after the last
// user_input step).
func outputMediaFromSteps(steps []Step) (*ImageContent, *AudioContent, *VideoContent) {
	var image *ImageContent
	var audio *AudioContent
	var video *VideoContent
	for idx := len(steps) - 1; idx >= 0; idx-- {
		step := steps[idx]
		if step.Type == StepTypeUserInput {
			break
		}
		if step.Type != StepTypeModelOutput || step.ModelOutputStep == nil {
			continue
		}
		content := step.ModelOutputStep.Content
		for j := len(content) - 1; j >= 0; j-- {
			item := content[j]
			switch item.Type {
			case ContentTypeImage:
				if image == nil {
					image = item.ImageContent
				}
			case ContentTypeAudio:
				if audio == nil {
					audio = item.AudioContent
				}
			case ContentTypeVideo:
				if video == nil {
					video = item.VideoContent
				}
			}
		}
	}
	return image, audio, video
}
