package sub2api_async

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
		{name: "image to image", modelName: ModelGPTImage2ImageToImage, images: []string{"https://example.com/a.png", "https://example.com/b.png"}, wantKey: "image"},
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
			if _, hasImages := body["images"]; hasImages {
				t.Fatalf("input should not also include generic images key: %#v", body)
			}
			if _, hasInputURLs := body["input_urls"]; hasInputURLs {
				t.Fatalf("input should use OpenAI-compatible image key, not input_urls: %#v", body)
			}
		})
	}
}

func TestConvertGPTImageImageToImageUsesSingleOpenAIImageReference(t *testing.T) {
	a := &TaskAdaptor{}
	const imageURL = "https://example.com/reference.png"
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGPTImage2ImageToImage,
		Prompt: "use the reference",
		Image:  imageURL,
		Images: []string{imageURL},
		Size:   "1440x2560",
	}

	body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelGPTImage2ImageToImage}})
	if err != nil {
		t.Fatal(err)
	}

	got, ok := body["image"].(string)
	if !ok {
		t.Fatalf("image should be a single OpenAI-compatible string, got %T: %#v", body["image"], body["image"])
	}
	if got != imageURL {
		t.Fatalf("image = %q want %q", got, imageURL)
	}
	if _, hasInputURLs := body["input_urls"]; hasInputURLs {
		t.Fatalf("input should not include input_urls: %#v", body)
	}
}

func TestConvertGPTImageImageToImagePreservesImageArrayFromJSON(t *testing.T) {
	a := &TaskAdaptor{}
	const imageURL = "https://example.com/reference.png"
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGPTImage2ImageToImage,
		Prompt: "use the reference",
		Images: []string{imageURL},
		Size:   "1440x2560",
	}

	body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelGPTImage2ImageToImage}})
	if err != nil {
		t.Fatal(err)
	}

	got, ok := body["image"].([]string)
	if !ok {
		t.Fatalf("image should preserve client JSON array shape, got %T: %#v", body["image"], body["image"])
	}
	if len(got) != 1 || got[0] != imageURL {
		t.Fatalf("image = %#v want [%q]", got, imageURL)
	}
	if _, hasInputURLs := body["input_urls"]; hasInputURLs {
		t.Fatalf("input should not include input_urls: %#v", body)
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

func TestDoSyncImageGenerationRoutesByGPTImage2Mode(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		wantPath string
	}{
		{
			name:     "text to image uses generations",
			model:    ModelGPTImage2TextToImage,
			wantPath: "/v1/images/generations",
		},
		{
			name:     "image to image uses edits",
			model:    ModelGPTImage2ImageToImage,
			wantPath: "/v1/images/edits",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seenPath := ""
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				if r.URL.Path != tc.wantPath {
					http.Error(w, "wrong path", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"abc123"}]}`))
			}))
			defer server.Close()

			body := []byte(`{"model":"` + tc.model + `","prompt":"make an image"}`)
			_, err := doSyncImageGeneration(t.Context(), server.URL, "test-key", body)
			if err != nil {
				t.Fatalf("doSyncImageGeneration error = %v, seen path %q", err, seenPath)
			}
			if seenPath != tc.wantPath {
				t.Fatalf("path = %q, want %q", seenPath, tc.wantPath)
			}
		})
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
	dataItems, ok := got["data"].([]any)
	if !ok || len(dataItems) != 1 {
		t.Fatalf("data = %#v", got["data"])
	}
	first, ok := dataItems[0].(map[string]any)
	if !ok {
		t.Fatalf("data[0] = %#v", dataItems[0])
	}
	if first["url"] != "https://example.com/image.png" {
		t.Fatalf("data[0].url = %#v", first["url"])
	}
	if got["status"] != "completed" {
		t.Fatalf("status = %#v", got["status"])
	}
}

func TestConvertToOpenAIAsyncImageReturnsStoredUpstreamBase64(t *testing.T) {
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
		Data:        []byte(`{"created":1,"data":[{"b64_json":"abc123"}]}`),
		PrivateData: model.TaskPrivateData{ResultURL: "http://localhost:3001/v1/videos/task_public/content"},
	}

	data, err := a.ConvertToOpenAIAsyncImage(task)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := common.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	dataItems, ok := got["data"].([]any)
	if !ok || len(dataItems) != 1 {
		t.Fatalf("data = %#v", got["data"])
	}
	first, ok := dataItems[0].(map[string]any)
	if !ok {
		t.Fatalf("data[0] = %#v", dataItems[0])
	}
	if first["b64_json"] != "abc123" {
		t.Fatalf("data[0].b64_json = %#v", first["b64_json"])
	}
	if _, ok := got["url"]; ok {
		t.Fatalf("url should be omitted when b64_json is available: %#v", got["url"])
	}
}

func TestConvertToOpenAIAsyncImageConvertsStoredDataURLToBase64(t *testing.T) {
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
		Data:        []byte(`{"created":1,"data":[{"url":"data:image/png;base64,abc123"}]}`),
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
	dataItems, ok := got["data"].([]any)
	if !ok || len(dataItems) != 1 {
		t.Fatalf("data = %#v", got["data"])
	}
	first, ok := dataItems[0].(map[string]any)
	if !ok {
		t.Fatalf("data[0] = %#v", dataItems[0])
	}
	if first["b64_json"] != "abc123" {
		t.Fatalf("data[0].b64_json = %#v", first["b64_json"])
	}
	if _, ok := got["url"]; ok {
		t.Fatalf("url should be omitted when data URL can be represented as b64_json: %#v", got["url"])
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
