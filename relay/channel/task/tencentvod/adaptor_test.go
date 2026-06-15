package tencentvod

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
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
		OriginModelName: "kling-image-3.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-image-3.0",
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
	if body.OutputConfig["Resolution"] != "2K" || body.OutputConfig["AspectRatio"] != "16:9" {
		t.Fatalf("output config = %#v", body.OutputConfig)
	}
	if _, ok := body.OutputConfig["Count"]; ok {
		t.Fatalf("Tencent VOD does not accept OutputConfig.Count: %#v", body.OutputConfig)
	}
	if len(body.FileInfos) != 1 || body.FileInfos[0].URL != "https://example.com/in.png" {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	if body.FileInfos[0].Type != "Url" || body.FileInfos[0].Category != "" || body.FileInfos[0].Usage != "" {
		t.Fatalf("image file info should only use Tencent image input fields: %#v", body.FileInfos[0])
	}
}

func TestConvertImageRequestPayloadAcceptsOpenAIImageArray(t *testing.T) {
	c := taskContext(t, `{
		"model": "og-image2-high",
		"prompt": "edit image",
		"image": ["https://example.com/input.jpg"],
		"size": "2496x1664"
	}`)
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		t.Fatal(err)
	}
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&req, &relaycommon.RelayInfo{
		OriginModelName: "og-image2-high",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "og-image2-high",
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
	if body.ModelName != "OG" || body.ModelVersion != "image2_high" {
		t.Fatalf("model = %s/%s", body.ModelName, body.ModelVersion)
	}
	if len(body.FileInfos) != 1 {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	got := body.FileInfos[0]
	if got.Type != "Url" || got.Category != "" || got.URL != "https://example.com/input.jpg" || got.Usage != "" || got.ID != "" {
		t.Fatalf("file info = %#v", got)
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
		OriginModelName: "vidu-q3-turbo",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3-turbo",
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
	if len(body.FileInfos) != 1 {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	if body.FileInfos[0].Type != "Url" || body.FileInfos[0].Category != "Image" || body.FileInfos[0].Usage != "Reference" {
		t.Fatalf("video file info should include category and usage: %#v", body.FileInfos[0])
	}
}

func TestDocumentedModelMatrixIncludesRequestedVendors(t *testing.T) {
	cases := []struct {
		publicModel  string
		kind         string
		modelName    string
		modelVersion string
	}{
		{publicModel: ModelGG31Image, kind: modelKindImage, modelName: "GG", modelVersion: "3.1"},
		{publicModel: ModelMJv7Image, kind: modelKindImage, modelName: "MJ", modelVersion: "v7"},
		{publicModel: ModelQwen0925Image, kind: modelKindImage, modelName: "Qwen", modelVersion: "0925"},
		{publicModel: ModelSI50LiteImage, kind: modelKindImage, modelName: "SI", modelVersion: "5.0-lite"},
		{publicModel: ModelJimeng40, kind: modelKindVideo, modelName: "Jimeng", modelVersion: "4.0"},
		{publicModel: ModelSV10Pro, kind: modelKindVideo, modelName: "SV", modelVersion: "1.0-pro"},
		{publicModel: ModelOS20, kind: modelKindVideo, modelName: "OS", modelVersion: "2.0"},
	}

	for _, tc := range cases {
		t.Run(tc.publicModel, func(t *testing.T) {
			spec, ok := lookupModelSpec(tc.publicModel)
			if !ok {
				t.Fatalf("missing model spec for %s", tc.publicModel)
			}
			if spec.Kind != tc.kind || spec.TencentModelName != tc.modelName || spec.TencentModelVersion != tc.modelVersion {
				t.Fatalf("spec = %+v", spec)
			}
			if !containsModel(ModelList, tc.publicModel) {
				t.Fatalf("%s missing from ModelList", tc.publicModel)
			}
		})
	}
}

func TestEstimateBillingUsesResolutionDurationAndCount(t *testing.T) {
	a := &TaskAdaptor{}
	c := taskContext(t, `{"model":"vidu-q3-turbo","prompt":"video","resolution":"1080P","duration":5}`)
	video := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "vidu-q3-turbo"})
	if video["duration"] != 5 || video["resolution"] != 1.75 {
		t.Fatalf("video ratios = %#v", video)
	}

	c = taskContext(t, `{"model":"kling-image-3.0","prompt":"image","resolution":"4K","n":4}`)
	image := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-image-3.0"})
	if image["resolution"] != 1.8 || image["count"] != 4 {
		t.Fatalf("image ratios = %#v", image)
	}
}

func containsModel(models []string, target string) bool {
	for _, model := range models {
		if model == target {
			return true
		}
	}
	return false
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
		OriginModelName: "gv-3.1",
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
	var response map[string]any
	if err := common.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["object"]; ok {
		t.Fatalf("Tencent VOD submit response should not include object: %#v", response)
	}
	if response["id"] != "task_public" || response["task_id"] != "task_public" || response["status"] != dto.VideoStatusQueued {
		t.Fatalf("response = %#v", response)
	}
}

func TestConvertToOpenAIVideoReturnsNeutralTaskStatus(t *testing.T) {
	a := &TaskAdaptor{}
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 1781495472,
		UpdatedAt: 1781495572,
		Properties: model.Properties{
			OriginModelName: "og-image2-high",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/out.png",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := common.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["object"]; ok {
		t.Fatalf("Tencent VOD poll response should not include video object: %#v", response)
	}
	if response["id"] != "task_public" || response["status"] != dto.VideoStatusCompleted || response["url"] != "https://example.com/out.png" {
		t.Fatalf("response = %#v", response)
	}
}

func TestConvertToOpenAIVideoSuppressesGatewayProxyURL(t *testing.T) {
	a := &TaskAdaptor{}
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:   "task_public",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Properties: model.Properties{
			OriginModelName: "hailuo-2.3-fast",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://lizh.ai/v1/videos/task_public/content",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := common.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["url"]; ok {
		t.Fatalf("Tencent VOD response must not expose gateway proxy URL: %#v", response)
	}
}

func TestConvertToOpenAIVideoRecoversDirectURLFromStoredTencentData(t *testing.T) {
	a := &TaskAdaptor{}
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:   "task_public",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Properties: model.Properties{
			OriginModelName: "hailuo-2.3-fast",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamKind: "video",
			ResultURL:    "https://lizh.ai/v1/videos/task_public/content",
		},
		Data: []byte(`{"Response":{"Status":"FINISH","Output":{"FileInfos":[{"MediaBasicInfo":{"MediaUrl":"https://example.com/out.mp4"}}]}}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := common.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response["url"] != "https://example.com/out.mp4" {
		t.Fatalf("response = %#v", response)
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

func TestParseTaskResultFindsNestedTencentMediaURL(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{
		"Response": {
			"Status": "FINISH",
			"Output": {
				"FileInfos": [
					{"Name": "preview"},
					{"MediaBasicInfo": {"MediaUrl": "https://example.com/direct-video.mp4"}}
				]
			},
			"RequestId": "req-1"
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != model.TaskStatusSuccess || info.Url != "https://example.com/direct-video.mp4" {
		t.Fatalf("info = %+v", info)
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
