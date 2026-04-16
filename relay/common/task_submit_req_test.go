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
