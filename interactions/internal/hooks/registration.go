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

package hooks

func initHooks(h *Hooks) {
	// Hooks are registered per SDK instance, and are valid for the lifetime of the SDK instance.
	// Add any hooks you wish to add here. Feel free to define your hooks in this file or in
	// separate files in the hooks package.
	//
	// The following methods are available for registering hooks:
	_ = h.registerSDKInitHook
	_ = h.registerBeforeRequestHook
	_ = h.registerAfterSuccessHook
	_ = h.registerAfterErrorHook

	h.registerBeforeRequestHook(&GoogleGenAIAuthHook{})
}
