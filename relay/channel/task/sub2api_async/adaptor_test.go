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
		{name: "image to image", modelName: ModelGPTImage2ImageToImage, images: []string{"https://example.com/a.png", "https://example.com/b.png"}, wantKey: "images"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Resolution is sent via the authoritative TaskSubmitReq.Resolution
			// field, not through Metadata. The adapter now strips authoritative
			// keys (model / prompt / size / resolution) out of metadata before
			// merging, so a caller that wants to set resolution must use the
			// typed field — this prevents a future caller from silently
			// shadowing the request shape via metadata.
			req := relaycommon.TaskSubmitReq{
				Model:      tc.modelName,
				Prompt:     "make an image",
				Images:     tc.images,
				Size:       "1024x1024",
				Resolution: "2K",
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
			// `resolution` comes from the explicit metadata above. aspect_ratio
			// is no longer auto-derived from size — the channel forwards the
			// W×H request shape verbatim so the caller controls what upstream
			// sees.
			assertInput(t, body, "resolution", "2K")
			if _, hasAspect := body["aspect_ratio"]; hasAspect {
				t.Fatalf("aspect_ratio must not be auto-derived from size: %#v", body)
			}
			if tc.wantKey == "" {
				if _, ok := body["input_urls"]; ok {
					t.Fatalf("text-to-image should not include input_urls: %#v", body)
				}
				return
			}
			got, ok := body[tc.wantKey].([]map[string]string)
			if !ok {
				t.Fatalf("%s has type %T", tc.wantKey, body[tc.wantKey])
			}
			if len(got) != 2 || got[0]["image_url"] != "https://example.com/a.png" || got[1]["image_url"] != "https://example.com/b.png" {
				t.Fatalf("%s = %#v", tc.wantKey, got)
			}
			if _, hasImage := body["image"]; hasImage {
				t.Fatalf("input should not include OpenAI image key for Sub2API edits: %#v", body)
			}
			if _, hasInputURLs := body["input_urls"]; hasInputURLs {
				t.Fatalf("input should not include input_urls: %#v", body)
			}
		})
	}
}

// TestConvertGPTImageMetadataCannotShadowAuthoritativeFields codifies the
// field-injection guard added by convertToRequestPayload's
// stripAuthoritativeMetadataFields. Authoritative request shape comes from
// the typed TaskSubmitReq fields; metadata is for *extra* fields the upstream
// understands, not a backdoor that can override size / resolution / prompt
// / model after they have been set.
func TestConvertGPTImageMetadataCannotShadowAuthoritativeFields(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:      ModelGPTImage2TextToImage,
		Prompt:     "authoritative prompt",
		Size:       "2K",
		Resolution: "2K",
		Metadata: map[string]any{
			"size":       "8K",       // attempted backdoor
			"resolution": "8K",       // attempted backdoor
			"prompt":     "injected", // attempted backdoor
			"model":      "phantom",  // already stripped by UnmarshalMetadata
			"quality":    "high",     // legitimate metadata passthrough
		},
	}

	body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelGPTImage2TextToImage}})
	if err != nil {
		t.Fatal(err)
	}

	// Authoritative fields must survive verbatim from TaskSubmitReq.
	assertInput(t, body, "model", ModelGPTImage2TextToImage)
	assertInput(t, body, "prompt", "authoritative prompt")
	assertInput(t, body, "size", "2K")
	assertInput(t, body, "resolution", "2K")

	// Non-authoritative metadata still passes through.
	assertInput(t, body, "quality", "high")
}

// TestConvertGPTImageDoesNotInjectResolutionOrAspectRatio codifies the
// pass-through contract: when the caller sends only `size` (pixel form, e.g.
// "2560x1440") and does not supply `resolution` or `aspect_ratio`, the
// upstream payload MUST NOT have those fields auto-derived. The earlier
// applySize() helper reverse-derived them with a broken tier mapping
// (anything ≥ 1080 short-side → "1080p"), which produced a contradictory
// resolution="1080p" / size="2560x1440" pair and triggered 502s on
// 16:9 1440p / 4K requests.
func TestConvertGPTImageDoesNotInjectResolutionOrAspectRatio(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGPTImage2TextToImage,
		Prompt: "high-fidelity QHD mockup",
		Size:   "2560x1440",
	}

	body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelGPTImage2TextToImage}})
	if err != nil {
		t.Fatal(err)
	}

	assertInput(t, body, "size", "2560x1440")
	if _, hasAspect := body["aspect_ratio"]; hasAspect {
		t.Fatalf("aspect_ratio must not be auto-derived from size: %#v", body)
	}
	if _, hasResolution := body["resolution"]; hasResolution {
		t.Fatalf("resolution must not be auto-derived from size: %#v", body)
	}
}

