package pingxingshijie

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func TestJsonAnyFromBytes(t *testing.T) {
	v := jsonAnyFromBytes([]byte(`{"code":0,"msg":"ok","data":{"id":"x"}}`))
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", v)
	}
	if m["code"].(float64) != 0 {
		t.Fatalf("code: %v", m["code"])
	}
}

func TestConvertToOpenAIVideo_IncludesUpstreamMetadata(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:     "task_x",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		Data:       []byte(`{"code":0,"msg":"ok","data":{"id":"u1","status":"succeeded","content":{"video_url":"https://ex/v.mp4"},"usage":{"total_tokens":1}}}`),
		Properties: model.Properties{OriginModelName: "m1"},
	}
	b, err := a.ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := common.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	meta, _ := out["metadata"].(map[string]any)
	if meta == nil {
		t.Fatal("missing metadata")
	}
	up, ok := meta["upstream"].(map[string]any)
	if !ok || up["code"].(float64) != 0 {
		t.Fatalf("metadata.upstream: %#v", meta["upstream"])
	}
}
