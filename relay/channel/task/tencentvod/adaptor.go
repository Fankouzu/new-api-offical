package tencentvod

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
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	ChannelName           = "Tencent VOD AIGC"
	actionCreateImageTask = "CreateAigcImageTask"
	actionCreateVideoTask = "CreateAigcVideoTask"
	actionDescribeTask    = "DescribeTaskDetail"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL       string
	pendingAction string
}

type fileInfo struct {
	Type     string `json:"Type,omitempty"`
	Category string `json:"Category,omitempty"`
	URL      string `json:"Url,omitempty"`
	ID       string `json:"FileId,omitempty"`
	Usage    string `json:"Usage,omitempty"`
}

type tencentPayload struct {
	SubAppID     int64          `json:"SubAppId"`
	ModelName    string         `json:"ModelName,omitempty"`
	ModelVersion string         `json:"ModelVersion,omitempty"`
	Prompt       string         `json:"Prompt,omitempty"`
	FileInfos    []fileInfo     `json:"FileInfos,omitempty"`
	LastFrameURL string         `json:"LastFrameUrl,omitempty"`
	LastFrameID  string         `json:"LastFrameFileId,omitempty"`
	OutputConfig map[string]any `json:"OutputConfig,omitempty"`
	SceneType    string         `json:"SceneType,omitempty"`
	ExtInfo      map[string]any `json:"ExtInfo,omitempty"`
	TaskID       string         `json:"TaskId,omitempty"`
}