func TestConvertGPTImageImageToImageUsesSingleSub2APIImageReference(t *testing.T) {
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

	got, ok := body["images"].([]map[string]string)
	if !ok {
		t.Fatalf("images should be Sub2API image_url objects, got %T: %#v", body["images"], body["images"])
	}
	if len(got) != 1 || got[0]["image_url"] != imageURL {
		t.Fatalf("images = %#v want image_url %q", got, imageURL)
	}
	if _, hasImage := body["image"]; hasImage {
		t.Fatalf("input should not include image key: %#v", body)
	}
	if _, hasInputURLs := body["input_urls"]; hasInputURLs {
		t.Fatalf("input should not include input_urls: %#v", body)
	}
}

func TestConvertGPTImageImageToImageConvertsImageArrayToSub2APIImageURLs(t *testing.T) {
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

	got, ok := body["images"].([]map[string]string)
	if !ok {
		t.Fatalf("images should be Sub2API image_url objects, got %T: %#v", body["images"], body["images"])
	}
	if len(got) != 1 || got[0]["image_url"] != imageURL {
		t.Fatalf("images = %#v want image_url %q", got, imageURL)
	}
	if _, hasImage := body["image"]; hasImage {
		t.Fatalf("input should not include image key: %#v", body)
	}
	if _, hasInputURLs := body["input_urls"]; hasInputURLs {
		t.Fatalf("input should not include input_urls: %#v", body)
	}
}

