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

func isTaskResultProxyURL(u string) bool {
	u = strings.TrimSpace(u)
	return strings.Contains(u, "/api/task/") && strings.Contains(u, "/result")
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

func looksLikeVideoAssetURL(u string) bool {
	lower := strings.ToLower(strings.TrimSpace(u))
	if !strings.HasPrefix(lower, "http") {
		return false
	}
	return strings.Contains(lower, ".mp4") ||
		strings.Contains(lower, ".webm") ||
		strings.Contains(lower, ".mov") ||
		strings.Contains(lower, ".m4v") ||
		strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, "video")
}

func isImageDataURL(u string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(u)), "data:image/")
}

func isMediaDataURL(u string) bool {
	lower := strings.ToLower(strings.TrimSpace(u))
	return strings.HasPrefix(lower, "data:image/") || strings.HasPrefix(lower, "data:video/")
}

func looksLikeMediaResultURL(u string) bool {
	return looksLikeImageAssetURL(u) || looksLikeVideoAssetURL(u)
}

func isInputMediaKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(lower, "request") ||
		strings.Contains(lower, "input") ||
		strings.Contains(lower, "prompt") ||
		strings.Contains(lower, "source") ||
		strings.Contains(lower, "reference") ||
		strings.Contains(lower, "mask")
}

func isMediaURLKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(lower, "url") ||
		strings.Contains(lower, "image") ||
		strings.Contains(lower, "video") ||
		strings.Contains(lower, "thumbnail") ||
		strings.Contains(lower, "cover")
}

func mediaURLKeyHintsResult(key string, value string) bool {
	if isInputMediaKey(key) || !isMediaURLKey(key) {
		return false
	}
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lowerValue, "http") &&
		(strings.Contains(lowerKey, "image") ||
			strings.Contains(lowerKey, "img") ||
			strings.Contains(lowerKey, "video") ||
			strings.Contains(lowerKey, "thumbnail") ||
			strings.Contains(lowerKey, "cover"))
}

func walkFirstMediaLikeURL(v any) string {
	switch x := v.(type) {
	case map[string]any:
		if b64, ok := x["b64_json"].(string); ok && strings.TrimSpace(b64) != "" {
			return "data:image/png;base64," + strings.TrimSpace(b64)
		}
		if u, ok := x["url"].(string); ok {
			switch {
			case strings.HasPrefix(u, "http") && looksLikeMediaResultURL(u):
				return u
			case isMediaDataURL(u):
				return u
			}
		}
		for key, vv := range x {
			if isInputMediaKey(key) {
				continue
			}
			if s, ok := vv.(string); ok && isMediaURLKey(key) && !isInputMediaKey(key) {
				if isMediaDataURL(s) || looksLikeMediaResultURL(s) || mediaURLKeyHintsResult(key, s) {
					return strings.TrimSpace(s)
				}
			}
			if s := walkFirstMediaLikeURL(vv); s != "" {
				return s
			}
		}
	case string:
		if isMediaDataURL(x) {
			return x
		}
	case []any:
		for _, item := range x {
			if s := walkFirstMediaLikeURL(item); s != "" {
				return s
			}
		}
	}
	return ""
}

func walkFirstImageLikeURL(v any) string {
	switch x := v.(type) {
	case map[string]any:
		if b64, ok := x["b64_json"].(string); ok && strings.TrimSpace(b64) != "" {
			return "data:image/png;base64," + strings.TrimSpace(b64)
		}
		if u, ok := x["url"].(string); ok {
			switch {
			case strings.HasPrefix(u, "http") && looksLikeImageAssetURL(u):
				return u
			case isImageDataURL(u):
				return u
			}
		}
		for _, vv := range x {
			if s := walkFirstImageLikeURL(vv); s != "" {
				return s
			}
		}
	case string:
		if isImageDataURL(x) {
			return x
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

// extractFirstMediaLikeURLFromJSON scans nested task payload JSON for the first image/video result URL.
func extractFirstMediaLikeURLFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var root any
	if err := common.Unmarshal(raw, &root); err != nil {
		return ""
	}
	return walkFirstMediaLikeURL(root)
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
