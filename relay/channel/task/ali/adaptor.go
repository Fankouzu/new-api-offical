package ali

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/samber/lo"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

// AliVideoRequest 阿里通义万相视频生成请求
type AliVideoRequest struct {
	Model      string              `json:"model"`
	Input      AliVideoInput       `json:"input"`
	Parameters *AliVideoParameters `json:"parameters,omitempty"`
}

// AliVideoInput 视频输入参数
type AliVideoInput struct {
	Prompt         string      `json:"prompt,omitempty"`          // 文本提示词
	ImgURL         string      `json:"img_url,omitempty"`         // 首帧图像URL或Base64（图生视频）
	FirstFrameURL  string      `json:"first_frame_url,omitempty"` // 首帧图片URL（首尾帧生视频）
	LastFrameURL   string      `json:"last_frame_url,omitempty"`  // 尾帧图片URL（首尾帧生视频）
	AudioURL       string      `json:"audio_url,omitempty"`       // 音频URL（wan2.5支持）
	NegativePrompt string      `json:"negative_prompt,omitempty"` // 反向提示词
	Template       string      `json:"template,omitempty"`        // 视频特效模板
	Media          []AliMedia  `json:"media,omitempty"`           // PixVerse 媒体素材
}

// AliMedia PixVerse 媒体素材
type AliMedia struct {
	Type    string `json:"type"`                // image_url / first_frame / last_frame
	URL     string `json:"url"`
	RefName string `json:"ref_name,omitempty"` // 参考生视频引用名
}

// AliVideoParameters 视频参数
type AliVideoParameters struct {
	Resolution   string `json:"resolution,omitempty"`    // 分辨率: 360P/480P/540P/720P/1080P
	Size         string `json:"size,omitempty"`          // 尺寸: 如 "832*480"（文生视频）
	Duration     int    `json:"duration,omitempty"`      // 时长: 3-10秒
	PromptExtend bool   `json:"prompt_extend,omitempty"` // 是否开启prompt智能改写
	Watermark    bool   `json:"watermark,omitempty"`     // 是否添加水印
	Audio        *bool  `json:"audio,omitempty"`         // 是否添加音频
	ShotType     string `json:"shot_type,omitempty"`     // 镜头类型: single/multi（PixVerse v6）
	Seed         int    `json:"seed,omitempty"`          // 随机数种子
}

// AliVideoResponse 阿里通义万相响应
type AliVideoResponse struct {
	Output    AliVideoOutput `json:"output"`
	RequestID string         `json:"request_id"`
	Code      string         `json:"code,omitempty"`
	Message   string         `json:"message,omitempty"`
	Usage     *AliUsage      `json:"usage,omitempty"`
}

