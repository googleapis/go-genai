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

// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package interactions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"google.golang.org/genai/interactions/internal/apijson"
	"google.golang.org/genai/interactions/internal/apijson/unmarshalinfo"
	"google.golang.org/genai/interactions/internal/apiquery"
	"google.golang.org/genai/interactions/internal/requestconfig"
	"google.golang.org/genai/interactions/option"
	"google.golang.org/genai/interactions/packages/apidata"
	"google.golang.org/genai/interactions/packages/ssestream"
	"google.golang.org/genai/interactions/shared/constant"
)

// InteractionService contains methods and other services that help with
// interacting with the gemini-next-gen-api API.
//
// Experimental: This service is experimental and may change in future versions.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewInteractionService] method instead.
type InteractionService struct {
	Options []option.RequestOption
}

// NewInteractionService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewInteractionService(opts ...option.RequestOption) (r InteractionService) {
	r = InteractionService{}
	r.Options = opts
	return
}

// Deletes the interaction by id.
func (r *InteractionService) Delete(ctx context.Context, id string, body DeleteParams, opts ...option.RequestOption) (res *DeleteResponse, err error) {
	opts = slices.Concat(r.Options, opts)
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return nil, err
	}
	if body.APIVersion == "" && precfg.APIVersion != nil {
		body.APIVersion = *precfg.APIVersion
	}
	if body.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return nil, err
	}
	if id == "" {
		err = errors.New("missing required id parameter")
		return nil, err
	}
	path := fmt.Sprintf("%s/interactions/%s", url.PathEscape(body.APIVersion), url.PathEscape(id))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodDelete, path, nil, &res, opts...)
	return res, err
}

// Cancels an interaction by id. This only applies to background interactions that
// are still running.
func (r *InteractionService) Cancel(ctx context.Context, id string, body CancelParams, opts ...option.RequestOption) (res *Interaction, err error) {
	opts = slices.Concat(r.Options, opts)
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return nil, err
	}
	if body.APIVersion == "" && precfg.APIVersion != nil {
		body.APIVersion = *precfg.APIVersion
	}
	if body.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return nil, err
	}
	if id == "" {
		err = errors.New("missing required id parameter")
		return nil, err
	}
	path := fmt.Sprintf("%s/interactions/%s/cancel", url.PathEscape(body.APIVersion), url.PathEscape(id))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, nil, &res, opts...)
	return res, err
}

// Creates a new interaction.
func (r *InteractionService) NewAgent(ctx context.Context, params NewAgentParams, opts ...option.RequestOption) (res *Interaction, err error) {
	opts = slices.Concat(r.Options, opts)
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return nil, err
	}
	if params.APIVersion == "" && precfg.APIVersion != nil {
		params.APIVersion = *precfg.APIVersion
	}
	if params.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return nil, err
	}
	path := fmt.Sprintf("%s/interactions?agent", url.PathEscape(params.APIVersion))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return res, err
}

// Creates a new interaction.
func (r *InteractionService) NewAgentStreaming(ctx context.Context, params NewAgentParams, opts ...option.RequestOption) (stream *ssestream.Stream[InteractionSSEEvent]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append(opts, option.WithJSONSet("stream", true))
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	if params.APIVersion == "" && precfg.APIVersion != nil {
		params.APIVersion = *precfg.APIVersion
	}
	if params.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	path := fmt.Sprintf("%s/interactions?agent", url.PathEscape(params.APIVersion))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &raw, opts...)
	return ssestream.NewStream[InteractionSSEEvent](ssestream.NewDecoder(raw), err)
}

// Creates a new interaction.
func (r *InteractionService) NewModel(ctx context.Context, params NewModelParams, opts ...option.RequestOption) (res *Interaction, err error) {
	opts = slices.Concat(r.Options, opts)
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return nil, err
	}
	if params.APIVersion == "" && precfg.APIVersion != nil {
		params.APIVersion = *precfg.APIVersion
	}
	if params.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return nil, err
	}
	path := fmt.Sprintf("%s/interactions?model", url.PathEscape(params.APIVersion))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &res, opts...)
	return res, err
}

// Creates a new interaction.
func (r *InteractionService) NewModelStreaming(ctx context.Context, params NewModelParams, opts ...option.RequestOption) (stream *ssestream.Stream[InteractionSSEEvent]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append(opts, option.WithJSONSet("stream", true))
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	if params.APIVersion == "" && precfg.APIVersion != nil {
		params.APIVersion = *precfg.APIVersion
	}
	if params.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	path := fmt.Sprintf("%s/interactions?model", url.PathEscape(params.APIVersion))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, params, &raw, opts...)
	return ssestream.NewStream[InteractionSSEEvent](ssestream.NewDecoder(raw), err)
}

// Retrieves the full details of a single interaction based on its
// `Interaction.id`.
func (r *InteractionService) Get(ctx context.Context, id string, params GetParams, opts ...option.RequestOption) (res *Interaction, err error) {
	opts = slices.Concat(r.Options, opts)
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return nil, err
	}
	if params.APIVersion == "" && precfg.APIVersion != nil {
		params.APIVersion = *precfg.APIVersion
	}
	if params.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return nil, err
	}
	if id == "" {
		err = errors.New("missing required id parameter")
		return nil, err
	}
	path := fmt.Sprintf("%s/interactions/%s", url.PathEscape(params.APIVersion), url.PathEscape(id))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, params, &res, opts...)
	return res, err
}

