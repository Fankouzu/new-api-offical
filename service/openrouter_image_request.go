package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	OpenRouterImageGenerationMode = "generation"
	OpenRouterImageEditMode       = "edit"
)

var openRouterImageEditModels = map[string]struct{}{
	"Qwen-Image-Edit":    {},
	"Qwen-Image-Edit-2k": {},
}

type OpenRouterImageRequest struct {
	Model             string                          `json:"model"`
	Prompt            string                          `json:"prompt"`
	N                 *uint                           `json:"n,omitempty"`
	Size              string                          `json:"size,omitempty"`
	Resolution        string                          `json:"resolution,omitempty"`
	AspectRatio       string                          `json:"aspect_ratio,omitempty"`
	Quality           string                          `json:"quality,omitempty"`
	OutputFormat      json.RawMessage                 `json:"output_format,omitempty"`
	Background        json.RawMessage                 `json:"background,omitempty"`
	OutputCompression json.RawMessage                 `json:"output_compression,omitempty"`
	Seed              json.RawMessage                 `json:"seed,omitempty"`
	Stream            *bool                           `json:"stream,omitempty"`
	InputReferences   []OpenRouterImageInputReference `json:"input_references,omitempty"`
	Provider          *OpenRouterImageProviderOptions `json:"provider,omitempty"`
	Extra             map[string]json.RawMessage      `json:"-"`
}

type OpenRouterImageInputReference struct {
	ImageURL *OpenRouterImageURL `json:"image_url,omitempty"`
	URL      string              `json:"url,omitempty"`
	Type     string              `json:"type,omitempty"`
}

type OpenRouterImageURL struct {
	URL string `json:"url"`
}

type OpenRouterImageProviderOptions struct {
	Options map[string]any `json:"options,omitempty"`
}

func (r *OpenRouterImageRequest) UnmarshalJSON(data []byte) error {
	type Alias OpenRouterImageRequest
	var known Alias
	if err := common.Unmarshal(data, &known); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := common.Unmarshal(data, &raw); err != nil {
		return err
	}
	known.Extra = make(map[string]json.RawMessage)
	knownFields := map[string]struct{}{
		"model":              {},
		"prompt":             {},
		"n":                  {},
		"size":               {},
		"resolution":         {},
		"aspect_ratio":       {},
		"quality":            {},
		"output_format":      {},
		"background":         {},
		"output_compression": {},
		"seed":               {},
		"stream":             {},
		"input_references":   {},
		"provider":           {},
	}
	for key, value := range raw {
		if _, ok := knownFields[key]; !ok {
			known.Extra[key] = value
		}
	}
	*r = OpenRouterImageRequest(known)
	return nil
}

func ConvertOpenRouterImageRequest(req OpenRouterImageRequest) (*dto.ImageRequest, string, error) {
	req.Model = strings.TrimSpace(req.Model)
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Model == "" {
		return nil, "", errors.New("model is required")
	}
	if req.Prompt == "" {
		return nil, "", errors.New("prompt is required")
	}
	if req.N != nil && *req.N > 1 {
		return nil, "", errors.New("OpenRouter image provider endpoint supports n=1 only")
	}
	if req.Stream != nil && *req.Stream {
		return nil, "", errors.New("OpenRouter image streaming is not supported for these models")
	}

	imageReq := &dto.ImageRequest{
		Model:             req.Model,
		Prompt:            req.Prompt,
		N:                 common.GetPointer(uint(1)),
		Size:              normalizeOpenRouterSize(req.Model, req.Size, req.Resolution, req.AspectRatio),
		Quality:           req.Quality,
		OutputFormat:      req.OutputFormat,
		Background:        req.Background,
		OutputCompression: req.OutputCompression,
		Stream:            req.Stream,
		Extra:             cloneRawMessageMap(req.Extra),
	}
	if len(req.Seed) > 0 {
		imageReq.Extra["seed"] = cloneRawMessage(req.Seed)
	}
	mergeOpenRouterProviderOptions(imageReq.Extra, req.Provider)
	mergeOpenRouterExtraFields(imageReq, req)

	if isOpenRouterEditImageModel(req.Model) {
		imageURL := firstOpenRouterInputReferenceURL(req.InputReferences)
		if imageURL == "" {
			return nil, "", errors.New("input_references with image_url.url is required for edit image models")
		}
		raw, err := common.Marshal(imageURL)
		if err != nil {
			return nil, "", fmt.Errorf("marshal image reference: %w", err)
		}
		imageReq.Image = raw
		return imageReq, OpenRouterImageEditMode, nil
	}
	return imageReq, OpenRouterImageGenerationMode, nil
}

