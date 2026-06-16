package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

type capturePollingAdaptor struct {
	body map[string]any
}

func (a *capturePollingAdaptor) Init(_ *relaycommon.RelayInfo) {}

func (a *capturePollingAdaptor) FetchTask(_ string, _ string, body map[string]any, _ string) (*http.Response, error) {
	a.body = body
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"Response":{"Status":"PROCESSING","Progress":10}}`)),
	}, nil
}

func (a *capturePollingAdaptor) ParseTaskResult(_ []byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{
		Status:   string(model.TaskStatusInProgress),
		Progress: "10%",
	}, nil
}

func (a *capturePollingAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int {
	return 0
}

func TestUpdateVideoSingleTaskPassesChannelOtherAsRegion(t *testing.T) {
	adaptor := &capturePollingAdaptor{}
	task := &model.Task{
		TaskID:    "public-task",
		Action:    constant.TaskActionGenerate,
		Status:    model.TaskStatusSubmitted,
		ChannelId: 62,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "vod-task-123",
		},
	}
	ch := &model.Channel{
		Type:  62,
		Key:   "sid|skey|1500044236",
		Other: "ap-guangzhou",
	}

	if err := updateVideoSingleTask(context.Background(), adaptor, ch, "public-task", map[string]*model.Task{"public-task": task}); err != nil {
		t.Fatal(err)
	}

	if adaptor.body["region"] != "ap-guangzhou" || adaptor.body["api_version"] != "ap-guangzhou" {
		t.Fatalf("fetch body = %#v", adaptor.body)
	}
}