// Retrieves the full details of a single interaction based on its
// `Interaction.id`.
func (r *InteractionService) GetStreaming(ctx context.Context, id string, params GetParams, opts ...option.RequestOption) (stream *ssestream.Stream[InteractionSSEEvent]) {
	var (
		raw *http.Response
		err error
	)
	opts = slices.Concat(r.Options, opts)
	opts = append(opts, option.WithJSONSet("stream", true))
	precfg, err := requestconfig.PreRequestOptions(opts...)
	if err != nil {
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	if params.APIVersion == "" && precfg.APIVersion != nil {
		params.APIVersion = *precfg.APIVersion
	}
	if params.APIVersion == "" {
		err = errors.New("missing required api_version parameter")
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	if id == "" {
		err = errors.New("missing required id parameter")
		return ssestream.NewStream[InteractionSSEEvent](nil, err)
	}
	path := fmt.Sprintf("%s/interactions/%s", url.PathEscape(params.APIVersion), url.PathEscape(id))
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, params, &raw, opts...)
	return ssestream.NewStream[InteractionSSEEvent](ssestream.NewDecoder(raw), err)
}

// The configuration for allowed tools.
type AllowedTools struct {
	// The mode of the tool choice.
	//
	// Any of "auto", "any", "none", "validated".
	Mode ToolChoiceType `json:"mode,omitzero"`
	// The names of the allowed tools.
	Tools []string `json:"tools,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type Annotation struct {
	URLCitation   *URLCitation   `json:",omitzero,inline" discriminator:"url_citation"`
	FileCitation  *FileCitation  `json:",omitzero,inline" discriminator:"file_citation"`
	PlaceCitation *PlaceCitation `json:",omitzero,inline" discriminator:"place_citation"`

	metadata `api:"union"`
}

func (u Annotation) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *Annotation) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

// An audio content block.
type AudioContent struct {
	// The number of audio channels.
	Channels *int `json:"channels,omitzero"`
	// The audio content.
	Data string `json:"data,omitzero" format:"byte"`
	// The mime type of the audio.
	//
	// Any of "audio/wav", "audio/mp3", "audio/aiff", "audio/aac", "audio/ogg",
	// "audio/flac", "audio/mpeg", "audio/m4a", "audio/l16", "audio/opus",
	// "audio/alaw", "audio/mulaw".
	MimeType string `json:"mime_type,omitzero"`
	// The sample rate of the audio.
	Rate *int `json:"rate,omitzero"`
	// The URI of the audio.
	Uri string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Audio `json:"type" default:"audio"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type AudioDelta struct {
	// The number of audio channels.
	Channels *int   `json:"channels,omitzero"`
	Data     string `json:"data,omitzero" format:"byte"`
	// Any of "audio/wav", "audio/mp3", "audio/aiff", "audio/aac", "audio/ogg",
	// "audio/flac", "audio/mpeg", "audio/m4a", "audio/l16", "audio/opus",
	// "audio/alaw", "audio/mulaw".
	MimeType string `json:"mime_type,omitzero"`
	// The sample rate of the audio.
	Rate *int   `json:"rate,omitzero"`
	Uri  string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Audio `json:"type" default:"audio"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The arguments to pass to the code execution.
type CodeExecutionCallArguments struct {
	// The code to be executed.
	Code string `json:"code,omitzero"`
	// Programming language of the `code`.
	//
	// Any of "python".
	Language string `json:"language,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Code execution content.
type CodeExecutionCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// Required. The arguments to pass to the code execution.
	Arguments CodeExecutionCallArguments `json:"arguments" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.CodeExecutionCall `json:"type" default:"code_execution_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type CodeExecutionCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// The arguments to pass to the code execution.
	Arguments CodeExecutionCallArguments `json:"arguments" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.CodeExecutionCall `json:"type" default:"code_execution_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Code execution result content.
type CodeExecutionResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// Required. The output of the code execution.
	Result string `json:"result" api:"required"`
	// Whether the code execution resulted in an error.
	IsError *bool `json:"is_error,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.CodeExecutionResult `json:"type" default:"code_execution_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type CodeExecutionResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID  string `json:"call_id" api:"required"`
	Result  string `json:"result" api:"required"`
	IsError *bool  `json:"is_error,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.CodeExecutionResult `json:"type" default:"code_execution_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type Content struct {
	Text                *TextContent                `json:",omitzero,inline" discriminator:"text"`
	Image               *ImageContent               `json:",omitzero,inline" discriminator:"image"`
	Audio               *AudioContent               `json:",omitzero,inline" discriminator:"audio"`
	Document            *DocumentContent            `json:",omitzero,inline" discriminator:"document"`
	Video               *VideoContent               `json:",omitzero,inline" discriminator:"video"`
	Thought             *ThoughtContent             `json:",omitzero,inline" discriminator:"thought"`
	FunctionCall        *FunctionCallContent        `json:",omitzero,inline" discriminator:"function_call"`
	CodeExecutionCall   *CodeExecutionCallContent   `json:",omitzero,inline" discriminator:"code_execution_call"`
	URLContextCall      *URLContextCallContent      `json:",omitzero,inline" discriminator:"url_context_call"`
	MCPServerToolCall   *MCPServerToolCallContent   `json:",omitzero,inline" discriminator:"mcp_server_tool_call"`
	GoogleSearchCall    *GoogleSearchCallContent    `json:",omitzero,inline" discriminator:"google_search_call"`
	FileSearchCall      *FileSearchCallContent      `json:",omitzero,inline" discriminator:"file_search_call"`
	GoogleMapsCall      *GoogleMapsCallContent      `json:",omitzero,inline" discriminator:"google_maps_call"`
	FunctionResult      *FunctionResultContent      `json:",omitzero,inline" discriminator:"function_result"`
	CodeExecutionResult *CodeExecutionResultContent `json:",omitzero,inline" discriminator:"code_execution_result"`
	URLContextResult    *URLContextResultContent    `json:",omitzero,inline" discriminator:"url_context_result"`
	GoogleSearchResult  *GoogleSearchResultContent  `json:",omitzero,inline" discriminator:"google_search_result"`
	MCPServerToolResult *MCPServerToolResultContent `json:",omitzero,inline" discriminator:"mcp_server_tool_result"`
	FileSearchResult    *FileSearchResultContent    `json:",omitzero,inline" discriminator:"file_search_result"`
	GoogleMapsResult    *GoogleMapsResultContent    `json:",omitzero,inline" discriminator:"google_maps_result"`

	metadata `api:"union"`
}

func (u Content) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *Content) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type ContentDelta struct {
	// The delta content data for a content block.
	Delta ContentDeltaDelta `json:"delta" api:"required"`
	Index int               `json:"index" api:"required"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.ContentDelta `json:"event_type" default:"content.delta"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type ContentDeltaDelta struct {
	Text                *TextDelta                `json:",omitzero,inline" discriminator:"text"`
	Image               *ImageDelta               `json:",omitzero,inline" discriminator:"image"`
	Audio               *AudioDelta               `json:",omitzero,inline" discriminator:"audio"`
	Document            *DocumentDelta            `json:",omitzero,inline" discriminator:"document"`
	Video               *VideoDelta               `json:",omitzero,inline" discriminator:"video"`
	ThoughtSummary      *ThoughtSummaryDelta      `json:",omitzero,inline" discriminator:"thought_summary"`
	ThoughtSignature    *ThoughtSignatureDelta    `json:",omitzero,inline" discriminator:"thought_signature"`
	FunctionCall        *FunctionCallDelta        `json:",omitzero,inline" discriminator:"function_call"`
	CodeExecutionCall   *CodeExecutionCallDelta   `json:",omitzero,inline" discriminator:"code_execution_call"`
	URLContextCall      *URLContextCallDelta      `json:",omitzero,inline" discriminator:"url_context_call"`
	GoogleSearchCall    *GoogleSearchCallDelta    `json:",omitzero,inline" discriminator:"google_search_call"`
	MCPServerToolCall   *MCPServerToolCallDelta   `json:",omitzero,inline" discriminator:"mcp_server_tool_call"`
	FileSearchCall      *FileSearchCallDelta      `json:",omitzero,inline" discriminator:"file_search_call"`
	GoogleMapsCall      *GoogleMapsCallDelta      `json:",omitzero,inline" discriminator:"google_maps_call"`
	FunctionResult      *FunctionResultDelta      `json:",omitzero,inline" discriminator:"function_result"`
	CodeExecutionResult *CodeExecutionResultDelta `json:",omitzero,inline" discriminator:"code_execution_result"`
	URLContextResult    *URLContextResultDelta    `json:",omitzero,inline" discriminator:"url_context_result"`
	GoogleSearchResult  *GoogleSearchResultDelta  `json:",omitzero,inline" discriminator:"google_search_result"`
	MCPServerToolResult *MCPServerToolResultDelta `json:",omitzero,inline" discriminator:"mcp_server_tool_result"`
	FileSearchResult    *FileSearchResultDelta    `json:",omitzero,inline" discriminator:"file_search_result"`
	GoogleMapsResult    *GoogleMapsResultDelta    `json:",omitzero,inline" discriminator:"google_maps_result"`
	TextAnnotation      *TextAnnotationDelta      `json:",omitzero,inline" discriminator:"text_annotation"`

	metadata `api:"union"`
}

func (u ContentDeltaDelta) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *ContentDeltaDelta) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type ContentStart struct {
	// The content of the response.
	Content Content `json:"content" api:"required"`
	Index   int     `json:"index" api:"required"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.ContentStart `json:"event_type" default:"content.start"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type ContentStop struct {
	Index int `json:"index" api:"required"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.ContentStop `json:"event_type" default:"content.stop"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Configuration for the Deep Research agent.
type DeepResearchAgentConfig struct {
	// Enables human-in-the-loop planning for the Deep Research agent. If set to true,
	// the Deep Research agent will provide a research plan in its response. The agent
	// will then proceed only if the user confirms the plan in the next turn. Relevant
	// issue: b/482352502.
	CollaborativePlanning *bool `json:"collaborative_planning,omitzero"`
	// Whether to include thought summaries in the response.
	//
	// Any of "auto", "none".
	ThinkingSummaries string `json:"thinking_summaries,omitzero"`
	// Whether to include visualizations in the response.
	//
	// Any of "off", "auto".
	Visualization string `json:"visualization,omitzero"`
	// This field doesn't need to be set.
	Type constant.DeepResearch `json:"type" default:"deep-research"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A document content block.
type DocumentContent struct {
	// The document content.
	Data string `json:"data,omitzero" format:"byte"`
	// The mime type of the document.
	//
	// Any of "application/pdf".
	MimeType string `json:"mime_type,omitzero"`
	// The URI of the document.
	Uri string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Document `json:"type" default:"document"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type DocumentDelta struct {
	Data string `json:"data,omitzero" format:"byte"`
	// Any of "application/pdf".
	MimeType string `json:"mime_type,omitzero"`
	Uri      string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Document `json:"type" default:"document"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Configuration for dynamic agents.
type DynamicAgentConfig struct {
	// This field doesn't need to be set.
	Type   constant.Dynamic `json:"type" default:"dynamic"`
	Fields map[string]any   `json:",inline"`

	metadata
}

type ErrorEvent struct {
	// Error message from an interaction.
	Error ErrorEventError `json:"error,omitzero"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.Error `json:"event_type" default:"error"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Error message from an interaction.
type ErrorEventError struct {
	// A URI that identifies the error type.
	Code string `json:"code,omitzero"`
	// A human-readable error message.
	Message string `json:"message,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A file citation annotation.
type FileCitation struct {
	// The URI of the file.
	DocumentUri string `json:"document_uri,omitzero"`
	// End of the attributed segment, exclusive.
	EndIndex *int `json:"end_index,omitzero"`
	// The name of the file.
	FileName string `json:"file_name,omitzero"`
	// Source attributed for a portion of the text.
	Source string `json:"source,omitzero"`
	// Start of segment of the response that is attributed to this source.
	//
	// Index indicates the start of the segment, measured in bytes.
	StartIndex *int `json:"start_index,omitzero"`
	// This field doesn't need to be set.
	Type constant.FileCitation `json:"type" default:"file_citation"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// File Search content.
type FileSearchCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FileSearchCall `json:"type" default:"file_search_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type FileSearchCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FileSearchCall `json:"type" default:"file_search_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The result of the File Search.
type FileSearchResult struct {
	// User provided metadata about the FileSearchResult.
	CustomMetadata []any `json:"custom_metadata,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// File Search result content.
type FileSearchResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// Required. The results of the File Search.
	Result []FileSearchResult `json:"result" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FileSearchResult `json:"type" default:"file_search_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type FileSearchResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID string             `json:"call_id" api:"required"`
	Result []FileSearchResult `json:"result" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FileSearchResult `json:"type" default:"file_search_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model.
type Function struct {
	// A description of the function.
	Description string `json:"description,omitzero"`
	// The name of the function.
	Name string `json:"name,omitzero"`
	// The JSON Schema for the function's parameters.
	Parameters any `json:"parameters,omitzero"`
	// This field doesn't need to be set.
	Type constant.Function `json:"type" default:"function"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A function tool call content block.
type FunctionCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// Required. The arguments to pass to the function.
	Arguments map[string]any `json:"arguments" api:"required"`
	// Required. The name of the tool to call.
	Name string `json:"name" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FunctionCall `json:"type" default:"function_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type FunctionCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID        string         `json:"id" api:"required"`
	Arguments map[string]any `json:"arguments" api:"required"`
	Name      string         `json:"name" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FunctionCall `json:"type" default:"function_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A function tool result content block.
type FunctionResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// The result of the tool call.
	Result FunctionResultContentResult `json:"result" api:"required"`
	// Whether the tool call resulted in an error.
	IsError *bool `json:"is_error,omitzero"`
	// The name of the tool that was called.
	Name string `json:"name,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FunctionResult `json:"type" default:"function_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type FunctionResultContentResult struct {
	FunctionResultSubcontentList []FunctionResultContentResultFunctionResultSubcontentListItem `json:",omitzero,inline"`
	String                       string                                                        `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u FunctionResultContentResult) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *FunctionResultContentResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// Only one field in this union will be nonzero
type FunctionResultContentResultFunctionResultSubcontentListItem struct {
	Text  *TextContent  `json:",omitzero,inline" discriminator:"text"`
	Image *ImageContent `json:",omitzero,inline" discriminator:"image"`

	metadata `api:"union"`
}

func (u FunctionResultContentResultFunctionResultSubcontentListItem) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *FunctionResultContentResultFunctionResultSubcontentListItem) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type FunctionResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID  string                    `json:"call_id" api:"required"`
	Result  FunctionResultDeltaResult `json:"result" api:"required"`
	IsError *bool                     `json:"is_error,omitzero"`
	Name    string                    `json:"name,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.FunctionResult `json:"type" default:"function_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type FunctionResultDeltaResult struct {
	FunctionResultSubcontentList []FunctionResultDeltaResultFunctionResultSubcontentListItem `json:",omitzero,inline"`
	String                       string                                                      `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u FunctionResultDeltaResult) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *FunctionResultDeltaResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// Only one field in this union will be nonzero
type FunctionResultDeltaResultFunctionResultSubcontentListItem struct {
	Text  *TextContent  `json:",omitzero,inline" discriminator:"text"`
	Image *ImageContent `json:",omitzero,inline" discriminator:"image"`

	metadata `api:"union"`
}

func (u FunctionResultDeltaResultFunctionResultSubcontentListItem) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *FunctionResultDeltaResultFunctionResultSubcontentListItem) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

// Configuration parameters for model interactions.
type GenerationConfig struct {
	// Configuration for image interaction.
	ImageConfig ImageConfig `json:"image_config,omitzero"`
	// The maximum number of tokens to include in the response.
	MaxOutputTokens *int `json:"max_output_tokens,omitzero"`
	// Seed used in decoding for reproducibility.
	Seed *int `json:"seed,omitzero"`
	// Configuration for speech interaction.
	SpeechConfig []SpeechConfig `json:"speech_config,omitzero"`
	// A list of character sequences that will stop output interaction.
	StopSequences []string `json:"stop_sequences,omitzero"`
	// Controls the randomness of the output.
	Temperature *float64 `json:"temperature,omitzero"`
	// The level of thought tokens that the model should generate.
	//
	// Any of "minimal", "low", "medium", "high".
	ThinkingLevel ThinkingLevel `json:"thinking_level,omitzero"`
	// Whether to include thought summaries in the response.
	//
	// Any of "auto", "none".
	ThinkingSummaries string `json:"thinking_summaries,omitzero"`
	// The tool choice configuration.
	ToolChoice *GenerationConfigToolChoice `json:"tool_choice,omitzero"`
	// The maximum cumulative probability of tokens to consider when sampling.
	TopP *float64 `json:"top_p,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type GenerationConfigToolChoice struct {
	ToolChoiceType   ToolChoiceType    `json:",omitzero,inline"`
	ToolChoiceConfig *ToolChoiceConfig `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u GenerationConfigToolChoice) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *GenerationConfigToolChoice) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// The arguments to pass to the Google Maps tool.
type GoogleMapsCallArguments struct {
	// The queries to be executed.
	Queries []string `json:"queries,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Google Maps content.
type GoogleMapsCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// The arguments to pass to the Google Maps tool.
	Arguments GoogleMapsCallArguments `json:"arguments,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleMapsCall `json:"type" default:"google_maps_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type GoogleMapsCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// The arguments to pass to the Google Maps tool.
	Arguments GoogleMapsCallArguments `json:"arguments,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleMapsCall `json:"type" default:"google_maps_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The result of the Google Maps.
type GoogleMapsResult struct {
	// The places that were found.
	Places []GoogleMapsResultPlace `json:"places,omitzero"`
	// Resource name of the Google Maps widget context token.
	WidgetContextToken string `json:"widget_context_token,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type GoogleMapsResultPlace struct {
	// Title of the place.
	Name string `json:"name,omitzero"`
	// The ID of the place, in `places/{place_id}` format.
	PlaceID string `json:"place_id,omitzero"`
	// Snippets of reviews that are used to generate answers about the features of a
	// given place in Google Maps.
	ReviewSnippets []GoogleMapsResultPlaceReviewSnippet `json:"review_snippets,omitzero"`
	// URI reference of the place.
	URL string `json:"url,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Encapsulates a snippet of a user review that answers a question about the
// features of a specific place in Google Maps.
type GoogleMapsResultPlaceReviewSnippet struct {
	// The ID of the review snippet.
	ReviewID string `json:"review_id,omitzero"`
	// Title of the review.
	Title string `json:"title,omitzero"`
	// A link that corresponds to the user review on Google Maps.
	URL string `json:"url,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Google Maps result content.
type GoogleMapsResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// Required. The results of the Google Maps.
	Result []GoogleMapsResult `json:"result" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleMapsResult `json:"type" default:"google_maps_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type GoogleMapsResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// The results of the Google Maps.
	Result []GoogleMapsResult `json:"result,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleMapsResult `json:"type" default:"google_maps_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The arguments to pass to Google Search.
type GoogleSearchCallArguments struct {
	// Web search queries for the following-up web search.
	Queries []string `json:"queries,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Google Search content.
type GoogleSearchCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// Required. The arguments to pass to Google Search.
	Arguments GoogleSearchCallArguments `json:"arguments" api:"required"`
	// The type of search grounding enabled.
	//
	// Any of "web_search", "image_search", "enterprise_web_search".
	SearchType string `json:"search_type,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleSearchCall `json:"type" default:"google_search_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type GoogleSearchCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// The arguments to pass to Google Search.
	Arguments GoogleSearchCallArguments `json:"arguments" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleSearchCall `json:"type" default:"google_search_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The result of the Google Search.
type GoogleSearchResult struct {
	// Web content snippet that can be embedded in a web page or an app webview.
	SearchSuggestions string `json:"search_suggestions,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Google Search result content.
type GoogleSearchResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// Required. The results of the Google Search.
	Result []GoogleSearchResult `json:"result" api:"required"`
	// Whether the Google Search resulted in an error.
	IsError *bool `json:"is_error,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleSearchResult `json:"type" default:"google_search_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type GoogleSearchResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID  string               `json:"call_id" api:"required"`
	Result  []GoogleSearchResult `json:"result" api:"required"`
	IsError *bool                `json:"is_error,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.GoogleSearchResult `json:"type" default:"google_search_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The configuration for image interaction.
type ImageConfig struct {
	// Any of "1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9",
	// "1:8", "8:1", "1:4", "4:1".
	AspectRatio string `json:"aspect_ratio,omitzero"`
	// Any of "1K", "2K", "4K", "512".
	ImageSize string `json:"image_size,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// An image content block.
type ImageContent struct {
	// The image content.
	Data string `json:"data,omitzero" format:"byte"`
	// The mime type of the image.
	//
	// Any of "image/png", "image/jpeg", "image/webp", "image/heic", "image/heif",
	// "image/gif", "image/bmp", "image/tiff".
	MimeType string `json:"mime_type,omitzero"`
	// The resolution of the media.
	//
	// Any of "low", "medium", "high", "ultra_high".
	Resolution string `json:"resolution,omitzero"`
	// The URI of the image.
	Uri string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Image `json:"type" default:"image"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type ImageDelta struct {
	Data string `json:"data,omitzero" format:"byte"`
	// Any of "image/png", "image/jpeg", "image/webp", "image/heic", "image/heif",
	// "image/gif", "image/bmp", "image/tiff".
	MimeType string `json:"mime_type,omitzero"`
	// The resolution of the media.
	//
	// Any of "low", "medium", "high", "ultra_high".
	Resolution string `json:"resolution,omitzero"`
	Uri        string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Image `json:"type" default:"image"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type Input struct {
	ContentList                []Content                   `json:",omitzero,inline"`
	String                     string                      `json:",omitzero,inline"`
	TurnList                   []Turn                      `json:",omitzero,inline"`
	TextContent                *TextContent                `json:",omitzero,inline"`
	ImageContent               *ImageContent               `json:",omitzero,inline"`
	AudioContent               *AudioContent               `json:",omitzero,inline"`
	DocumentContent            *DocumentContent            `json:",omitzero,inline"`
	VideoContent               *VideoContent               `json:",omitzero,inline"`
	ThoughtContent             *ThoughtContent             `json:",omitzero,inline"`
	FunctionCallContent        *FunctionCallContent        `json:",omitzero,inline"`
	CodeExecutionCallContent   *CodeExecutionCallContent   `json:",omitzero,inline"`
	URLContextCallContent      *URLContextCallContent      `json:",omitzero,inline"`
	MCPServerToolCallContent   *MCPServerToolCallContent   `json:",omitzero,inline"`
	GoogleSearchCallContent    *GoogleSearchCallContent    `json:",omitzero,inline"`
	FileSearchCallContent      *FileSearchCallContent      `json:",omitzero,inline"`
	GoogleMapsCallContent      *GoogleMapsCallContent      `json:",omitzero,inline"`
	FunctionResultContent      *FunctionResultContent      `json:",omitzero,inline"`
	CodeExecutionResultContent *CodeExecutionResultContent `json:",omitzero,inline"`
	URLContextResultContent    *URLContextResultContent    `json:",omitzero,inline"`
	GoogleSearchResultContent  *GoogleSearchResultContent  `json:",omitzero,inline"`
	MCPServerToolResultContent *MCPServerToolResultContent `json:",omitzero,inline"`
	FileSearchResultContent    *FileSearchResultContent    `json:",omitzero,inline"`
	GoogleMapsResultContent    *GoogleMapsResultContent    `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u Input) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *Input) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// The Interaction resource.
type Interaction struct {
	// Required. Output only. A unique identifier for the interaction completion.
	ID string `json:"id" api:"required"`
	// Required. Output only. The time at which the response was created in ISO 8601
	// format (YYYY-MM-DDThh:mm:ssZ).
	Created time.Time `json:"created" api:"required" format:"date-time"`
	// Required. Output only. The status of the interaction.
	//
	// Any of "in_progress", "requires_action", "completed", "failed", "cancelled",
	// "incomplete".
	Status string `json:"status" api:"required"`
	// Required. Output only. The time at which the response was last updated in ISO
	// 8601 format (YYYY-MM-DDThh:mm:ssZ).
	Updated time.Time `json:"updated" api:"required" format:"date-time"`
	// The name of the `Agent` used for generating the interaction.
	Agent string `json:"agent,omitzero"`
	// Configuration parameters for the agent interaction.
	AgentConfig *InteractionAgentConfig `json:"agent_config,omitzero"`
	// Input only. Configuration parameters for the model interaction.
	GenerationConfig GenerationConfig `json:"generation_config,omitzero"`
	// The input for the interaction.
	Input *Input `json:"input,omitzero"`
	// The name of the `Model` used for generating the interaction.
	Model string `json:"model,omitzero"`
	// Output only. Responses from the model.
	Outputs []Content `json:"outputs,omitzero"`
	// The ID of the previous interaction, if any.
	PreviousInteractionID string `json:"previous_interaction_id,omitzero"`
	// Enforces that the generated response is a JSON object that complies with the
	// JSON schema specified in this field.
	ResponseFormat any `json:"response_format,omitzero"`
	// The mime type of the response. This is required if response_format is set.
	ResponseMimeType string `json:"response_mime_type,omitzero"`
	// The requested modalities of the response (TEXT, IMAGE, AUDIO).
	//
	// Any of "text", "image", "audio", "video", "document".
	ResponseModalities []string `json:"response_modalities,omitzero"`
	// Output only. The role of the interaction.
	Role string `json:"role,omitzero"`
	// The service tier for the interaction.
	//
	// Any of "flex", "standard", "priority".
	ServiceTier string `json:"service_tier,omitzero"`
	// System instruction for the interaction.
	SystemInstruction string `json:"system_instruction,omitzero"`
	// A list of tool declarations the model may call during interaction.
	Tools []Tool `json:"tools,omitzero"`
	// Output only. Statistics on the interaction request's token usage.
	Usage Usage `json:"usage,omitzero"`
	// Optional. Webhook configuration for receiving notifications when the interaction
	// completes.
	WebhookConfig WebhookConfig `json:"webhook_config,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type InteractionAgentConfig struct {
	Dynamic      *DynamicAgentConfig      `json:",omitzero,inline" discriminator:"dynamic"`
	DeepResearch *DeepResearchAgentConfig `json:",omitzero,inline" discriminator:"deep-research"`

	metadata `api:"union"`
}

func (u InteractionAgentConfig) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *InteractionAgentConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type InteractionCompleteEvent struct {
	// Required. The completed interaction with empty outputs to reduce the payload
	// size. Use the preceding ContentDelta events for the actual output.
	Interaction Interaction `json:"interaction" api:"required"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.InteractionComplete `json:"event_type" default:"interaction.complete"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type InteractionSSEEvent struct {
	InteractionStart        *InteractionStartEvent    `json:",omitzero,inline" discriminator:"interaction.start"`
	InteractionComplete     *InteractionCompleteEvent `json:",omitzero,inline" discriminator:"interaction.complete"`
	InteractionStatusUpdate *InteractionStatusUpdate  `json:",omitzero,inline" discriminator:"interaction.status_update"`
	ContentStart            *ContentStart             `json:",omitzero,inline" discriminator:"content.start"`
	ContentDelta            *ContentDelta             `json:",omitzero,inline" discriminator:"content.delta"`
	ContentStop             *ContentStop              `json:",omitzero,inline" discriminator:"content.stop"`
	Error                   *ErrorEvent               `json:",omitzero,inline" discriminator:"error"`

	metadata `api:"union"`
}

func (u InteractionSSEEvent) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *InteractionSSEEvent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "event_type", u, &u.metadata)
}

type InteractionStartEvent struct {
	// The Interaction resource.
	Interaction Interaction `json:"interaction" api:"required"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.InteractionStart `json:"event_type" default:"interaction.start"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type InteractionStatusUpdate struct {
	InteractionID string `json:"interaction_id" api:"required"`
	// Any of "in_progress", "requires_action", "completed", "failed", "cancelled",
	// "incomplete".
	Status string `json:"status" api:"required"`
	// The event_id token to be used to resume the interaction stream, from this event.
	EventID string `json:"event_id,omitzero"`
	// This field doesn't need to be set.
	EventType constant.InteractionStatusUpdate `json:"event_type" default:"interaction.status_update"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// MCPServer tool call content.
type MCPServerToolCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// Required. The JSON object of arguments for the function.
	Arguments map[string]any `json:"arguments" api:"required"`
	// Required. The name of the tool which was called.
	Name string `json:"name" api:"required"`
	// Required. The name of the used MCP server.
	ServerName string `json:"server_name" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.MCPServerToolCall `json:"type" default:"mcp_server_tool_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type MCPServerToolCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID         string         `json:"id" api:"required"`
	Arguments  map[string]any `json:"arguments" api:"required"`
	Name       string         `json:"name" api:"required"`
	ServerName string         `json:"server_name" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.MCPServerToolCall `json:"type" default:"mcp_server_tool_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// MCPServer tool result content.
type MCPServerToolResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// The output from the MCP server call. Can be simple text or rich content.
	Result MCPServerToolResultContentResult `json:"result" api:"required"`
	// Name of the tool which is called for this specific tool call.
	Name string `json:"name,omitzero"`
	// The name of the used MCP server.
	ServerName string `json:"server_name,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.MCPServerToolResult `json:"type" default:"mcp_server_tool_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type MCPServerToolResultContentResult struct {
	FunctionResultSubcontentList []MCPServerToolResultContentResultFunctionResultSubcontentListItem `json:",omitzero,inline"`
	String                       string                                                             `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u MCPServerToolResultContentResult) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *MCPServerToolResultContentResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// Only one field in this union will be nonzero
type MCPServerToolResultContentResultFunctionResultSubcontentListItem struct {
	Text  *TextContent  `json:",omitzero,inline" discriminator:"text"`
	Image *ImageContent `json:",omitzero,inline" discriminator:"image"`

	metadata `api:"union"`
}

func (u MCPServerToolResultContentResultFunctionResultSubcontentListItem) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *MCPServerToolResultContentResultFunctionResultSubcontentListItem) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type MCPServerToolResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID     string                         `json:"call_id" api:"required"`
	Result     MCPServerToolResultDeltaResult `json:"result" api:"required"`
	Name       string                         `json:"name,omitzero"`
	ServerName string                         `json:"server_name,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.MCPServerToolResult `json:"type" default:"mcp_server_tool_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type MCPServerToolResultDeltaResult struct {
	FunctionResultSubcontentList []MCPServerToolResultDeltaResultFunctionResultSubcontentListItem `json:",omitzero,inline"`
	String                       string                                                           `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u MCPServerToolResultDeltaResult) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *MCPServerToolResultDeltaResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// Only one field in this union will be nonzero
type MCPServerToolResultDeltaResultFunctionResultSubcontentListItem struct {
	Text  *TextContent  `json:",omitzero,inline" discriminator:"text"`
	Image *ImageContent `json:",omitzero,inline" discriminator:"image"`

	metadata `api:"union"`
}

func (u MCPServerToolResultDeltaResultFunctionResultSubcontentListItem) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *MCPServerToolResultDeltaResultFunctionResultSubcontentListItem) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

// Only one field in this union will be nonzero
type Model struct {
	Model  string `json:",omitzero,inline"`
	String string `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u Model) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *Model) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// A place citation annotation.
type PlaceCitation struct {
	// End of the attributed segment, exclusive.
	EndIndex *int `json:"end_index,omitzero"`
	// Title of the place.
	Name string `json:"name,omitzero"`
	// The ID of the place, in `places/{place_id}` format.
	PlaceID string `json:"place_id,omitzero"`
	// Snippets of reviews that are used to generate answers about the features of a
	// given place in Google Maps.
	ReviewSnippets []PlaceCitationReviewSnippet `json:"review_snippets,omitzero"`
	// Start of segment of the response that is attributed to this source.
	//
	// Index indicates the start of the segment, measured in bytes.
	StartIndex *int `json:"start_index,omitzero"`
	// URI reference of the place.
	URL string `json:"url,omitzero"`
	// This field doesn't need to be set.
	Type constant.PlaceCitation `json:"type" default:"place_citation"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Encapsulates a snippet of a user review that answers a question about the
// features of a specific place in Google Maps.
type PlaceCitationReviewSnippet struct {
	// The ID of the review snippet.
	ReviewID string `json:"review_id,omitzero"`
	// Title of the review.
	Title string `json:"title,omitzero"`
	// A link that corresponds to the user review on Google Maps.
	URL string `json:"url,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The configuration for speech interaction.
type SpeechConfig struct {
	// The language of the speech.
	Language string `json:"language,omitzero"`
	// The speaker's name, it should match the speaker name given in the prompt.
	Speaker string `json:"speaker,omitzero"`
	// The voice of the speaker.
	Voice string `json:"voice,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type TextAnnotationDelta struct {
	// Citation information for model-generated content.
	Annotations []Annotation `json:"annotations,omitzero"`
	// This field doesn't need to be set.
	Type constant.TextAnnotation `json:"type" default:"text_annotation"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A text content block.
type TextContent struct {
	// Required. The text content.
	Text string `json:"text" api:"required"`
	// Citation information for model-generated content.
	Annotations []Annotation `json:"annotations,omitzero"`
	// This field doesn't need to be set.
	Type constant.Text `json:"type" default:"text"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type TextDelta struct {
	Text string `json:"text" api:"required"`
	// This field doesn't need to be set.
	Type constant.Text `json:"type" default:"text"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type ThinkingLevel string

const (
	ThinkingLevelMinimal ThinkingLevel = "minimal"
	ThinkingLevelLow     ThinkingLevel = "low"
	ThinkingLevelMedium  ThinkingLevel = "medium"
	ThinkingLevelHigh    ThinkingLevel = "high"
)

// A thought content block.
type ThoughtContent struct {
	// Signature to match the backend source to be part of the generation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// A summary of the thought.
	Summary []ThoughtSummaryContent `json:"summary,omitzero"`
	// This field doesn't need to be set.
	Type constant.Thought `json:"type" default:"thought"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type ThoughtSignatureDelta struct {
	// Signature to match the backend source to be part of the generation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.ThoughtSignature `json:"type" default:"thought_signature"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type ThoughtSummaryContent struct {
	Text  *TextContent  `json:",omitzero,inline" discriminator:"text"`
	Image *ImageContent `json:",omitzero,inline" discriminator:"image"`

	metadata `api:"union"`
}

func (u ThoughtSummaryContent) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *ThoughtSummaryContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type ThoughtSummaryDelta struct {
	// A new summary item to be added to the thought.
	Content *ThoughtSummaryContent `json:"content,omitzero"`
	// This field doesn't need to be set.
	Type constant.ThoughtSummary `json:"type" default:"thought_summary"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type Tool struct {
	Function      *Function          `json:",omitzero,inline" discriminator:"function"`
	CodeExecution *ToolCodeExecution `json:",omitzero,inline" discriminator:"code_execution"`
	URLContext    *ToolURLContext    `json:",omitzero,inline" discriminator:"url_context"`
	ComputerUse   *ToolComputerUse   `json:",omitzero,inline" discriminator:"computer_use"`
	MCPServer     *ToolMCPServer     `json:",omitzero,inline" discriminator:"mcp_server"`
	GoogleSearch  *ToolGoogleSearch  `json:",omitzero,inline" discriminator:"google_search"`
	FileSearch    *ToolFileSearch    `json:",omitzero,inline" discriminator:"file_search"`
	GoogleMaps    *ToolGoogleMaps    `json:",omitzero,inline" discriminator:"google_maps"`
	Retrieval     *ToolRetrieval     `json:",omitzero,inline" discriminator:"retrieval"`

	metadata `api:"union"`
}

func (u Tool) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *Tool) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

// A tool that can be used by the model to execute code.
type ToolCodeExecution struct {
	// This field doesn't need to be set.
	Type constant.CodeExecution `json:"type" default:"code_execution"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model to fetch URL context.
type ToolURLContext struct {
	// This field doesn't need to be set.
	Type constant.URLContext `json:"type" default:"url_context"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model to interact with the computer.
type ToolComputerUse struct {
	// The environment being operated.
	//
	// Any of "browser".
	Environment string `json:"environment,omitzero"`
	// The list of predefined functions that are excluded from the model call.
	ExcludedPredefinedFunctions []string `json:"excludedPredefinedFunctions,omitzero"`
	// This field doesn't need to be set.
	Type constant.ComputerUse `json:"type" default:"computer_use"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A MCPServer is a server that can be called by the model to perform actions.
type ToolMCPServer struct {
	// The allowed tools.
	AllowedTools []AllowedTools `json:"allowed_tools,omitzero"`
	// Optional: Fields for authentication headers, timeouts, etc., if needed.
	Headers map[string]string `json:"headers,omitzero"`
	// The name of the MCPServer.
	Name string `json:"name,omitzero"`
	// The full URL for the MCPServer endpoint. Example: "https://api.example.com/mcp"
	URL string `json:"url,omitzero"`
	// This field doesn't need to be set.
	Type constant.MCPServer `json:"type" default:"mcp_server"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model to search Google.
type ToolGoogleSearch struct {
	// The types of search grounding to enable.
	//
	// Any of "web_search", "image_search", "enterprise_web_search".
	SearchTypes []string `json:"search_types,omitzero"`
	// This field doesn't need to be set.
	Type constant.GoogleSearch `json:"type" default:"google_search"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model to search files.
type ToolFileSearch struct {
	// The file search store names to search.
	FileSearchStoreNames []string `json:"file_search_store_names,omitzero"`
	// Metadata filter to apply to the semantic retrieval documents and chunks.
	MetadataFilter string `json:"metadata_filter,omitzero"`
	// The number of semantic retrieval chunks to retrieve.
	TopK *int `json:"top_k,omitzero"`
	// This field doesn't need to be set.
	Type constant.FileSearch `json:"type" default:"file_search"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model to call Google Maps.
type ToolGoogleMaps struct {
	// Whether to return a widget context token in the tool call result of the
	// response.
	EnableWidget *bool `json:"enable_widget,omitzero"`
	// The latitude of the user's location.
	Latitude *float64 `json:"latitude,omitzero"`
	// The longitude of the user's location.
	Longitude *float64 `json:"longitude,omitzero"`
	// This field doesn't need to be set.
	Type constant.GoogleMaps `json:"type" default:"google_maps"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A tool that can be used by the model to retrieve files.
type ToolRetrieval struct {
	// The types of file retrieval to enable.
	//
	// Any of "vertex_ai_search".
	RetrievalTypes []string `json:"retrieval_types,omitzero"`
	// Used to specify configuration for VertexAISearch.
	VertexAISearchConfig ToolRetrievalVertexAISearchConfig `json:"vertex_ai_search_config,omitzero"`
	// This field doesn't need to be set.
	Type constant.Retrieval `json:"type" default:"retrieval"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Used to specify configuration for VertexAISearch.
type ToolRetrievalVertexAISearchConfig struct {
	// Optional. Used to specify Vertex AI Search datastores.
	Datastores []string `json:"datastores,omitzero"`
	// Optional. Used to specify Vertex AI Search engine.
	Engine string `json:"engine,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The tool choice configuration containing allowed tools.
type ToolChoiceConfig struct {
	// The allowed tools.
	AllowedTools AllowedTools `json:"allowed_tools,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type ToolChoiceType string

const (
	ToolChoiceTypeAuto      ToolChoiceType = "auto"
	ToolChoiceTypeAny       ToolChoiceType = "any"
	ToolChoiceTypeNone      ToolChoiceType = "none"
	ToolChoiceTypeValidated ToolChoiceType = "validated"
)

type Turn struct {
	Content *TurnContent `json:"content,omitzero"`
	// The originator of this turn. Must be user for input or model for model output.
	Role string `json:"role,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Only one field in this union will be nonzero
type TurnContent struct {
	ContentList []Content `json:",omitzero,inline"`
	String      string    `json:",omitzero,inline"`

	metadata `api:"union"`
}

func (u TurnContent) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *TurnContent) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalUnion(data, u, &u.metadata)
}

// A URL citation annotation.
type URLCitation struct {
	// End of the attributed segment, exclusive.
	EndIndex *int `json:"end_index,omitzero"`
	// Start of segment of the response that is attributed to this source.
	//
	// Index indicates the start of the segment, measured in bytes.
	StartIndex *int `json:"start_index,omitzero"`
	// The title of the URL.
	Title string `json:"title,omitzero"`
	// The URL.
	URL string `json:"url,omitzero"`
	// This field doesn't need to be set.
	Type constant.URLCitation `json:"type" default:"url_citation"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The arguments to pass to the URL context.
type URLContextCallArguments struct {
	// The URLs to fetch.
	URLs []string `json:"urls,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// URL context content.
type URLContextCallContent struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// Required. The arguments to pass to the URL context.
	Arguments URLContextCallArguments `json:"arguments" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.URLContextCall `json:"type" default:"url_context_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type URLContextCallDelta struct {
	// Required. A unique ID for this specific tool call.
	ID string `json:"id" api:"required"`
	// The arguments to pass to the URL context.
	Arguments URLContextCallArguments `json:"arguments" api:"required"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.URLContextCall `json:"type" default:"url_context_call"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The result of the URL context.
type URLContextResult struct {
	// The status of the URL retrieval.
	//
	// Any of "success", "error", "paywall", "unsafe".
	Status string `json:"status,omitzero"`
	// The URL that was fetched.
	URL string `json:"url,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// URL context result content.
type URLContextResultContent struct {
	// Required. ID to match the ID from the function call block.
	CallID string `json:"call_id" api:"required"`
	// Required. The results of the URL context.
	Result []URLContextResult `json:"result" api:"required"`
	// Whether the URL context resulted in an error.
	IsError *bool `json:"is_error,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.URLContextResult `json:"type" default:"url_context_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type URLContextResultDelta struct {
	// Required. ID to match the ID from the function call block.
	CallID  string             `json:"call_id" api:"required"`
	Result  []URLContextResult `json:"result" api:"required"`
	IsError *bool              `json:"is_error,omitzero"`
	// A signature hash for backend validation.
	Signature string `json:"signature,omitzero" format:"byte"`
	// This field doesn't need to be set.
	Type constant.URLContextResult `json:"type" default:"url_context_result"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Statistics on the interaction request's token usage.
type Usage struct {
	// A breakdown of cached token usage by modality.
	CachedTokensByModality []UsageCachedTokensByModality `json:"cached_tokens_by_modality,omitzero"`
	// A breakdown of input token usage by modality.
	InputTokensByModality []UsageInputTokensByModality `json:"input_tokens_by_modality,omitzero"`
	// A breakdown of output token usage by modality.
	OutputTokensByModality []UsageOutputTokensByModality `json:"output_tokens_by_modality,omitzero"`
	// A breakdown of tool-use token usage by modality.
	ToolUseTokensByModality []UsageToolUseTokensByModality `json:"tool_use_tokens_by_modality,omitzero"`
	// Number of tokens in the cached part of the prompt (the cached content).
	TotalCachedTokens *int `json:"total_cached_tokens,omitzero"`
	// Number of tokens in the prompt (context).
	TotalInputTokens *int `json:"total_input_tokens,omitzero"`
	// Total number of tokens across all the generated responses.
	TotalOutputTokens *int `json:"total_output_tokens,omitzero"`
	// Number of tokens of thoughts for thinking models.
	TotalThoughtTokens *int `json:"total_thought_tokens,omitzero"`
	// Total token count for the interaction request (prompt + responses + other
	// internal tokens).
	TotalTokens *int `json:"total_tokens,omitzero"`
	// Number of tokens present in tool-use prompt(s).
	TotalToolUseTokens *int `json:"total_tool_use_tokens,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The token count for a single response modality.
type UsageCachedTokensByModality struct {
	// The modality associated with the token count.
	//
	// Any of "text", "image", "audio", "video", "document".
	Modality string `json:"modality,omitzero"`
	// Number of tokens for the modality.
	Tokens *int `json:"tokens,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The token count for a single response modality.
type UsageInputTokensByModality struct {
	// The modality associated with the token count.
	//
	// Any of "text", "image", "audio", "video", "document".
	Modality string `json:"modality,omitzero"`
	// Number of tokens for the modality.
	Tokens *int `json:"tokens,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The token count for a single response modality.
type UsageOutputTokensByModality struct {
	// The modality associated with the token count.
	//
	// Any of "text", "image", "audio", "video", "document".
	Modality string `json:"modality,omitzero"`
	// Number of tokens for the modality.
	Tokens *int `json:"tokens,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// The token count for a single response modality.
type UsageToolUseTokensByModality struct {
	// The modality associated with the token count.
	//
	// Any of "text", "image", "audio", "video", "document".
	Modality string `json:"modality,omitzero"`
	// Number of tokens for the modality.
	Tokens *int `json:"tokens,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// A video content block.
type VideoContent struct {
	// The video content.
	Data string `json:"data,omitzero" format:"byte"`
	// The mime type of the video.
	//
	// Any of "video/mp4", "video/mpeg", "video/mpg", "video/mov", "video/avi",
	// "video/x-flv", "video/webm", "video/wmv", "video/3gpp".
	MimeType string `json:"mime_type,omitzero"`
	// The resolution of the media.
	//
	// Any of "low", "medium", "high", "ultra_high".
	Resolution string `json:"resolution,omitzero"`
	// The URI of the video.
	Uri string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Video `json:"type" default:"video"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type VideoDelta struct {
	Data string `json:"data,omitzero" format:"byte"`
	// Any of "video/mp4", "video/mpeg", "video/mpg", "video/mov", "video/avi",
	// "video/x-flv", "video/webm", "video/wmv", "video/3gpp".
	MimeType string `json:"mime_type,omitzero"`
	// The resolution of the media.
	//
	// Any of "low", "medium", "high", "ultra_high".
	Resolution string `json:"resolution,omitzero"`
	Uri        string `json:"uri,omitzero"`
	// This field doesn't need to be set.
	Type constant.Video `json:"type" default:"video"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

// Message for configuring webhook events for a request.
type WebhookConfig struct {
	// Optional. If set, these webhook URIs will be used for webhook events instead of
	// the registered webhooks.
	Uris []string `json:"uris,omitzero"`
	// Optional. The user metadata that will be returned on each event emission to the
	// webhooks.
	UserMetadata map[string]any `json:"user_metadata,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

type DeleteResponse = any

type DeleteParams struct {
	// Defaults to "v1beta" if not set.
	APIVersion string `path:"api_version" api:"required" json:"-"`
}

type CancelParams struct {
	// Defaults to "v1beta" if not set.
	APIVersion string `path:"api_version" api:"required" json:"-"`
}

type NewAgentParams struct {
	// Defaults to "v1beta" if not set.
	APIVersion string `path:"api_version" api:"required" json:"-"`
	// The name of the `Agent` used for generating the interaction.
	Agent string `json:"agent" api:"required"`
	// The input for the interaction.
	Input Input `json:"input,omitzero" api:"required"`
	// Configuration parameters for the agent interaction.
	AgentConfig *NewAgentParamsAgentConfig `json:"agent_config,omitzero"`
	// Input only. Whether to run the model interaction in the background.
	Background *bool `json:"background,omitzero"`
	// The ID of the previous interaction, if any.
	PreviousInteractionID string `json:"previous_interaction_id,omitzero"`
	// Enforces that the generated response is a JSON object that complies with the
	// JSON schema specified in this field.
	ResponseFormat any `json:"response_format,omitzero"`
	// The mime type of the response. This is required if response_format is set.
	ResponseMimeType string `json:"response_mime_type,omitzero"`
	// The requested modalities of the response (TEXT, IMAGE, AUDIO).
	//
	// Any of "text", "image", "audio", "video", "document".
	ResponseModalities []string `json:"response_modalities,omitzero"`
	// The service tier for the interaction.
	//
	// Any of "flex", "standard", "priority".
	ServiceTier string `json:"service_tier,omitzero"`
	// Input only. Whether to store the response and request for later retrieval.
	Store *bool `json:"store,omitzero"`
	// System instruction for the interaction.
	SystemInstruction string `json:"system_instruction,omitzero"`
	// A list of tool declarations the model may call during interaction.
	Tools []Tool `json:"tools,omitzero"`
	// Optional. Webhook configuration for receiving notifications when the interaction
	// completes.
	WebhookConfig WebhookConfig `json:"webhook_config,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

func (r NewAgentParams) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}
func (r *NewAgentParams) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	for _, val := range r.ResponseModalities {
		unmarshalinfo.ExpectEnum(&r.metadata, val, "text", "image", "audio", "video", "document")
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.ServiceTier, "flex", "standard", "priority")
	return nil
}

// Only one field in this union will be nonzero
type NewAgentParamsAgentConfig struct {
	Dynamic      *DynamicAgentConfig      `json:",omitzero,inline" discriminator:"dynamic"`
	DeepResearch *DeepResearchAgentConfig `json:",omitzero,inline" discriminator:"deep-research"`

	metadata `api:"union"`
}

func (u NewAgentParamsAgentConfig) MarshalJSON() ([]byte, error) {
	return apijson.MarshalUnionStruct(u)
}

func (u *NewAgentParamsAgentConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalDiscriminatedUnion(data, "type", u, &u.metadata)
}

type NewModelParams struct {
	// Defaults to "v1beta" if not set.
	APIVersion string `path:"api_version" api:"required" json:"-"`
	// The input for the interaction.
	Input Input `json:"input,omitzero" api:"required"`
	// The name of the `Model` used for generating the interaction.
	Model string `json:"model" api:"required"`
	// Input only. Whether to run the model interaction in the background.
	Background *bool `json:"background,omitzero"`
	// Input only. Configuration parameters for the model interaction.
	GenerationConfig GenerationConfig `json:"generation_config,omitzero"`
	// The ID of the previous interaction, if any.
	PreviousInteractionID string `json:"previous_interaction_id,omitzero"`
	// Enforces that the generated response is a JSON object that complies with the
	// JSON schema specified in this field.
	ResponseFormat any `json:"response_format,omitzero"`
	// The mime type of the response. This is required if response_format is set.
	ResponseMimeType string `json:"response_mime_type,omitzero"`
	// The requested modalities of the response (TEXT, IMAGE, AUDIO).
	//
	// Any of "text", "image", "audio", "video", "document".
	ResponseModalities []string `json:"response_modalities,omitzero"`
	// The service tier for the interaction.
	//
	// Any of "flex", "standard", "priority".
	ServiceTier string `json:"service_tier,omitzero"`
	// Input only. Whether to store the response and request for later retrieval.
	Store *bool `json:"store,omitzero"`
	// System instruction for the interaction.
	SystemInstruction string `json:"system_instruction,omitzero"`
	// A list of tool declarations the model may call during interaction.
	Tools []Tool `json:"tools,omitzero"`
	// Optional. Webhook configuration for receiving notifications when the interaction
	// completes.
	WebhookConfig WebhookConfig `json:"webhook_config,omitzero"`

	// DynamicFields can be used to add, omit, or overwrite fields
	apidata.DynamicFields `json:"-" api:"extras"`
	metadata
}

func (r NewModelParams) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}
func (r *NewModelParams) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	for _, val := range r.ResponseModalities {
		unmarshalinfo.ExpectEnum(&r.metadata, val, "text", "image", "audio", "video", "document")
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.ServiceTier, "flex", "standard", "priority")
	return nil
}

type GetParams struct {
	// Defaults to "v1beta" if not set.
	APIVersion string `path:"api_version" api:"required" json:"-"`
	// If set to true, includes the input in the response.
	IncludeInput bool `query:"include_input,omitzero" json:"-"`
	// Optional. If set, resumes the interaction stream from the next chunk after the
	// event marked by the event id. Can only be used if `stream` is true.
	LastEventID string `query:"last_event_id,omitzero" json:"-"`
}

// URLQuery serializes [GetParams]'s query parameters as `url.Values`.
func (r GetParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Marshaling/unmarshaling boilerplate below

func (r AllowedTools) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r AudioContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r AudioDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r CodeExecutionCallArguments) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r CodeExecutionCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r CodeExecutionCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r CodeExecutionResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r CodeExecutionResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ContentDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ContentStart) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ContentStop) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r DeepResearchAgentConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r DocumentContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r DocumentDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r DynamicAgentConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ErrorEvent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ErrorEventError) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FileCitation) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FileSearchCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FileSearchCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FileSearchResult) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FileSearchResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FileSearchResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r Function) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FunctionCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FunctionCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FunctionResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r FunctionResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GenerationConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsCallArguments) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsResult) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsResultPlace) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsResultPlaceReviewSnippet) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleMapsResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleSearchCallArguments) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleSearchCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleSearchCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleSearchResult) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleSearchResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r GoogleSearchResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ImageConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ImageContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ImageDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r Interaction) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r InteractionCompleteEvent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r InteractionStartEvent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r InteractionStatusUpdate) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r MCPServerToolCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r MCPServerToolCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r MCPServerToolResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r MCPServerToolResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r PlaceCitation) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r PlaceCitationReviewSnippet) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r SpeechConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r TextAnnotationDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r TextContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r TextDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ThoughtContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ThoughtSignatureDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ThoughtSummaryDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolCodeExecution) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolURLContext) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolComputerUse) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolMCPServer) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolGoogleSearch) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolFileSearch) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolGoogleMaps) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolRetrieval) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolRetrievalVertexAISearchConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r ToolChoiceConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r Turn) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLCitation) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLContextCallArguments) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLContextCallContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLContextCallDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLContextResult) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLContextResultContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r URLContextResultDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r Usage) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r UsageCachedTokensByModality) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r UsageInputTokensByModality) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r UsageOutputTokensByModality) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r UsageToolUseTokensByModality) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r VideoContent) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r VideoDelta) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r WebhookConfig) MarshalJSON() (data []byte, err error) {
	return apijson.MarshalRoot(r)
}

