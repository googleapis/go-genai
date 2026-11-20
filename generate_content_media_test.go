// Copyright 2026 WG ModelHub contributors
// Licensed under the Apache License, Version 2.0. See LICENSE.

package genai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type mediaPatchFixture struct {
	name     string
	contents []*Content
	config   *GenerateContentConfig
}

func mediaPatchClient(t testing.TB, backend Backend) *apiClient {
	t.Helper()
	cc := &ClientConfig{Backend: backend, HTTPClient: &http.Client{}, envVarProvider: func() map[string]string { return map[string]string{} }}
	if backend == BackendVertexAI {
		cc.Project, cc.Location = "audit-project", "global"
	} else {
		cc.APIKey = "offline-placeholder"
	}
	c, err := NewClient(context.Background(), cc)
	if err != nil {
		t.Fatal(err)
	}
	return c.Models.apiClient
}

func mediaPatchFixtures() []mediaPatchFixture {
	data := []byte{0, 1, 127, 128, 255}
	content := func(parts ...*Part) []*Content { return []*Content{{Role: "user", Parts: parts}} }
	text := func() []*Content { return content(NewPartFromText("中文 <>& \"quote\" \\ newline\n")) }
	schema := &Schema{Type: TypeObject, Properties: map[string]*Schema{"name": {Type: TypeString, MinLength: Ptr[int64](0)}}, Required: []string{"name"}}
	return []mediaPatchFixture{
		{"text", text(), nil},
		{"image", content(NewPartFromText("describe"), NewPartFromBytes(data, "image/png")), nil},
		{"multiple_media", content(NewPartFromBytes(data, "image/jpeg"), NewPartFromBytes(data, "audio/wav"), NewPartFromBytes(data, "video/mp4")), nil},
		{"zero_temperature", text(), &GenerateContentConfig{Temperature: Ptr[float32](0), TopP: Ptr[float32](0)}},
		{"tools", content(NewPartFromBytes(data, "image/png")), &GenerateContentConfig{Tools: []*Tool{{FunctionDeclarations: []*FunctionDeclaration{{Name: "lookup", Parameters: schema}}}}}},
		{"system_media", content(NewPartFromBytes(data, "image/png")), &GenerateContentConfig{SystemInstruction: &Content{Parts: []*Part{NewPartFromText("system"), NewPartFromBytes(data, "image/png")}}}},
		{"thinking", content(NewPartFromBytes(data, "image/png")), &GenerateContentConfig{ThinkingConfig: &ThinkingConfig{ThinkingBudget: Ptr[int32](0), IncludeThoughts: true}}},
		{"schema", content(NewPartFromBytes(data, "image/png")), &GenerateContentConfig{ResponseMIMEType: "application/json", ResponseSchema: schema}},
		{"json_schema", content(NewPartFromBytes(data, "image/png")), &GenerateContentConfig{ResponseMIMEType: "application/json", ResponseJsonSchema: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}}}}},
		{"thought_signature", []*Content{{Role: "model", Parts: []*Part{{Text: "thought", Thought: true, ThoughtSignature: data}, NewPartFromBytes(data, "image/png")}}}, nil},
		{"tool_history", []*Content{{Role: "model", Parts: []*Part{{FunctionCall: &FunctionCall{Name: "lookup", Args: map[string]any{"id": int64(123)}}}}}, {Role: "user", Parts: []*Part{{FunctionResponse: &FunctionResponse{Name: "lookup", Response: map[string]any{"result": "ok"}}}, NewPartFromBytes(data, "image/png")}}}, nil},
		{"video_metadata", content(&Part{InlineData: &Blob{Data: data, MIMEType: "video/mp4"}, VideoMetadata: &VideoMetadata{EndOffset: 2 * time.Second}}), nil},
		{"empty_media", content(&Part{InlineData: &Blob{Data: []byte{}, MIMEType: "image/png"}}, &Part{InlineData: &Blob{MIMEType: "image/png"}}), nil},
		{"empty_blob_metadata", content(&Part{InlineData: &Blob{Data: data}}), nil},
		{"http_extra_body", content(NewPartFromBytes(data, "image/png")), &GenerateContentConfig{HTTPOptions: &HTTPOptions{ExtraBody: map[string]any{"auditExtra": "kept"}}}},
	}
}

