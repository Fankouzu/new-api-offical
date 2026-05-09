package fal

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

type falSubmitRequest struct {
	Prompt    string   `json:"prompt"`
	ImageSize any      `json:"image_size,omitempty"`
	Quality   string   `json:"quality,omitempty"`
	NumImages int      `json:"num_images,omitempty"`
	OutputFmt string   `json:"output_format,omitempty"`
	ImageURLs []string `json:"image_urls,omitempty"`
	MaskURL   string   `json:"mask_url,omitempty"`
}

type falSubmitResponse struct {
	RequestID string `json:"request_id"`
}

type falStatusResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Logs      []struct {
		Message string `json:"message"`
	} `json:"logs,omitempty"`
}

type falLogEntry struct {
	Message string `json:"message"`
}

type falResultResponse struct {
	RequestID string           `json:"request_id"`
	Status    string           `json:"status"`
	Data      struct {
		Images []falResultImage `json:"images"`
	} `json:"data"`
	Images []falResultImage `json:"images"`
	// Error fields — fal.ai may return these without a "status" field on failure.
	Detail string        `json:"detail"`
	Error  string        `json:"error"`
	Logs   []falLogEntry `json:"logs,omitempty"`
}

type falResultImage struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
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
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	quality := "medium"
	if q, ok := req.Metadata["quality"].(string); ok && q != "" {
		quality = strings.ToLower(strings.TrimSpace(q))
	}
	size := strings.ReplaceAll(strings.TrimSpace(req.Size), " ", "")
	if size == "" {
		size = "1024x1024"
	}
	multiplier, ok := getPricingMultiplier(info.OriginModelName, size, quality)
	if !ok {
		return nil
	}
	return map[string]float64{"quality": multiplier}
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	model := info.UpstreamModelName
	if model == "" {
		model = info.OriginModelName
	}
	if model == "" {
		return "", fmt.Errorf("fal: model name is required")
	}
	return fmt.Sprintf("%s/%s", a.baseURL, model), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Key "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	cfg := getModelConfig(info.OriginModelName)
	quality := "medium"
	if q, ok := taskReq.Metadata["quality"].(string); ok && q != "" {
		quality = strings.ToLower(strings.TrimSpace(q))
	}

	imageSize := taskReq.Size
	if imageSize == "" {
		imageSize = "square_hd"
	}

	req := falSubmitRequest{
		Prompt:    taskReq.Prompt,
		Quality:   quality,
		NumImages: 1,
		OutputFmt: "png",
	}

	if taskReq.Size != "" {
		if _, _, err := parseDimensions(taskReq.Size); err == nil {
			w, h, _ := parseDimensions(taskReq.Size)
			req.ImageSize = map[string]int{"width": w, "height": h}
		} else {
			req.ImageSize = imageSize
		}
	} else {
		req.ImageSize = imageSize
	}

	if cfg.ImageKey == "image_urls" && len(taskReq.Images) > 0 {
		req.ImageURLs = taskReq.Images
	}

	data, err := common.Marshal(req)
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

	var taskResp falSubmitResponse
	if err := common.Unmarshal(responseBody, &taskResp); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", string(responseBody)), "unmarshal_failed", http.StatusInternalServerError)
	}
	if taskResp.RequestID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("request_id is empty"), "invalid_response", http.StatusInternalServerError)
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return taskResp.RequestID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	modelName, _ := body["model_name"].(string)
	if modelName == "" {
		modelName = ModelGPTImage2T2I
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	uri := fmt.Sprintf("%s/%s/requests/%s", baseURL, modelName, url.PathEscape(taskID))
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Key "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	// Let the caller read the error body — 4xx/5xx may still contain parseable status.
	return resp, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var res falResultResponse
	if err := common.Unmarshal(respBody, &res); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := &relaycommon.TaskInfo{Code: 0, TaskID: res.RequestID}
	status := strings.ToLower(strings.TrimSpace(res.Status))
	images := res.Data.Images
	if len(images) == 0 {
		images = res.Images
	}
	switch status {
	case "completed":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		if len(images) > 0 {
			taskResult.Url = images[0].URL
		}
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
	case "in_progress", "in_queue":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	default:
		// Check for error responses that don't contain a "status" field.
		if res.Detail != "" {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Progress = "100%"
			taskResult.Reason = "fal: " + res.Detail
		} else if res.Error != "" {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Progress = "100%"
			taskResult.Reason = "fal: " + res.Error
		} else if hasErrorInLogs(res.Logs) {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Progress = "100%"
			taskResult.Reason = "fal: error in logs"
		} else if len(images) > 0 {
			// Status field may be absent; presence of images means completion.
			taskResult.Status = model.TaskStatusSuccess
			taskResult.Progress = "100%"
			taskResult.Url = images[0].URL
		} else {
			taskResult.Status = model.TaskStatusInProgress
			taskResult.Progress = "50%"
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

func (a *TaskAdaptor) ConvertToOpenAIVideo(_ *model.Task) ([]byte, error) {
	return nil, fmt.Errorf("fal: video conversion not supported")
}

func (a *TaskAdaptor) ConvertToOpenAIAsyncImage(originTask *model.Task) ([]byte, error) {
	resultURL := originTask.PrivateData.ResultURL
	if resultURL == "" {
		return nil, fmt.Errorf("fal: result URL is empty")
	}
	imgObj := map[string]any{
		"url": resultURL,
	}
	resp := map[string]any{
		"created": originTask.CreatedAt,
		"data":    []any{imgObj},
	}
	return common.Marshal(resp)
}

func parseDimensions(size string) (int, int, error) {
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size format: %s", size)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

func hasErrorInLogs(logs []falLogEntry) bool {
	for _, log := range logs {
		msg := strings.ToLower(log.Message)
		if strings.Contains(msg, "error") ||
			strings.Contains(msg, "failed") ||
			strings.Contains(msg, "policy") ||
			strings.Contains(msg, "violation") ||
			strings.Contains(msg, "rejected") ||
			strings.Contains(msg, "blocked") ||
			strings.Contains(msg, "content") {
			return true
		}
	}
	return false
}