func (r *AllowedTools) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Mode, "auto", "any", "none", "validated")
	return nil
}

func (r *AudioContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "audio")
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.MimeType,
		"audio/wav", "audio/mp3", "audio/aiff", "audio/aac", "audio/ogg", "audio/flac", "audio/mpeg", "audio/m4a", "audio/l16", "audio/opus", "audio/alaw", "audio/mulaw",
	)
	return nil
}

func (r *AudioDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "audio")
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.MimeType,
		"audio/wav", "audio/mp3", "audio/aiff", "audio/aac", "audio/ogg", "audio/flac", "audio/mpeg", "audio/m4a", "audio/l16", "audio/opus", "audio/alaw", "audio/mulaw",
	)
	return nil
}

func (r *CodeExecutionCallArguments) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Language, "python")
	return nil
}

func (r *CodeExecutionCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "code_execution_call")
	return nil
}

func (r *CodeExecutionCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "code_execution_call")
	return nil
}

func (r *CodeExecutionResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "code_execution_result")
	return nil
}

func (r *CodeExecutionResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "code_execution_result")
	return nil
}

func (r *ContentDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "content.delta")
	return nil
}

func (r *ContentStart) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "content.start")
	return nil
}

func (r *ContentStop) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "content.stop")
	return nil
}

