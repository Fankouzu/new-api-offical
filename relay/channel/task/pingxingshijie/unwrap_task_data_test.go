package pingxingshijie

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func TestUnwrapInnerForTaskData_EnvelopeEmptyDataUsesRaw(t *testing.T) {
	// Success envelope with missing/empty data must not return empty inner (would break json.Unmarshal).
	const body = `{"code":0,"msg":"ok"}`
	inner, err := unwrapInnerForTaskData([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if string(inner) != body {
		t.Fatalf("expected full body fallback, got %q", string(inner))
	}
}

func TestConvertToOpenAIVideo_EnvelopeShapeWithoutInnerTask(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:     "task_test",
		Status:     model.TaskStatusInProgress,
		Progress:   "30%",
		Data:       []byte(`{"code":0,"msg":"ok"}`),
		Properties: model.Properties{OriginModelName: "m"},
	}
	b, err := a.ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := common.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
}
