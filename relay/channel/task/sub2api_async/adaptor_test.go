package sub2api_async

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

func TestBuildRequestURLUsesConfiguredBaseURL(t *testing.T) {
	a := &TaskAdaptor{}
	a.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://example.sub2api.local/", ApiKey: "test-key"}})

	got, err := a.BuildRequestURL(&relaycommon.RelayInfo{})
	if err != nil {
		t.Fatal(err)
	}

	if got != "https://example.sub2api.local/v1/images/generations" {
		t.Fatalf("BuildRequestURL = %q", got)
	}
}

func TestConvertGPTImagePayloadsFromUnifiedRequest(t *testing.T) {
	a := &TaskAdaptor{}
	cases := []struct {
		name      string
		modelName string
		images    []string
		wantKey   string
	}{
		{name: "text to image", modelName: ModelGPTImage2TextToImage},
		{name: "image to image", modelName: ModelGPTImage2ImageToImage, images: []string{"https://example.com/a.png", "https://example.com/b.png"}, wantKey: "input_urls"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := relaycommon.TaskSubmitReq{
				Model:  tc.modelName,
				Prompt: "make an image",
				Images: tc.images,
				Size:   "1024x1024",
				Metadata: map[string]any{
					"resolution": "2K",
				},
			}

			body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tc.modelName}})
			if err != nil {
				t.Fatal(err)
			}

			if body["model"] != tc.modelName {
				t.Fatalf("model = %q", body["model"])
			}
			assertInput(t, body, "prompt", "make an image")
			assertInput(t, body, "size", "1024x1024")
			assertInput(t, body, "aspect_ratio", "1:1")
			assertInput(t, body, "resolution", "2K")
			if tc.wantKey == "" {
				if _, ok := body["input_urls"]; ok {
					t.Fatalf("text-to-image should not include input_urls: %#v", body)
				}
				return
			}
			got, ok := body[tc.wantKey].([]string)
			if !ok {
				t.Fatalf("%s has type %T", tc.wantKey, body[tc.wantKey])
			}
			if len(got) != 2 || got[0] != "https://example.com/a.png" || got[1] != "https://example.com/b.png" {
				t.Fatalf("%s = %#v", tc.wantKey, got)
			}
		})
	}
}

func TestDoResponseReturnsPublicTaskAndSchedulesBackgroundWorker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := &TaskAdaptor{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{"model":"gpt-image-2-text-to-image","prompt":"make an image"}`)),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: ModelGPTImage2TextToImage,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public_123"},
	}

	upstreamTaskID, rawBody, taskErr := a.DoResponse(c, resp, info)
	if taskErr != nil {
		t.Fatalf("DoResponse error = %+v", taskErr)
	}
	if upstreamTaskID != "task_public_123" {
		t.Fatalf("upstreamTaskID = %q", upstreamTaskID)
	}
	if len(rawBody) == 0 {
		t.Fatal("expected raw response body")
	}

	var got map[string]any
	if err := common.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != "task_public_123" || got["task_id"] != "task_public_123" {
		t.Fatalf("public task response = %s", recorder.Body.String())
	}
	if info.AfterTaskInserted == nil {
		t.Fatal("expected background worker callback to be registered")
	}
}

func TestDoRequestCarriesBuiltRequestBodyWithoutCallingUpstream(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"model":"gpt-image-2-text-to-image","prompt":"make an image"}`)
	resp, err := a.DoRequest(&gin.Context{}, &relaycommon.RelayInfo{}, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("body = %s, want %s", string(got), string(body))
	}
}

