# Changelog

## [0.2.0](https://github.com/googleapis/go-genai/compare/v0.1.0...v0.2.0) (2025-02-05)


### Features

* Add enhanced_prompt to GeneratedImage class ([449f0fb](https://github.com/googleapis/go-genai/commit/449f0fbc1f57b5ce5e20eef587f67f2d0d93a889))
* Add labels for GenerateContent requests ([98231e5](https://github.com/googleapis/go-genai/commit/98231e5e7fa2483004841b50ceee841078e6d951))


### Bug Fixes

* remove unsupported parameter from Gemini API ([39c8868](https://github.com/googleapis/go-genai/commit/39c88682acbf554bad4d7a8ca92a854a7005052a))
* Use camel case for Go function parameters ([94765e6](https://github.com/googleapis/go-genai/commit/94765e68aef1258054711cc601e070e4ef7c80e5))

## [0.1.0](https://github.com/googleapis/go-genai/compare/v0.0.1...v0.1.0) (2025-01-29)


### ⚠ BREAKING CHANGES

* Make some numeric fields to pointer type and bool fields to value type, and rename ControlReferenceTypeControlType* constants

### Features

* [genai-modules][models] Add HttpOptions to all method configs for models. ([765c9b7](https://github.com/googleapis/go-genai/commit/765c9b7311884554c352ec00a0253c2cbbbf665c))
* Add Imagen generate_image support for Go SDK ([068fe54](https://github.com/googleapis/go-genai/commit/068fe541801ced806714662af023a481271402c4))
* Add support for audio_timestamp to types.GenerateContentConfig (fixes [#132](https://github.com/googleapis/go-genai/issues/132)) ([cfede62](https://github.com/googleapis/go-genai/commit/cfede6255a13b4977450f65df80b576342f44b5a))
* Add support for enhance_prompt to model.generate_image ([a35f52a](https://github.com/googleapis/go-genai/commit/a35f52a318a874935a1e615dbaa24bb91625c5de))
* Add ThinkingConfig to generate content config. ([ad73778](https://github.com/googleapis/go-genai/commit/ad73778cf6f1c6d9b240cf73fce52b87ae70378f))
* enable Text() and FunctionCalls() quick accessor for GenerateContentResponse ([3f3a450](https://github.com/googleapis/go-genai/commit/3f3a450954283fa689c9c19a29b0487c177f7aeb))
* Images - Added Image.mime_type ([3333511](https://github.com/googleapis/go-genai/commit/3333511a656b796065cafff72168c112c74de293))
* introducing HTTPOptions to Client ([e3d1d8e](https://github.com/googleapis/go-genai/commit/e3d1d8e6aa0cbbb3f2950c571f5c0a70b7ce8656))
* make Part, FunctionDeclaration, Image, and GenerateContentResponse classmethods argument keyword only ([f7d1043](https://github.com/googleapis/go-genai/commit/f7d1043bb791930d82865a11b83fea785e313922))
* Make some numeric fields to pointer type and bool fields to value type, and rename ControlReferenceTypeControlType* constants ([ee4e5a4](https://github.com/googleapis/go-genai/commit/ee4e5a414640226e9b685a7d67673992f2c63dee))
* support caches create/update/get/update in Go SDK ([0620d97](https://github.com/googleapis/go-genai/commit/0620d97e32b3e535edab8f3f470e08746ace4d60))
* support usability constructor functions for Part struct ([831b879](https://github.com/googleapis/go-genai/commit/831b879ea15a82506299152e9f790f34bbe511f9))


### Miscellaneous Chores

* Released as 0.1.0 ([e046125](https://github.com/googleapis/go-genai/commit/e046125c8b378b5acb05e64ed46c4aac51dd9456))


### Code Refactoring

* rename GenerateImage() to GenerateImage(), rename GenerateImageConfig to GenerateImagesConfig, rename GenerateImageResponse to GenerateImagesResponse, rename GenerateImageParameters to GenerateImagesParameters ([ebb231f](https://github.com/googleapis/go-genai/commit/ebb231f0c86bb30f013301e26c562ccee8380ee0))

## 0.0.1 (2025-01-10)


### Features

* enable response_logprobs and logprobs for Google AI ([#17](https://github.com/googleapis/go-genai/issues/17)) ([51f2744](https://github.com/googleapis/go-genai/commit/51f274411ea770fa8fc16ce316085310875e5d68))
* Go SDK Live module implementation for GoogleAI backend ([f88e65a](https://github.com/googleapis/go-genai/commit/f88e65a7f8fda789b0de5ecc4e2ed9d2bd02cc89))
* Go SDK Live module initial implementation for VertexAI. ([4d82dc0](https://github.com/googleapis/go-genai/commit/4d82dc0c478151221d31c0e3ccde9ac215f2caf2))


### Bug Fixes

* change string type to numeric types ([bfdc94f](https://github.com/googleapis/go-genai/commit/bfdc94fd1b38fb61976f0386eb73e486cc3bc0f8))
* fix README typo ([5ae8aa6](https://github.com/googleapis/go-genai/commit/5ae8aa6deec520f33d1746be411ed55b2b10d74f))
