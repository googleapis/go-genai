// Copyright 2024 Google LLC
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

package genai

import "cloud.google.com/go/civil"

// Outcome of the code execution.
type Outcome string

const (
	// Unspecified status. This value should not be used.
	OutcomeUnspecified Outcome = "OUTCOME_UNSPECIFIED"
	// Code execution completed successfully.
	OutcomeOK Outcome = "OUTCOME_OK"
	// Code execution finished but with a failure. `stderr` should contain the reason.
	OutcomeFailed Outcome = "OUTCOME_FAILED"
	// Code execution ran for too long, and was cancelled. There may or may not be a partial
	// output present.
	OutcomeDeadlineExceeded Outcome = "OUTCOME_DEADLINE_EXCEEDED"
)

// Programming language of the `code`.
type Language string

const (
	// Unspecified language. This value should not be used.
	LanguageUnspecified Language = "LANGUAGE_UNSPECIFIED"
	// Python >= 3.10, with numpy and simpy available.
	LanguagePython Language = "PYTHON"
)

// A basic data type.
type Type string

const (
	TypeUnspecified Type = "TYPE_UNSPECIFIED"
	TypeString      Type = "STRING"
	TypeNumber      Type = "NUMBER"
	TypeInteger     Type = "INTEGER"
	TypeBoolean     Type = "BOOLEAN"
	TypeArray       Type = "ARRAY"
	TypeObject      Type = "OBJECT"
)

// Harm category.
type HarmCategory string

const (
	// The harm category is unspecified.
	HarmCategoryUnspecified HarmCategory = "HARM_CATEGORY_UNSPECIFIED"
	// The harm category is hate speech.
	HarmCategoryHateSpeech HarmCategory = "HARM_CATEGORY_HATE_SPEECH"
	// The harm category is dangerous content.
	HarmCategoryDangerousContent HarmCategory = "HARM_CATEGORY_DANGEROUS_CONTENT"
	// The harm category is harassment.
	HarmCategoryHarassment HarmCategory = "HARM_CATEGORY_HARASSMENT"
	// The harm category is sexually explicit content.
	HarmCategorySexuallyExplicit HarmCategory = "HARM_CATEGORY_SEXUALLY_EXPLICIT"
	// The harm category is civic integrity.
	HarmCategoryCivicIntegrity HarmCategory = "HARM_CATEGORY_CIVIC_INTEGRITY"
)

// Specify if the threshold is used for probability or severity score. If not specified,
// the threshold is used for probability score.
type HarmBlockMethod string

const (
	// The harm block method is unspecified.
	HarmBlockMethodUnspecified HarmBlockMethod = "HARM_BLOCK_METHOD_UNSPECIFIED"
	// The harm block method uses both probability and severity scores.
	HarmBlockMethodSeverity HarmBlockMethod = "SEVERITY"
	// The harm block method uses the probability score.
	HarmBlockMethodProbability HarmBlockMethod = "PROBABILITY"
)

// The harm block threshold.
type HarmBlockThreshold string

const (
	// Unspecified harm block threshold.
	HarmBlockThresholdUnspecified HarmBlockThreshold = "HARM_BLOCK_THRESHOLD_UNSPECIFIED"
	// Block low threshold and above (i.e. block more).
	HarmBlockThresholdBlockLowAndAbove HarmBlockThreshold = "BLOCK_LOW_AND_ABOVE"
	// Block medium threshold and above.
	HarmBlockThresholdBlockMediumAndAbove HarmBlockThreshold = "BLOCK_MEDIUM_AND_ABOVE"
	// Block only high threshold (i.e. block less).
	HarmBlockThresholdBlockOnlyHigh HarmBlockThreshold = "BLOCK_ONLY_HIGH"
	// Block none.
	HarmBlockThresholdBlockNone HarmBlockThreshold = "BLOCK_NONE"
	// Turn off the safety filter.
	HarmBlockThresholdOff HarmBlockThreshold = "OFF"
)

// The mode of the predictor to be used in dynamic retrieval.
type Mode string

const (
	// Always trigger retrieval.
	ModeUnspecified Mode = "MODE_UNSPECIFIED"
	// Run retrieval only when system decides it is necessary.
	ModeDynamic Mode = "MODE_DYNAMIC"
)

// The reason why the model stopped generating tokens. If empty, the model has not stopped
// generating the tokens.
type FinishReason string

const (
	// The finish reason is unspecified.
	FinishReasonUnspecified FinishReason = "FINISH_REASON_UNSPECIFIED"
	// Token generation reached a natural stopping point or a configured stop sequence.
	FinishReasonStop FinishReason = "STOP"
	// Token generation reached the configured maximum output tokens.
	FinishReasonMaxTokens FinishReason = "MAX_TOKENS"
	// Token generation stopped because the content potentially contains safety violations.
	// NOTE: When streaming, content is empty if content filters blocks the output.
	FinishReasonSafety FinishReason = "SAFETY"
	// The token generation stopped because of potential recitation.
	FinishReasonRecitation FinishReason = "RECITATION"
	// All other reasons that stopped the token generation.
	FinishReasonOther FinishReason = "OTHER"
	// Token generation stopped because the content contains forbidden terms.
	FinishReasonBlocklist FinishReason = "BLOCKLIST"
	// Token generation stopped for potentially containing prohibited content.
	FinishReasonProhibitedContent FinishReason = "PROHIBITED_CONTENT"
	// Token generation stopped because the content potentially contains Sensitive Personally
	// Identifiable Information (SPII).
	FinishReasonSPII FinishReason = "SPII"
	// The function call generated by the model is invalid.
	FinishReasonMalformedFunctionCall FinishReason = "MALFORMED_FUNCTION_CALL"
)

// Harm probability levels in the content.
type HarmProbability string

const (
	// Harm probability unspecified.
	HarmProbabilityUnspecified HarmProbability = "HARM_PROBABILITY_UNSPECIFIED"
	// Negligible level of harm.
	HarmProbabilityNegligible HarmProbability = "NEGLIGIBLE"
	// Low level of harm.
	HarmProbabilityLow HarmProbability = "LOW"
	// Medium level of harm.
	HarmProbabilityMedium HarmProbability = "MEDIUM"
	// High level of harm.
	HarmProbabilityHigh HarmProbability = "HIGH"
)

// Harm severity levels in the content.
type HarmSeverity string