// AliVideoOutput 输出信息
type AliVideoOutput struct {
	TaskID        string `json:"task_id"`
	TaskStatus    string `json:"task_status"`
	SubmitTime    string `json:"submit_time,omitempty"`
	ScheduledTime string `json:"scheduled_time,omitempty"`
	EndTime       string `json:"end_time,omitempty"`
	OrigPrompt    string `json:"orig_prompt,omitempty"`
	ActualPrompt  string `json:"actual_prompt,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
}

// AliUsage 使用统计
type AliUsage struct {
	Duration   int `json:"duration,omitempty"`
	VideoCount int `json:"video_count,omitempty"`
	SR         int `json:"SR,omitempty"`
}

type AliMetadata struct {
	// Input 相关
	AudioURL       string     `json:"audio_url,omitempty"`       // 音频URL
	ImgURL         string     `json:"img_url,omitempty"`         // 图片URL（图生视频）
	FirstFrameURL  string     `json:"first_frame_url,omitempty"` // 首帧图片URL（首尾帧生视频）
	LastFrameURL   string     `json:"last_frame_url,omitempty"`  // 尾帧图片URL（首尾帧生视频）
	NegativePrompt string     `json:"negative_prompt,omitempty"` // 反向提示词
	Template       string     `json:"template,omitempty"`        // 视频特效模板
	Media          []AliMedia `json:"media,omitempty"`           // PixVerse 媒体素材

	// Parameters 相关
	Resolution   *string `json:"resolution,omitempty"`    // 分辨率: 360P/480P/540P/720P/1080P
	Size         *string `json:"size,omitempty"`          // 尺寸: 如 "832*480"
	Duration     *int    `json:"duration,omitempty"`      // 时长
	PromptExtend *bool   `json:"prompt_extend,omitempty"` // 是否开启prompt智能改写
	Watermark    *bool   `json:"watermark,omitempty"`     // 是否添加水印
	Audio        *bool   `json:"audio,omitempty"`         // 是否添加音频
	ShotType     string  `json:"shot_type,omitempty"`     // 镜头类型: single/multi
	Seed         *int    `json:"seed,omitempty"`          // 随机数种子
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	// ValidateMultipartDirect 负责解析并将原始 TaskSubmitReq 存入 context
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/v1/services/aigc/video-generation/video-synthesis", a.baseURL), nil
}

// BuildRequestHeader sets required headers for Ali API
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-Async", "enable") // 阿里异步任务必须设置
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_task_request_failed")
	}

	aliReq, err := a.convertToAliRequest(info, taskReq)
	if err != nil {
		return nil, errors.Wrap(err, "convert_to_ali_request_failed")
	}
	logger.LogJson(c, "ali video request body", aliReq)

	bodyBytes, err := common.Marshal(aliReq)
	if err != nil {
		return nil, errors.Wrap(err, "marshal_ali_request_failed")
	}
	return bytes.NewReader(bodyBytes), nil
}

var (
	size480p = []string{
		"832*480",
		"480*832",
		"624*624",
	}
	size720p = []string{
		"1280*720",
		"720*1280",
		"960*960",
		"1088*832",
		"832*1088",
	}
	size1080p = []string{
		"1920*1080",
		"1080*1920",
		"1440*1440",
		"1632*1248",
		"1248*1632",
	}
)

var (
	size360p = []string{
		"640*360", "360*640", "640*480", "480*640", "640*640",
		"640*432", "432*640", "640*288",
	}
	size540p = []string{
		"1024*576", "576*1024", "1024*768", "768*1024", "1024*1024",
		"1024*688", "688*1024", "1024*448",
	}
)

// isPixverseModel 判断是否为 PixVerse 系列模型
func isPixverseModel(model string) bool {
	return strings.HasPrefix(model, "pixverse/")
}

func sizeToResolution(size string) (string, error) {
	if lo.Contains(size360p, size) {
		return "360P", nil
	} else if lo.Contains(size480p, size) {
		return "480P", nil
	} else if lo.Contains(size540p, size) {
		return "540P", nil
	} else if lo.Contains(size720p, size) {
		return "720P", nil
	} else if lo.Contains(size1080p, size) {
		return "1080P", nil
	}
	return "", fmt.Errorf("invalid size: %s", size)
}

func ProcessAliOtherRatios(aliReq *AliVideoRequest) (map[string]float64, error) {
	otherRatios := make(map[string]float64)
	aliRatios := map[string]map[string]float64{
		"wan2.6-i2v": {
			"720P":  1,
			"1080P": 1 / 0.6,
		},
		"wan2.5-t2v-preview": {
			"480P":  1,
			"720P":  2,
			"1080P": 1 / 0.3,
		},
		"wan2.2-t2v-plus": {
			"480P":  1,
			"1080P": 0.7 / 0.14,
		},
		"wan2.5-i2v-preview": {
			"480P":  1,
			"720P":  2,
			"1080P": 1 / 0.3,
		},
		"wan2.2-i2v-plus": {
			"480P":  1,
			"1080P": 0.7 / 0.14,
		},
		"wan2.2-kf2v-flash": {
			"480P":  1,
			"720P":  2,
			"1080P": 4.8,
		},
		"wan2.2-i2v-flash": {
			"480P": 1,
			"720P": 2,
		},
		"wan2.2-s2v": {
			"480P": 1,
			"720P": 0.9 / 0.5,
		},
	}
	var resolution string

	// size match
	if aliReq.Parameters.Size != "" {
		toResolution, err := sizeToResolution(aliReq.Parameters.Size)
		if err != nil {
			return nil, err
		}
		resolution = toResolution
	} else {
		resolution = strings.ToUpper(aliReq.Parameters.Resolution)
		if !strings.HasSuffix(resolution, "P") {
			resolution = resolution + "P"
		}
	}
	if otherRatio, ok := aliRatios[aliReq.Model]; ok {
		if ratio, ok := otherRatio[resolution]; ok {
			otherRatios[fmt.Sprintf("resolution-%s", resolution)] = ratio
		}
	}

	// PixVerse 系列模型计费倍率
	if isPixverseModel(aliReq.Model) {
		audioKey := "no-audio"
		if aliReq.Parameters.Audio != nil && *aliReq.Parameters.Audio {
			audioKey = "audio"
		}
		ratioKey := fmt.Sprintf("%s-%s", resolution, audioKey)

		if strings.Contains(aliReq.Model, "v6") {
				// PixVerse v6 基准：360P 无声 = 1.0 (0.15元/秒)
			pixverseV6Ratios := map[string]float64{
				"360P-no-audio":  1.0,
				"360P-audio":     0.21 / 0.15,
				"480P-no-audio":  1.0,
				"540P-no-audio":  0.21 / 0.15,
				"540P-audio":     0.27 / 0.15,
				"720P-no-audio":  0.27 / 0.15,
				"720P-audio":     0.36 / 0.15,
				"1080P-no-audio": 0.53 / 0.15,
				"1080P-audio":    0.68 / 0.15,
			}
			if ratio, ok := pixverseV6Ratios[ratioKey]; ok {
				otherRatios[fmt.Sprintf("resolution-%s", resolution)] = ratio
			}
		} else {
				// PixVerse v5.6 基准：360P/540P 无声 = 1.0 (0.21元/秒)
			pixverseV56Ratios := map[string]float64{
				"360P-no-audio":  1.0,
				"360P-audio":     0.47 / 0.21,
				"480P-no-audio":  1.0,
				"540P-no-audio":  1.0,
				"540P-audio":     0.47 / 0.21,
				"720P-no-audio":  0.27 / 0.21,
				"720P-audio":     0.53 / 0.21,
				"1080P-no-audio": 0.44 / 0.21,
				"1080P-audio":    0.70 / 0.21,
			}
			if ratio, ok := pixverseV56Ratios[ratioKey]; ok {
				otherRatios[fmt.Sprintf("resolution-%s", resolution)] = ratio
			}
		}
	}

	return otherRatios, nil
}

func (a *TaskAdaptor) convertToAliRequest(info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) (*AliVideoRequest, error) {
	upstreamModel := req.Model
	if info.IsModelMapped {
		upstreamModel = info.UpstreamModelName
	}

	aliReq := &AliVideoRequest{
		Model: upstreamModel,
		Input: AliVideoInput{
			Prompt: req.Prompt,
		},
		Parameters: &AliVideoParameters{
			Watermark: false,
		},
	}

	if isPixverseModel(upstreamModel) {
		if err := a.buildPixverseRequest(aliReq, req); err != nil {
			return nil, err
		}
	} else {
		// wan 系列模型：使用 img_url / first_frame_url
		aliReq.Input.ImgURL = req.InputReference
		aliReq.Parameters.PromptExtend = true
		a.buildWanResolution(aliReq, req)
	}

	// 处理时长
	if req.Duration > 0 {
		aliReq.Parameters.Duration = req.Duration
	} else if req.Seconds != "" {
		seconds, err := strconv.Atoi(req.Seconds)
		if err != nil {
			return nil, errors.Wrap(err, "convert seconds to int failed")
		}
		aliReq.Parameters.Duration = seconds
	} else {
		aliReq.Parameters.Duration = 5 // 默认5秒
	}

	// 从 metadata 中提取额外参数（覆盖默认值）
	if req.Metadata != nil {
		if metadataBytes, err := common.Marshal(req.Metadata); err == nil {
			err = common.Unmarshal(metadataBytes, aliReq)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshal metadata failed")
			}
		} else {
			return nil, errors.Wrap(err, "marshal metadata failed")
		}
	}

	if aliReq.Model != upstreamModel {
		return nil, errors.New("can't change model with metadata")
	}

	return aliReq, nil
}

// buildWanResolution 处理 wan 系列模型的分辨率参数
func (a *TaskAdaptor) buildWanResolution(aliReq *AliVideoRequest, req relaycommon.TaskSubmitReq) {
	if req.Size != "" {
		// wan t2v size must contain *
		if strings.Contains(req.Model, "t2v") && !strings.Contains(req.Size, "*") {
			return // will be caught by validation later
		}
		if strings.Contains(req.Size, "*") {
			aliReq.Parameters.Size = req.Size
		} else {
			resolution := strings.ToUpper(req.Size)
			if !strings.HasSuffix(resolution, "P") {
				resolution = resolution + "P"
			}
			aliReq.Parameters.Resolution = resolution
		}
	} else {
		if strings.Contains(req.Model, "t2v") {
			if strings.HasPrefix(req.Model, "wan2.5") || strings.HasPrefix(req.Model, "wan2.2") {
				aliReq.Parameters.Size = "1920*1080"
			} else {
				aliReq.Parameters.Size = "1280*720"
			}
		} else {
			if strings.HasPrefix(req.Model, "wan2.6") || strings.HasPrefix(req.Model, "wan2.5") {
				aliReq.Parameters.Resolution = "1080P"
			} else if strings.HasPrefix(req.Model, "wan2.2-i2v-flash") {
				aliReq.Parameters.Resolution = "720P"
			} else if strings.HasPrefix(req.Model, "wan2.2-i2v-plus") {
				aliReq.Parameters.Resolution = "1080P"
			} else {
				aliReq.Parameters.Resolution = "720P"
			}
		}
	}
}

// buildPixverseRequest 构建 PixVerse 系列模型的请求
func (a *TaskAdaptor) buildPixverseRequest(aliReq *AliVideoRequest, req relaycommon.TaskSubmitReq) error {
	// 确定模型后缀类型
	modelSuffix := ""
	for _, suffix := range []string{"-t2v", "-it2v", "-kf2v", "-r2v"} {
		if strings.HasSuffix(aliReq.Model, suffix) {
			modelSuffix = suffix
			break
		}
	}

	switch modelSuffix {
	case "-t2v":
		// 文生视频：使用 size（像素值），无 media
		a.setPixverseSize(aliReq, req, "1280*720")
	case "-it2v":
		// 图生视频：使用 resolution（档位），media[{type:"image_url", url:img}]
		a.setPixverseResolution(aliReq, req, "720P")
		if req.InputReference != "" {
			aliReq.Input.Media = []AliMedia{{Type: "image_url", URL: req.InputReference}}
		} else if len(req.Images) > 0 {
			aliReq.Input.Media = []AliMedia{{Type: "image_url", URL: req.Images[0]}}
		}
	case "-kf2v":
		// 首尾帧生视频：使用 resolution（档位），media[{type:"first_frame"}, {type:"last_frame"}]
		a.setPixverseResolution(aliReq, req, "720P")
		if len(req.Images) >= 2 {
			aliReq.Input.Media = []AliMedia{
				{Type: "first_frame", URL: req.Images[0]},
				{Type: "last_frame", URL: req.Images[1]},
			}
		}
	case "-r2v":
		// 参考生视频：使用 size（像素值），media[{type:"image_url", url:img, ref_name:...}]
		a.setPixverseSize(aliReq, req, "1280*720")
		if len(req.Images) > 0 {
			media := make([]AliMedia, 0, len(req.Images))
			for _, imgURL := range req.Images {
				media = append(media, AliMedia{Type: "image_url", URL: imgURL})
			}
			aliReq.Input.Media = media
		}
	}

	return nil
}

// setPixverseSize 设置 PixVerse t2v/r2v 的 size 参数（像素值格式）
func (a *TaskAdaptor) setPixverseSize(aliReq *AliVideoRequest, req relaycommon.TaskSubmitReq, defaultSize string) {
	if req.Size != "" && strings.Contains(req.Size, "*") {
		aliReq.Parameters.Size = req.Size
	} else {
		aliReq.Parameters.Size = defaultSize
	}
}

// setPixverseResolution 设置 PixVerse it2v/kf2v 的 resolution 参数（档位格式）
func (a *TaskAdaptor) setPixverseResolution(aliReq *AliVideoRequest, req relaycommon.TaskSubmitReq, defaultResolution string) {
	if req.Size != "" && !strings.Contains(req.Size, "*") {
		resolution := strings.ToUpper(req.Size)
		if !strings.HasSuffix(resolution, "P") {
			resolution = resolution + "P"
		}
		aliReq.Parameters.Resolution = resolution
	} else {
		aliReq.Parameters.Resolution = defaultResolution
	}
}

// EstimateBilling 根据用户请求参数计算 OtherRatios（时长、分辨率等）。
// 在 ValidateRequestAndSetAction 之后、价格计算之前调用。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	aliReq, err := a.convertToAliRequest(info, taskReq)
	if err != nil {
		return nil
	}

	otherRatios := map[string]float64{
		"seconds": float64(aliReq.Parameters.Duration),
	}
	ratios, err := ProcessAliOtherRatios(aliReq)
	if err != nil {
		return otherRatios
	}
	for k, v := range ratios {
		otherRatios[k] = v
	}
	return otherRatios
}

// DoRequest delegates to common helper
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// 解析阿里响应
	var aliResp AliVideoResponse
	if err := common.Unmarshal(responseBody, &aliResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	// 检查错误
	if aliResp.Code != "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("%s: %s", aliResp.Code, aliResp.Message), "ali_api_error", resp.StatusCode)
		return
	}

	if aliResp.Output.TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 转换为 OpenAI 格式响应
	openAIResp := dto.NewOpenAIVideo()
	openAIResp.ID = info.PublicTaskID
	openAIResp.TaskID = info.PublicTaskID
	openAIResp.Model = c.GetString("model")
	if openAIResp.Model == "" && info != nil {
		openAIResp.Model = info.OriginModelName
	}
	openAIResp.Status = convertAliStatus(aliResp.Output.TaskStatus)
	openAIResp.CreatedAt = common.GetTimestamp()

	// 返回 OpenAI 格式
	c.JSON(http.StatusOK, openAIResp)

	return aliResp.Output.TaskID, responseBody, nil
}

// FetchTask 查询任务状态
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/v1/tasks/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// ParseTaskResult 解析任务结果
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var aliResp AliVideoResponse
	if err := common.Unmarshal(respBody, &aliResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	// 状态映射
	switch aliResp.Output.TaskStatus {
	case "PENDING":
		taskResult.Status = model.TaskStatusQueued
	case "RUNNING":
		taskResult.Status = model.TaskStatusInProgress
	case "SUCCEEDED":
		taskResult.Status = model.TaskStatusSuccess
		// 阿里直接返回视频URL，不需要额外的代理端点
		taskResult.Url = aliResp.Output.VideoURL
	case "FAILED", "CANCELED", "UNKNOWN":
		taskResult.Status = model.TaskStatusFailure
		if aliResp.Message != "" {
			taskResult.Reason = aliResp.Message
		} else if aliResp.Output.Message != "" {
			taskResult.Reason = fmt.Sprintf("task failed, code: %s , message: %s", aliResp.Output.Code, aliResp.Output.Message)
		} else {
			taskResult.Reason = "task failed"
		}
	default:
		taskResult.Status = model.TaskStatusQueued
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var aliResp AliVideoResponse
	if err := common.Unmarshal(task.Data, &aliResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal ali response failed")
	}

	openAIResp := dto.NewOpenAIVideo()
	openAIResp.ID = task.TaskID
	openAIResp.Status = convertAliStatus(aliResp.Output.TaskStatus)
	openAIResp.Model = task.Properties.OriginModelName
	openAIResp.SetProgressStr(task.Progress)
	openAIResp.CreatedAt = task.CreatedAt
	openAIResp.CompletedAt = task.UpdatedAt

	// 设置视频URL（核心字段）
	openAIResp.SetMetadata("url", aliResp.Output.VideoURL)

	// 错误处理
	if aliResp.Code != "" {
		openAIResp.Error = &dto.OpenAIVideoError{
			Code:    aliResp.Code,
			Message: aliResp.Message,
		}
	} else if aliResp.Output.Code != "" {
		openAIResp.Error = &dto.OpenAIVideoError{
			Code:    aliResp.Output.Code,
			Message: aliResp.Output.Message,
		}
	}

	return common.Marshal(openAIResp)
}

func convertAliStatus(aliStatus string) string {
	switch aliStatus {
	case "PENDING":
		return dto.VideoStatusQueued
	case "RUNNING":
		return dto.VideoStatusInProgress
	case "SUCCEEDED":
		return dto.VideoStatusCompleted
	case "FAILED", "CANCELED", "UNKNOWN":
		return dto.VideoStatusFailed
	default:
		return dto.VideoStatusUnknown
	}
}
