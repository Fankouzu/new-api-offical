package sub2api_async

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// ─── Upstream Gemini request DTOs ────────────────────────────────────────────

type geminiImageRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text     string           `json:"text,omitempty"`
	FileData *geminiFileData  `json:"fileData,omitempty"`
	// InlineData is here for completeness; upstream responses carry it but we
	// never send it upstream (callers supply URLs, not raw bytes).
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

// geminiFileData lets Gemini fetch the image directly from a public URL,
// avoiding the need to download and re-upload image bytes on the relay side.
type geminiFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

// geminiInlineData carries raw base64 image bytes in upstream responses.
type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	ResponseModalities []string           `json:"responseModalities"`
	ImageConfig        *geminiImageConfig `json:"imageConfig,omitempty"`
}

// geminiImageConfig maps to the generationConfig.imageConfig block.
// Both fields are optional — Gemini uses sensible defaults when omitted.
type geminiImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	ImageSize   string `json:"imageSize,omitempty"`
}

// ─── Upstream Gemini response DTOs ───────────────────────────────────────────

type geminiImageResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
	ModelVersion  string              `json:"modelVersion"`
	Error         *geminiError        `json:"error"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// ─── Request builder ─────────────────────────────────────────────────────────

const maxGeminiInputImages = 14

// buildGeminiImageRequest converts a TaskSubmitReq into the upstream Gemini
// generateContent request body.
//
// Parameter mapping from downstream caller:
//   - req.Prompt         → parts[0].text
//   - req.Images / req.Image (input_urls) → parts[1..N].fileData (max 14)
//   - metadata.aspect_ratio → generationConfig.imageConfig.aspectRatio
//   - metadata.image_size   → generationConfig.imageConfig.imageSize
//
// The caller controls responseModalities; we always request both TEXT and IMAGE
// so that model reasoning text is captured alongside the image.
func buildGeminiImageRequest(req *relaycommon.TaskSubmitReq, _ *relaycommon.RelayInfo) (*geminiImageRequest, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt is required for gemini-3.1-flash-image")
	}

	parts := []geminiPart{{Text: req.Prompt}}

	// Collect image URLs from req.Image (single) and req.Images (array),
	// de-duplicated and capped at maxGeminiInputImages.
	imageURLs := collectImageURLs(req)
	if len(imageURLs) > maxGeminiInputImages {
		imageURLs = imageURLs[:maxGeminiInputImages]
	}
	for _, u := range imageURLs {
		mimeType := guessImageMIMEFromURL(u)
		parts = append(parts, geminiPart{
			FileData: &geminiFileData{
				MimeType: mimeType,
				FileURI:  u,
			},
		})
	}

	imgCfg := resolveGeminiImageConfig(req)
	genCfg := geminiGenerationConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}
	if imgCfg != nil {
		genCfg.ImageConfig = imgCfg
	}

	return &geminiImageRequest{
		Contents: []geminiContent{
			{Role: "user", Parts: parts},
		},
		GenerationConfig: genCfg,
	}, nil
}

// collectImageURLs gathers URLs from req.Image and req.Images, preserving
// order and removing duplicates.
func collectImageURLs(req *relaycommon.TaskSubmitReq) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(req.Images)+1)
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	if req.Image != "" {
		add(req.Image)
	}
	for _, u := range req.Images {
		add(u)
	}
	// Also check metadata.input_urls (downstream may send them there).
	if inputURLs, ok := req.Metadata["input_urls"]; ok {
		switch v := inputURLs.(type) {
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					add(s)
				}
			}
		case string:
			add(v)
		}
	}
	return out
}

// guessImageMIMEFromURL returns a best-effort MIME type based on the URL
// file extension. Gemini accepts the type hint but does not strictly require
// it to be accurate. Falls back to "image/jpeg".
func guessImageMIMEFromURL(u string) string {
	lower := strings.ToLower(u)
	// Strip query string before checking extension.
	if idx := strings.Index(lower, "?"); idx != -1 {
		lower = lower[:idx]
	}
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// resolveGeminiImageConfig builds the imageConfig block from downstream params.
//
// Priority (highest first):
//  1. metadata.aspect_ratio / metadata.image_size (explicit Gemini params)
//  2. metadata.aspectRatio / metadata.imageSize   (camelCase aliases)
//
// Returns nil when neither aspect_ratio nor image_size is present, causing the
// imageConfig block to be omitted and Gemini to use its defaults.
func resolveGeminiImageConfig(req *relaycommon.TaskSubmitReq) *geminiImageConfig {
	aspectRatio := metadataStringAlt(req.Metadata, "aspect_ratio", "aspectRatio")
	imageSize := metadataStringAlt(req.Metadata, "image_size", "imageSize")

	// Validate values; silently drop invalid ones so bad params do not cause
	// an upstream 400 with a confusing error message.
	if aspectRatio != "" && !validGeminiAspectRatios[aspectRatio] {
		aspectRatio = ""
	}
	if imageSize != "" && !validGeminiImageSizes[imageSize] {
		imageSize = ""
	}

	if aspectRatio == "" && imageSize == "" {
		return nil
	}
	return &geminiImageConfig{
		AspectRatio: aspectRatio,
		ImageSize:   imageSize,
	}
}

// metadataStringAlt reads the first non-empty string value from metadata by
// trying keys in order.
func metadataStringAlt(metadata map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := metadata[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// ─── Response parser ──────────────────────────────────────────────────────────

// parseGeminiImageResult extracts the base64 image data and MIME type from a
// Gemini generateContent response. Returns a "data:<mime>;base64,<b64>" URL
// (to match the existing parseSyncImageGenerationResult contract) or an error.
func parseGeminiImageResult(respBody []byte) (string, error) {
	var res geminiImageResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return "", fmt.Errorf("unmarshal gemini image response failed: %w", err)
	}

	if res.Error != nil && res.Error.Message != "" {
		return "", fmt.Errorf("gemini upstream error %d: %s", res.Error.Code, res.Error.Message)
	}

	for _, candidate := range res.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && strings.TrimSpace(part.InlineData.Data) != "" {
				mime := part.InlineData.MimeType
				if mime == "" {
					mime = detectImageMIME(part.InlineData.Data)
				}
				return "data:" + mime + ";base64," + strings.TrimSpace(part.InlineData.Data), nil
			}
		}
	}

	return "", fmt.Errorf("gemini image response contains no inline image data")
}

// ─── Billing helper ───────────────────────────────────────────────────────────

// resolveGeminiImageSize returns the imageSize tier for billing purposes.
// Reads metadata.image_size (or its camelCase alias) and normalises the value.
// Falls back to "1K" (the Gemini default) when absent or invalid.
func resolveGeminiImageSize(req relaycommon.TaskSubmitReq) string {
	raw := metadataStringAlt(req.Metadata, "image_size", "imageSize")
	upper := strings.ToUpper(strings.TrimSpace(raw))
	if validGeminiImageSizes[upper] {
		return upper
	}
	return "1K"
}
