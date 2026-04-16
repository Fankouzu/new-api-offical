package pingxingshijie

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestNormalizeSeedance15DraftUpscaleResolution(t *testing.T) {
	m := "doubao-seedance-1-5-pro-251215"
	if got := normalizeSeedance15DraftUpscaleResolution(m, "480p"); got != "720p" {
		t.Fatalf("480p: got %q", got)
	}
	if got := normalizeSeedance15DraftUpscaleResolution(m, "720p"); got != "720p" {
		t.Fatalf("720p: got %q", got)
	}
	if got := normalizeSeedance15DraftUpscaleResolution(m, "1080p"); got != "1080p" {
		t.Fatalf("1080p: got %q", got)
	}
	if got := normalizeSeedance15DraftUpscaleResolution("other-model", "480p"); got != "480p" {
		t.Fatalf("other model: got %q", got)
	}
}

func TestConvertToRequestPayload_DraftTaskClearsDraftAndSkipsTopLevelSeconds(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:   "doubao-seedance-1-5-pro-251215",
		Prompt:  "Generate from draft",
		Seconds: "10",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":       "draft_task",
					"draft_task": map[string]interface{}{"id": "cgt-20260416103233-xccct"},
				},
			},
			"draft":        true,
			"resolution":   "1080p",
			"watermark":    false,
		},
	}
	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		t.Fatal(err)
	}
	if body.Draft != nil {
		t.Fatalf("draft_task upscale should omit draft flag from upstream body, got %+v", body.Draft)
	}
	if body.Duration != nil {
		t.Fatalf("draft_task upscale should not merge top-level seconds into duration, got %+v", body.Duration)
	}
	if body.Resolution != "1080p" {
		t.Fatalf("resolution: got %q", body.Resolution)
	}
}

// TestDraftTaskUpscale_UpstreamJSONShape documents the contract: downstream OpenAI-style body
// with metadata.content draft_task maps to upstream Ark body without generate_audio/duration/text.
func TestDraftTaskUpscale_UpstreamJSONShape(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-1-5-pro-251215",
		Prompt: "Generate from draft",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "draft_task",
					"draft_task": map[string]interface{}{
						"id": "cgt-20260416103233-xccct",
					},
				},
			},
			"watermark":         false,
			"resolution":      "720p",
			"return_last_frame": true,
		},
	}
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		t.Fatal(err)
	}
	// Mirrors BuildRequestBody: default generate_audio is skipped when draft_task is present.
	if body.GenerateAudio != nil {
		t.Fatalf("generate_audio must not be set for draft_task before marshal, got %+v", body.GenerateAudio)
	}
	if !contentHasDraftTask(body.Content) {
		t.Fatal("expected draft_task in content")
	}
	data, err := common.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"generate_audio", "draft", "duration", "prompt"} {
		if _, ok := got[k]; ok {
			t.Errorf("upstream JSON must not include %q", k)
		}
	}
	for _, k := range []string{"model", "content", "watermark", "resolution", "return_last_frame"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing expected key %q in %s", k, string(data))
		}
	}
	content, ok := got["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("content: got %#v", got["content"])
	}
	item, ok := content[0].(map[string]interface{})
	if !ok || item["type"] != "draft_task" {
		t.Fatalf("content[0]: %#v", content[0])
	}
	dt, ok := item["draft_task"].(map[string]interface{})
	if !ok || dt["id"] != "cgt-20260416103233-xccct" {
		t.Fatalf("draft_task: %#v", item["draft_task"])
	}
}
