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

package operations

import (
	"encoding/json"
	"io"

	"google.golang.org/genai/interactions/internal/utils"
	"google.golang.org/genai/interactions/models/components"
	"google.golang.org/genai/interactions/models/environments"
)

type UploadEnvironmentFileGlobals struct {
	// Which version of the API to use.
	APIVersion *string `pathParam:"style=simple,explode=false,name=api_version"`
}

func (g *UploadEnvironmentFileGlobals) GetAPIVersion() *string {
	if g == nil {
		return nil
	}
	return g.APIVersion
}

// UploadEnvironmentFileRequest represents a request to upload a file or archive to an environment workspace.
type UploadEnvironmentFileRequest struct {
	// Which version of the API to use.
	APIVersion *string `pathParam:"style=simple,explode=false,name=api_version"`
	// The environment ID or resource name.
	Environment string `pathParam:"style=simple,explode=false,name=environment"`
	// The relative destination path within the environment workspace.
	Path string `pathParam:"style=simple,explode=false,name=path"`
	// In-memory binary file content to upload.
	Content []byte
	// A readable stream of binary data to upload.
	Reader io.Reader
	// Path to a local file to upload.
	FilePath *string
	// Content length in bytes. Inferred automatically when Content or FilePath is provided.
	SizeBytes *int64
	// The MIME type of the file. Inferred if not provided.
	MimeType *string
	// Whether to overwrite existing files. Defaults to true.
	Overwrite *bool `queryParam:"style=form,explode=true,name=overwrite"`
	// If true and uploading a zip/tar archive, extracts the archive contents into the destination directory. Defaults to false.
	Extract *bool `queryParam:"style=form,explode=true,name=extract"`
}

func (u *UploadEnvironmentFileRequest) GetAPIVersion() *string {
	if u == nil {
		return nil
	}
	return u.APIVersion
}

func (u *UploadEnvironmentFileRequest) GetEnvironment() string {
	if u == nil {
		return ""
	}
	return u.Environment
}

func (u *UploadEnvironmentFileRequest) GetPath() string {
	if u == nil {
		return ""
	}
	return u.Path
}

func (u *UploadEnvironmentFileRequest) GetContent() []byte {
	if u == nil {
		return nil
	}
	return u.Content
}

func (u *UploadEnvironmentFileRequest) GetReader() io.Reader {
	if u == nil {
		return nil
	}
	return u.Reader
}

func (u *UploadEnvironmentFileRequest) GetFilePath() *string {
	if u == nil {
		return nil
	}
	return u.FilePath
}

func (u *UploadEnvironmentFileRequest) GetSizeBytes() *int64 {
	if u == nil {
		return nil
	}
	return u.SizeBytes
}

func (u *UploadEnvironmentFileRequest) GetMimeType() *string {
	if u == nil {
		return nil
	}
	return u.MimeType
}

func (u *UploadEnvironmentFileRequest) GetOverwrite() *bool {
	if u == nil {
		return nil
	}
	return u.Overwrite
}

func (u *UploadEnvironmentFileRequest) GetExtract() *bool {
	if u == nil {
		return nil
	}
	return u.Extract
}

// UploadEnvironmentFileResponse represents the response returned after uploading a file.
type UploadEnvironmentFileResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Extracted or uploaded files metadata.
	Files *environments.GetEnvironmentFilesResponse `json:"files,omitzero"`
}

func (u UploadEnvironmentFileResponse) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(u, "", false)
}

func (u *UploadEnvironmentFileResponse) UnmarshalJSON(data []byte) error {
	var probe struct {
		Files []any           `json:"files"`
		File  json.RawMessage `json:"file"`
	}
	if err := json.Unmarshal(data, &probe); err == nil {
		if probe.Files != nil {
			var getEnvFiles environments.GetEnvironmentFilesResponse
			if err := utils.UnmarshalJSON(data, &getEnvFiles, "", false, nil); err != nil {
				return err
			}
			u.Files = &getEnvFiles
			return nil
		}
		if len(probe.File) > 0 {
			var envFile environments.EnvironmentFile
			if err := utils.UnmarshalJSON(probe.File, &envFile, "", false, nil); err != nil {
				return err
			}
			u.Files = &environments.GetEnvironmentFilesResponse{
				Files: []environments.EnvironmentFile{envFile},
			}
			return nil
		}
	}

	var envFile environments.EnvironmentFile
	if err := utils.UnmarshalJSON(data, &envFile, "", false, nil); err != nil {
		return err
	}
	u.Files = &environments.GetEnvironmentFilesResponse{
		Files: []environments.EnvironmentFile{envFile},
	}
	return nil
}

func (u *UploadEnvironmentFileResponse) GetHTTPMeta() components.HTTPMetadata {
	if u == nil {
		return components.HTTPMetadata{}
	}
	return u.HTTPMeta
}

func (u *UploadEnvironmentFileResponse) GetFiles() *environments.GetEnvironmentFilesResponse {
	if u == nil {
		return nil
	}
	return u.Files
}