func (r *DeepResearchAgentConfig) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "deep-research")
	unmarshalinfo.PreferEnum(&r.metadata, &r.ThinkingSummaries, "auto", "none")
	unmarshalinfo.PreferEnum(&r.metadata, &r.Visualization, "off", "auto")
	return nil
}

func (r *DocumentContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "document")
	unmarshalinfo.PreferEnum(&r.metadata, &r.MimeType, "application/pdf")
	return nil
}

func (r *DocumentDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "document")
	unmarshalinfo.PreferEnum(&r.metadata, &r.MimeType, "application/pdf")
	return nil
}

func (r *DynamicAgentConfig) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "dynamic")
	return nil
}

func (r *ErrorEvent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "error")
	return nil
}

func (r *ErrorEventError) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *FileCitation) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "file_citation")
	return nil
}

func (r *FileSearchCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "file_search_call")
	return nil
}

func (r *FileSearchCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "file_search_call")
	return nil
}

func (r *FileSearchResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *FileSearchResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "file_search_result")
	return nil
}

func (r *FileSearchResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "file_search_result")
	return nil
}

func (r *Function) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "function")
	return nil
}

func (r *FunctionCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "function_call")
	return nil
}

func (r *FunctionCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "function_call")
	return nil
}

func (r *FunctionResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "function_result")
	return nil
}

