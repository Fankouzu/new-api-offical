package pingxingshijie

import (
	"math"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
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
			"draft":      true,
			"resolution": "1080p",
			"watermark":  false,
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
			"resolution":        "720p",
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

func TestConvertToRequestPayload_TopLevelSeedanceContentAndParams(t *testing.T) {
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
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(payload, &req); err != nil {
		t.Fatal(err)
	}
	body, err := (&TaskAdaptor{}).convertToRequestPayload(&req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := common.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	content, ok := got["content"].([]interface{})
	if !ok || len(content) != 3 {
		t.Fatalf("content: got %#v in %s", got["content"], string(data))
	}
	for i, wantRole := range []string{"first_frame", "last_frame"} {
		item, ok := content[i+1].(map[string]interface{})
		if !ok || item["role"] != wantRole {
			t.Fatalf("content[%d] role: got %#v want %q", i+1, content[i+1], wantRole)
		}
	}
	if got["duration"] != float64(6) {
		t.Fatalf("duration: got %#v", got["duration"])
	}
	if got["resolution"] != "720p" {
		t.Fatalf("resolution: got %#v", got["resolution"])
	}
	if got["ratio"] != "adaptive" {
		t.Fatalf("ratio: got %#v", got["ratio"])
	}
	if got["watermark"] != false {
		t.Fatalf("watermark: got %#v", got["watermark"])
	}
	if got["generate_audio"] != true {
		t.Fatalf("generate_audio: got %#v", got["generate_audio"])
	}
}

func TestEstimateBilling_Seedance20OnlyApplies1080PResolutionRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		model      string
		resolution string
		content    []interface{}
		want       map[string]float64
	}{
		{
			name:       "seedance 2.0 1080p without video input uses 51 base price ratio",
			model:      "doubao-seedance-2-0-260128",
			resolution: "1080p",
			want:       map[string]float64{"resolution": 51.0 / 46.0},
		},
		{
			name:       "seedance 2.0 1080p with video input uses corrected 31 over 51 combined ratio",
			model:      "doubao-seedance-2-0-260128",
			resolution: "1080p",
			content: []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/input.mp4"},
				},
			},
			want: map[string]float64{"video_input": 31.0 / 51.0, "resolution": 51.0 / 46.0},
		},
		{
			name:       "seedance 2.0 720p without video input has no adjustment",
			model:      "doubao-seedance-2-0-260128",
			resolution: "720p",
		},
		{
			name:       "seedance 2.0 480p with video input keeps normal video input pricing",
			model:      "doubao-seedance-2-0-260128",
			resolution: "480p",
			content: []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/input.mp4"},
				},
			},
			want: map[string]float64{"video_input": 28.0 / 46.0},
		},
		{
			name:       "seedance 2.0 720p with video input keeps normal video input pricing",
			model:      "doubao-seedance-2-0-260128",
			resolution: "720p",
			content: []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/input.mp4"},
				},
			},
			want: map[string]float64{"video_input": 28.0 / 46.0},
		},
		{
			name:       "seedance 2.0 fast 1080p is not adjusted by seedance 2.0 pricing",
			model:      "doubao-seedance-2-0-fast-260128",
			resolution: "1080p",
		},
		{
			name:       "seedance 2.0 fast with video input keeps fast pricing",
			model:      "doubao-seedance-2-0-fast-260128",
			resolution: "1080p",
			content: []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/input.mp4"},
				},
			},
			want: map[string]float64{"video_input": 22.0 / 37.0},
		},
		{
			name:       "seedance 1.5 pro 1080p is not adjusted by seedance 2.0 pricing",
			model:      "doubao-seedance-1-5-pro-251215",
			resolution: "1080p",
		},
		{
			name:       "seedance 1.0 is not adjusted by seedance 2.0 pricing",
			model:      "doubao-seedance-1-0-pro-fast-251015",
			resolution: "1080p",
			content: []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/input.mp4"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			metadata := map[string]interface{}{"resolution": tt.resolution}
			if tt.content != nil {
				metadata["content"] = tt.content
			}
			c.Set("task_request", relaycommon.TaskSubmitReq{
				Model:    tt.model,
				Prompt:   "test",
				Metadata: metadata,
			})
			got := (&TaskAdaptor{}).EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: tt.model})
			assertRatios(t, got, tt.want)
		})
	}
}

func assertRatios(t *testing.T, got, want map[string]float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ratio count: got %#v want %#v", got, want)
	}
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			t.Fatalf("missing ratio %q in %#v", key, got)
		}
		if math.Abs(gotValue-wantValue) > 1e-12 {
			t.Fatalf("ratio %q: got %.12f want %.12f", key, gotValue, wantValue)
		}
	}
}