func TestConvertGPTImageImageToImageDoesNotLeakImageBracketMetadata(t *testing.T) {
	a := &TaskAdaptor{}
	const imageURL = "https://example.com/reference.png"
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGPTImage2ImageToImage,
		Prompt: "use the reference",
		Images: []string{imageURL},
		Size:   "1440x2560",
		Metadata: map[string]any{
			"image[]": []any{imageURL},
		},
	}

	body, err := a.convertToRequestPayload(&req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: ModelGPTImage2ImageToImage}})
	if err != nil {
		t.Fatal(err)
	}

	got, ok := body["images"].([]map[string]string)
	if !ok {
		t.Fatalf("images should be Sub2API image_url objects, got %T: %#v", body["images"], body["images"])
	}
	if len(got) != 1 || got[0]["image_url"] != imageURL {
		t.Fatalf("images = %#v want image_url %q", got, imageURL)
	}
	if _, hasBracketImage := body["image[]"]; hasBracketImage {
		t.Fatalf("input should not leak image[] metadata: %#v", body)
	}
	if _, hasImage := body["image"]; hasImage {
		t.Fatalf("input should not include image key: %#v", body)
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
			_, err := doSyncImageGeneration(t.Context(), server.URL, "test-key", tc.model, body)
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
	urlResult, err := parseSyncImageGenerationResult(ModelGPTImage2TextToImage, []byte(`{"created":1,"data":[{"url":"https://example.com/out.png"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if urlResult != "https://example.com/out.png" {
		t.Fatalf("url result = %q", urlResult)
	}

	b64Result, err := parseSyncImageGenerationResult(ModelGPTImage2TextToImage, []byte(`{"created":1,"data":[{"b64_json":"abc123"}]}`))
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

// ─── Gemini image tests ───────────────────────────────────────────────────────

func TestBuildGeminiImageRequestTextOnly(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "a shiba inu in a spacesuit",
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Contents) != 1 {
		t.Fatalf("contents len = %d", len(got.Contents))
	}
	if got.Contents[0].Role != "user" {
		t.Fatalf("role = %q", got.Contents[0].Role)
	}
	parts := got.Contents[0].Parts
	if len(parts) != 1 {
		t.Fatalf("parts len = %d, want 1 (text only)", len(parts))
	}
	if parts[0].Text != req.Prompt {
		t.Fatalf("text = %q", parts[0].Text)
	}
	if parts[0].FileData != nil {
		t.Fatalf("expected no fileData for text-only request")
	}
	modalities := got.GenerationConfig.ResponseModalities
	if len(modalities) != 2 || modalities[0] != "TEXT" || modalities[1] != "IMAGE" {
		t.Fatalf("responseModalities = %v", modalities)
	}
}

func TestBuildGeminiImageRequestWithSingleInputURL(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "convert to van gogh style",
		Image:  "https://example.com/photo.jpg",
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	parts := got.Contents[0].Parts
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want 2 (text + 1 image)", len(parts))
	}
	if parts[0].Text != req.Prompt {
		t.Fatalf("text part = %q", parts[0].Text)
	}
	fd := parts[1].FileData
	if fd == nil {
		t.Fatal("expected fileData in second part")
	}
	if fd.FileURI != req.Image {
		t.Fatalf("fileUri = %q, want %q", fd.FileURI, req.Image)
	}
	if fd.MimeType != "image/jpeg" {
		t.Fatalf("mimeType = %q", fd.MimeType)
	}
}

func TestBuildGeminiImageRequestWithMultipleInputURLs(t *testing.T) {
	urls := []string{
		"https://example.com/img1.png",
		"https://example.com/img2.jpg",
		"https://example.com/img3.webp",
	}
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "merge these styles",
		Images: urls,
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	parts := got.Contents[0].Parts
	if len(parts) != 4 {
		t.Fatalf("parts len = %d, want 4 (text + 3 images)", len(parts))
	}
	for i, u := range urls {
		fd := parts[i+1].FileData
		if fd == nil {
			t.Fatalf("parts[%d].fileData is nil", i+1)
		}
		if fd.FileURI != u {
			t.Fatalf("parts[%d].fileUri = %q, want %q", i+1, fd.FileURI, u)
		}
	}
}

func TestBuildGeminiImageRequestWithInputURLsFromMetadata(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "style transfer",
		Metadata: map[string]any{
			"input_urls": []any{"https://example.com/ref.png"},
		},
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	parts := got.Contents[0].Parts
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want 2 (text + 1 from input_urls)", len(parts))
	}
	if parts[1].FileData == nil || parts[1].FileData.FileURI != "https://example.com/ref.png" {
		t.Fatalf("unexpected fileData: %+v", parts[1].FileData)
	}
}

func TestBuildGeminiImageRequestDeduplicatesURLs(t *testing.T) {
	const dupURL = "https://example.com/same.png"
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "test dedup",
		Image:  dupURL,
		Images: []string{dupURL, "https://example.com/other.jpg"},
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	// text + dupURL (once) + other = 3 parts total
	parts := got.Contents[0].Parts
	if len(parts) != 3 {
		t.Fatalf("parts len = %d, want 3 after dedup", len(parts))
	}
}

func TestBuildGeminiImageRequestWithAspectRatioAndImageSize(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "wide banner",
		Metadata: map[string]any{
			"aspect_ratio": "16:9",
			"image_size":   "2K",
		},
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg := got.GenerationConfig.ImageConfig
	if cfg == nil {
		t.Fatal("expected imageConfig to be set")
	}
	if cfg.AspectRatio != "16:9" {
		t.Fatalf("aspectRatio = %q", cfg.AspectRatio)
	}
	if cfg.ImageSize != "2K" {
		t.Fatalf("imageSize = %q", cfg.ImageSize)
	}
}

func TestBuildGeminiImageRequestInvalidAspectRatioDropped(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "test",
		Metadata: map[string]any{
			"aspect_ratio": "3:7", // not in the valid set
		},
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Invalid aspect_ratio should be silently dropped → no imageConfig block.
	if got.GenerationConfig.ImageConfig != nil {
		t.Fatalf("expected imageConfig to be nil for invalid aspect_ratio, got %+v", got.GenerationConfig.ImageConfig)
	}
}

func TestBuildGeminiImageRequestRequiresPrompt(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model: ModelGeminiFlashImage,
	}
	_, err := buildGeminiImageRequest(&req, nil)
	if err == nil {
		t.Fatal("expected error for missing prompt")
	}
}

func TestBuildGeminiImageRequestCapsAt14Images(t *testing.T) {
	urls := make([]string, 20)
	for i := range urls {
		urls[i] = "https://example.com/" + string(rune('a'+i)) + ".jpg"
	}
	req := relaycommon.TaskSubmitReq{
		Model:  ModelGeminiFlashImage,
		Prompt: "merge all",
		Images: urls,
	}
	got, err := buildGeminiImageRequest(&req, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 1 text + 14 images = 15 parts
	if len(got.Contents[0].Parts) != 15 {
		t.Fatalf("parts len = %d, want 15 (capped at 14 images)", len(got.Contents[0].Parts))
	}
}

func TestParseGeminiImageResultExtractsBase64(t *testing.T) {
	respJSON := `{
		"candidates": [{
			"content": {
				"parts": [
					{"text": ""},
					{"inlineData": {"data": "abc123xyz", "mimeType": "image/jpeg"}}
				],
				"role": "model"
			},
			"finishReason": "STOP"
		}],
		"modelVersion": "gemini-3.1-flash-image"
	}`
	result, err := parseGeminiImageResult([]byte(respJSON))
	if err != nil {
		t.Fatal(err)
	}
	if result != "data:image/jpeg;base64,abc123xyz" {
		t.Fatalf("result = %q", result)
	}
}

func TestParseGeminiImageResultErrorResponse(t *testing.T) {
	respJSON := `{
		"error": {
			"code": 400,
			"message": "invalid request: prompt too long",
			"status": "INVALID_ARGUMENT"
		}
	}`
	_, err := parseGeminiImageResult([]byte(respJSON))
	if err == nil {
		t.Fatal("expected error for upstream error response")
	}
	if !strings.Contains(err.Error(), "invalid request") {
		t.Fatalf("error = %q, want to contain upstream message", err.Error())
	}
}

func TestParseGeminiImageResultNoImageData(t *testing.T) {
	respJSON := `{
		"candidates": [{
			"content": {
				"parts": [{"text": "I cannot generate that image."}],
				"role": "model"
			},
			"finishReason": "SAFETY"
		}]
	}`
	_, err := parseGeminiImageResult([]byte(respJSON))
	if err == nil {
		t.Fatal("expected error when no inlineData present")
	}
}

func TestResolveGeminiImageSizeDefaults(t *testing.T) {
	cases := []struct {
		name     string
		metadata map[string]any
		want     string
	}{
		{name: "no metadata → 1K default", metadata: nil, want: "1K"},
		{name: "explicit 2K", metadata: map[string]any{"image_size": "2K"}, want: "2K"},
		{name: "explicit 4K", metadata: map[string]any{"image_size": "4K"}, want: "4K"},
		{name: "explicit 512", metadata: map[string]any{"image_size": "512"}, want: "512"},
		{name: "camelCase alias imageSize", metadata: map[string]any{"imageSize": "4K"}, want: "4K"},
		{name: "invalid value → 1K", metadata: map[string]any{"image_size": "8K"}, want: "1K"},
		{name: "lowercase 2k normalised", metadata: map[string]any{"image_size": "2k"}, want: "2K"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := relaycommon.TaskSubmitReq{Metadata: tc.metadata}
			got := resolveGeminiImageSize(req)
			if got != tc.want {
				t.Fatalf("resolveGeminiImageSize = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEstimateBillingGeminiFlashImageResolutionRatios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name      string
		imageSize string
		wantRatio float64
	}{
		{name: "512 is cheapest", imageSize: "512", wantRatio: 1.0 / 3.0},
		{name: "1K is base", imageSize: "1K", wantRatio: 1.0},
		{name: "2K scales correctly", imageSize: "2K", wantRatio: 5.0 / 3.0},
		{name: "4K scales correctly", imageSize: "4K", wantRatio: 8.0 / 3.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &TaskAdaptor{}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Set("task_request", relaycommon.TaskSubmitReq{
				Metadata: map[string]any{"image_size": tc.imageSize},
			})
			ratios := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: ModelGeminiFlashImage})
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

func TestBuildRequestURLGeminiUsesKeyQueryParam(t *testing.T) {
	a := &TaskAdaptor{}
	a.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelBaseUrl: "https://api.sub2api.com/",
		ApiKey:         "my-secret-key",
	}})
	got, err := a.BuildRequestURL(&relaycommon.RelayInfo{OriginModelName: ModelGeminiFlashImage})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://api.sub2api.com" + UpstreamPathGeminiFlashImage + "?key=my-secret-key"
	if got != want {
		t.Fatalf("BuildRequestURL = %q, want %q", got, want)
	}
}

func TestDoSyncImageGenerationGeminiUsesKeyQueryParam(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		if r.Header.Get("Authorization") != "" {
			http.Error(w, "unexpected Authorization header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"parts": [{"text":""}, {"inlineData":{"data":"abc","mimeType":"image/jpeg"}}],
					"role": "model"
				},
				"finishReason": "STOP"
			}]
		}`))
	}))
	defer server.Close()

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"test"}]}],"generationConfig":{"responseModalities":["TEXT","IMAGE"]}}`)
	_, err := doSyncImageGeneration(t.Context(), server.URL, "my-api-key", ModelGeminiFlashImage, body)
	if err != nil {
		t.Fatal(err)
	}
	if seenQuery != "key=my-api-key" {
		t.Fatalf("query = %q, want key=my-api-key", seenQuery)
	}
}

func TestGuessImageMIMEFromURL(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://example.com/photo.jpg", "image/jpeg"},
		{"https://example.com/photo.jpeg", "image/jpeg"},
		{"https://example.com/photo.PNG", "image/png"},
		{"https://example.com/anim.gif", "image/gif"},
		{"https://example.com/modern.webp", "image/webp"},
		{"https://example.com/image?format=jpg&w=100", "image/jpeg"},
		{"https://example.com/noext", "image/jpeg"},
	}
	for _, tc := range cases {
		got := guessImageMIMEFromURL(tc.url)
		if got != tc.want {
			t.Fatalf("guessImageMIMEFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}
