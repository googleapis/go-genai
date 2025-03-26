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

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
)

type apiClient struct {
	clientConfig *ClientConfig
}

// sendStreamRequest issues an server streaming API request and returns a map of the response contents.
func sendStreamRequest[T responseStream[R], R any](ctx context.Context, ac *apiClient, path string, method string, body map[string]any, httpOptions *HTTPOptions, output *responseStream[R]) error {
	req, err := buildRequest(ctx, ac, path, body, method, httpOptions)
	if err != nil {
		return err
	}

	resp, err := doRequest(ac, req)
	if err != nil {
		return err
	}

	// resp.Body will be closed by the iterator
	return deserializeStreamResponse(resp, output)
}

// sendRequest issues an API request and returns a map of the response contents.
func sendRequest(ctx context.Context, ac *apiClient, path string, method string, body map[string]any, httpOptions *HTTPOptions) (map[string]any, error) {
	req, err := buildRequest(ctx, ac, path, body, method, httpOptions)
	if err != nil {
		return nil, err
	}

	resp, err := doRequest(ac, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return deserializeUnaryResponse(resp)
}

func mapToStruct[R any](input map[string]any, output *R) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(input)
	if err != nil {
		return fmt.Errorf("mapToStruct: error encoding input %#v: %w", input, err)
	}
	err = json.Unmarshal(b.Bytes(), output)
	if err != nil {
		return fmt.Errorf("mapToStruct: error unmarshalling input %#v: %w", input, err)
	}
	return nil
}

func (ac *apiClient) createAPIURL(suffix string, httpOptions *HTTPOptions) (*url.URL, error) {
	if ac.clientConfig.Backend == BackendVertexAI {
		if !strings.HasPrefix(suffix, "projects/") {
			suffix = fmt.Sprintf("projects/%s/locations/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, suffix)
		}
		u, err := url.Parse(fmt.Sprintf("%s/%s/%s", httpOptions.BaseURL, httpOptions.APIVersion, suffix))
		if err != nil {
			return nil, fmt.Errorf("createAPIURL: error parsing Vertex AI URL: %w", err)
		}
		return u, nil
	} else {
		u, err := url.Parse(fmt.Sprintf("%s/%s/%s", httpOptions.BaseURL, httpOptions.APIVersion, suffix))
		if err != nil {
			return nil, fmt.Errorf("createAPIURL: error parsing ML Dev URL: %w", err)
		}
		return u, nil
	}
}

func buildRequest(ctx context.Context, ac *apiClient, path string, body map[string]any, method string, httpOptions *HTTPOptions) (*http.Request, error) {
	url, err := ac.createAPIURL(path, httpOptions)
	if err != nil {
		return nil, err
	}
	b := new(bytes.Buffer)
	if len(body) > 0 {
		if err := json.NewEncoder(b).Encode(body); err != nil {
			return nil, fmt.Errorf("buildRequest: error encoding body %#v: %w", body, err)
		}
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url.String(), b)
	if err != nil {
		return nil, err
	}
	// Set headers
	doMergeHeaders(httpOptions.Headers, &req.Header)
	doMergeHeaders(sdkHeader(ac), &req.Header)
	return req, nil
}

func sdkHeader(ac *apiClient) http.Header {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	if ac.clientConfig.APIKey != "" {
		header.Set("x-goog-api-key", ac.clientConfig.APIKey)
	}
	// TODO(b/381108714): Automate revisions to the SDK library version.
	libraryLabel := "google-genai-sdk/0.0.1"
	languageLabel := fmt.Sprintf("gl-go/%s", runtime.Version())
	versionHeaderValue := fmt.Sprintf("%s %s", libraryLabel, languageLabel)
	header.Set("user-agent", versionHeaderValue)
	header.Set("x-goog-api-client", versionHeaderValue)
	return header
}

func doRequest(ac *apiClient, req *http.Request) (*http.Response, error) {
	// Create a new HTTP client and send the request
	client := ac.clientConfig.HTTPClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doRequest: error sending request: %w", err)
	}
	return resp, nil
}

