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

package interactions

import (
	"io"
	"time"
)

func Int(v int64) *int64          { return &v }
func Bool(v bool) *bool           { return &v }
func Float(v float64) *float64    { return &v }
func String(v string) *string     { return &v }
func Time(v time.Time) *time.Time { return &v }

func File(rdr io.Reader, filename string, contentType string) file {
	return file{rdr, filename, contentType}
}

type file struct {
	io.Reader
	name        string
	contentType string
}

func (f file) Filename() string {
	if f.name != "" {
		return f.name
	} else if named, ok := f.Reader.(interface{ Name() string }); ok {
		return named.Name()
	}
	return ""
}

func (f file) ContentType() string {
	return f.contentType
}
