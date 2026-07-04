package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

	size, err := normalizeOpenRouterSize(req.Model, req.Size, req.Resolution, req.AspectRatio)
	if err != nil {
		return nil, "", err
	}

	imageReq := &dto.ImageRequest{
		Model:             req.Model,
		Prompt:            req.Prompt,
		Size:              size,
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

	if isOpenRouterEditImageModel(req.Model) {
		imageURL := firstOpenRouterInputReferenceURL(req.InputReferences)
		if imageURL == "" {
			return nil, "", errors.New("input_references with image_url.url is required for edit image models")
		}
		raw, err := common.Marshal(imageURL)
		if err != nil {
			return nil, "", fmt.Errorf("marshal image reference: %w", err)
		}
		imageReq.Extra["image_url"] = raw
		mergeOpenRouterExtraFields(imageReq, req)
		return imageReq, OpenRouterImageEditMode, nil
	}
	mergeOpenRouterExtraFields(imageReq, req)
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
	if resolution == "" && strings.Contains(model, "2k") {
		maxSide = 2048
	}
	if strings.Contains(resolution, "2k") || strings.Contains(resolution, "2048") {
		maxSide = 2048
	}

	switch aspectRatio {
	case "", "1:1":
		return fmt.Sprintf("%dx%d", maxSide, maxSide)
	case "16:9":
		if maxSide == 2048 {
			return "2048x1152"
		}
		return "1024x576"
	case "9:16":
		if maxSide == 2048 {
			return "1152x2048"
		}
		return "576x1024"
	case "16:10":
		if maxSide == 2048 {
			return "2048x1280"
		}
		return "1024x640"
	case "10:16":
		if maxSide == 2048 {
			return "1280x2048"
		}
		return "640x1024"
	case "4:3":
		if maxSide == 2048 {
			return "2048x1536"
		}
		return "1024x768"
	case "3:4":
		if maxSide == 2048 {
			return "1536x2048"
		}
		return "768x1024"
	case "3:2":
		if maxSide == 2048 {
			return "2048x1360"
		}
		return "1024x682"
	case "2:3":
		if maxSide == 2048 {
			return "1360x2048"
		}
		return "682x1024"
	default:
		return ""
	}
}

func mergeOpenRouterExtraFields(imageReq *dto.ImageRequest, req OpenRouterImageRequest) {
	if imageReq.Extra == nil {
		imageReq.Extra = map[string]json.RawMessage{}
	}

	allowed := map[string]struct{}{
		"negative_prompt":     {},
		"num_inference_steps": {},
		"guidance_scale":      {},
		"seed":                {},
	}
	if isOpenRouterEditImageModel(req.Model) {
		allowed["image_url"] = struct{}{}
		allowed["mask_url"] = struct{}{}
		allowed["task_types"] = struct{}{}
	} else {
		allowed["control_image"] = struct{}{}
		allowed["control_mode"] = struct{}{}
		allowed["image_scale"] = struct{}{}
	}

	for key := range imageReq.Extra {
		if _, ok := allowed[key]; !ok {
			delete(imageReq.Extra, key)
		}
	}

	extraFields := make(map[string]json.RawMessage, len(imageReq.Extra))
	for key, value := range imageReq.Extra {
		extraFields[key] = cloneRawMessage(value)
	}
	raw, err := common.Marshal(extraFields)
	if err == nil && len(bytes.TrimSpace(raw)) > 2 {
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

func normalizeOpenRouterSize(model, size, resolution, aspectRatio string) (string, error) {
	size = strings.TrimSpace(size)
	if strings.Contains(strings.ToLower(size), "x") {
		return size, validateOpenRouterUpstreamSize(model, size)
	}
	if size != "" {
		resolution = size
	}
	normalized := sizeFromOpenRouterDimensions(model, resolution, aspectRatio)
	if normalized == "" {
		if strings.TrimSpace(resolution) != "" || strings.TrimSpace(aspectRatio) != "" {
			return "", fmt.Errorf("unsupported image size parameters for model %s: resolution=%q aspect_ratio=%q", model, resolution, aspectRatio)
		}
		return "", nil
	}
	return normalized, validateOpenRouterUpstreamSize(model, normalized)
}

func cloneRawMessage(source json.RawMessage) json.RawMessage {
	if len(source) == 0 {
		return nil
	}
	result := make([]byte, len(source))
	copy(result, source)
	return result
}

func validateOpenRouterUpstreamSize(model, size string) error {
	width, height, ok := parseOpenRouterSize(size)
	if !ok {
		return fmt.Errorf("unsupported image size %q for model %s", size, model)
	}
	if _, ok := openRouterSupportedUpstreamSizes[fmt.Sprintf("%dx%d", width, height)]; !ok {
		return fmt.Errorf("unsupported image size %q for model %s", size, model)
	}

	lowerModel := strings.ToLower(strings.TrimSpace(model))
	if !strings.Contains(lowerModel, "2k") && (width > 1024 || height > 1024) {
		return fmt.Errorf("image size %s exceeds upstream limit for non-2k model %s; use a -2k model", size, model)
	}
	if width > 2048 || height > 2048 {
		return fmt.Errorf("image size %s exceeds upstream 2k limit for model %s", size, model)
	}
	if strings.HasPrefix(lowerModel, "qwen-image-2512") && width == 2048 && height == 2048 {
		return fmt.Errorf("image size %s exceeds upstream limit for model %s; Qwen-Image-2512 does not support 2048x2048", size, model)
	}
	return nil
}

func parseOpenRouterSize(size string) (int, int, bool) {
	left, right, ok := strings.Cut(strings.ToLower(strings.TrimSpace(size)), "x")
	if !ok {
		return 0, 0, false
	}
	width, err := strconv.Atoi(strings.TrimSpace(left))
	if err != nil {
		return 0, 0, false
	}
	height, err := strconv.Atoi(strings.TrimSpace(right))
	if err != nil {
		return 0, 0, false
	}
	return width, height, width > 0 && height > 0
}

var openRouterSupportedUpstreamSizes = map[string]struct{}{
	"512x512":   {},
	"1024x1024": {},
	"1024x576":  {},
	"576x1024":  {},
	"1024x640":  {},
	"640x1024":  {},
	"1024x768":  {},
	"768x1024":  {},
	"1024x682":  {},
	"682x1024":  {},
	"2048x2048": {},
	"2048x1152": {},
	"1152x2048": {},
	"2048x1280": {},
	"1280x2048": {},
	"2048x1360": {},
	"1360x2048": {},
	"2048x1536": {},
	"1536x2048": {},
}
