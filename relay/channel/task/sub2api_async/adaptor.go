package sub2api_async

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

type createTaskResponse struct {
	ID     string                `json:"id"`
	TaskID string                `json:"task_id"`
	Status string                `json:"status"`
	Error  *dto.OpenAIVideoError `json:"error"`
	Code   int                   `json:"code"`
	Msg    string                `json:"msg"`
	Data   struct {
		TaskID string `json:"taskId"`
	} `json:"data"`
}

type recordInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskID     string `json:"taskId"`
		Model      string `json:"model"`
		State      string `json:"state"`
		ResultJSON string `json:"resultJson"`
		FailCode   string `json:"failCode"`
		FailMsg    string `json:"failMsg"`
	} `json:"data"`
}

type resultJSONPayload struct {
	ResultURLs    []string `json:"resultUrls"`
	FirstFrameURL []string `json:"firstFrameUrl"`
	LastFrameURL  []string `json:"lastFrameUrl"`
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
	return a.baseURL + "/v1/images/generations/async", nil
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
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var taskResp createTaskResponse
	if err := common.Unmarshal(responseBody, &taskResp); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", string(responseBody)), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if taskResp.Code != 0 && taskResp.Code != http.StatusOK {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("sub2api async error: %s", taskResp.Msg), strconv.Itoa(taskResp.Code), http.StatusBadRequest)
	}
	if taskResp.Error != nil && strings.TrimSpace(taskResp.Error.Message) != "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("sub2api async error: %s", taskResp.Error.Message), "sub2api_async_error", http.StatusBadRequest)
	}
	upstreamTaskID := firstNonEmpty(taskResp.Data.TaskID, taskResp.TaskID, taskResp.ID)
	if upstreamTaskID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return upstreamTaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	uri := fmt.Sprintf("%s/v1/images/generations/%s", baseURL, url.PathEscape(taskID))
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var res recordInfoResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	if res.Data.State == "" && res.Data.TaskID == "" && res.Code == 0 {
		return parseOpenAIAsyncImageTaskResult(respBody)
	}
	if res.Code != 0 && res.Code != http.StatusOK {
		return &relaycommon.TaskInfo{Code: res.Code, Status: model.TaskStatusFailure, Progress: taskcommon.ProgressComplete, Reason: res.Msg}, nil
	}

	taskResult := &relaycommon.TaskInfo{Code: 0, TaskID: res.Data.TaskID}
	switch strings.ToLower(strings.TrimSpace(res.Data.State)) {
	case "waiting":
		taskResult.Status = model.TaskStatusSubmitted
		taskResult.Progress = taskcommon.ProgressSubmitted
	case "queuing":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = taskcommon.ProgressQueued
	case "generating":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	case "success":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.Url = firstResultURL(res.Data.ResultJSON)
	case "fail":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		taskResult.Reason = strings.TrimSpace(res.Data.FailMsg)
		if taskResult.Reason == "" {
			taskResult.Reason = strings.TrimSpace(res.Data.FailCode)
		}
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	}
	return taskResult, nil
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
	if u := originTask.GetResultURL(); u != "" {
		out["url"] = u
	}
	if originTask.FailReason != "" {
		out["error"] = map[string]any{"message": originTask.FailReason}
	}
	return common.Marshal(out)
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

func firstResultURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var payload resultJSONPayload
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return ""
	}
	for _, urls := range [][]string{payload.ResultURLs, payload.FirstFrameURL, payload.LastFrameURL} {
		for _, u := range urls {
			if strings.TrimSpace(u) != "" {
				return u
			}
		}
	}
	return ""
}
