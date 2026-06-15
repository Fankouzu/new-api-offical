package tencentvod

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestParseConfigSupportsDelimitedAndJSONKeys(t *testing.T) {
	delimited, err := parseConfig("sid|skey|1500044236", "ap-guangzhou")
	if err != nil {
		t.Fatal(err)
	}
	if delimited.SecretID != "sid" || delimited.SecretKey != "skey" || delimited.SubAppID != 1500044236 || delimited.Region != "ap-guangzhou" {
		t.Fatalf("delimited config = %+v", delimited)
	}

	jsonCfg, err := parseConfig(`{"secret_id":"json_sid","secret_key":"json_skey","sub_app_id":123}`, "ap-shanghai")
	if err != nil {
		t.Fatal(err)
	}
	if jsonCfg.SecretID != "json_sid" || jsonCfg.SecretKey != "json_skey" || jsonCfg.SubAppID != 123 || jsonCfg.Region != "ap-shanghai" {
		t.Fatalf("json config = %+v", jsonCfg)
	}
}

func TestParseConfigRequiresRegionAndSecretFields(t *testing.T) {
	if _, err := parseConfig("sid|skey|1", ""); err == nil {
		t.Fatal("expected missing region error")
	}
	if _, err := parseConfig("sid|skey", "ap-guangzhou"); err == nil {
		t.Fatal("expected invalid key format error")
	}
}

func TestConvertImageRequestPayload(t *testing.T) {
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt: "make an image",
		Images: []string{"https://example.com/in.png"},
		Size:   "2K",
		Metadata: map[string]any{
			"n":            3,
			"aspect_ratio": "16:9",
		},
	}, &relaycommon.RelayInfo{
		OriginModelName: "tencent-vod/kling-image-3.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "tencent-vod/kling-image-3.0",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if action != actionCreateImageTask {
		t.Fatalf("action = %q", action)
	}
	if body.SubAppID != 1500044236 || body.ModelName != "Kling" || body.ModelVersion != "3.0" || body.Prompt != "make an image" {
		t.Fatalf("body = %+v", body)
	}
	if body.OutputConfig["Resolution"] != "2K" || body.OutputConfig["Count"] != 3 || body.OutputConfig["AspectRatio"] != "16:9" {
		t.Fatalf("output config = %#v", body.OutputConfig)
	}
	if len(body.FileInfos) != 1 || body.FileInfos[0].URL != "https://example.com/in.png" {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
}

func TestConvertVideoRequestPayload(t *testing.T) {
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt:     "make a video",
		Image:      "https://example.com/first.png",
		Resolution: "1080P",
		Duration:   5,
	}, &relaycommon.RelayInfo{
		OriginModelName: "tencent-vod/vidu-q3-turbo",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "tencent-vod/vidu-q3-turbo",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if action != actionCreateVideoTask {
		t.Fatalf("action = %q", action)
	}
	if body.ModelName != "Vidu" || body.ModelVersion != "q3-turbo" {
		t.Fatalf("model = %s/%s", body.ModelName, body.ModelVersion)
	}
	if body.OutputConfig["Resolution"] != "1080P" || body.OutputConfig["Duration"] != 5 {
		t.Fatalf("output config = %#v", body.OutputConfig)
	}
}

func TestEstimateBillingUsesResolutionDurationAndCount(t *testing.T) {
	a := &TaskAdaptor{}
	c := taskContext(t, `{"model":"tencent-vod/vidu-q3-turbo","prompt":"video","resolution":"1080P","duration":5}`)
	video := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "tencent-vod/vidu-q3-turbo"})
	if video["duration"] != 5 || video["resolution"] != 1.75 {
		t.Fatalf("video ratios = %#v", video)
	}

	c = taskContext(t, `{"model":"tencent-vod/kling-image-3.0","prompt":"image","resolution":"4K","n":4}`)
	image := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "tencent-vod/kling-image-3.0"})
	if image["resolution"] != 1.8 || image["count"] != 4 {
		t.Fatalf("image ratios = %#v", image)
	}
}

func TestBuildRequestHeaderSignsTencentVODRequest(t *testing.T) {
	a := &TaskAdaptor{pendingAction: actionCreateVideoTask}
	body := []byte(`{"SubAppId":1500044236}`)
	req := httptest.NewRequest(http.MethodPost, "https://vod.tencentcloudapi.com/", bytes.NewReader(body))

	err := a.BuildRequestHeader(nil, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ApiKey:     "sid|skey|1500044236",
		ApiVersion: "ap-guangzhou",
	}})
	if err != nil {
		t.Fatal(err)
	}

	if req.Header.Get("X-TC-Action") != actionCreateVideoTask || req.Header.Get("X-TC-Region") != "ap-guangzhou" {
		t.Fatalf("headers = %#v", req.Header)
	}
	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "TC3-HMAC-SHA256 Credential=sid/") || strings.Contains(auth, "skey") {
		t.Fatalf("authorization header leaks secret or has wrong format: %q", auth)
	}
}

func TestDoResponseReturnsPublicTaskAndStoresUpstreamID(t *testing.T) {
	a := &TaskAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"Response":{"TaskId":"vod-task-123","RequestId":"req-1"}}`)),
	}
	taskID, rawBody, taskErr := a.DoResponse(c, resp, &relaycommon.RelayInfo{
		OriginModelName: "tencent-vod/gv-3.1",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	})
	if taskErr != nil {
		t.Fatalf("DoResponse error = %+v", taskErr)
	}
	if taskID != "vod-task-123" {
		t.Fatalf("taskID = %q", taskID)
	}
	if !strings.Contains(string(rawBody), "vod-task-123") || w.Code != http.StatusOK {
		t.Fatalf("response code=%d raw=%s", w.Code, rawBody)
	}
}

func TestParseTaskResultMapsTencentStatesAndURLs(t *testing.T) {
	a := &TaskAdaptor{}
	success, err := a.ParseTaskResult([]byte(`{"Response":{"Status":"FINISH","Output":{"VideoUrl":"https://example.com/out.mp4"},"RequestId":"req-1"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if success.Status != model.TaskStatusSuccess || success.Url != "https://example.com/out.mp4" {
		t.Fatalf("success = %+v", success)
	}

	failed, err := a.ParseTaskResult([]byte(`{"Response":{"Status":"FAIL","ErrMsg":"bad prompt"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.TaskStatusFailure || failed.Reason != "bad prompt" {
		t.Fatalf("failed = %+v", failed)
	}

	running, err := a.ParseTaskResult([]byte(`{"Response":{"TaskStatus":"PROCESSING","Progress":42}}`))
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != model.TaskStatusInProgress || running.Progress != "42%" {
		t.Fatalf("running = %+v", running)
	}
}

func taskContext(t *testing.T, raw string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	var req relaycommon.TaskSubmitReq
	if err := common.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	c.Set("task_request", req)
	return c
}