func mediaPatchBaseline(model string, contents []*Content, config *GenerateContentConfig) (map[string]any, error) {
	var m map[string]any
	err := deepMarshal(map[string]any{"model": model, "contents": contents, "config": config}, &m)
	return m, err
}

func mediaPatchBuild(ac *apiClient, params map[string]any, cfg *GenerateContentConfig, stream bool) (*http.Request, error) {
	converter := generateContentParametersToMldev
	if ac.clientConfig.Backend == BackendVertexAI {
		converter = generateContentParametersToVertex
	}
	body, err := converter(ac, params, nil, params)
	if err != nil {
		return nil, err
	}
	suffix := ":generateContent"
	if stream {
		suffix = ":streamGenerateContent?alt=sse"
	}
	path, err := InternalFormatMap("{model}"+suffix, body["_url"].(map[string]any))
	if err != nil {
		return nil, err
	}
	delete(body, "_url")
	delete(body, "_query")
	opts := &HTTPOptions{Headers: http.Header{}}
	if cfg != nil && cfg.HTTPOptions != nil {
		opts = cfg.HTTPOptions
	}
	r, _, err := buildRequest(context.Background(), ac, path, body, http.MethodPost, opts)
	return r, err
}

func mediaPatchJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestMediaPatchParameters(t *testing.T) {
	fixtures := append(mediaPatchFixtures(), mediaPatchFixture{"nil_contents", nil, nil}, mediaPatchFixture{"empty_contents", []*Content{}, nil}, mediaPatchFixture{"nil_nodes", []*Content{nil, {Parts: []*Part{nil, NewPartFromBytes([]byte{1}, "image/png")}}}, nil})
	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			before := mediaPatchJSON(t, map[string]any{"contents": tc.contents, "config": tc.config})
			want, err := mediaPatchBaseline("gemini-2.5-flash", tc.contents, tc.config)
			if err != nil {
				t.Fatal(err)
			}
			got, err := marshalGenerateContentParameters("gemini-2.5-flash", tc.contents, tc.config)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(mediaPatchJSON(t, want), mediaPatchJSON(t, got)) {
				t.Fatal("parameter JSON changed")
			}
			if !bytes.Equal(before, mediaPatchJSON(t, map[string]any{"contents": tc.contents, "config": tc.config})) {
				t.Fatal("input mutated")
			}
			for i, c := range tc.contents {
				if c == nil {
					continue
				}
				for j, p := range c.Parts {
					if p == nil || p.InlineData == nil || len(p.InlineData.Data) == 0 {
						continue
					}
					data := got["contents"].([]any)[i].(map[string]any)["parts"].([]any)[j].(map[string]any)["inlineData"].(map[string]any)["data"].([]byte)
					if &data[0] != &p.InlineData.Data[0] {
						t.Fatal("media array copied")
					}
				}
			}
			if tc.name == "system_media" {
				data := got["config"].(map[string]any)["systemInstruction"].(map[string]any)["parts"].([]any)[1].(map[string]any)["inlineData"].(map[string]any)["data"].([]byte)
				if &data[0] != &tc.config.SystemInstruction.Parts[1].InlineData.Data[0] {
					t.Fatal("system media copied")
				}
			}
		})
	}
}

type mediaPatchTransport func(*http.Request) (*http.Response, error)