func isOpenRouterEditImageModel(model string) bool {
	_, ok := openRouterImageEditModels[model]
	return ok
}

func firstOpenRouterInputReferenceURL(refs []OpenRouterImageInputReference) string {
	for _, ref := range refs {
		if ref.ImageURL != nil && strings.TrimSpace(ref.ImageURL.URL) != "" {
			return strings.TrimSpace(ref.ImageURL.URL)
		}
		if strings.TrimSpace(ref.URL) != "" {
			return strings.TrimSpace(ref.URL)
		}
	}
	return ""
}

func sizeFromOpenRouterDimensions(model, resolution, aspectRatio string) string {
	model = strings.TrimSpace(strings.ToLower(model))
	resolution = strings.TrimSpace(strings.ToLower(resolution))
	aspectRatio = strings.TrimSpace(aspectRatio)
	if resolution == "" && aspectRatio == "" {
		return ""
	}

	maxSide := 1024
	if strings.Contains(model, "2k") || strings.Contains(resolution, "2k") || strings.Contains(resolution, "2048") {
		maxSide = 2048
	}

	switch aspectRatio {
	case "", "1:1":
		return fmt.Sprintf("%dx%d", maxSide, maxSide)
	case "16:9":
		if maxSide == 2048 {
			return "1536x864"
		}
		return "1024x576"
	case "9:16":
		if maxSide == 2048 {
			return "864x1536"
		}
		return "576x1024"
	case "4:3":
		if maxSide == 2048 {
			return "1536x1152"
		}
		return "1024x768"
	case "3:4":
		if maxSide == 2048 {
			return "1152x1536"
		}
		return "768x1024"
	default:
		return ""
	}
}

func mergeOpenRouterExtraFields(imageReq *dto.ImageRequest, req OpenRouterImageRequest) {
	if imageReq.Extra == nil {
		imageReq.Extra = map[string]json.RawMessage{}
	}

	parameters := map[string]any{
		"size": imageReq.Size,
		"n":    1,
	}
	if len(req.Seed) > 0 {
		var seed any
		if err := common.Unmarshal(req.Seed, &seed); err == nil {
			parameters["seed"] = seed
		}
	}
	for key, value := range imageReq.Extra {
		var decoded any
		if err := common.Unmarshal(value, &decoded); err == nil {
			parameters[key] = decoded
		}
	}

	input := map[string]any{
		"prompt": req.Prompt,
	}
	if raw, ok := imageReq.Extra["negative_prompt"]; ok {
		var negativePrompt any
		if err := common.Unmarshal(raw, &negativePrompt); err == nil {
			input["negative_prompt"] = negativePrompt
			delete(parameters, "negative_prompt")
		}
	}
	if isOpenRouterEditImageModel(req.Model) {
		if imageURL := firstOpenRouterInputReferenceURL(req.InputReferences); imageURL != "" {
			input["messages"] = []map[string]any{{
				"role": "user",
				"content": []map[string]any{
					{"image": imageURL},
					{"text": req.Prompt},
				},
			}}
		}
	}

	extraFields := map[string]any{
		"parameters": parameters,
		"input":      input,
	}
	raw, err := common.Marshal(extraFields)
	if err == nil && len(bytes.TrimSpace(raw)) > 0 {
		imageReq.ExtraFields = json.RawMessage(raw)
	}
}

func mergeOpenRouterProviderOptions(target map[string]json.RawMessage, provider *OpenRouterImageProviderOptions) {
	if provider == nil || len(provider.Options) == 0 {
		return
	}
	raw, ok := provider.Options[OpenRouterProviderSlug()]
	if !ok {
		return
	}
	options, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for key, value := range options {
		if strings.TrimSpace(key) == "" {
			continue
		}
		jsonData, err := common.Marshal(value)
		if err != nil {
			continue
		}
		target[key] = json.RawMessage(jsonData)
	}
}

func cloneRawMessageMap(source map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(source))
	for key, value := range source {
		result[key] = cloneRawMessage(value)
	}
	return result
}

func normalizeOpenRouterSize(model, size, resolution, aspectRatio string) string {
	size = strings.TrimSpace(size)
	if strings.Contains(strings.ToLower(size), "x") {
		return size
	}
	if size != "" {
		resolution = size
	}
	return sizeFromOpenRouterDimensions(model, resolution, aspectRatio)
}

func cloneRawMessage(source json.RawMessage) json.RawMessage {
	if len(source) == 0 {
		return nil
	}
	result := make([]byte, len(source))
	copy(result, source)
	return result
}
