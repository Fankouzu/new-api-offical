package common

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestTaskSubmitReq_UnmarshalJSON_ImageArray(t *testing.T) {
	const payload = `{
		"model": "doubao-seedream-4-5-251128",
		"prompt": "test",
		"image": ["https://example.com/a.jpg", "https://example.com/b.jpg"]
	}`
	var req TaskSubmitReq
	if err := common.UnmarshalJsonStr(payload, &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Images) != 2 {
		t.Fatalf("Images: got %d want 2", len(req.Images))
	}
	if req.Images[0] != "https://example.com/a.jpg" {
		t.Fatalf("unexpected first URL: %q", req.Images[0])
	}
	if !req.HasImage() {
		t.Fatal("HasImage() should be true")
	}
}

func TestTaskSubmitReq_UnmarshalJSON_ImageString(t *testing.T) {
	const payload = `{"prompt":"x","image":"https://example.com/one.png"}`
	var req TaskSubmitReq
	if err := common.UnmarshalJsonStr(payload, &req); err != nil {
		t.Fatal(err)
	}
	if req.Image != "https://example.com/one.png" {
		t.Fatalf("Image: %q", req.Image)
	}
	if !req.HasImage() {
		t.Fatal("HasImage() should be true for single image string")
	}
}

func TestTaskSubmitReq_UnmarshalJSON_PreservesProviderTopLevelFieldsAsMetadata(t *testing.T) {
	const payload = `{
		"model": "doubao-seedance-1-5-pro-251215",
		"prompt": "首帧过渡到尾帧",
		"content": [
			{"type":"text","text":"首帧过渡到尾帧"},
			{"type":"image_url","image_url":{"url":"https://example.com/first.jpg"},"role":"first_frame"},
			{"type":"image_url","image_url":{"url":"https://example.com/last.jpg"},"role":"last_frame"}
		],
		"generate_audio": true,
		"ratio": "adaptive",
		"duration": 6,
		"watermark": false,
		"resolution": "720p"
	}`
	var req TaskSubmitReq
	if err := common.UnmarshalJsonStr(payload, &req); err != nil {
		t.Fatal(err)
	}
	if req.Duration != 6 {
		t.Fatalf("Duration: got %d want 6", req.Duration)
	}
	if req.Resolution != "720p" {
		t.Fatalf("Resolution: got %q", req.Resolution)
	}
	content, ok := req.Metadata["content"].([]interface{})
	if !ok || len(content) != 3 {
		t.Fatalf("metadata content: got %#v", req.Metadata["content"])
	}
	second, ok := content[1].(map[string]interface{})
	if !ok || second["role"] != "first_frame" {
		t.Fatalf("second content item: %#v", content[1])
	}
	if req.Metadata["generate_audio"] != true {
		t.Fatalf("generate_audio metadata: %#v", req.Metadata["generate_audio"])
	}
	if req.Metadata["ratio"] != "adaptive" {
		t.Fatalf("ratio metadata: %#v", req.Metadata["ratio"])
	}
	if req.Metadata["watermark"] != false {
		t.Fatalf("watermark metadata: %#v", req.Metadata["watermark"])
	}
}