func (f mediaPatchTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMediaPatchSDKHTTP(t *testing.T) {
	const response = `{"candidates":[{"content":{"role":"model","parts":[{"text":"OK"},{"inlineData":{"mimeType":"image/png","data":"AAECAw=="}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":2,"totalTokenCount":12},"responseId":"audit-response"}`
	for _, backend := range []Backend{BackendGeminiAPI, BackendVertexAI} {
		for _, stream := range []bool{false, true} {
			for _, tc := range mediaPatchFixtures() {
				t.Run(fmt.Sprintf("%v/stream=%v/%s", backend, stream, tc.name), func(t *testing.T) {
					ac := mediaPatchClient(t, backend)
					// Public SDK methods apply defaults before parameter conversion.
					if tc.config != nil {
						tc.config.setDefaults()
					}
					params, err := mediaPatchBaseline("gemini-2.5-flash", tc.contents, tc.config)
					if err != nil {
						t.Fatal(err)
					}
					want, err := mediaPatchBuild(ac, params, tc.config, stream)
					if err != nil {
						t.Fatal(err)
					}
					wantBody, err := io.ReadAll(want.Body)
					if err != nil {
						t.Fatal(err)
					}
					want.Body.Close()
					calls := 0
					ac.clientConfig.HTTPClient.Transport = mediaPatchTransport(func(r *http.Request) (*http.Response, error) {
						calls++
						body, err := io.ReadAll(r.Body)
						if err != nil {
							return nil, err
						}
						if !bytes.Equal(wantBody, body) {
							t.Error("HTTP request bytes changed")
						}
						if r.Method != want.Method || r.URL.String() != want.URL.String() || r.ContentLength != want.ContentLength {
							t.Error("HTTP envelope changed")
						}
						for _, key := range []string{"Content-Type", "x-goog-api-key", "x-goog-api-client"} {
							if r.Header.Get(key) != want.Header.Get(key) {
								t.Errorf("header changed: %s", key)
							}
						}
						if r.GetBody == nil {
							t.Fatal("request cannot be replayed")
						}
						replay, err := r.GetBody()
						if err != nil {
							return nil, err
						}
						copyBody, err := io.ReadAll(replay)
						replay.Close()
						if err != nil {
							return nil, err
						}
						if !bytes.Equal(body, copyBody) {
							t.Error("GetBody bytes changed")
						}
						payload := response
						contentType := "application/json"
						if stream {
							payload = "data: " + response + "\n\ndata: " + `{"candidates":[{"content":{"parts":[{"text":"!"}]},"finishReason":"STOP"}],"usageMetadata":{"totalTokenCount":12}}` + "\n\n"
							contentType = "text/event-stream"
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
					})
					m := Models{apiClient: ac}
					var responses []*GenerateContentResponse
					if stream {
						for r, e := range m.GenerateContentStream(context.Background(), "gemini-2.5-flash", tc.contents, tc.config) {
							if e != nil {
								t.Fatal(e)
							}
							responses = append(responses, r)
						}
					} else {
						r, e := m.GenerateContent(context.Background(), "gemini-2.5-flash", tc.contents, tc.config)
						if e != nil {
							t.Fatal(e)
						}
						responses = append(responses, r)
					}
					if calls != 1 {
						t.Fatalf("HTTP call count=%d", calls)
					}
					wantCount := 1
					if stream {
						wantCount = 2
					}
					if len(responses) != wantCount {
						t.Fatalf("response count=%d", len(responses))
					}
					first := responses[0]
					if first.ResponseID != "audit-response" || first.Candidates[0].Content.Parts[0].Text != "OK" || !bytes.Equal(first.Candidates[0].Content.Parts[1].InlineData.Data, []byte{0, 1, 2, 3}) {
						t.Fatal("response decoding changed")
					}
					last := responses[len(responses)-1]
					if last.UsageMetadata.TotalTokenCount != 12 || last.Candidates[0].FinishReason != FinishReasonStop {
						t.Fatal("terminal metadata changed")
					}
				})
			}
		}
	}
}

func TestMediaPatchConcurrent(t *testing.T) {
	tc := mediaPatchFixtures()[5]
	before := mediaPatchJSON(t, map[string]any{"contents": tc.contents, "config": tc.config})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if _, err := marshalGenerateContentParameters("gemini-2.5-flash", tc.contents, tc.config); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if !reflect.DeepEqual(before, mediaPatchJSON(t, map[string]any{"contents": tc.contents, "config": tc.config})) {
		t.Fatal("shared input mutated")
	}
}

func TestMediaPatchCallbackSnapshot(t *testing.T) {
	for _, backend := range []Backend{BackendGeminiAPI, BackendVertexAI} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/stream=%v", backend, stream), func(t *testing.T) {
				original := []byte("before")
				encoded := base64.StdEncoding.EncodeToString(original)
				contents := []*Content{NewContentFromBytes(original, "image/png", RoleUser)}
				// Gemini converters return []map; Vertex retains JSON-decoded []any.
				first := func(items any) map[string]any {
					if maps, ok := items.([]map[string]any); ok {
						return maps[0]
					}
					return items.([]any)[0].(map[string]any)
				}
				config := &GenerateContentConfig{
					SystemInstruction: NewContentFromBytes(original, "image/png", RoleUser),
					HTTPOptions: &HTTPOptions{ExtrasRequestProvider: func(body map[string]any) map[string]any {
						for _, content := range []map[string]any{
							first(body["contents"]),
							body["systemInstruction"].(map[string]any),
						} {
							data := first(content["parts"])["inlineData"].(map[string]any)["data"]
							if text, ok := data.(string); !ok || text != encoded {
								t.Fatalf("callback lost Base64 snapshot: type=%T", data)
							}
						}
						// A same-goroutine mutation is deterministic, without a data race.
						copy(original, "after!")
						body["callbackApplied"] = true
						return body
					}},
				}
				ac := mediaPatchClient(t, backend)
				calls := 0
				ac.clientConfig.HTTPClient.Transport = mediaPatchTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					for _, content := range []map[string]any{
						body["contents"].([]any)[0].(map[string]any), body["systemInstruction"].(map[string]any),
					} {
						data := content["parts"].([]any)[0].(map[string]any)["inlineData"].(map[string]any)["data"]
						if data != encoded {
							t.Fatal("original-array mutation changed the outgoing snapshot")
						}
					}
					if body["callbackApplied"] != true || string(original) != "after!" {
						t.Fatal("callback did not run")
					}
					payload := `{"candidates":[{"content":{"parts":[{"text":"OK"}]},"finishReason":"STOP"}]}`
					contentType := "application/json"
					if stream {
						payload = "data: " + payload + "\n\n"
						contentType = "text/event-stream"
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
				})
				model := Models{apiClient: ac}
				if stream {
					for _, err := range model.GenerateContentStream(context.Background(), "gemini-2.5-flash", contents, config) {
						if err != nil {
							t.Fatal(err)
						}
					}
				} else if _, err := model.GenerateContent(context.Background(), "gemini-2.5-flash", contents, config); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatalf("requests=%d", calls)
				}
			})
		}
	}
}

func BenchmarkMediaPatch(b *testing.B) {
	for _, size := range []int{1 << 20, 10 << 20} {
		data := bytes.Repeat([]byte{0, 1, 127, 128, 255}, size/5+1)[:size]
		contents := []*Content{NewContentFromBytes(data, "image/png", RoleUser)}
		for _, backend := range []Backend{BackendGeminiAPI, BackendVertexAI} {
			for _, candidate := range []bool{false, true} {
				name := "baseline"
				if candidate {
					name = "patched"
				}
				b.Run(fmt.Sprintf("%v/%dMiB/%s", backend, size>>20, name), func(b *testing.B) {
					ac := mediaPatchClient(b, backend)
					marshal := mediaPatchBaseline
					if candidate {
						marshal = marshalGenerateContentParameters
					}
					b.SetBytes(int64(size))
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						params, err := marshal("gemini-2.5-flash", contents, nil)
						if err != nil {
							b.Fatal(err)
						}
						r, err := mediaPatchBuild(ac, params, nil, false)
						if err != nil {
							b.Fatal(err)
						}
						if _, err = io.Copy(io.Discard, r.Body); err != nil {
							b.Fatal(err)
						}
						r.Body.Close()
					}
				})
			}
		}
	}
}
