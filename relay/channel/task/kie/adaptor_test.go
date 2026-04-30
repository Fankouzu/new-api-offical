package kie

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestBuildRequestURLUsesConfiguredBaseURL(t *testing.T) {
	a := &TaskAdaptor{}
	a.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://example.kie.ai/", ApiKey: "test-key"}})

	got, err := a.BuildRequestURL(&relaycommon.RelayInfo{})
	if err != nil {
		t.Fatal(err)
	}

	if got != "https://example.kie.ai/api/v1/jobs/createTask" {
		t.Fatalf("BuildRequestURL = %q", got)
	}
}

func TestConvertSeedance2RequestPayloadFromUnifiedRequest(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    ModelSeedance2,
		Prompt:   "make a video",
		Image:    "https://example.com/first.png",
		Size:     "1280x720",
		Duration: 6,
		Metadata: map[string]any{
			"last_frame_url": "https://example.com/last.png",
			"generate_audio": false,
			"web_search":     true,
		},
	}

	body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelSeedance2}})
	if err != nil {
		t.Fatal(err)
	}

	if body.Model != ModelSeedance2 {
		t.Fatalf("model = %q", body.Model)
	}
	assertInput(t, body.Input, "prompt", "make a video")
	assertInput(t, body.Input, "first_frame_url", "https://example.com/first.png")
	assertInput(t, body.Input, "last_frame_url", "https://example.com/last.png")
	assertInput(t, body.Input, "resolution", "720p")
	assertInput(t, body.Input, "aspect_ratio", "16:9")
	assertInput(t, body.Input, "duration", float64(6))
	assertInput(t, body.Input, "generate_audio", false)
	assertInput(t, body.Input, "web_search", true)
}

func TestConvertImageModelPayloadsFromUnifiedImages(t *testing.T) {
	a := &TaskAdaptor{}
	cases := []struct {
		name      string
		modelName string
		wantKey   string
	}{
		{name: "gpt image 2 image-to-image", modelName: ModelGPTImage2ImageToImage, wantKey: "input_urls"},
		{name: "nano banana 2", modelName: ModelNanoBanana2, wantKey: "image_input"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := relaycommon.TaskSubmitReq{
				Model:  tc.modelName,
				Prompt: "make an image",
				Images: []string{"https://example.com/a.png", "https://example.com/b.png"},
				Size:   "1024x1024",
				Metadata: map[string]any{
					"resolution": "2K",
				},
			}

			body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tc.modelName}})
			if err != nil {
				t.Fatal(err)
			}

			assertInput(t, body.Input, "prompt", "make an image")
			assertInput(t, body.Input, "aspect_ratio", "1:1")
			assertInput(t, body.Input, "resolution", "2K")
			got, ok := body.Input[tc.wantKey].([]string)
			if !ok {
				t.Fatalf("%s has type %T", tc.wantKey, body.Input[tc.wantKey])
			}
			if len(got) != 2 || got[0] != "https://example.com/a.png" || got[1] != "https://example.com/b.png" {
				t.Fatalf("%s = %#v", tc.wantKey, got)
			}
		})
	}
}

func TestResolveDefaultModelsForGenericFallbacks(t *testing.T) {
	a := &TaskAdaptor{}

	imageBody, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Prompt: "make an image",
	}, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dall-e"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}, RequestURLPath: "/v1/images/generations/async"})
	if err != nil {
		t.Fatal(err)
	}
	if imageBody.Model != ModelSeedream45TextToImage {
		t.Fatalf("image default model = %q", imageBody.Model)
	}

	imageEditBody, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Prompt: "edit image",
		Image:  "https://example.com/input.png",
	}, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "dall-e"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}, RequestURLPath: "/v1/images/generations/async"})
	if err != nil {
		t.Fatal(err)
	}
	if imageEditBody.Model != ModelSeedream45ImageToImage {
		t.Fatalf("image edit default model = %q", imageEditBody.Model)
	}

	videoBody, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Prompt: "make a video",
	}, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "59_generate"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{Action: "generate"}, RequestURLPath: "/v1/videos/generations"})
	if err != nil {
		t.Fatal(err)
	}
	if videoBody.Model != ModelSeedance2 {
		t.Fatalf("video default model = %q", videoBody.Model)
	}
}

func TestDoResponseStoresUpstreamTaskIDAndReturnsPublicTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := &TaskAdaptor{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString(`{"code":200,"msg":"success","data":{"taskId":"kie_task_123"}}`)),
	}

	upstreamTaskID, rawBody, taskErr := a.DoResponse(c, resp, &relaycommon.RelayInfo{
		OriginModelName: ModelSeedance2,
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public_123"},
	})
	if taskErr != nil {
		t.Fatalf("DoResponse error = %+v", taskErr)
	}
	if upstreamTaskID != "kie_task_123" {
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
}

func TestFetchTaskUsesRecordInfoEndpointAndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs/recordInfo" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("taskId") != "task/id with space" {
			t.Fatalf("taskId query = %q", r.URL.Query().Get("taskId"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"msg":"success","data":{"state":"generating"}}`))
	}))
	defer server.Close()

	a := &TaskAdaptor{}
	resp, err := a.FetchTask(server.URL, "test-key", map[string]any{"task_id": "task/id with space"}, "")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body = %s", resp.StatusCode, string(b))
	}
}

func TestParseTaskResultMapsKieStatesAndResultURL(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{"code":200,"msg":"success","data":{"state":"success","resultJson":"{\"resultUrls\":[\"https://example.com/out.mp4\"]}"}}`))
	if err != nil {
		t.Fatal(err)
	}

	if info.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %q", info.Status)
	}
	if info.Progress != "100%" {
		t.Fatalf("progress = %q", info.Progress)
	}
	if info.Url != "https://example.com/out.mp4" {
		t.Fatalf("url = %q", info.Url)
	}

	failed, err := a.ParseTaskResult([]byte(`{"code":200,"msg":"success","data":{"state":"fail","failMsg":"bad prompt"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.TaskStatusFailure || failed.Reason != "bad prompt" {
		t.Fatalf("failed result = %+v", failed)
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
			OriginModelName: ModelNanoBanana2,
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
	if got["url"] != "https://example.com/image.png" {
		t.Fatalf("url = %#v", got["url"])
	}
	if got["status"] != "completed" {
		t.Fatalf("status = %#v", got["status"])
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