const (
	// Harm severity unspecified.
	HarmSeverityUnspecified HarmSeverity = "HARM_SEVERITY_UNSPECIFIED"
	// Negligible level of harm severity.
	HarmSeverityNegligible HarmSeverity = "HARM_SEVERITY_NEGLIGIBLE"
	// Low level of harm severity.
	HarmSeverityLow HarmSeverity = "HARM_SEVERITY_LOW"
	// Medium level of harm severity.
	HarmSeverityMedium HarmSeverity = "HARM_SEVERITY_MEDIUM"
	// High level of harm severity.
	HarmSeverityHigh HarmSeverity = "HARM_SEVERITY_HIGH"
)

// Blocked reason.
type BlockedReason string

const (
	// Unspecified blocked reason.
	BlockedReasonUnspecified BlockedReason = "BLOCKED_REASON_UNSPECIFIED"
	// Candidates blocked due to safety.
	BlockedReasonSafety BlockedReason = "SAFETY"
	// Candidates blocked due to other reason.
	BlockedReasonOther BlockedReason = "OTHER"
	// Candidates blocked due to the terms which are included from the terminology blocklist.
	BlockedReasonBlocklist BlockedReason = "BLOCKLIST"
	// Candidates blocked due to prohibited content.
	BlockedReasonProhibitedContent BlockedReason = "PROHIBITED_CONTENT"
)

// Config class for the dynamic retrieval config mode.
type DynamicRetrievalConfigMode string

const (
	// Always trigger retrieval.
	DynamicRetrievalConfigModeUnspecified DynamicRetrievalConfigMode = "MODE_UNSPECIFIED"
	// Run retrieval only when system decides it is necessary.
	DynamicRetrievalConfigModeDynamic DynamicRetrievalConfigMode = "MODE_DYNAMIC"
)

// Config class for the function calling config mode.
type FunctionCallingConfigMode string

const (
	// The function calling config mode is unspecified. Should not be used.
	FunctionCallingConfigModeUnspecified FunctionCallingConfigMode = "MODE_UNSPECIFIED"
	// Default model behavior, model decides to predict either function calls or natural
	// language response.
	FunctionCallingConfigModeAuto FunctionCallingConfigMode = "AUTO"
	// Model is constrained to always predicting function calls only. If "allowed_function_names"
	// are set, the predicted function calls will be limited to any one of "allowed_function_names",
	// else the predicted function calls will be any one of the provided "function_declarations".
	FunctionCallingConfigModeAny FunctionCallingConfigMode = "ANY"
	// Model will not predict any function calls. Model behavior is same as when not passing
	// any function declarations.
	FunctionCallingConfigModeNone FunctionCallingConfigMode = "NONE"
)

// The media resolution to use.
type MediaResolution string

const (
	// Media resolution has not been set
	MediaResolutionUnspecified MediaResolution = "MEDIA_RESOLUTION_UNSPECIFIED"
	// Media resolution set to low (64 tokens).
	MediaResolutionLow MediaResolution = "MEDIA_RESOLUTION_LOW"
	// Media resolution set to medium (256 tokens).
	MediaResolutionMedium MediaResolution = "MEDIA_RESOLUTION_MEDIUM"
	// Media resolution set to high (zoomed reframing with 256 tokens).
	MediaResolutionHigh MediaResolution = "MEDIA_RESOLUTION_HIGH"
)

// Enum representing the mask mode of a mask reference image.
type MaskReferenceMode string

const (
	MaskReferenceModeMaskModeDefault      MaskReferenceMode = "MASK_MODE_DEFAULT"
	MaskReferenceModeMaskModeUserProvided MaskReferenceMode = "MASK_MODE_USER_PROVIDED"
	MaskReferenceModeMaskModeBackground   MaskReferenceMode = "MASK_MODE_BACKGROUND"
	MaskReferenceModeMaskModeForeground   MaskReferenceMode = "MASK_MODE_FOREGROUND"
	MaskReferenceModeMaskModeSemantic     MaskReferenceMode = "MASK_MODE_SEMANTIC"
)

// Enum representing the control type of a control reference image.
type ControlReferenceType string

const (
	ControlReferenceTypeControlTypeDefault  ControlReferenceType = "CONTROL_TYPE_DEFAULT"
	ControlReferenceTypeControlTypeCanny    ControlReferenceType = "CONTROL_TYPE_CANNY"
	ControlReferenceTypeControlTypeScribble ControlReferenceType = "CONTROL_TYPE_SCRIBBLE"
	ControlReferenceTypeControlTypeFaceMesh ControlReferenceType = "CONTROL_TYPE_FACE_MESH"
)

// Enum representing the subject type of a subject reference image.
type SubjectReferenceType string

const (
	SubjectReferenceTypeSubjectTypeDefault SubjectReferenceType = "SUBJECT_TYPE_DEFAULT"
	SubjectReferenceTypeSubjectTypePerson  SubjectReferenceType = "SUBJECT_TYPE_PERSON"
	SubjectReferenceTypeSubjectTypeAnimal  SubjectReferenceType = "SUBJECT_TYPE_ANIMAL"
	SubjectReferenceTypeSubjectTypeProduct SubjectReferenceType = "SUBJECT_TYPE_PRODUCT"
)

// Metadata describes the input video content.
type VideoMetadata struct {
	// Optional. The end offset of the video.
	EndOffset string `json:"endOffset,omitempty"`
	// Optional. The start offset of the video.
	StartOffset string `json:"startOffset,omitempty"`
}

// Result of executing the [ExecutableCode]. Always follows a `part` containing the
// [ExecutableCode].
type CodeExecutionResult struct {
	// Required. Outcome of the code execution.
	Outcome Outcome `json:"outcome,omitempty"`
	// Optional. Contains stdout when code execution is successful, stderr or other description
	// otherwise.
	Output string `json:"output,omitempty"`
}

// Code generated by the model that is meant to be executed, and the result returned
// to the model. Generated when using the [FunctionDeclaration] tool and [FunctionCallingConfig]
// mode is set to [Mode.CODE].
type ExecutableCode struct {
	// Required. The code to be executed.
	Code string `json:"code,omitempty"`
	// Required. Programming language of the `code`.
	Language Language `json:"language,omitempty"`
}

// URI based data.
type FileData struct {
	// Required. URI.
	FileURI string `json:"fileUri,omitempty"`
	// Required. The IANA standard MIME type of the source data.
	MIMEType string `json:"mimeType,omitempty"`
}

// A function call.
type FunctionCall struct {
	// The unique ID of the function call. If populated, the client to execute the
	// `function_call` and return the response with the matching `id`.
	ID string `json:"id,omitempty"`
	// Optional. Required. The function parameters and values in JSON object format. See
	// [FunctionDeclaration.parameters] for parameter details.
	Args map[string]any `json:"args,omitempty"`
	// Required. The name of the function to call. Matches [FunctionDeclaration.name].
	Name string `json:"name,omitempty"`
}

