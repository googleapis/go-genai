package genai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const precisionArgsJSON = `{"id":9007199254740993,"nested":{"decimal":1.234567890123456789,"values":[-9007199254740993,42,0,0.5]},"label":"unchanged","enabled":true,"empty":null}`

func TestFunctionCallNumericPrecision(t *testing.T) {
	// Arrange.
	var call FunctionCall
	input := `{"name":"lookup","args":` + precisionArgsJSON + `}`
	// Act.
	if err := json.Unmarshal([]byte(input), &call); err != nil {
		t.Fatal(err)
	}
	// Assert.
	assertPrecisionArgs(t, call.Args)
	if _, ok := call.Args["id"].(json.Number); !ok {
		t.Fatalf("Args[id] has type %T, want json.Number", call.Args["id"])
	}
}

func TestGenerateContentFunctionCallNumericPrecision(t *testing.T) {
	for _, backend := range []Backend{BackendGeminiAPI, BackendVertexAI} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("backend=%v/stream=%v", backend, stream), func(t *testing.T) {
				// Arrange: the response is literal JSON so the fixture cannot round it first.
				response := `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"lookup","args":` + precisionArgsJSON + `},"thoughtSignature":"AQID"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":42}}`
				requests := make(chan []byte, 2)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						w.WriteHeader(500)
						return
					}
					requests <- body
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprintf(w, "data: %s\n\n", response)
					} else {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, response)
					}
				}))
				defer server.Close()
				client, err := NewClient(context.Background(), &ClientConfig{Backend: backend, APIKey: "test-key", HTTPClient: server.Client(), HTTPOptions: HTTPOptions{BaseURL: server.URL}, envVarProvider: func() map[string]string { return map[string]string{} }})
				if err != nil {
					t.Fatal(err)
				}
				var first *GenerateContentResponse
				// Act: receive a tool call and send its content back on the next turn.
				contents := Text("lookup")
				for turn := 0; turn < 2; turn++ {
					var got *GenerateContentResponse
					if stream {
						for item, err := range client.Models.GenerateContentStream(context.Background(), "gemini-test", contents, nil) {
							if err != nil {
								t.Fatal(err)
							}
							got = item
						}
					} else {
						got, err = client.Models.GenerateContent(context.Background(), "gemini-test", contents, nil)
						if err != nil {
							t.Fatal(err)
						}
					}
					if got == nil {
						t.Fatal("missing response")
					}
					if turn == 0 {
						first = got
						contents = []*Content{first.Candidates[0].Content}
					}
				}
				// Assert: decoded args and the next request retain the original numbers.
				assertPrecisionArgs(t, first.FunctionCalls()[0].Args)
				if first.UsageMetadata.PromptTokenCount != 42 {
					t.Fatal("typed usage changed")
				}
				if string(first.Candidates[0].Content.Parts[0].ThoughtSignature) != "\x01\x02\x03" {
					t.Fatal("signature changed")
				}
				<-requests
				body := <-requests
				for _, number := range []string{"9007199254740993", "1.234567890123456789", "-9007199254740993"} {
					if !strings.Contains(string(body), number) {
						t.Errorf("next request lost %s: %s", number, body)
					}
				}
			})
		}
	}
}

func assertPrecisionArgs(t *testing.T, args map[string]any) {
	t.Helper()
	got, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	var expected, actual any
	decode := func(s string, out *any) {
		d := json.NewDecoder(strings.NewReader(s))
		d.UseNumber()
		if err := d.Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	decode(precisionArgsJSON, &expected)
	decode(string(got), &actual)
	wantBytes, _ := json.Marshal(expected)
	gotBytes, _ := json.Marshal(actual)
	if string(wantBytes) != string(gotBytes) {
		t.Errorf("args = %s, want %s", gotBytes, wantBytes)
	}
}

func TestFunctionCallNumericPrecisionInvalidJSON(t *testing.T) {
	for _, input := range []string{``, `{"args":{"id":1}} {}`, `{"args":{"id":1}} garbage`, `{"args":{"id":01}}`, `{"args":{"id":NaN}}`, `{"args":[1]}`} {
		t.Run(input, func(t *testing.T) {
			// Arrange.
			var call FunctionCall
			// Act.
			err := json.Unmarshal([]byte(input), &call)
			// Assert.
			if err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestDeepMarshalNumericPrecision(t *testing.T) {
	// Arrange.
	call := FunctionCall{Name: "lookup", Args: map[string]any{"id": int64(9007199254740993), "decimal": json.Number("1.234567890123456789")}}
	var result map[string]any
	// Act.
	if err := deepMarshal(call, &result); err != nil {
		t.Fatal(err)
	}
	// Assert.
	got, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"args":{"decimal":1.234567890123456789,"id":9007199254740993},"name":"lookup"}` {
		t.Fatalf("request values changed: %s", got)
	}
}

func TestUnmarshalJSONWithNumbersRejectsTrailingData(t *testing.T) {
	for _, input := range []string{``, `{} {}`, `{} garbage`, `{"id":01}`} {
		t.Run(input, func(t *testing.T) {
			// Arrange.
			var got map[string]any
			var want json.RawMessage
			wantErr := json.Unmarshal([]byte(input), &want)
			// Act.
			err := unmarshalJSONWithNumbers([]byte(input), &got)
			// Assert.
			if err == nil || wantErr == nil || err.Error() != wantErr.Error() {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestNumericQueryAfterDeepMarshal(t *testing.T) {
	// Arrange: pagination/config maps are also built through deepMarshal.
	var query map[string]any
	if err := deepMarshal(map[string]any{"pageSize": 42}, &query); err != nil {
		t.Fatal(err)
	}
	// Act.
	got, err := createURLQuery(query)
	// Assert.
	if err != nil {
		t.Fatal(err)
	}
	if got != "pageSize=42" {
		t.Fatalf("query = %q, want pageSize=42", got)
	}
}