func TestParseTaskResultMapsStatesAndResultURL(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"id":"sub2_task_123","task_id":"sub2_task_123","status":"completed","url":"https://example.com/out.png"}`))
	if err != nil {
		t.Fatal(err)
	}

	if info.TaskID != "sub2_task_123" {
		t.Fatalf("task id = %q", info.TaskID)
	}
	if info.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %q", info.Status)
	}
	if info.Progress != "100%" {
		t.Fatalf("progress = %q", info.Progress)
	}
	if info.Url != "https://example.com/out.png" {
		t.Fatalf("url = %q", info.Url)
	}

	failed, err := a.ParseTaskResult([]byte(`{"id":"sub2_task_123","status":"failed","error":{"message":"bad prompt"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.TaskStatusFailure || failed.Reason != "bad prompt" {
		t.Fatalf("failed result = %+v", failed)
	}

	inProgress, err := a.ParseTaskResult([]byte(`{"id":"task_local","task_id":"task_local","status":"in_progress","progress":"30%"}`))
	if err != nil {
		t.Fatal(err)
	}
	if inProgress.Status != model.TaskStatusInProgress || inProgress.Progress != "30%" {
		t.Fatalf("in progress result = %+v", inProgress)
	}
}

func TestParseSyncImageGenerationResult(t *testing.T) {
	urlResult, err := parseSyncImageGenerationResult([]byte(`{"created":1,"data":[{"url":"https://example.com/out.png"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if urlResult != "https://example.com/out.png" {
		t.Fatalf("url result = %q", urlResult)
	}

	b64Result, err := parseSyncImageGenerationResult([]byte(`{"created":1,"data":[{"b64_json":"abc123"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if b64Result != "data:image/png;base64,abc123" {
		t.Fatalf("b64 result = %q", b64Result)
	}
}

func TestConvertToOpenAIAsyncImageUsesStoredResultURL(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 123,
		UpdatedAt: 456,
		Properties: model.Properties{
			OriginModelName: ModelGPTImage2TextToImage,
		},
		PrivateData: model.TaskPrivateData{ResultURL: "https://example.com/image.png"},
	}

	data, err := a.ConvertToOpenAIAsyncImage(task)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["object"] != "sub2api_async.image.generation.task" {
		t.Fatalf("object = %#v", got["object"])
	}
	if got["url"] != "https://example.com/image.png" {
		t.Fatalf("url = %#v", got["url"])
	}
	if got["status"] != "completed" {
		t.Fatalf("status = %#v", got["status"])
	}
}

func TestConvertToOpenAIAsyncImageUsesProxyURLForStoredDataURL(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 123,
		UpdatedAt: 456,
		Properties: model.Properties{
			OriginModelName: ModelGPTImage2TextToImage,
		},
		PrivateData: model.TaskPrivateData{ResultURL: "data:image/png;base64,abc123"},
	}

	data, err := a.ConvertToOpenAIAsyncImage(task)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	url, _ := got["url"].(string)
	if strings.HasPrefix(url, "data:") {
		t.Fatalf("url should not expose data URL: %q", url)
	}
	if !strings.Contains(url, "/v1/videos/task_public/content") {
		t.Fatalf("url = %q, want local content proxy URL", url)
	}
}

func TestEstimateBillingAppliesGPTImage2ResolutionRatios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name       string
		modelName  string
		resolution string
		wantRatio  float64
	}{
		{name: "text 1K uses base price", modelName: ModelGPTImage2TextToImage, resolution: "1K", wantRatio: 1},
		{name: "text 2K scales from 1K base price", modelName: ModelGPTImage2TextToImage, resolution: "2K", wantRatio: 5.0 / 3.0},
		{name: "image 4K scales from 1K base price", modelName: ModelGPTImage2ImageToImage, resolution: "4K", wantRatio: 8.0 / 3.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &TaskAdaptor{}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Set("task_request", relaycommon.TaskSubmitReq{Metadata: map[string]any{"resolution": tc.resolution}})

			ratios := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: tc.modelName})
			got, ok := ratios["resolution"]
			if !ok {
				t.Fatalf("missing resolution ratio in %#v", ratios)
			}
			if got != tc.wantRatio {
				t.Fatalf("resolution ratio = %v, want %v", got, tc.wantRatio)
			}
		})
	}
}

func assertInput(t *testing.T, input map[string]any, key string, want any) {
	t.Helper()
	got, ok := input[key]
	if !ok {
		t.Fatalf("missing input[%q] in %#v", key, input)
	}
	if got != want {
		t.Fatalf("input[%q] = %#v, want %#v", key, got, want)
	}
}
