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

func TestConvertToOpenAIAsyncImage_SubmitAckReturnsPendingResponse(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:   "task_image_ack",
		Status:   model.TaskStatusSubmitted,
		Progress: "10%",
		Data: []byte(`{
			"code": 0,
			"msg": "ok",
			"data": {"data": {"id": "I20260401210457-4767-8dc442"}}
		}`),
		Properties: model.Properties{OriginModelName: "doubao-seedream-4-5-251128"},
	}

	b, err := a.ConvertToOpenAIAsyncImage(task)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := common.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["task_id"] != "task_image_ack" {
		t.Fatalf("task_id: got %#v", out["task_id"])
	}
	if out["status"] != model.TaskStatus(model.TaskStatusSubmitted).ToVideoStatus() {
		t.Fatalf("status: got %#v", out["status"])
	}
	if _, ok := out["url"]; ok {
		t.Fatalf("submit ack must not invent url: %#v", out["url"])
	}
	if _, ok := out["upstream"]; !ok {
		t.Fatal("expected upstream submit ack metadata")
	}
}

func TestExtractAssetCreateID_ResponseMetadataResultShape(t *testing.T) {
	const body = `{
		"ResponseMetadata": {
			"RequestId": "20260319162446B5DF7E4FBBC56F78E6DA"
		},
		"Result": {
			"Id": "asset-20260319082447-qrrjp"
		}
	}`

	if got := extractAssetCreateID([]byte(body)); got != "asset-20260319082447-qrrjp" {
		t.Fatalf("extractAssetCreateID() = %q", got)
	}
}