func (r *FunctionResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "function_result")
	return nil
}

func (r *GenerationConfig) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.ThinkingLevel, "minimal", "low", "medium", "high")
	unmarshalinfo.PreferEnum(&r.metadata, &r.ThinkingSummaries, "auto", "none")
	return nil
}

func (r *GoogleMapsCallArguments) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *GoogleMapsCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_maps_call")
	return nil
}

func (r *GoogleMapsCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_maps_call")
	return nil
}

func (r *GoogleMapsResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *GoogleMapsResultPlace) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *GoogleMapsResultPlaceReviewSnippet) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *GoogleMapsResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_maps_result")
	return nil
}

func (r *GoogleMapsResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_maps_result")
	return nil
}

func (r *GoogleSearchCallArguments) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *GoogleSearchCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_search_call")
	unmarshalinfo.PreferEnum(&r.metadata, &r.SearchType, "web_search", "image_search", "enterprise_web_search")
	return nil
}

func (r *GoogleSearchCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_search_call")
	return nil
}

func (r *GoogleSearchResult) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *GoogleSearchResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_search_result")
	return nil
}

func (r *GoogleSearchResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_search_result")
	return nil
}

func (r *ImageConfig) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.AspectRatio,
		"1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9", "1:8", "8:1", "1:4", "4:1",
	)
	unmarshalinfo.PreferEnum(&r.metadata, &r.ImageSize, "1K", "2K", "4K", "512")
	return nil
}