// A function response.
type FunctionResponse struct {
	// The ID of the function call this response is for. Populated by the client
	// to match the corresponding function call `id`.
	ID string `json:"id,omitempty"`
	// Required. The name of the function to call. Matches [FunctionDeclaration.name] and
	// [FunctionCall.name].
	Name string `json:"name,omitempty"`
	// Required. The function response in JSON object format. Use "output" key to specify
	// function output and "error" key to specify error details (if any). If "output" and
	// "error" keys are not specified, then whole "response" is treated as function output.
	Response map[string]any `json:"response,omitempty"`
}

// Content blob. It's preferred to send as text directly rather than raw bytes.
type Blob struct {
	// Required. Raw bytes.
	Data []byte `json:"data,omitempty"`
	// Required. The IANA standard MIME type of the source data.
	MIMEType string `json:"mimeType,omitempty"`
}

// A datatype containing media content.
// Exactly one field within a Part should be set, representing the specific type
// of content being conveyed. Using multiple fields within the same `Part`
// instance is considered invalid.
type Part struct {
	// Metadata for a given video.
	VideoMetadata *VideoMetadata `json:"videoMetadata,omitempty"`
	// Optional. Result of executing the [ExecutableCode].
	CodeExecutionResult *CodeExecutionResult `json:"codeExecutionResult,omitempty"`
	// Optional. Code generated by the model that is meant to be executed.
	ExecutableCode *ExecutableCode `json:"executableCode,omitempty"`
	// Optional. URI based data.
	FileData *FileData `json:"fileData,omitempty"`
	// Optional. A predicted [FunctionCall] returned from the model that contains a string
	// representing the [FunctionDeclaration.name] with the parameters and their values.
	FunctionCall *FunctionCall `json:"functionCall,omitempty"`
	// Optional. The result output of a [FunctionCall] that contains a string representing
	// the [FunctionDeclaration.name] and a structured JSON object containing any output
	// from the function call. It is used as context to the model.
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	// Optional. Inlined bytes data.
	InlineData *Blob `json:"inlineData,omitempty"`
	// Optional. Text part (can be code).
	Text string `json:"text,omitempty"`
}

// Contains the multi-part content of a message.
type Content struct {
	// List of parts that constitute a single message. Each part may have
	// a different IANA MIME type.
	Parts []*Part `json:"parts,omitempty"`
	// Optional. The producer of the content. Must be either 'user' or
	// 'model'. Useful to set for multi-turn conversations, otherwise can be
	// left blank or unset. If role is not specified, SDK will determine the role.
	Role string `json:"role,omitempty"`
}

// Schema that defines the format of input and output data.
// Represents a select subset of an OpenAPI 3.0 schema object.
type Schema struct {
	// Optional. Minimum number of the elements for Type.ARRAY.
	MinItems string `json:"minItems,omitempty"`
	// Optional. Example of the object. Will only populated when the object is the root.
	Example any `json:"example,omitempty"`
	// Optional. The order of the properties. Not a standard field in open API spec. Only
	// used to support the order of the properties.
	PropertyOrdering []string `json:"propertyOrdering,omitempty"`
	// Optional. Pattern of the Type.STRING to restrict a string to a regular expression.
	Pattern string `json:"pattern,omitempty"`
	// Optional. SCHEMA FIELDS FOR TYPE INTEGER and NUMBER Minimum value of the Type.INTEGER
	// and Type.NUMBER
	Minimum *float64 `json:"minimum,omitempty"`
	// Optional. Default value of the data.
	Default any `json:"default,omitempty"`
	// Optional. The value should be validated against any (one or more) of the subschemas
	// in the list.
	AnyOf []*Schema `json:"anyOf,omitempty"`
	// Optional. Maximum length of the Type.STRING
	MaxLength string `json:"maxLength,omitempty"`
	// Optional. The title of the Schema.
	Title string `json:"title,omitempty"`
	// Optional. SCHEMA FIELDS FOR TYPE STRING Minimum length of the Type.STRING
	MinLength string `json:"minLength,omitempty"`
	// Optional. Minimum number of the properties for Type.OBJECT.
	MinProperties string `json:"minProperties,omitempty"`
	// Optional. Maximum number of the elements for Type.ARRAY.
	MaxItems string `json:"maxItems,omitempty"`
	// Optional. Maximum value of the Type.INTEGER and Type.NUMBER
	Maximum *float64 `json:"maximum,omitempty"`
	// Optional. Indicates if the value may be null.
	Nullable *bool `json:"nullable,omitempty"`
	// Optional. Maximum number of the properties for Type.OBJECT.
	MaxProperties string `json:"maxProperties,omitempty"`
	// Optional. The type of the data.
	Type Type `json:"type,omitempty"`
	// Optional. The description of the data.
	Description string `json:"description,omitempty"`
	// Optional. Possible values of the element of primitive type with enum format. Examples:
	// 1. We can define direction as : {type:STRING, format:enum, enum:["EAST", NORTH",
	// "SOUTH", "WEST"]} 2. We can define apartment number as : {type:INTEGER, format:enum,
	// enum:["101", "201", "301"]}
	Enum []string `json:"enum,omitempty"`
	// Optional. The format of the data. Supported formats: for NUMBER type: "float", "double"
	// for INTEGER type: "int32", "int64" for STRING type: "email", "byte", etc
	Format string `json:"format,omitempty"`
	// Optional. SCHEMA FIELDS FOR TYPE ARRAY Schema of the elements of Type.ARRAY.
	Items *Schema `json:"items,omitempty"`
	// Optional. SCHEMA FIELDS FOR TYPE OBJECT Properties of Type.OBJECT.
	Properties map[string]*Schema `json:"properties,omitempty"`
	// Optional. Required properties of Type.OBJECT.
	Required []string `json:"required,omitempty"`
}

// Safety settings.
type SafetySetting struct {
	// Determines if the harm block method uses probability or probability
	// and severity scores.
	Method HarmBlockMethod `json:"method,omitempty"`
	// Required. Harm category.
	Category HarmCategory `json:"category,omitempty"`
	// Required. The harm block threshold.
	Threshold HarmBlockThreshold `json:"threshold,omitempty"`
}

