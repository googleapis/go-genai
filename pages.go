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
	"context"
	"fmt"
	"io"
	"iter"
	"reflect"
)

// PageDone is the error returned by Next when no more page is available.
var PageDone = errors.New("PageDone")

type Page[T any, C any] struct {
	Name  string // The name of the paged items.
	Items []*T   // The items in the current page.

	config        C                                                          // The configuration used for the API call.
	listFunc      func(ctx context.Context, config *C) ([]*T, string, error) // The function used to retrieve the next page.
	nextPageToken string                                                     // The token to use to retrieve the next page of results.
}

func newPage[T any, C any](ctx context.Context, name string, config C, listFunc func(ctx context.Context, config *C) ([]*T, string, error)) (page[T, C], error) {
	p := Page[T, C]{
		Name:     name,
		config:   config,
		listFunc: listFunc,
	}
	items, nextPageToken, err := listFunc(ctx, &config)
	if err != nil {
		return p, err
	}
	p.Items = items
	p.nextPageToken = nextPageToken
	return p, nil
}

// all returns an iterator that yields all items across all pages of results.
//
// The iterator retrieves each page sequentially and yields each item within
// the page.  If an error occurs during retrieval, the iterator will stop
// and the error will be returned as the second value in the next call to Next().
// An io.EOF error indicates that all pages have been processed.
func (p page[T, C]) all(ctx context.Context) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for {
			for _, item := range p.Items {
				if !yield(item, nil) {
					return
				}
			}
			var err error
			p, err = p.next(ctx)
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(nil, err)
				return
			}
		}
	}
}

// next retrieves the next page of results.
//
// If there are no more pages, PageDone is returned.  Otherwise,
// a new Page struct containing the next set of results is returned.
// Any other errors encountered during retrieval will also be returned.
func (p page[T, C]) Next(ctx context.Context) (page[T, C], error) {
	if p.nextPageToken == "" {
		return p, PageDone
	}
	configPtr := reflect.ValueOf(&p.config) // Note the & operator
	configValue := configPtr.Elem()
	pageTokenField := configValue.FieldByName("PageToken")
	if !pageTokenField.IsValid() || !pageTokenField.CanSet() {
		return p, fmt.Errorf("pageToken field is invalid or not settable")
	}
	pageTokenField.SetString(p.nextPageToken)

	return newPage[T, C](ctx, p.Name, p.config, p.listFunc)
}