func (r *ImageContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "image")
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.MimeType,
		"image/png", "image/jpeg", "image/webp", "image/heic", "image/heif", "image/gif", "image/bmp", "image/tiff",
	)
	unmarshalinfo.PreferEnum(&r.metadata, &r.Resolution, "low", "medium", "high", "ultra_high")
	return nil
}

func (r *ImageDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "image")
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.MimeType,
		"image/png", "image/jpeg", "image/webp", "image/heic", "image/heif", "image/gif", "image/bmp", "image/tiff",
	)
	unmarshalinfo.PreferEnum(&r.metadata, &r.Resolution, "low", "medium", "high", "ultra_high")
	return nil
}

func (r *Interaction) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectEnum(
		&r.metadata, r.Status,
		"in_progress", "requires_action", "completed", "failed", "cancelled", "incomplete",
	)
	for _, val := range r.ResponseModalities {
		unmarshalinfo.ExpectEnum(&r.metadata, val, "text", "image", "audio", "video", "document")
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.ServiceTier, "flex", "standard", "priority")
	return nil
}

func (r *InteractionCompleteEvent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "interaction.complete")
	return nil
}

func (r *InteractionStartEvent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "interaction.start")
	return nil
}

func (r *InteractionStatusUpdate) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.EventType, "interaction.status_update")
	unmarshalinfo.ExpectEnum(
		&r.metadata, r.Status,
		"in_progress", "requires_action", "completed", "failed", "cancelled", "incomplete",
	)
	return nil
}