// Defines a function that the model can generate JSON inputs for.
// The inputs are based on `OpenAPI 3.0 specifications
// <https://spec.openapis.org/oas/v3.0.3>`_.
type FunctionDeclaration struct {
	// Describes the output from the function in the OpenAPI JSON Schema
	// Object format.
	Response *Schema `json:"response,omitempty"`
	// Optional. Description and purpose of the function. Model uses it to decide how and
	// whether to call the function.
	Description string `json:"description,omitempty"`
	// Required. The name of the function to call. Must start with a letter or an underscore.
	// Must be a-z, A-Z, 0-9, or contain underscores, dots and dashes, with a maximum length
	// of 64.
	Name string `json:"name,omitempty"`
	// Optional. Describes the parameters to this function in JSON Schema Object format.
	// Reflects the Open API 3.03 Parameter Object. string Key: the name of the parameter.
	// Parameter names are case sensitive. Schema Value: the Schema defining the type used
	// for the parameter. For function with no parameters, this can be left unset. Parameter
	// names must start with a letter or an underscore and must only contain chars a-z,
	// A-Z, 0-9, or underscores with a maximum length of 64. Example with 1 required and
	// 1 optional parameter: type: OBJECT properties: param1: type: STRING param2: type:
	// INTEGER required: - param1
	Parameters *Schema `json:"parameters,omitempty"`
}

// Tool to support Google Search in Model. Powered by Google.
type GoogleSearch struct {
}

// Describes the options to customize dynamic retrieval.
type DynamicRetrievalConfig struct {
	// The mode of the predictor to be used in dynamic retrieval.
	Mode DynamicRetrievalConfigMode `json:"mode,omitempty"`
	// Optional. The threshold to be used in dynamic retrieval. If not set, a system default
	// value is used.
	DynamicThreshold *float64 `json:"dynamicThreshold,omitempty"`
}

// Tool to retrieve public web data for grounding, powered by Google.
type GoogleSearchRetrieval struct {
	// Specifies the dynamic retrieval configuration for the given source.
	DynamicRetrievalConfig *DynamicRetrievalConfig `json:"dynamicRetrievalConfig,omitempty"`
}

// Retrieve from Vertex AI Search datastore for grounding. See https://cloud.google.com/products/agent-builder
type VertexAISearch struct {
	// Required. Fully-qualified Vertex AI Search data store resource ID. Format: `projects/{project}/locations/{location}/collections/{collection}/dataStores/{dataStore}`
	Datastore string `json:"datastore,omitempty"`
}

// The definition of the RAG resource.
type VertexRAGStoreRAGResource struct {
	// Optional. RAGCorpora resource name. Format: `projects/{project}/locations/{location}/ragCorpora/{rag_corpus}`
	RAGCorpus string `json:"ragCorpus,omitempty"`
	// Optional. rag_file_id. The files should be in the same rag_corpus set in rag_corpus
	// field.
	RAGFileIDs []string `json:"ragFileIds,omitempty"`
}

// Retrieve from Vertex RAG Store for grounding.
type VertexRAGStore struct {
	// Optional. Deprecated. Please use rag_resources instead.
	RAGCorpora []string `json:"ragCorpora,omitempty"`
	// Optional. The representation of the RAG source. It can be used to specify corpus
	// only or ragfiles. Currently only support one corpus or multiple files from one corpus.
	// In the future we may open up multiple corpora support.
	RAGResources []*VertexRAGStoreRAGResource `json:"ragResources,omitempty"`
	// Optional. Number of top k results to return from the selected corpora.
	SimilarityTopK *int64 `json:"similarityTopK,omitempty"`
	// Optional. Only return results with vector distance smaller than the threshold.
	VectorDistanceThreshold *float64 `json:"vectorDistanceThreshold,omitempty"`
}

// Defines a retrieval tool that model can call to access external knowledge.
type Retrieval struct {
	// Set to use data source powered by Vertex AI Search.
	VertexAISearch *VertexAISearch `json:"vertexAiSearch,omitempty"`
	// Set to use data source powered by Vertex RAG store. User data is uploaded via the
	// VertexRAGDataService.
	VertexRAGStore *VertexRAGStore `json:"vertexRagStore,omitempty"`
}

// Tool that executes code generated by the model, and automatically returns the result
// to the model. See also [ExecutableCode]and [CodeExecutionResult] which are input
// and output to this tool.
type ToolCodeExecution struct {
}

// Tool details of a tool that the model may use to generate a response.
type Tool struct {
	// List of function declarations that the tool supports.
	FunctionDeclarations []*FunctionDeclaration `json:"functionDeclarations,omitempty"`
	// Optional. Retrieval tool type. System will always execute the provided retrieval
	// tool(s) to get external knowledge to answer the prompt. Retrieval results are presented
	// to the model for generation.
	Retrieval *Retrieval `json:"retrieval,omitempty"`
	// Optional. Google Search tool type. Specialized retrieval tool
	// that is powered by Google Search.
	GoogleSearch *GoogleSearch `json:"googleSearch,omitempty"`
	// Optional. GoogleSearchRetrieval tool type. Specialized retrieval tool that is powered
	// by Google search.
	GoogleSearchRetrieval *GoogleSearchRetrieval `json:"googleSearchRetrieval,omitempty"`
	// Optional. CodeExecution tool type. Enables the model to execute code as part of generation.
	// This field is only used by the Gemini Developer API services.
	CodeExecution *ToolCodeExecution `json:"codeExecution,omitempty"`
}

