package sub2api_async

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

type openAIImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url"`
		B64JSON       string `json:"b64_json"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Error *dto.OpenAIVideoError `json:"error"`
}

// Single weight table shared by every Sub2API GPT Image 2 variant — t2i and
// i2i are priced identically per resolution tier today. Keeping one source
// prevents drift when prices change (the previous duplicated-per-model table
// required updating two parallel maps in lock-step). If a future variant
// needs different weights, add it as a separate constant and reference it
// from imageResolutionRatioWeights below — do not silently fork this map.
var gptImage2ResolutionWeights = map[string]float64{
	"1K": 3,
	"2K": 5,
	"4K": 8,
}

var imageResolutionRatioWeights = map[string]map[string]float64{
	ModelGPTImage2TextToImage:  gptImage2ResolutionWeights,
	ModelGPTImage2ImageToImage: gptImage2ResolutionWeights,
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(DefaultBaseURL, "/")
	if info != nil && strings.TrimSpace(info.ChannelBaseUrl) != "" {
		a.baseURL = strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
	}
	if info != nil {
		a.apiKey = info.ApiKey
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	if info == nil {
		return nil
	}
	weights, ok := imageResolutionRatioWeights[info.OriginModelName]
	if !ok {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	resolution := normalizeImageResolution(resolveBillingResolution(req))
	if resolution == "" {
		resolution = "1K"
	}
	weight, ok := weights[resolution]
	if !ok {
		return nil
	}
	baseWeight := weights["1K"]
	if baseWeight == 0 {
		return nil
	}
	return map[string]float64{"resolution": weight / baseWeight}
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + UpstreamPathImagesGenerations, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, err
	}
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	// Sub2API-async exposes an async contract to clients while the upstream
	// only supports synchronous OpenAI-compatible image generation. Do not call
	// upstream on the request path; DoResponse schedules the sync call after
	// the local task row is inserted.
	body, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, fmt.Errorf("read request body failed: %w", err)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}, nil
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	requestBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	publicTaskID := strings.TrimSpace(info.PublicTaskID)
	if publicTaskID == "" {
		publicTaskID = model.GenerateTaskID()
		info.PublicTaskID = publicTaskID
	}
	if info.TaskRelayInfo != nil {
		// Capture the inbound request id NOW (synchronously, while the gin
		// context is still valid). The closure below fires after the task row
		// is inserted — possibly after this handler has returned — and uses
		// it to seed the background ctx so logger.LogError etc. carry the
		// original request id instead of "SYSTEM". context.Background() is
		// still the base so the goroutine outlives the http request.
		var requestID string
		if c != nil && c.Request != nil {
			if v, ok := c.Request.Context().Value(common.RequestIdKey).(string); ok {
				requestID = v
			}
		}
		info.AfterTaskInserted = func(localTaskID int64) {
			bgCtx := context.Background()
			if requestID != "" {
				bgCtx = context.WithValue(bgCtx, common.RequestIdKey, requestID)
			}
			a.scheduleSyncImageGeneration(bgCtx, localTaskID, requestBody)
		}
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = publicTaskID
	ov.TaskID = publicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return publicTaskID, requestBody, nil
}

func (a *TaskAdaptor) FetchTask(_ string, _ string, body map[string]any, _ string) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)
	task, exists, err := model.GetByOnlyTaskId(taskID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	respBody, err := a.ConvertToOpenAIAsyncImage(task)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
	}, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	return parseOpenAIAsyncImageTaskResult(respBody)
}

func (a *TaskAdaptor) scheduleSyncImageGeneration(ctx context.Context, localTaskID int64, requestBody []byte) {
	baseURL := a.baseURL
	apiKey := a.apiKey
	gopool.Go(func() {
		runSub2APIAsyncImageGeneration(ctx, localTaskID, baseURL, apiKey, requestBody)
	})
}

func runSub2APIAsyncImageGeneration(ctx context.Context, localTaskID int64, baseURL, apiKey string, requestBody []byte) {
	task, exists, err := model.GetTaskByID(localTaskID)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("sub2api async get task failed: %v", err))
		return
	}
	if !exists {
		logger.LogError(ctx, fmt.Sprintf("sub2api async task not found: %d", localTaskID))
		return
	}

	startedAt := time.Now().Unix()
	task.Status = model.TaskStatusInProgress
	task.Progress = taskcommon.ProgressInProgress
	task.StartTime = startedAt
	if won, err := task.UpdateWithStatus(model.TaskStatusNotStart); err != nil {
		logger.LogError(ctx, fmt.Sprintf("sub2api async mark running failed: %v", err))
		return
	} else if !won {
		// Another goroutine already moved this task out of NotStart. Surface
		// it so duplicate scheduling (gopool re-entry, process race, etc.) is
		// observable — the success / failure CAS sites further down already
		// log this case; keep all three CAS branches symmetric.
		logger.LogWarn(ctx, fmt.Sprintf("sub2api async task status changed before start update: %s", task.TaskID))
		return
	}

	respBody, err := doSyncImageGeneration(ctx, baseURL, apiKey, requestBody)
	if err != nil {
		markSub2APIAsyncTaskFailed(ctx, task, err.Error())
		return
	}

	resultURL, err := parseSyncImageGenerationResult(respBody)
	if err != nil {
		markSub2APIAsyncTaskFailed(ctx, task, err.Error())
		return
	}

	task.Data = respBody
	task.PrivateData.ResultURL = resultURL
	task.Status = model.TaskStatusSuccess
	task.Progress = taskcommon.ProgressComplete
	task.FinishTime = time.Now().Unix()
	if won, err := task.UpdateWithStatus(model.TaskStatusInProgress); err != nil {
		logger.LogError(ctx, fmt.Sprintf("sub2api async mark success failed: %v", err))
	} else if !won {
		logger.LogWarn(ctx, fmt.Sprintf("sub2api async task status changed before success update: %s", task.TaskID))
	}
}

func doSyncImageGeneration(ctx context.Context, baseURL, apiKey string, requestBody []byte) ([]byte, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	requestPath, err := syncImageGenerationRequestPath(requestBody)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, syncImageGenerationTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+requestPath, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("new upstream request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: syncImageGenerationTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream sync image request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upstream response failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// Truncate the body before it becomes part of an error message that
		// will land in task.FailReason. Upstream 502s commonly return full
		// HTML error pages (multiple KB / occasionally MB); bloating the
		// task row hurts admin list queries and rotating logs.
		return nil, fmt.Errorf("upstream status %d: %s", resp.StatusCode, truncateUpstreamErrorBody(respBody))
	}
	return respBody, nil
}

// syncImageGenerationTimeout caps how long the deferred sync upstream call may
// run. Sub2API's t2i / edits endpoints typically respond in < 90s; the 5-minute
// ceiling is the worst-case allowance and is applied to BOTH the request
// context deadline and the underlying http.Client timeout so a single source
// of truth governs the bound.
const syncImageGenerationTimeout = 5 * time.Minute

// truncateUpstreamErrorBody keeps the leading 1KB of an upstream error
// response and appends a marker indicating the original size, so admins can
// still tell how big the truncated payload was. Operates on bytes not runes
// because the body may not be UTF-8 (HTML error pages can be other encodings).
const upstreamErrorBodyMaxBytes = 1024

func truncateUpstreamErrorBody(body []byte) string {
	if len(body) <= upstreamErrorBodyMaxBytes {
		return string(body)
	}
	return fmt.Sprintf("%s...(truncated, %d bytes total)", string(body[:upstreamErrorBodyMaxBytes]), len(body))
}

func syncImageGenerationRequestPath(requestBody []byte) (string, error) {
	var req struct {
		Model string `json:"model"`
	}
	if err := common.Unmarshal(requestBody, &req); err != nil {
		return "", errors.Wrap(err, "unmarshal upstream request body failed")
	}
	switch strings.TrimSpace(req.Model) {
	case ModelGPTImage2TextToImage:
		return UpstreamPathImagesGenerations, nil
	case ModelGPTImage2ImageToImage:
		return UpstreamPathImagesEdits, nil
	default:
		return "", fmt.Errorf("unsupported model: %s", req.Model)
	}
}

func parseSyncImageGenerationResult(respBody []byte) (string, error) {
	var res openAIImageGenerationResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return "", errors.Wrap(err, "unmarshal upstream image response failed")
	}
	if res.Error != nil {
		// Some upstreams return code-only errors (e.g. {"error":{"code":"quota_exceeded"}})
		// without a human-readable message. Surface whichever is present so the task's
		// FailReason names the actual upstream error class instead of degrading to
		// the misleading "no image data" branch further down.
		if reason := firstNonEmpty(res.Error.Message, res.Error.Code); reason != "" {
			return "", fmt.Errorf("upstream image error: %s", reason)
		}
	}
	for _, item := range res.Data {
		if strings.TrimSpace(item.URL) != "" {
			return strings.TrimSpace(item.URL), nil
		}
		if b64 := strings.TrimSpace(item.B64JSON); b64 != "" {
			return "data:" + detectImageMIME(b64) + ";base64," + b64, nil
		}
	}
	return "", fmt.Errorf("upstream image response has no image data")
}

// detectImageMIME inspects the first few decoded bytes of a base64 image
// payload to figure out the correct MIME type. The previous code hardcoded
// "image/png" for every b64 response, which mislabeled JPEG / WebP / GIF
// outputs and broke strict downstream MIME consumers (browsers tolerate it,
// some HTTP clients and CDN edge rules do not). Falls back to image/png only
// when the magic bytes match nothing recognised — that matches the previous
// default so we never produce a *worse* label than before.
func detectImageMIME(b64 string) string {
	// Only need the first ~12 decoded bytes. Decoding a small prefix is cheap
	// and avoids materialising the full image in memory.
	const sniffLen = 16
	enc := base64.StdEncoding
	prefixLen := enc.EncodedLen(sniffLen)
	if len(b64) < prefixLen {
		prefixLen = len(b64)
	}
	// base64 length must be a multiple of 4 to decode; round down.
	prefixLen -= prefixLen % 4
	if prefixLen <= 0 {
		return "image/png"
	}
	buf, err := enc.DecodeString(b64[:prefixLen])
	if err != nil || len(buf) < 4 {
		return "image/png"
	}
	switch {
	case bytes.HasPrefix(buf, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(buf, []byte{0x89, 0x50, 0x4E, 0x47}):
		return "image/png"
	case bytes.HasPrefix(buf, []byte("GIF87a")), bytes.HasPrefix(buf, []byte("GIF89a")):
		return "image/gif"
	case len(buf) >= 12 && bytes.Equal(buf[:4], []byte("RIFF")) && bytes.Equal(buf[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return "image/png"
	}
}

func markSub2APIAsyncTaskFailed(ctx context.Context, task *model.Task, reason string) {
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = strings.TrimSpace(reason)
	if task.FailReason == "" {
		task.FailReason = "sub2api async image generation failed"
	}
	task.FinishTime = time.Now().Unix()
	if won, err := task.UpdateWithStatus(model.TaskStatusInProgress); err != nil {
		logger.LogError(ctx, fmt.Sprintf("sub2api async mark failed failed: %v", err))
	} else if !won {
		logger.LogWarn(ctx, fmt.Sprintf("sub2api async task status changed before failure update: %s", task.TaskID))
	} else if task.Quota != 0 {
		service.RefundTaskQuota(ctx, task, task.FailReason)
	}
}

func parseOpenAIAsyncImageTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var res struct {
		ID       string `json:"id"`
		TaskID   string `json:"task_id"`
		Status   string `json:"status"`
		Progress any    `json:"progress"`
		URL      string `json:"url"`
		Error    *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := common.Unmarshal(respBody, &res); err != nil {
		return nil, errors.Wrap(err, "unmarshal openai async image result failed")
	}

	taskResult := &relaycommon.TaskInfo{Code: 0, TaskID: firstNonEmpty(res.TaskID, res.ID)}
	switch strings.ToLower(strings.TrimSpace(res.Status)) {
	case dto.VideoStatusQueued, "submitted", "pending":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = normalizeProgress(res.Progress, taskcommon.ProgressQueued)
	case dto.VideoStatusInProgress, "running", "generating":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = normalizeProgress(res.Progress, taskcommon.ProgressInProgress)
	case dto.VideoStatusCompleted, "success", "succeeded":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.Url = firstNonEmpty(res.URL, metadataString(res.Metadata, "url"))
	case dto.VideoStatusFailed, "failure", "fail":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		if res.Error != nil {
			taskResult.Reason = firstNonEmpty(strings.TrimSpace(res.Error.Message), strings.TrimSpace(res.Error.Code))
		}
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
	default:
		// Unknown upstream status string. Previously this branch silently
		// classified the task as InProgress, which left tasks polling forever
		// when the upstream protocol drifted or a channel was misconfigured.
		// Fail the task explicitly so the operator gets a visible signal
		// instead of an indefinite-running phantom.
		rawStatus := strings.TrimSpace(res.Status)
		logger.LogWarn(context.Background(), fmt.Sprintf("sub2api async unknown upstream task status: %q", rawStatus))
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		if rawStatus == "" {
			taskResult.Reason = "upstream returned empty task status"
		} else {
			taskResult.Reason = fmt.Sprintf("unknown upstream task status: %s", rawStatus)
		}
	}
	return taskResult, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	openAIVideo := originTask.ToOpenAIVideo()
	if originTask.FailReason != "" && originTask.Status == model.TaskStatusFailure {
		openAIVideo.Error = &dto.OpenAIVideoError{Message: originTask.FailReason}
	}
	return common.Marshal(openAIVideo)
}

func (a *TaskAdaptor) ConvertToOpenAIAsyncImage(originTask *model.Task) ([]byte, error) {
	out := map[string]any{
		"object":     "sub2api_async.image.generation.task",
		"id":         originTask.TaskID,
		"task_id":    originTask.TaskID,
		"status":     originTask.Status.ToVideoStatus(),
		"progress":   originTask.Progress,
		"model":      originTask.Properties.OriginModelName,
		"created_at": originTask.CreatedAt,
		"updated_at": originTask.UpdatedAt,
	}
	b64, b64Err := firstOpenAIImageBase64(originTask.Data, originTask.GetResultURL())
	if b64Err != nil {
		// Stored upstream payload is malformed. Surface it so admins notice
		// the corruption, but degrade gracefully to the stored result URL
		// so the task response is still served — the alternative is a 500 on
		// every FetchTask hitting this row.
		logger.LogError(context.Background(), fmt.Sprintf("sub2api async task %s base64 extraction failed: %v", originTask.TaskID, b64Err))
	}
	if b64 != "" && originTask.Status != model.TaskStatusFailure {
		out["data"] = []any{map[string]any{"b64_json": b64}}
	} else if u := originTask.GetResultURL(); u != "" && originTask.Status != model.TaskStatusFailure {
		out["data"] = []any{map[string]any{"url": u}}
	}
	if originTask.FailReason != "" {
		out["error"] = map[string]any{"message": originTask.FailReason}
	}
	return common.Marshal(out)
}

func firstOpenAIImageBase64(raw []byte, fallbackURL string) (string, error) {
	if len(raw) != 0 {
		var res openAIImageGenerationResponse
		if err := common.Unmarshal(raw, &res); err != nil {
			// Stored response cannot be parsed; let the caller decide whether
			// to fail or fall back. Returning the URL-shaped fallback here too
			// is harmless — callers that want to know about the parse failure
			// inspect the returned error.
			return dataImageBase64(fallbackURL), errors.Wrap(err, "unmarshal stored sub2api image response failed")
		}
		for _, item := range res.Data {
			if b64 := strings.TrimSpace(item.B64JSON); b64 != "" {
				return b64, nil
			}
			if b64 := dataImageBase64(item.URL); b64 != "" {
				return b64, nil
			}
		}
	}
	return dataImageBase64(fallbackURL), nil
}

func dataImageBase64(dataURL string) string {
	dataURL = strings.TrimSpace(dataURL)
	if !strings.HasPrefix(dataURL, "data:image/") {
		return ""
	}
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 || !strings.Contains(parts[0], ";base64") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (map[string]any, error) {
	modelName := resolveModelName(req, info)
	if modelName == "" {
		return nil, fmt.Errorf("model is required")
	}
	if _, ok := modelConfigs[modelName]; !ok {
		return nil, fmt.Errorf("unsupported model: %s", modelName)
	}

	// The W×H request shape (size, aspect_ratio, resolution) is owned by the
	// client / pipeline. Sub2API-async is a pure pass-through channel: whatever
	// the caller supplies is forwarded verbatim. Previously this branch called
	// applySize() to reverse-derive `aspect_ratio` and `resolution` from a
	// pixel-form `size` like "2560x1440"; that injected fields the caller never
	// asked for and used a broken tier mapping (anything ≥ 1080 short-side was
	// reported as "1080p", so a 1440p QHD request shipped a contradictory
	// resolution="1080p" / size="2560x1440" pair to the actual upstream and
	// triggered a 502). Do not re-introduce that derivation — if the upstream
	// needs aspect_ratio or resolution, the caller must send them explicitly
	// (kie/adaptor.go intentionally still derives them for the KIE channel
	// because that upstream requires them; that pattern is NOT appropriate
	// for sub2api).
	input := map[string]any{"model": modelName}
	if req.Prompt != "" {
		input["prompt"] = req.Prompt
	}
	if req.Size != "" {
		input["size"] = req.Size
	}
	if req.Resolution != "" {
		input["resolution"] = req.Resolution
	}
	// Strip authoritative request fields from metadata before the merge below.
	// Sub2API-async treats `prompt`, `size`, and `resolution` as caller-owned
	// request-shape inputs (read directly off TaskSubmitReq above); allowing
	// metadata to silently override them would create a field-injection face
	// — a caller (or a future caller) could send size=2K with metadata.size=8K
	// and the upstream would see 8K. UnmarshalMetadata already deletes `model`
	// for the same reason (billing-bypass guard); extend the same guard to the
	// other authoritative fields.
	sanitizedMetadata := stripAuthoritativeMetadataFields(req.Metadata)
	if err := taskcommon.UnmarshalMetadata(sanitizedMetadata, &input); err != nil {
		return nil, err
	}

	cfg := getModelConfig(modelName)
	images := requestImages(req)
	if len(images) > 0 {
		if cfg.ImageKey != "" {
			if cfg.ImageURLKey != "" {
				input[cfg.ImageKey] = imageURLObjects(images, cfg.ImageURLKey)
			} else {
				input[cfg.ImageKey] = images
			}
		} else if len(images) == 1 {
			input["image"] = images[0]
		} else {
			input["images"] = images
		}
	}

	return input, nil
}

func imageURLObjects(images []string, key string) []map[string]string {
	objects := make([]map[string]string, 0, len(images))
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" {
			continue
		}
		objects = append(objects, map[string]string{key: image})
	}
	return objects
}

func resolveModelName(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) string {
	modelName := strings.TrimSpace(req.Model)
	if info != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		modelName = strings.TrimSpace(info.UpstreamModelName)
	}
	return modelName
}

func requestImages(req *relaycommon.TaskSubmitReq) []string {
	images := make([]string, 0, len(req.Images)+1)
	seen := make(map[string]struct{}, len(req.Images)+1)
	add := func(image string) {
		image = strings.TrimSpace(image)
		if image == "" {
			return
		}
		if _, ok := seen[image]; ok {
			return
		}
		seen[image] = struct{}{}
		images = append(images, image)
	}
	if req.Image != "" {
		add(req.Image)
	}
	for _, image := range req.Images {
		add(image)
	}
	return images
}

func resolveBillingResolution(req relaycommon.TaskSubmitReq) string {
	// Billing tier lookup. Only consults what the caller explicitly sent:
	// req.Resolution first, then metadata.resolution. Does NOT reverse-derive
	// from pixel size — see convertToRequestPayload for the rationale.
	// Callers that want tier-accurate billing for size-only requests must
	// send `resolution` alongside (e.g. via metadata).
	input := map[string]any{}
	if req.Resolution != "" {
		input["resolution"] = req.Resolution
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &input); err != nil {
		return ""
	}
	resolution, _ := input["resolution"].(string)
	return resolution
}

// stripAuthoritativeMetadataFields returns a copy of metadata with keys that
// duplicate authoritative request fields removed. `model`, `prompt`, `size`,
// and `resolution` are owned by the typed TaskSubmitReq; allowing metadata to
// shadow them would let a caller silently override the request shape after
// the typed fields have already been read.
func stripAuthoritativeMetadataFields(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	sanitized := make(map[string]any, len(metadata))
	for k, v := range metadata {
		switch k {
		case "model", "prompt", "size", "resolution":
			continue
		}
		sanitized[k] = v
	}
	return sanitized
}

func normalizeImageResolution(resolution string) string {
	resolution = strings.ToUpper(strings.TrimSpace(resolution))
	switch resolution {
	case "1K", "2K", "4K":
		return resolution
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func normalizeProgress(progress any, fallback string) string {
	switch v := progress.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			if strings.HasSuffix(strings.TrimSpace(v), "%") {
				return strings.TrimSpace(v)
			}
			return strings.TrimSpace(v) + "%"
		}
	case float64:
		return fmt.Sprintf("%.0f%%", v)
	case int:
		return fmt.Sprintf("%d%%", v)
	}
	return fallback
}
