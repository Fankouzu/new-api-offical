package model

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// isVideoProxyContentURL reports whether u is this gateway's /v1/videos/:task_id/content proxy URL.
func isVideoProxyContentURL(u, taskID string) bool {
	u = strings.TrimSpace(u)
	if u == "" || taskID == "" {
		return false
	}
	return strings.Contains(u, "/v1/videos/") && strings.Contains(u, taskID) && strings.Contains(u, "/content")
}

// looksLikeImageAssetURL is a light heuristic for HTTP URLs that point to raster images (TOS, CDN, etc.).
func looksLikeImageAssetURL(u string) bool {
	lower := strings.ToLower(strings.TrimSpace(u))
	if !strings.HasPrefix(lower, "http") {
		return false
	}
	if strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".jpg") ||
		strings.Contains(lower, ".png") || strings.Contains(lower, ".webp") ||
		strings.Contains(lower, ".gif") {
		return true
	}
	// Seedream and similar APIs may omit extension in signed URLs
	if strings.Contains(lower, "seedream") || strings.Contains(lower, "image") && strings.Contains(lower, "generation") {
		return true
	}
	return false
}

func walkFirstImageLikeURL(v any) string {
	switch x := v.(type) {
	case map[string]any:
		if u, ok := x["url"].(string); ok && strings.HasPrefix(u, "http") && looksLikeImageAssetURL(u) {
			return u
		}
		for _, vv := range x {
			if s := walkFirstImageLikeURL(vv); s != "" {
				return s
			}
		}
	case []any:
		for _, item := range x {
			if s := walkFirstImageLikeURL(item); s != "" {
				return s
			}
		}
	}
	return ""
}

// extractFirstImageLikeHTTPURLFromJSON scans nested task payload JSON for the first image result URL.
func extractFirstImageLikeHTTPURLFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var root any
	if err := common.Unmarshal(raw, &root); err != nil {
		return ""
	}
	return walkFirstImageLikeURL(root)
}

// ExtractImageURLFromJSONBytes parses arbitrary JSON bytes (e.g. upstream poll body) for an image URL.
func ExtractImageURLFromJSONBytes(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var root any
	if err := common.Unmarshal(raw, &root); err != nil {
		return ""
	}
	return walkFirstImageLikeURL(root)
}