// Function calling config.
type FunctionCallingConfig struct {
	// Optional. Function calling mode.
	Mode FunctionCallingConfigMode `json:"mode,omitempty"`
	// Optional. Function names to call. Only set when the Mode is ANY. Function names should
	// match [FunctionDeclaration.name]. With mode set to ANY, model will predict a function
	// call from the set of function names provided.
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

// Tool config.
// This config is shared for all tools provided in the request.
type ToolConfig struct {
	// Optional. Function calling config.
	FunctionCallingConfig *FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// The configuration for the prebuilt speaker to use.
type PrebuiltVoiceConfig struct {
	// The name of the prebuilt voice to use.
	VoiceName string `json:"voiceName,omitempty"`
}

// The configuration for the voice to use.
type VoiceConfig struct {
	// The configuration for the speaker to use.
	PrebuiltVoiceConfig *PrebuiltVoiceConfig `json:"prebuiltVoiceConfig,omitempty"`
}

// The speech generation configuration.
type SpeechConfig struct {
	// The configuration for the speaker to use.
	VoiceConfig *VoiceConfig `json:"voiceConfig,omitempty"`
}

// When automated routing is specified, the routing will be determined by the pretrained
// routing model and customer provided model routing preference.
type GenerationConfigRoutingConfigAutoRoutingMode struct {
	// The model routing preference.
	ModelRoutingPreference string `json:"modelRoutingPreference,omitempty"`
}

// When manual routing is set, the specified model will be used directly.
type GenerationConfigRoutingConfigManualRoutingMode struct {
	// The model name to use. Only the public LLM models are accepted. e.g. 'gemini-1.5-pro-001'.
	ModelName string `json:"modelName,omitempty"`
}

// The configuration for routing the request to a specific model.
type GenerationConfigRoutingConfig struct {
	// Automated routing.
	AutoMode *GenerationConfigRoutingConfigAutoRoutingMode `json:"autoMode,omitempty"`
	// Manual routing.
	ManualMode *GenerationConfigRoutingConfigManualRoutingMode `json:"manualMode,omitempty"`
}

// Class for configuring optional model parameters.
// For more information, see `Content generation parameters
// <https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/content-generation-parameters>`_.
type GenerateContentConfig struct {
	// Instructions for the model to steer it toward better performance.
	// For example, "Answer as concisely as possible" or "Don't use technical
	// terms in your response".
	SystemInstruction *Content `json:"systemInstruction,omitempty"`
	// Value that controls the degree of randomness in token selection.
	// Lower temperatures are good for prompts that require a less open-ended or
	// creative response, while higher temperatures can lead to more diverse or
	// creative results.
	Temperature *float64 `json:"temperature,omitempty"`
	// Tokens are selected from the most to least probable until the sum
	// of their probabilities equals this value. Use a lower value for less
	// random responses and a higher value for more random responses.
	TopP *float64 `json:"topP,omitempty"`
	// For each token selection step, the ``top_k`` tokens with the
	// highest probabilities are sampled. Then tokens are further filtered based
	// on ``top_p`` with the final token selected using temperature sampling. Use
	// a lower number for less random responses and a higher number for more
	// random responses.
	TopK *float64 `json:"topK,omitempty"`
	// Number of response variations to return.
	// If CandidateCount is zero, then syetem will return only one candidate.
	CandidateCount int64 `json:"candidateCount,omitempty"`
	// Maximum number of tokens that can be generated in the response.
	MaxOutputTokens *int64 `json:"maxOutputTokens,omitempty"`
	// List of strings that tells the model to stop generating text if one
	// of the strings is encountered in the response.
	StopSequences []string `json:"stopSequences,omitempty"`
	// Whether to return the log probabilities of the tokens that were
	// chosen by the model at each step.
	// If ResponseLogprobs is zero value, then no token log probabilities will be returned
	// from API.
	ResponseLogprobs bool `json:"responseLogprobs,omitempty"`
	// Number of top candidate tokens to return the log probabilities for
	// at each generation step.
	Logprobs *int64 `json:"logprobs,omitempty"`
	// Positive values penalize tokens that already appear in the
	// generated text, increasing the probability of generating more diverse
	// content.
	PresencePenalty *float64 `json:"presencePenalty,omitempty"`
	// Positive values penalize tokens that repeatedly appear in the
	// generated text, increasing the probability of generating more diverse
	// content.
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`
	// When ``seed`` is fixed to a specific number, the model makes a best
	// effort to provide the same response for repeated requests. By default, a
	// random number is used.
	Seed *int64 `json:"seed,omitempty"`
	// Output response media type of the generated candidate text.
	ResponseMIMEType string `json:"responseMimeType,omitempty"`
	// Schema that the generated candidate text must adhere to.
	ResponseSchema *Schema `json:"responseSchema,omitempty"`
	// Configuration for model router requests.
	RoutingConfig *GenerationConfigRoutingConfig `json:"routingConfig,omitempty"`
	// Safety settings in the request to block unsafe content in the
	// response.
	SafetySettings []*SafetySetting `json:"safetySettings,omitempty"`
	// Code that enables the system to interact with external systems to
	// perform an action outside of the knowledge and scope of the model.
	Tools []*Tool `json:"tools,omitempty"`
	// Associates model output to a specific function call.
	ToolConfig *ToolConfig `json:"toolConfig,omitempty"`
	// Resource name of a context cache that can be used in subsequent
	// requests.
	CachedContent string `json:"cachedContent,omitempty"`
	// The requested modalities of the response. Represents the set of
	// modalities that the model can return.
	ResponseModalities []string `json:"responseModalities,omitempty"`
	// If specified, the media resolution specified will be used.
	MediaResolution MediaResolution `json:"mediaResolution,omitempty"`
	// The speech generation configuration.
	SpeechConfig *SpeechConfig `json:"speechConfig,omitempty"`
}

// Class for configuring the content of the request to the model.
type GenerateContentParameters struct {
	// ID of the model to use. For a list of models, see `Google models
	// <https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models>`_.
	Model string `json:"model,omitempty"`
	// Content of the request.
	Contents []*Content `json:"contents,omitempty"`
	// Configuration that contains optional model parameters.
	Config *GenerateContentConfig `json:"config,omitempty"`
}

// Source attributions for content.
type Citation struct {
	// Output only. End index into the content.
	// If EndIndex is zero value, then EndIndex is either empty or zero value in API response.
	EndIndex int64 `json:"endIndex,omitempty"`
	// Output only. License of the attribution.
	License string `json:"license,omitempty"`
	// Output only. Publication date of the attribution.
	PublicationDate *civil.Date `json:"publicationDate,omitempty"`
	// Output only. Start index into the content.
	// If StartIndex is zero value, then StartIndex is either empty or zero value in API
	// response.
	StartIndex int64 `json:"startIndex,omitempty"`
	// Output only. Title of the attribution.
	Title string `json:"title,omitempty"`
	// Output only. Url reference of the attribution.
	URI string `json:"uri,omitempty"`
}

// Class for citation information when the model quotes another source.
type CitationMetadata struct {
	// Contains citation information when the model directly quotes, at
	// length, from another source. Can include traditional websites and code
	// repositories.
	Citations []*Citation `json:"citations,omitempty"`
}

// Chunk from context retrieved by the retrieval tools.
type GroundingChunkRetrievedContext struct {
	// Text of the attribution.
	Text string `json:"text,omitempty"`
	// Title of the attribution.
	Title string `json:"title,omitempty"`
	// URI reference of the attribution.
	URI string `json:"uri,omitempty"`
}

// Chunk from the web.
type GroundingChunkWeb struct {
	// Title of the chunk.
	Title string `json:"title,omitempty"`
	// URI reference of the chunk.
	URI string `json:"uri,omitempty"`
}

// Grounding chunk.
type GroundingChunk struct {
	// Grounding chunk from context retrieved by the retrieval tools.
	RetrievedContext *GroundingChunkRetrievedContext `json:"retrievedContext,omitempty"`
	// Grounding chunk from the web.
	Web *GroundingChunkWeb `json:"web,omitempty"`
}

// Segment of the content.
type Segment struct {
	// Output only. End index in the given Part, measured in bytes. Offset from the start
	// of the Part, exclusive, starting at zero.
	// If EndIndex is zero value, then EndIndex is either empty or zero value in API response.
	EndIndex int64 `json:"endIndex,omitempty"`
	// Output only. The index of a Part object within its parent Content object.
	// If PartIndex is zero value, then PartIndex is either empty or zero value in API response.
	PartIndex int64 `json:"partIndex,omitempty"`
	// Output only. Start index in the given Part, measured in bytes. Offset from the start
	// of the Part, inclusive, starting at zero.
	// If StartIndex is zero value, then StartIndex is either empty or zero value in API
	// response.
	StartIndex int64 `json:"startIndex,omitempty"`
	// Output only. The text corresponding to the segment from the response.
	Text string `json:"text,omitempty"`
}

// Grounding support.
type GroundingSupport struct {
	// Confidence score of the support references. Ranges from 0 to 1. 1 is the most confident.
	// This list must have the same size as the grounding_chunk_indices.
	ConfidenceScores []float64 `json:"confidenceScores,omitempty"`
	// A list of indices (into 'grounding_chunk') specifying the citations associated with
	// the claim. For instance [1,3,4] means that grounding_chunk[1], grounding_chunk[3],
	// grounding_chunk[4] are the retrieved content attributed to the claim.
	GroundingChunkIndices []int64 `json:"groundingChunkIndices,omitempty"`
	// Segment of the content this support belongs to.
	Segment *Segment `json:"segment,omitempty"`
}

// Metadata related to retrieval in the grounding flow.
type RetrievalMetadata struct {
	// Optional. Score indicating how likely information from Google Search could help answer
	// the prompt. The score is in the range `[0, 1]`, where 0 is the least likely and 1
	// is the most likely. This score is only populated when Google Search grounding and
	// dynamic retrieval is enabled. It will be compared to the threshold to determine whether
	// to trigger Google Search.
	GoogleSearchDynamicRetrievalScore *float64 `json:"googleSearchDynamicRetrievalScore,omitempty"`
}

// Google search entry point.
type SearchEntryPoint struct {
	// Optional. Web content snippet that can be embedded in a web page or an app webview.
	RenderedContent string `json:"renderedContent,omitempty"`
	// Optional. Base64 encoded JSON representing array of tuple.
	SDKBlob []byte `json:"sdkBlob,omitempty"`
}

// Metadata returned to client when grounding is enabled.
type GroundingMetadata struct {
	// List of supporting references retrieved from specified grounding source.
	GroundingChunks []*GroundingChunk `json:"groundingChunks,omitempty"`
	// Optional. List of grounding support.
	GroundingSupports []*GroundingSupport `json:"groundingSupports,omitempty"`
	// Optional. Output only. Retrieval metadata.
	RetrievalMetadata *RetrievalMetadata `json:"retrievalMetadata,omitempty"`
	// Optional. Queries executed by the retrieval tools.
	RetrievalQueries []string `json:"retrievalQueries,omitempty"`
	// Optional. Google search entry for the following-up web searches.
	SearchEntryPoint *SearchEntryPoint `json:"searchEntryPoint,omitempty"`
	// Optional. Web search queries for the following-up web search.
	WebSearchQueries []string `json:"webSearchQueries,omitempty"`
}

// Candidate for the logprobs token and score.
type LogprobsResultCandidate struct {
	// The candidate's log probability.
	// If LogProbability is zero value, then LogProbability is either empty or zero value
	// in API response.
	LogProbability float64 `json:"logProbability,omitempty"`
	// The candidate's token string value.
	Token string `json:"token,omitempty"`
	// The candidate's token ID value.
	// If TokenID is zero value, then TokenID is either empty or zero value in API response.
	TokenID int64 `json:"tokenId,omitempty"`
}

// Candidates with top log probabilities at each decoding step.
type LogprobsResultTopCandidates struct {
	// Sorted by log probability in descending order.
	Candidates []*LogprobsResultCandidate `json:"candidates,omitempty"`
}

// Logprobs Result
type LogprobsResult struct {
	// Length = total number of decoding steps. The chosen candidates may or may not be
	// in top_candidates.
	ChosenCandidates []*LogprobsResultCandidate `json:"chosenCandidates,omitempty"`
	// Length = total number of decoding steps.
	TopCandidates []*LogprobsResultTopCandidates `json:"topCandidates,omitempty"`
}

// Safety rating corresponding to the generated content.
type SafetyRating struct {
	// Output only. Indicates whether the content was filtered out because of this rating.
	// If Blocked is zero value, then the content is not blocked.
	Blocked bool `json:"blocked,omitempty"`
	// Output only. Harm category.
	Category HarmCategory `json:"category,omitempty"`
	// Output only. Harm probability levels in the content.
	Probability HarmProbability `json:"probability,omitempty"`
	// Output only. Harm probability score.
	// If ProbabilityScore is zero value, then ProbabilityScore is either empty or zero
	// value in API response.
	ProbabilityScore float64 `json:"probabilityScore,omitempty"`
	// Output only. Harm severity levels in the content.
	Severity HarmSeverity `json:"severity,omitempty"`
	// Output only. Harm severity score.
	// If SeverityScore is zero value, then SeverityScore is either empty or zero value
	// in API response.
	SeverityScore float64 `json:"severityScore,omitempty"`
}

// Class containing a response candidate generated from the model.
type Candidate struct {
	// Contains the multi-part content of the response.
	Content *Content `json:"content,omitempty"`
	// Source attribution of the generated content.
	CitationMetadata *CitationMetadata `json:"citationMetadata,omitempty"`
	// Describes the reason the model stopped generating tokens.
	FinishMessage string `json:"finishMessage,omitempty"`
	// Number of tokens for this candidate.
	// If TokenCount is zero value, then TokenCount is either empty or zero value in API
	// response.
	TokenCount int64 `json:"tokenCount,omitempty"`
	// Output only. Average log probability score of the candidate.
	// If AvgLogprobs is zero value, then AvgLogprobs is either empty or zero value in API
	// response.
	AvgLogprobs float64 `json:"avgLogprobs,omitempty"`
	// Output only. The reason why the model stopped generating tokens. If empty, the model
	// has not stopped generating the tokens.
	FinishReason FinishReason `json:"finishReason,omitempty"`
	// Output only. Metadata specifies sources used to ground generated content.
	GroundingMetadata *GroundingMetadata `json:"groundingMetadata,omitempty"`
	// Output only. Index of the candidate.
	// If Index is zero value, then Index is either empty or zero value in API response.
	Index int64 `json:"index,omitempty"`
	// Output only. Log-likelihood scores for the response tokens and top tokens
	LogprobsResult *LogprobsResult `json:"logprobsResult,omitempty"`
	// Output only. List of ratings for the safety of a response candidate. There is at
	// most one rating per category.
	SafetyRatings []*SafetyRating `json:"safetyRatings,omitempty"`
}

// Content filter results for a prompt sent in the request.
type GenerateContentResponsePromptFeedback struct {
	// Output only. Blocked reason.
	BlockReason BlockedReason `json:"blockReason,omitempty"`
	// Output only. A readable block reason message.
	BlockReasonMessage string `json:"blockReasonMessage,omitempty"`
	// Output only. Safety ratings.
	SafetyRatings []*SafetyRating `json:"safetyRatings,omitempty"`
}

// Usage metadata about response(s).
type GenerateContentResponseUsageMetadata struct {
	// Output only. Number of tokens in the cached part in the input (the cached content).
	// If zero value, then no cached content is used.
	CachedContentTokenCount int64 `json:"cachedContentTokenCount,omitempty"`
	// Number of tokens in the response(s).
	// If CandidatesTokenCount is zero value, then this field is empty in response.
	CandidatesTokenCount int64 `json:"candidatesTokenCount,omitempty"`
	// Number of tokens in the request. When `cached_content` is set, this is still the
	// total effective prompt size meaning this includes the number of tokens in the cached
	// content.
	// If PromptTokenCount is zero value, then this field is empty in response.
	PromptTokenCount int64 `json:"promptTokenCount,omitempty"`
	// Total token count for prompt and response candidates.
	// If TotalTokenCount is zero value, then this field is empty in response.
	TotalTokenCount int64 `json:"totalTokenCount,omitempty"`
}

// Response message for PredictionService.GenerateContent.
type GenerateContentResponse struct {
	// Response variations returned by the model.
	Candidates []*Candidate `json:"candidates,omitempty"`
	// Output only. The model version used to generate the response.
	ModelVersion string `json:"modelVersion,omitempty"`
	// Output only. Content filter results for a prompt sent in the request. Note: Sent
	// only in the first stream chunk. Only happens when no candidates were generated due
	// to content violations.
	PromptFeedback *GenerateContentResponsePromptFeedback `json:"promptFeedback,omitempty"`
	// Usage metadata about the response(s).
	UsageMetadata *GenerateContentResponseUsageMetadata `json:"usageMetadata,omitempty"`
}

type testTableItem struct {
	// The name of the test. This is used to derive the replay id.
	Name string `json:"name,omitempty"`
	// The parameters to the test. Use pydantic models.
	Parameters map[string]any `json:"parameters,omitempty"`
	// Expects an exception for MLDev matching the string.
	ExceptionIfMLDev string `json:"exceptionIfMldev,omitempty"`
	// Expects an exception for Vertex matching the string.
	ExceptionIfVertex string `json:"exceptionIfVertex,omitempty"`
	// Use if you don't want to use the default replay ID which is derived from the test
	// name.
	OverrideReplayID string `json:"overrideReplayId,omitempty"`
	// True if the parameters contain an unsupported union type. This test will be skipped
	// for languages that do not support the union type.
	HasUnion bool `json:"hasUnion,omitempty"`
	// When set to a reason string, this test will be skipped in the API mode. Use this
	// flag for tests that can not be reproduced with the real API. E.g. a test that deletes
	// a resource.
	SkipInAPIMode string `json:"skipInApiMode,omitempty"`
}

type testTableFile struct {
	Comment string `json:"comment,omitempty"`

	TestMethod string `json:"testMethod,omitempty"`

	ParameterNames []string `json:"parameterNames,omitempty"`

	TestTable []*testTableItem `json:"testTable,omitempty"`
}

// Represents a single request in a replay.
type replayRequest struct {
	Method string `json:"method,omitempty"`

	Url string `json:"url,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	BodySegments []map[string]any `json:"bodySegments,omitempty"`
}

// Represents a single response in a replay.
type replayResponse struct {
	StatusCode int64 `json:"statusCode,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	BodySegments []map[string]any `json:"bodySegments,omitempty"`

	SDKResponseSegments []map[string]any `json:"sdkResponseSegments,omitempty"`
}

// Represents a single interaction, request and response in a replay.
type replayInteraction struct {
	Request *replayRequest `json:"request,omitempty"`

	Response *replayResponse `json:"response,omitempty"`
}

// Represents a recorded session.
type replayFile struct {
	ReplayID string `json:"replayId,omitempty"`

	Interactions []*replayInteraction `json:"interactions,omitempty"`
}

// Used to override the default configuration.
type UploadFileConfig struct {
	// Used to override HTTP request options.
	HTTPOptions map[string]any `json:"httpOptions,omitempty"`
	// The name of the file in the destination (e.g., 'files/sample-image'. If not provided
	// one will be generated.
	Name string `json:"name,omitempty"`
	// mime_type: The MIME type of the file. If not provided, it will be inferred from the
	// file extension.
	MIMEType string `json:"mimeType,omitempty"`
	// Optional display name of the file.
	DisplayName string `json:"displayName,omitempty"`
}

// Configuration for upscaling an image.
// For more information on this configuration, refer to
// the `Imagen API reference documentation
// <https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/imagen-api>`_.
type UpscaleImageConfig struct {
	// Used to override HTTP request options.
	HTTPOptions map[string]any `json:"httpOptions,omitempty"`
	// Whether to include a reason for filtered-out images in the
	// response.
	// If IncludeRAIReason is zero value, then no RAI reason will applied in the generation.
	IncludeRAIReason bool `json:"includeRaiReason,omitempty"`
	// The image format that the output should be saved as.
	OutputMIMEType string `json:"outputMimeType,omitempty"`
	// The level of compression if the ``output_mime_type`` is
	// ``image/jpeg``.
	OutputCompressionQuality *int64 `json:"outputCompressionQuality,omitempty"`
}

// Class that represents an image.
type Image struct {
	// The Cloud Storage URI of the image. ``Image`` can contain a value
	// for this field or the ``image_bytes`` field but not both.
	GCSURI string `json:"gcsUri,omitempty"`
	// The image bytes data. ``Image`` can contain a value for this field
	// or the ``gcs_uri`` field but not both.
	ImageBytes []byte `json:"imageBytes,omitempty"`
}

// User-facing config UpscaleImageParameters.
type UpscaleImageParameters struct {
	// The model to use.
	Model string `json:"model,omitempty"`
	// The input image to upscale.
	Image *Image `json:"image,omitempty"`
	// The factor to upscale the image (x2 or x4).
	UpscaleFactor string `json:"upscaleFactor,omitempty"`
	// Configuration for upscaling.
	Config *UpscaleImageConfig `json:"config,omitempty"`
}

// Class that represents a Raw reference image.
// A raw reference image represents the base image to edit, provided by the user.
// It can optionally be provided in addition to a mask reference image or
// a style reference image.
type RawReferenceImage struct {
	// The reference image for the editing operation.
	ReferenceImage *Image `json:"referenceImage,omitempty"`
	// The type of the reference image.
	ReferenceType string `json:"referenceType,omitempty"`
}

// Configuration for a Mask reference image.
type MaskReferenceConfig struct {
	// Prompts the model to generate a mask instead of you needing to
	// provide one (unless MASK_MODE_USER_PROVIDED is used).
	MaskMode MaskReferenceMode `json:"maskMode,omitempty"`
	// A list of up to 5 class IDs to use for semantic segmentation.
	// Automatically creates an image mask based on specific objects.
	SegmentationClasses []int64 `json:"segmentationClasses,omitempty"`
}

// Class that represents a Mask reference image.
// This encapsulates either a mask image provided by the user and configs for
// the user provided mask, or only config parameters for the model to generate
// a mask.
// A mask image is an image whose non-zero values indicate where to edit the base
// image. If the user provides a mask image, the mask must be in the same
// dimensions as the raw image.
type MaskReferenceImage struct {
	// The reference image for the editing operation.
	ReferenceImage *Image `json:"referenceImage,omitempty"`
	// The type of the reference image.
	ReferenceType string `json:"referenceType,omitempty"`
	// Configuration for the mask reference image.
	Config *MaskReferenceConfig `json:"config,omitempty"`
}

// Configuration for a Control reference image.
type ControlReferenceConfig struct {
	// The type of control reference image to use.
	ControlType ControlReferenceType `json:"controlType,omitempty"`
}

// Class that represents a Control reference image.
// The image of the control reference image is either a control image provided
// by the user, or a regular image which the backend will use to generate a
// control image of. In the case of the latter, the
// enable_control_image_computation field in the config should be set to True.
// A control image is an image that represents a sketch image of areas for the
// model to fill in based on the prompt.
type ControlReferenceImage struct {
	// The reference image for the editing operation.
	ReferenceImage *Image `json:"referenceImage,omitempty"`
	// The type of the reference image.
	ReferenceType string `json:"referenceType,omitempty"`
	// Configuration for the control reference image.
	Config *ControlReferenceConfig `json:"config,omitempty"`
}

// Configuration for a Style reference image.
type StyleReferenceConfig struct {
	// A text description of the style to use for the generated image.
	StyleDescription string `json:"styleDescription,omitempty"`
}

// Class that represents a Style reference image.
// This encapsulates a style reference image provided by the user, and
// additionally optional config parameters for the style reference image.
// A raw reference image can also be provided as a destination for the style to
// be applied to.
type StyleReferenceImage struct {
	// The reference image for the editing operation.
	ReferenceImage *Image `json:"referenceImage,omitempty"`
	// The type of the reference image.
	ReferenceType string `json:"referenceType,omitempty"`
	// Configuration for the style reference image.
	Config *StyleReferenceConfig `json:"config,omitempty"`
}

// Configuration for a Subject reference image.
type SubjectReferenceConfig struct {
	// The subject type of a subject reference image.
	SubjectType SubjectReferenceType `json:"subjectType,omitempty"`
	// Subject description for the image.
	SubjectDescription string `json:"subjectDescription,omitempty"`
}

// Class that represents a Subject reference image.
// This encapsulates a subject reference image provided by the user, and
// additionally optional config parameters for the subject reference image.
// A raw reference image can also be provided as a destination for the subject to
// be applied to.
type SubjectReferenceImage struct {
	// The reference image for the editing operation.
	ReferenceImage *Image `json:"referenceImage,omitempty"`
	// The type of the reference image.
	ReferenceType string `json:"referenceType,omitempty"`
	// Configuration for the subject reference image.
	Config *SubjectReferenceConfig `json:"config,omitempty"`
}

// Generation config.
type GenerationConfig struct {
	// Optional. If enabled, audio timestamp will be included in the request to the model.
	// If zero value, then no audio timestamp will be included in model inference.
	AudioTimestamp bool `json:"audioTimestamp,omitempty"`
	// Optional. Number of candidates to generate.
	CandidateCount *int64 `json:"candidateCount,omitempty"`
	// Optional. Frequency penalties.
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`
	// Optional. Logit probabilities.
	Logprobs *int64 `json:"logprobs,omitempty"`
	// Optional. The maximum number of output tokens to generate per message.
	MaxOutputTokens *int64 `json:"maxOutputTokens,omitempty"`
	// Optional. Positive penalties.
	PresencePenalty *float64 `json:"presencePenalty,omitempty"`
	// Optional. If true, export the logprobs results in response.
	// If ResponseLogprobs is zero, then no token log probabilities will be returned from
	// API.
	ResponseLogprobs bool `json:"responseLogprobs,omitempty"`
	// Optional. Output response mimetype of the generated candidate text. Supported mimetype:
	// - `text/plain`: (default) Text output. - `application/json`: JSON response in the
	// candidates. The model needs to be prompted to output the appropriate response type,
	// otherwise the behavior is undefined. This is a preview feature.
	ResponseMIMEType string `json:"responseMimeType,omitempty"`
	// Optional. The `Schema` object allows the definition of input and output data types.
	// These types can be objects, but also primitives and arrays. Represents a select subset
	// of an [OpenAPI 3.0 schema object](https://spec.openapis.org/oas/v3.0.3#schema). If
	// set, a compatible response_mime_type must also be set. Compatible mimetypes: `application/json`:
	// Schema for JSON response.
	ResponseSchema *Schema `json:"responseSchema,omitempty"`
	// Optional. Routing configuration.
	RoutingConfig *GenerationConfigRoutingConfig `json:"routingConfig,omitempty"`
	// Optional. Seed.
	Seed *int64 `json:"seed,omitempty"`
	// Optional. Stop sequences.
	StopSequences []string `json:"stopSequences,omitempty"`
	// Optional. Controls the randomness of predictions.
	Temperature *float64 `json:"temperature,omitempty"`
	// Optional. If specified, top-k sampling will be used.
	TopK *float64 `json:"topK,omitempty"`
	// Optional. If specified, nucleus sampling will be used.
	TopP *float64 `json:"topP,omitempty"`
}