func (r *MCPServerToolCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "mcp_server_tool_call")
	return nil
}

func (r *MCPServerToolCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "mcp_server_tool_call")
	return nil
}

func (r *MCPServerToolResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "mcp_server_tool_result")
	return nil
}

func (r *MCPServerToolResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "mcp_server_tool_result")
	return nil
}

func (r *PlaceCitation) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "place_citation")
	return nil
}

func (r *PlaceCitationReviewSnippet) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *SpeechConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *TextAnnotationDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "text_annotation")
	return nil
}

func (r *TextContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "text")
	return nil
}

func (r *TextDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "text")
	return nil
}

func (r *ThoughtContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "thought")
	return nil
}

func (r *ThoughtSignatureDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "thought_signature")
	return nil
}

func (r *ThoughtSummaryDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "thought_summary")
	return nil
}

func (r *ToolCodeExecution) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "code_execution")
	return nil
}

func (r *ToolURLContext) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "url_context")
	return nil
}

func (r *ToolComputerUse) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "computer_use")
	unmarshalinfo.PreferEnum(&r.metadata, &r.Environment, "browser")
	return nil
}

func (r *ToolMCPServer) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "mcp_server")
	return nil
}

func (r *ToolGoogleSearch) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_search")
	for _, val := range r.SearchTypes {
		unmarshalinfo.ExpectEnum(&r.metadata, val, "web_search", "image_search", "enterprise_web_search")
	}
	return nil
}