func deserializeUnaryResponse(resp *http.Response) (map[string]any, error) {
	if !httpStatusOk(resp) {
		return nil, newAPIError(resp)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	output := make(map[string]any)
	err = json.Unmarshal(respBody, &output)
	if err != nil {
		return nil, fmt.Errorf("deserializeUnaryResponse: error unmarshalling response: %w\n%s", err, respBody)
	}
	return output, nil
}

type responseStream[R any] struct {
	r  *bufio.Scanner
	rc io.ReadCloser
}

func iterateResponseStream[R any](rs *responseStream[R], responseConverter func(responseMap map[string]any) (*R, error)) iter.Seq2[*R, error] {
	return func(yield func(*R, error) bool) {
		defer func() {
			// Close the response body range over function is done.
			if err := rs.rc.Close(); err != nil {
				log.Printf("Error closing response body: %v", err)
			}
		}()
		for rs.r.Scan() {
			line := rs.r.Bytes()
			if len(line) == 0 {
				continue
			}
			prefix, data, _ := bytes.Cut(line, []byte(":"))
			switch string(prefix) {
			case "data":
				// Step 1: Unmarshal the JSON into a map[string]any so that we can call fromConverter
				// in Step 2.
				respRaw := make(map[string]any)
				if err := json.Unmarshal(data, &respRaw); err != nil {
					err = fmt.Errorf("iterateResponseStream: error unmarshalling data %s:%s. error: %w", string(prefix), string(data), err)
					if !yield(nil, err) {
						return
					}
				}
				// Step 2: The toStruct function calls fromConverter(handle Vertex and MLDev schema
				// difference and get a unified response). Then toStruct function converts the unified
				// response from map[string]any to struct type.
				// var resp = new(R)
				resp, err := responseConverter(respRaw)
				if err != nil {
					if !yield(nil, err) {
						return
					}
				}

				// Step 3: yield the response.
				if !yield(resp, nil) {
					return
				}
			default:
				// Stream chunk not started with "data" is treated as an error.
				if !yield(nil, fmt.Errorf("iterateResponseStream: invalid stream chunk: %s:%s", string(prefix), string(data))) {
					return
				}
			}
		}
		if rs.r.Err() != nil {
			if rs.r.Err() == bufio.ErrTooLong {
				log.Printf("The response is too large to process in streaming mode. Please use a non-streaming method.")
			}
			log.Printf("Error %v", rs.r.Err())
		}
	}
}

// APIError contains an error response from the server.
type APIError struct {
	// Code is the HTTP response status code.
	Code int `json:"code,omitempty"`
	// Message is the server response message.
	Message string `json:"message,omitempty"`
	// Status is the server response status.
	Status string `json:"status,omitempty"`
	// Details provide more context to an error.
	Details []map[string]any `json:"details,omitempty"`
}

type responseWithError struct {
	ErrorInfo *APIError `json:"error,omitempty"`
}

func newAPIError(resp *http.Response) error {
	var respWithError = new(responseWithError)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("newAPIError: error reading response body: %w. Response: %v", err, string(body))
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, respWithError); err != nil {
			return fmt.Errorf("newAPIError: unmarshal response to error failed: %w. Response: %v", err, string(body))
		}
		return *respWithError.ErrorInfo
	}
	return APIError{Code: resp.StatusCode, Status: resp.Status}
}

// Error returns a string representation of the ServerError.
func (e APIError) Error() string {
	return fmt.Sprintf(
		"Error %d, Message: %s, Status: %s, Details: %v",
		e.Code, e.Message, e.Status, e.Details,
	)
}

func httpStatusOk(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func deserializeStreamResponse[T responseStream[R], R any](resp *http.Response, output *responseStream[R]) error {
	if !httpStatusOk(resp) {
		return newAPIError(resp)
	}
	output.r = bufio.NewScanner(resp.Body)
	// Scanner default buffer max size is 64*1024 (64KB).
	// We provide 1KB byte buffer to the scanner and set max to 256MB.
	// When data exceed 1KB, then scanner will allocate new memory up to 256MB.
	// When data exceed 256MB, scanner will stop and returns err: bufio.ErrTooLong.
	output.r.Buffer(make([]byte, 1024), 268435456)

	output.r.Split(scan)
	output.rc = resp.Body
	return nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func scan(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// Look for two consecutive newlines in the data
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		// We have a full two-newline-terminated token.
		return i + 2, dropCR(data[0:i]), nil
	}

	// Handle the case of Windows-style newlines (\r\n\r\n)
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		// We have a full Windows-style two-newline-terminated token.
		return i + 4, dropCR(data[0:i]), nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}