type tencentEnvelope struct {
	Response struct {
		TaskID    string `json:"TaskId"`
		RequestID string `json:"RequestId"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"Response"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	if a.baseURL == "" {
		a.baseURL = defaultBaseURL
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapper(err, "get_task_request_failed", http.StatusBadRequest)
	}
	spec, ok := lookupModelSpec(resolveModelName(req.Model, info))
	if !ok {
		return service.TaskErrorWrapperLocal(fmt.Errorf("unsupported Tencent VOD AIGC model"), "unsupported_model", http.StatusBadRequest)
	}
	if spec.Kind == modelKindImage {
		info.Action = constant.TaskActionGenerate
		a.pendingAction = actionCreateImageTask
	} else {
		if req.HasImage() {
			info.Action = constant.TaskActionGenerate
		} else {
			info.Action = constant.TaskActionTextGenerate
		}
		a.pendingAction = actionCreateVideoTask
	}
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	ratios, err := a.EstimateBillingWithError(c, info)
	if err != nil {
		return nil
	}
	return ratios
}

func (a *TaskAdaptor) EstimateBillingWithError(c *gin.Context, info *relaycommon.RelayInfo) (map[string]float64, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	spec, ok := lookupModelSpec(resolveModelName(req.Model, info))
	if !ok {
		return nil, fmt.Errorf("unsupported Tencent VOD AIGC model %s", resolveModelName(req.Model, info))
	}
	if !info.PriceData.UsePrice || info.PriceData.ModelPrice <= 0 {
		return nil, fmt.Errorf("Tencent VOD pricing requires ModelPrice base price for model %s", spec.PublicModel)
	}

	return estimatePreciseBillingRatios(&req, spec)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	base := strings.TrimRight(a.baseURL, "/")
	if base == "" {
		base = strings.TrimRight(info.ChannelBaseUrl, "/")
	}
	if base == "" {
		base = defaultBaseURL
	}
	return base + "/", nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	body, action, err := a.convertToTencentPayload(&req, info)
	if err != nil {
		return nil, err
	}
	a.pendingAction = action
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	cfg, err := parseConfig(info.ApiKey, info.ApiVersion)
	if err != nil {
		return err
	}
	action := a.pendingAction
	if action == "" {
		action = actionCreateVideoTask
	}
	return signRequest(req, body, cfg, action, time.Now().UTC())
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}

	var envelope tencentEnvelope
	if err := common.Unmarshal(responseBody, &envelope); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrap(err, string(responseBody)), "unmarshal_response_failed", http.StatusInternalServerError)
	}
	if envelope.Response.Error != nil {
		return "", responseBody, service.TaskErrorWrapperLocal(fmt.Errorf("%s: %s", envelope.Response.Error.Code, envelope.Response.Error.Message), envelope.Response.Error.Code, http.StatusBadRequest)
	}
	if envelope.Response.TaskID == "" {
		return "", responseBody, service.TaskErrorWrapperLocal(fmt.Errorf("missing Tencent VOD task id"), "missing_task_id", http.StatusBadGateway)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         info.PublicTaskID,
		"task_id":    info.PublicTaskID,
		"model":      info.OriginModelName,
		"status":     dto.VideoStatusQueued,
		"progress":   0,
		"created_at": time.Now().Unix(),
	})
	return envelope.Response.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	region, _ := body["region"].(string)
	if region == "" {
		region = metadataString(body, "api_version")
	}
	if region == "" {
		return nil, fmt.Errorf("X-TC-Region is required for Tencent VOD task polling")
	}
	cfg, err := parseConfig(key, region)
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	reqBody, err := common.Marshal(tencentPayload{SubAppID: cfg.SubAppID, TaskID: taskID})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	if err := signRequest(req, reqBody, cfg, actionDescribeTask, time.Now().UTC()); err != nil {
		return nil, err
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var root map[string]any
	if err := common.Unmarshal(respBody, &root); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	if errObj, ok := findMap(root, "Error"); ok {
		return &relaycommon.TaskInfo{Status: model.TaskStatusFailure, Reason: firstString(metadataString(errObj, "Message"), metadataString(errObj, "Code"))}, nil
	}

	taskRoot := tencentAIGCTaskRoot(root)
	if failureReason := tencentTaskFailureReason(taskRoot); failureReason != "" {
		return &relaycommon.TaskInfo{Status: model.TaskStatusFailure, Reason: failureReason}, nil
	}

	status := strings.ToUpper(firstString(findString(taskRoot, "Status"), findString(taskRoot, "TaskStatus"), findString(taskRoot, "State")))
	taskInfo := &relaycommon.TaskInfo{Progress: normalizeProgress(findAny(taskRoot, "Progress"))}
	switch status {
	case "SUBMITTED", "CREATED", "WAITING", "QUEUEING", "QUEUED", "PENDING":
		taskInfo.Status = model.TaskStatusSubmitted
	case "PROCESSING", "RUNNING", "IN_PROGRESS":
		taskInfo.Status = model.TaskStatusInProgress
	case "FINISH", "FINISHED", "SUCCESS", "SUCCEEDED", "DONE", "COMPLETED":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Url = firstString(
			findString(taskRoot, "VideoUrl"),
			findString(taskRoot, "ImageUrl"),
			findString(taskRoot, "MediaUrl"),
			findString(taskRoot, "FileUrl"),
			findString(taskRoot, "Url"),
			findMediaURL(taskRoot),
		)
		if taskInfo.Url == "" {
			taskInfo.Status = model.TaskStatusInProgress
			if taskInfo.Progress == "" || taskInfo.Progress == taskcommon.ProgressComplete {
				taskInfo.Progress = "95%"
			}
		}
	case "FAIL", "FAILED", "FAILURE", "ERROR":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Reason = firstString(findString(taskRoot, "ErrMsg"), findString(taskRoot, "ErrorMessage"), findString(taskRoot, "Message"), findString(taskRoot, "Reason"))
	default:
		return nil, fmt.Errorf("unknown Tencent VOD task status: %s", status)
	}
	if taskInfo.Progress == "" {
		if taskInfo.Status == model.TaskStatusSuccess {
			taskInfo.Progress = taskcommon.ProgressComplete
		} else if taskInfo.Status == model.TaskStatusSubmitted {
			taskInfo.Progress = taskcommon.ProgressSubmitted
		} else if taskInfo.Status == model.TaskStatusInProgress {
			taskInfo.Progress = taskcommon.ProgressInProgress
		}
	}
	return taskInfo, nil
}

func tencentAIGCTaskRoot(root map[string]any) map[string]any {
	for _, key := range []string{"AigcVideoTask", "AigcImageTask", "AigcTask", "AiTask"} {
		if taskRoot, ok := findMap(root, key); ok {
			return taskRoot
		}
	}
	return root
}

func tencentTaskFailureReason(taskRoot map[string]any) string {
	errCode := strings.TrimSpace(findString(taskRoot, "ErrCode"))
	errCodeExt := strings.TrimSpace(findString(taskRoot, "ErrCodeExt"))
	message := firstString(
		findString(taskRoot, "Message"),
		findString(taskRoot, "ErrMsg"),
		findString(taskRoot, "ErrorMessage"),
		findString(taskRoot, "Reason"),
	)
	if errCode != "" && errCode != "0" {
		return firstString(message, errCodeExt, errCode)
	}
	if errCodeExt != "" {
		return firstString(message, errCodeExt)
	}
	return ""
}

func (a *TaskAdaptor) GetModelList() []string {
	return append([]string(nil), ModelList...)
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	out := map[string]any{
		"id":         originTask.TaskID,
		"task_id":    originTask.TaskID,
		"status":     originTask.Status.ToVideoStatus(),
		"progress":   originTask.Progress,
		"model":      originTask.Properties.OriginModelName,
		"created_at": originTask.CreatedAt,
		"updated_at": originTask.UpdatedAt,
	}
	if originTask.Status == model.TaskStatusFailure {
		out["error"] = map[string]any{"message": originTask.FailReason}
	} else if resultURL := originTask.GetResultURL(); resultURL != "" && !isGatewayVideoContentURL(resultURL, originTask.TaskID) {
		out["url"] = resultURL
	}
	return common.Marshal(out)
}

func isGatewayVideoContentURL(rawURL string, taskID string) bool {
	rawURL = strings.TrimSpace(rawURL)
	return rawURL != "" &&
		taskID != "" &&
		strings.Contains(rawURL, "/v1/videos/") &&
		strings.Contains(rawURL, taskID) &&
		strings.Contains(rawURL, "/content")
}

func (a *TaskAdaptor) convertToTencentPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*tencentPayload, string, error) {
	spec, ok := lookupModelSpec(resolveModelName(req.Model, info))
	if !ok {
		return nil, "", fmt.Errorf("unsupported Tencent VOD AIGC model")
	}
	cfg, err := parseConfig(info.ApiKey, info.ApiVersion)
	if err != nil {
		return nil, "", err
	}
	output := map[string]any{
		"Resolution": normalizeResolution(firstString(req.Resolution, req.Size, metadataString(req.Metadata, "resolution"), metadataString(req.Metadata, "size"), spec.DefaultResolution)),
	}
	if aspectRatio := metadataString(req.Metadata, "aspect_ratio"); aspectRatio != "" {
		output["AspectRatio"] = aspectRatio
	}
	if metadataBool(req.Metadata, "off_peak") || metadataBool(req.Metadata, "offpeak") {
		output["OffPeak"] = "Enabled"
	}
	fileInfos, lastFrameURL, lastFrameID := buildTencentFileInputs(req, spec.Kind)
	body := &tencentPayload{
		SubAppID:     cfg.SubAppID,
		ModelName:    spec.TencentModelName,
		ModelVersion: spec.TencentModelVersion,
		Prompt:       req.Prompt,
		FileInfos:    fileInfos,
		LastFrameURL: lastFrameURL,
		LastFrameID:  lastFrameID,
		OutputConfig: output,
		SceneType:    spec.SceneType,
		ExtInfo:      metadataMap(req.Metadata, "ext_info"),
	}
	if spec.Kind == modelKindImage {
		return body, actionCreateImageTask, nil
	}
	body.OutputConfig["Duration"] = resolveDuration(req, spec)
	return body, actionCreateVideoTask, nil
}

func resolveModelName(requestModel string, info *relaycommon.RelayInfo) string {
	if info != nil {
		if info.ChannelMeta != nil && info.UpstreamModelName != "" {
			return info.UpstreamModelName
		}
		if info.OriginModelName != "" {
			return info.OriginModelName
		}
	}
	return requestModel
}

func buildTencentFileInputs(req *relaycommon.TaskSubmitReq, modelKind string) ([]fileInfo, string, string) {
	if modelKind != modelKindVideo {
		return buildImageFileInfos(req), "", ""
	}
	return buildVideoFileInputs(req)
}

func buildImageFileInfos(req *relaycommon.TaskSubmitReq) []fileInfo {
	files := make([]fileInfo, 0, len(req.Images)+2)
	if strings.TrimSpace(req.Image) != "" {
		files = append(files, newURLFileInfo(strings.TrimSpace(req.Image), modelKindImage, ""))
	}
	for _, image := range req.Images {
		if strings.TrimSpace(image) != "" {
			files = append(files, newURLFileInfo(strings.TrimSpace(image), modelKindImage, ""))
		}
	}
	if req.InputReference != "" {
		if parsed, err := url.Parse(req.InputReference); err == nil && parsed.Scheme != "" {
			files = append(files, newURLFileInfo(req.InputReference, modelKindImage, ""))
		} else {
			files = append(files, newFileIDInfo(req.InputReference, modelKindImage, ""))
		}
	}
	return files
}

func buildVideoFileInputs(req *relaycommon.TaskSubmitReq) ([]fileInfo, string, string) {
	generated := make([]string, 0, len(req.Images)+1)
	if strings.TrimSpace(req.Image) != "" {
		generated = append(generated, strings.TrimSpace(req.Image))
	}
	for _, image := range req.Images {
		if strings.TrimSpace(image) != "" {
			generated = append(generated, strings.TrimSpace(image))
		}
	}

	files := make([]fileInfo, 0, len(generated)+1)
	if len(generated) > 0 {
		files = append(files, mediaToFileInfo(generated[0], modelKindVideo, "FirstFrame"))
	}

	lastFrameURL := ""
	lastFrameID := ""
	if len(generated) > 1 {
		if isURLInput(generated[1]) {
			lastFrameURL = generated[1]
		} else {
			lastFrameID = generated[1]
		}
	}
	if len(generated) > 2 {
		for _, reference := range generated[2:] {
			files = append(files, mediaToFileInfo(reference, modelKindVideo, "Reference"))
		}
	}

	if strings.TrimSpace(req.InputReference) != "" {
		files = append(files, mediaToFileInfo(strings.TrimSpace(req.InputReference), modelKindVideo, "Reference"))
	}

	return files, lastFrameURL, lastFrameID
}

func mediaToFileInfo(value string, modelKind string, usage string) fileInfo {
	if isURLInput(value) {
		return newURLFileInfo(value, modelKind, usage)
	}
	return newFileIDInfo(value, modelKind, usage)
}

func isURLInput(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func newURLFileInfo(rawURL string, modelKind string, usage string) fileInfo {
	info := fileInfo{
		Type: "Url",
		URL:  rawURL,
	}
	if modelKind == modelKindVideo {
		info.Category = "Image"
		info.Usage = usage
	}
	return info
}

func newFileIDInfo(fileID string, modelKind string, usage string) fileInfo {
	info := fileInfo{
		Type: "File",
		ID:   fileID,
	}
	if modelKind == modelKindVideo {
		info.Category = "Image"
		info.Usage = usage
	}
	return info
}

func resolveDuration(req *relaycommon.TaskSubmitReq, spec modelSpec) int {
	if req.Duration > 0 {
		return req.Duration
	}
	if req.Seconds != "" {
		if seconds, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && seconds > 0 {
			return seconds
		}
	}
	if duration := metadataInt(req.Metadata, "duration", 0); duration > 0 {
		return duration
	}
	if duration := metadataInt(req.Metadata, "seconds", 0); duration > 0 {
		return duration
	}
	if spec.DefaultDuration > 0 {
		return spec.DefaultDuration
	}
	return 5
}

func normalizeResolution(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	if strings.HasSuffix(value, "P") || strings.HasSuffix(value, "K") {
		return value
	}
	return value
}

func metadataString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func metadataInt(m map[string]any, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	value, ok := m[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func metadataMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if value, ok := m[key].(map[string]any); ok {
		return value
	}
	return nil
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func findMap(root map[string]any, key string) (map[string]any, bool) {
	for k, value := range root {
		if strings.EqualFold(k, key) {
			if m, ok := value.(map[string]any); ok {
				return m, true
			}
		}
		if m, ok := value.(map[string]any); ok {
			if found, ok := findMap(m, key); ok {
				return found, true
			}
		}
		if items, ok := value.([]any); ok {
			for _, item := range items {
				if m, ok := item.(map[string]any); ok {
					if found, ok := findMap(m, key); ok {
						return found, true
					}
				}
			}
		}
	}
	return nil, false
}

func findString(root map[string]any, key string) string {
	if value := findAny(root, key); value != nil {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func findMediaURL(root map[string]any) string {
	return findMediaURLValue(root)
}

func findMediaURLValue(value any) string {
	switch v := value.(type) {
	case string:
		s := strings.TrimSpace(v)
		if isTencentVODMediaURL(s) {
			return s
		}
	case map[string]any:
		for _, key := range []string{"MediaUrl", "VideoUrl", "ImageUrl", "FileUrl", "Url", "URL"} {
			if found := findMediaURLValue(v[key]); found != "" {
				return found
			}
		}
		for _, nested := range v {
			if found := findMediaURLValue(nested); found != "" {
				return found
			}
		}
	case []any:
		for _, item := range v {
			if found := findMediaURLValue(item); found != "" {
				return found
			}
		}
	}
	return ""
}

func isTencentVODMediaURL(value string) bool {
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return false
	}
	lower := strings.ToLower(value)
	return strings.Contains(lower, ".mp4") ||
		strings.Contains(lower, ".mov") ||
		strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, ".webm") ||
		strings.Contains(lower, ".png") ||
		strings.Contains(lower, ".jpg") ||
		strings.Contains(lower, ".jpeg") ||
		strings.Contains(lower, ".webp") ||
		strings.Contains(lower, ".gif")
}

func findAny(root map[string]any, key string) any {
	for k, value := range root {
		if strings.EqualFold(k, key) {
			return value
		}
		if m, ok := value.(map[string]any); ok {
			if found := findAny(m, key); found != nil {
				return found
			}
		}
		if items, ok := value.([]any); ok {
			for _, item := range items {
				if m, ok := item.(map[string]any); ok {
					if found := findAny(m, key); found != nil {
						return found
					}
				}
			}
		}
	}
	return nil
}

func normalizeProgress(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		if strings.HasSuffix(v, "%") {
			return v
		}
		if v != "" {
			return v + "%"
		}
	case float64:
		return fmt.Sprintf("%.0f%%", v)
	case int:
		return fmt.Sprintf("%d%%", v)
	}
	return ""
}
