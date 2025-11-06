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

package genai

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewPage(t *testing.T) {
	ctx := context.Background()
	name := "test"
	config := make(map[string]any)
	listFunc := func(ctx context.Context, config map[string]any) ([]*string, string, *HTTPResponse, error) {
		return []*string{Ptr("item1"), Ptr("item2")}, "next_page_token", nil, nil
	}

	page, err := newPage[string](ctx, name, config, listFunc)
	if err != nil {
		t.Fatalf("newPage failed: %v", err)
	}

	if page.Name != name {
		t.Errorf("Name mismatch: got %q, want %q", page.Name, name)
	}
	if diff := cmp.Diff(page.Items, []*string{Ptr("item1"), Ptr("item2")}); diff != "" {
		t.Errorf("Items mismatch (-want +got):\n%s", diff)
	}

	if page.NextPageToken != "next_page_token" {
		t.Errorf("nextPageToken mismatch: got %q, want %q", page.NextPageToken, "next_page_token")
	}

	listFuncError := func(ctx context.Context, config map[string]any) ([]*string, string, *HTTPResponse, error) {
		return nil, "", nil, errors.New("list func error")
	}
	_, err = newPage[string](ctx, name, config, listFuncError)
	if err == nil {
		t.Fatal("newPage should return an error")
	}
}

func TestPageNext(t *testing.T) {
	ctx := context.Background()
	config := make(map[string]any)
	listFunc := func(ctx context.Context, config map[string]any) ([]*string, string, *HTTPResponse, error) {
		if config["PageToken"] == "next_page_token" {
			return []*string{Ptr("item3"), Ptr("item4")}, "", nil, nil
		}
		return []*string{Ptr("item1"), Ptr("item2")}, "next_page_token", nil, nil
	}

	page, err := newPage[string](ctx, "test", config, listFunc)
	if err != nil {
		t.Fatalf("newPage failed: %v", err)
	}

	nextPage, err := page.Next(ctx)
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if diff := cmp.Diff(nextPage.Items, []*string{Ptr("item3"), Ptr("item4")}); diff != "" {
		t.Errorf("Items mismatch (-want +got):\n%s", diff)
	}

	_, err = nextPage.Next(ctx)
	if !errors.Is(err, ErrPageDone) {
		t.Errorf("Expected PageDone error, got %v", err)
	}
}

func TestPageAll(t *testing.T) {
	ctx := context.Background()
	config := map[string]any{}
	listFunc := func(ctx context.Context, config map[string]any) ([]*string, string, *HTTPResponse, error) {
		if config["PageToken"] == "next_page_token" {
			return []*string{Ptr("item3"), Ptr("item4")}, "", nil, nil
		}
		return []*string{Ptr("item1"), Ptr("item2")}, "next_page_token", nil, nil
	}

	page, err := newPage[string](ctx, "test", config, listFunc)
	if err != nil {
		t.Fatalf("newPage failed: %v", err)
	}

	allItems := []string{}
	for item, err := range page.all(ctx) {
		if err != nil {
			if errors.Is(err, ErrPageDone) {
				break // Expected PageDone at the end of iteration.
			}
			t.Fatalf("Unexpected error during iteration: %v", err)
		}
		allItems = append(allItems, *item)

	}

	wantItems := []string{"item1", "item2", "item3", "item4"}
	if diff := cmp.Diff(allItems, wantItems); diff != "" {
		t.Errorf("Items mismatch (-want, +got):\n%s", diff)
	}

	// Test error handling within the iterator.
	listFuncError := func(ctx context.Context, config map[string]any) ([]*string, string, *HTTPResponse, error) {
		if config["PageToken"] == "next_page_token" {
			return nil, "", nil, errors.New("list func error")
		}
		return []*string{Ptr("item1"), Ptr("item2")}, "next_page_token", nil, nil
	}
	page, err = newPage[string](ctx, "test", config, listFuncError)
	if err != nil {
		t.Fatalf("newPage failed: %v", err)
	}

	for _, err := range page.all(ctx) {
		if err != nil {
			if err.Error() == "list func error" {
				return // Expected error.
			}

			t.Fatalf("Unexpected error during iteration: %v", err)
		}

	}

}