func (r *ToolFileSearch) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "file_search")
	return nil
}

func (r *ToolGoogleMaps) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "google_maps")
	return nil
}

func (r *ToolRetrieval) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "retrieval")
	for _, val := range r.RetrievalTypes {
		unmarshalinfo.ExpectEnum(&r.metadata, val, "vertex_ai_search")
	}
	return nil
}

func (r *ToolRetrievalVertexAISearchConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *ToolChoiceConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *Turn) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *URLCitation) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "url_citation")
	return nil
}

func (r *URLContextCallArguments) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *URLContextCallContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "url_context_call")
	return nil
}

func (r *URLContextCallDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "url_context_call")
	return nil
}

func (r *URLContextResult) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Status, "success", "error", "paywall", "unsafe")
	return nil
}

func (r *URLContextResultContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "url_context_result")
	return nil
}

func (r *URLContextResultDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "url_context_result")
	return nil
}

func (r *Usage) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}

func (r *UsageCachedTokensByModality) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Modality, "text", "image", "audio", "video", "document")
	return nil
}

func (r *UsageInputTokensByModality) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Modality, "text", "image", "audio", "video", "document")
	return nil
}

func (r *UsageOutputTokensByModality) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Modality, "text", "image", "audio", "video", "document")
	return nil
}

func (r *UsageToolUseTokensByModality) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.PreferEnum(&r.metadata, &r.Modality, "text", "image", "audio", "video", "document")
	return nil
}

func (r *VideoContent) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "video")
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.MimeType,
		"video/mp4", "video/mpeg", "video/mpg", "video/mov", "video/avi", "video/x-flv", "video/webm", "video/wmv", "video/3gpp",
	)
	unmarshalinfo.PreferEnum(&r.metadata, &r.Resolution, "low", "medium", "high", "ultra_high")
	return nil
}

func (r *VideoDelta) UnmarshalJSON(data []byte) error {
	if err := apijson.UnmarshalRoot(data, r, &r.metadata); err != nil {
		return err
	}
	unmarshalinfo.ExpectConstant(&r.metadata, r.Type, "video")
	unmarshalinfo.PreferEnum(
		&r.metadata, &r.MimeType,
		"video/mp4", "video/mpeg", "video/mpg", "video/mov", "video/avi", "video/x-flv", "video/webm", "video/wmv", "video/3gpp",
	)
	unmarshalinfo.PreferEnum(&r.metadata, &r.Resolution, "low", "medium", "high", "ultra_high")
	return nil
}

func (r *WebhookConfig) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r, &r.metadata)
}
