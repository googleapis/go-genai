// Copyright 2025 Google LLC
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

// Chats client.

package genai

import (
	"context"
)

// Chats can be used to facilitate multi-turn conversations with the model.
//
//	client, _ := genai.NewClient(ctx, &genai.ClientConfig{})
//	chat, _ := client.Chats.Create(ctx, "gemini-2.0-flash", nil, nil)
type Chats struct {
	apiClient *apiClient
}

// Chat represents a single chat session (multi-turn conversation) with the model.
//
//		client, _ := genai.NewClient(ctx, &genai.ClientConfig{})
//		chat, _ := client.Chats.Create(ctx, "gemini-2.0-flash", nil, nil)
//	  result, err = chat.SendMessage(ctx, genai.Part{Text: "What is 1 + 2?"})
type Chat struct {
	Models
	apiClient            *apiClient
	model                string
	config               *GenerateContentConfig
	comprehensiveHistory []*Content
}

// Create a new chat session.
func (c *Chats) Create(ctx context.Context, model string, config *GenerateContentConfig, history []*Content) (*Chat, error) {
	chat := &Chat{
		apiClient:            c.apiClient,
		model:                model,
		config:               config,
		comprehensiveHistory: history,
	}
	chat.Models.apiClient = c.apiClient
	return chat, nil
}

func (c *Chat) recordHistory(ctx context.Context, inputContent *Content, modelOutput *Content) {
	c.comprehensiveHistory = append(c.comprehensiveHistory, inputContent)
	c.comprehensiveHistory = append(c.comprehensiveHistory, modelOutput)
}

// copySanitizedModelContent creates a (shallow) copy of modelContent with role set to
// model and empty text parts removed.
func copySanitizedModelContent(modelContent *Content) *Content {
	newContent := &Content{Role: "model"}
	for _, part := range modelContent.Parts {
		text := (*part).Text
		if len(string(text)) > 0 {
			newContent.Parts = append(newContent.Parts, part)
		}
	}
	return newContent
}

// Sends the conversation history with the additional user's message and returns the model's response.
func (c *Chat) SendMessage(ctx context.Context, parts ...Part) (*GenerateContentResponse, error) {
	// Transform Parts to single Content
	p := make([]*Part, len(parts))
	for i, part := range parts {
		p[i] = &part
	}
	inputContent := &Content{Parts: p, Role: "user"}

	// Combine history with input content to send to model
	contents := append(c.comprehensiveHistory, inputContent)

	// Generate Content
	modelOutput, err := c.GenerateContent(ctx, c.model, contents, c.config)
	if err != nil {
		return nil, err
	}

	// Record history
	modelOutputContent := &Content{}
	if modelOutput.Candidates != nil && len(modelOutput.Candidates) > 0 && modelOutput.Candidates[0].Content != nil {
		modelOutputContent = modelOutput.Candidates[0].Content
	}
	modelOutputContent = copySanitizedModelContent(modelOutputContent)
	c.recordHistory(ctx, inputContent, modelOutputContent)

	return modelOutput, err
}
