package sub2api_async

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

var imageResolutionRatioWeights = map[string]map[string]float64{
	ModelGPTImage2TextToImage: {
		"1K": 3,
		"2K": 5,
		"4K": 8,
	},
	ModelGPTImage2ImageToImage: {
		"1K": 3,
		"2K": 5,
		"4K": 8,
	},
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
	return a.baseURL + "/v1/images/generations", nil
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
		info.AfterTaskInserted = func(localTaskID int64) {
			a.scheduleSyncImageGeneration(context.Background(), localTaskID, requestBody)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/images/generations", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("new upstream request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream sync image request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upstream response failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("upstream status %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func parseSyncImageGenerationResult(respBody []byte) (string, error) {
	var res openAIImageGenerationResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return "", errors.Wrap(err, "unmarshal upstream image response failed")
	}
	if res.Error != nil && strings.TrimSpace(res.Error.Message) != "" {
		return "", fmt.Errorf("upstream image error: %s", res.Error.Message)
	}
	for _, item := range res.Data {
		if strings.TrimSpace(item.URL) != "" {
			return strings.TrimSpace(item.URL), nil
		}
		if strings.TrimSpace(item.B64JSON) != "" {
			return "data:image/png;base64," + strings.TrimSpace(item.B64JSON), nil
		}
	}
	return "", fmt.Errorf("upstream image response has no image data")
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
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = normalizeProgress(res.Progress, taskcommon.ProgressInProgress)
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
	if b64 := firstOpenAIImageBase64(originTask.Data, originTask.GetResultURL()); b64 != "" && originTask.Status != model.TaskStatusFailure {
		out["data"] = []any{map[string]any{"b64_json": b64}}
	} else if u := originTask.GetResultURL(); u != "" && originTask.Status != model.TaskStatusFailure {
		out["data"] = []any{map[string]any{"url": u}}
	}
	if originTask.FailReason != "" {
		out["error"] = map[string]any{"message": originTask.FailReason}
	}
	return common.Marshal(out)
}

func firstOpenAIImageBase64(raw []byte, fallbackURL string) string {
	if len(raw) != 0 {
		var res openAIImageGenerationResponse
		if err := common.Unmarshal(raw, &res); err == nil {
			for _, item := range res.Data {
				if b64 := strings.TrimSpace(item.B64JSON); b64 != "" {
					return b64
				}
				if b64 := dataImageBase64(item.URL); b64 != "" {
					return b64
				}
			}
		}
	}
	return dataImageBase64(fallbackURL)
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

	input := map[string]any{"model": modelName}
	if req.Prompt != "" {
		input["prompt"] = req.Prompt
	}
	if req.Size != "" {
		input["size"] = req.Size
	}
	if req.Size != "" {
		applySize(input, req.Size)
	}
	if req.Resolution != "" {
		input["resolution"] = req.Resolution
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &input); err != nil {
		return nil, err
	}

	cfg := getModelConfig(modelName)
	images := requestImages(req)
	if len(images) == 1 {
		input["image"] = images[0]
	} else if len(images) > 1 {
		input["images"] = images
	}
	if len(images) > 0 && cfg.ImageKey != "" {
		input[cfg.ImageKey] = images
	}

	return input, nil
}

func resolveModelName(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) string {
	modelName := strings.TrimSpace(req.Model)
	if info != nil && strings.TrimSpace(info.UpstreamModelName) != "" {
		modelName = strings.TrimSpace(info.UpstreamModelName)
	}
	return modelName
}

func requestImages(req *relaycommon.TaskSubmitReq) []string {
	images := append([]string(nil), req.Images...)
	if req.Image != "" {
		images = append([]string{req.Image}, images...)
	}
	return images
}

func resolveBillingResolution(req relaycommon.TaskSubmitReq) string {
	input := map[string]any{}
	if req.Size != "" {
		applySize(input, req.Size)
	}
	if req.Resolution != "" {
		input["resolution"] = req.Resolution
	}
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &input); err != nil {
		return ""
	}
	resolution, _ := input["resolution"].(string)
	return resolution
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

func applySize(input map[string]any, size string) {
	size = strings.TrimSpace(size)
	if strings.Contains(size, "x") {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			w, wErr := strconv.Atoi(parts[0])
			h, hErr := strconv.Atoi(parts[1])
			if wErr == nil && hErr == nil && w > 0 && h > 0 {
				input["aspect_ratio"] = simplifyRatio(w, h)
				input["resolution"] = resolutionFromDimensions(w, h)
				return
			}
		}
	}
	if strings.HasSuffix(strings.ToLower(size), "p") || strings.HasSuffix(strings.ToUpper(size), "K") {
		input["resolution"] = size
	}
}

func simplifyRatio(w, h int) string {
	g := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/g, h/g)
}

func resolutionFromDimensions(w, h int) string {
	shorter := min(w, h)
	if shorter >= 1080 {
		return "1080p"
	}
	if shorter >= 720 {
		return "720p"
	}
	return "480p"
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
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
