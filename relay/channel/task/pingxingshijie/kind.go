package pingxingshijie

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Upstream kind for PingXingShiJie async APIs (stored in task private data for polling).
const (
	UpstreamKindVideo = "video"
	UpstreamKindImage = "image"
	UpstreamKindAsset = "asset"
)

// UpstreamKindFromPath returns video | image | asset from gateway path.
func UpstreamKindFromPath(path string) string {
	if strings.Contains(path, "/v1/assets/upload") {
		return UpstreamKindAsset
	}
	// POST /v1/images/generations/async or GET /v1/images/generations/:task_id
	if strings.Contains(path, "/v1/images/generations/") {
		return UpstreamKindImage
	}
	return UpstreamKindVideo
}

// UpstreamKindFromGin is a convenience wrapper.
func UpstreamKindFromGin(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return UpstreamKindVideo
	}
	return UpstreamKindFromPath(c.Request.URL.Path)
}
