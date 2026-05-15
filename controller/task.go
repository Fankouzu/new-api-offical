package controller

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// UpdateTaskBulk 薄入口，实际轮询逻辑在 service 层
func UpdateTaskBulk() {
	service.TaskPollingLoop()
}

func GetAllTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 解析其他查询参数
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ChannelID:      c.Query("channel_id"),
	}

	items := model.TaskGetAllTasks(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllTasks(queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, true))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	items := model.TaskGetAllUserTask(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTask(userId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, false))
	common.ApiSuccess(c, pageInfo)
}

func GetTaskByID(c *gin.Context) {
	detail, ok := loadTaskDetailByIDParam(c, false)
	if !ok {
		return
	}

	common.ApiSuccess(c, detail)
}

func GetUserTaskByID(c *gin.Context) {
	detail, ok := loadTaskDetailByIDParam(c, true)
	if !ok {
		return
	}
	if detail.UserId != c.GetInt("id") {
		taskNotFound(c)
		return
	}

	common.ApiSuccess(c, detail)
}

func GetTaskRawByID(c *gin.Context) {
	task, ok := loadTaskByIDParam(c)
	if !ok {
		return
	}

	common.ApiSuccess(c, sanitizedTaskRawDto(task))
}

func GetUserTaskRawByID(c *gin.Context) {
	task, ok := loadTaskByIDParam(c)
	if !ok {
		return
	}
	if task.UserId != c.GetInt("id") {
		taskNotFound(c)
		return
	}

	common.ApiSuccess(c, sanitizedTaskRawDto(task))
}

func GetTaskResultByID(c *gin.Context) {
	task, ok := loadTaskResultByIDParam(c)
	if !ok {
		return
	}
	writeTaskResult(c, task)
}

func GetUserTaskResultByID(c *gin.Context) {
	task, ok := loadTaskResultByIDParam(c)
	if !ok {
		return
	}
	if task.UserId != c.GetInt("id") {
		taskNotFound(c)
		return
	}
	writeTaskResult(c, task)
}

func loadTaskByIDParam(c *gin.Context) (*model.Task, bool) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		common.ApiErrorMsg(c, "invalid task id")
		return nil, false
	}

	task, exist, err := model.GetTaskByID(taskID)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if !exist {
		taskNotFound(c)
		return nil, false
	}
	return task, true
}

func sanitizedTaskRawDto(task *model.Task) *dto.TaskDto {
	taskDto := relay.TaskModel2Dto(task)
	resultURL := task.PrivateData.ResultURL
	if resultURL == "" {
		resultURL = task.FailReason
	}
	taskDto.ResultURL = summarizeLargeInlineMediaString(resultURL)
	taskDto.Data = sanitizeTaskRawJSON(taskDto.Data)
	return taskDto
}

func sanitizeTaskRawJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return raw
	}
	value = sanitizeTaskRawValue(value)
	b, err := common.Marshal(value)
	if err != nil {
		return raw
	}
	return b
}

func sanitizeTaskRawValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if s, ok := child.(string); ok {
				v[key] = summarizeTaskRawString(key, s)
				continue
			}
			v[key] = sanitizeTaskRawValue(child)
		}
		return v
	case []any:
		for i, child := range v {
			v[i] = sanitizeTaskRawValue(child)
		}
		return v
	case string:
		return summarizeTaskRawString("", v)
	default:
		return v
	}
}

func summarizeTaskRawString(key string, value string) string {
	if looksLikeHTTPURL(value) {
		return value
	}
	if strings.EqualFold(key, "b64_json") || looksLikeInlineImageData(value) || looksLikeLargeBase64(value) {
		return summarizeLargeInlineMediaString(value)
	}
	return value
}

func looksLikeHTTPURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func looksLikeInlineImageData(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "data:image/")
}

func looksLikeLargeBase64(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 512 {
		return false
	}
	base64Chars := 0
	for _, r := range trimmed[:512] {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '/' || r == '=' {
			base64Chars++
		}
	}
	return base64Chars > 500
}

func summarizeLargeInlineMediaString(value string) string {
	const prefixLen = 64
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= prefixLen || looksLikeHTTPURL(trimmed) {
		return value
	}
	if !looksLikeInlineImageData(trimmed) && !looksLikeLargeBase64(trimmed) {
		return value
	}
	prefix := trimmed
	if len(prefix) > prefixLen {
		prefix = prefix[:prefixLen]
	}
	return fmt.Sprintf("%s... [base64 omitted, original_length=%d]", prefix, len(trimmed))
}

func loadTaskResultByIDParam(c *gin.Context) (*model.Task, bool) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		common.ApiErrorMsg(c, "invalid task id")
		return nil, false
	}

	task, exist, err := getLightweightTaskResultByID(taskID)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if !exist {
		taskNotFound(c)
		return nil, false
	}
	return task, true
}

func loadTaskDetailByIDParam(c *gin.Context, self bool) (*dto.TaskDetailDto, bool) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		common.ApiErrorMsg(c, "invalid task id")
		return nil, false
	}

	detail, exist, err := getLightweightTaskDetailByID(taskID, self)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if !exist {
		taskNotFound(c)
		return nil, false
	}
	return detail, true
}

func taskNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"message": "task not found",
	})
}

type lightweightTaskDetailRow struct {
	ID         int64                 `gorm:"column:id"`
	CreatedAt  int64                 `gorm:"column:created_at"`
	UpdatedAt  int64                 `gorm:"column:updated_at"`
	TaskID     string                `gorm:"column:task_id"`
	Platform   constant.TaskPlatform `gorm:"column:platform"`
	UserId     int                   `gorm:"column:user_id"`
	Group      string                `gorm:"column:group"`
	ChannelId  int                   `gorm:"column:channel_id"`
	Quota      int                   `gorm:"column:quota"`
	Action     string                `gorm:"column:action"`
	Status     model.TaskStatus      `gorm:"column:status"`
	FailReason string                `gorm:"column:fail_reason"`
	SubmitTime int64                 `gorm:"column:submit_time"`
	StartTime  int64                 `gorm:"column:start_time"`
	FinishTime int64                 `gorm:"column:finish_time"`
	Progress   string                `gorm:"column:progress"`
	Properties model.Properties      `gorm:"column:properties"`
	Username   string                `gorm:"column:username"`
}

func getLightweightTaskResultByID(id int64) (*model.Task, bool, error) {
	if id <= 0 {
		return nil, false, nil
	}

	task := &model.Task{}
	err := model.DB.Select(
		"id",
		"task_id",
		"user_id",
		"status",
		"fail_reason",
		"private_data",
		"properties",
	).Where("id = ?", id).Take(task).Error
	exist, err := model.RecordExist(err)
	if err != nil || !exist {
		return nil, exist, err
	}

	resultURL := strings.TrimSpace(task.GetResultURL())
	if resultURL != "" && !isRecoverableStoredProxyResult(resultURL) {
		return task, true, nil
	}

	return model.GetTaskByID(id)
}

func isRecoverableStoredProxyResult(resultURL string) bool {
	resultURL = strings.TrimSpace(resultURL)
	return strings.Contains(resultURL, "/v1/videos/") && strings.Contains(resultURL, "/content") ||
		strings.Contains(resultURL, "/api/task/") && strings.Contains(resultURL, "/result")
}

func getLightweightTaskDetailByID(id int64, self bool) (*dto.TaskDetailDto, bool, error) {
	if id <= 0 {
		return nil, false, nil
	}

	row := lightweightTaskDetailRow{}
	err := model.DB.Table("tasks").
		Select(taskDetailSelectColumns()).
		Where("id = ?", id).
		Take(&row).Error
	exist, err := model.RecordExist(err)
	if err != nil || !exist {
		return nil, exist, err
	}
	return taskDetailRow2Dto(row, self), true, nil
}

func taskDetailSelectColumns() []string {
	columns := []string{
		"id",
		"created_at",
		"updated_at",
		"task_id",
		"platform",
		"user_id",
		taskDetailGroupColumn(),
		"channel_id",
		"quota",
		"action",
		"status",
		"submit_time",
		"start_time",
		"finish_time",
		"progress",
		"properties",
	}
	switch {
	case common.UsingPostgreSQL:
		columns = append(columns, "SUBSTRING(COALESCE(fail_reason, '') FROM 1 FOR 4096) AS fail_reason")
	case common.UsingMySQL:
		columns = append(columns, "LEFT(COALESCE(fail_reason, ''), 4096) AS fail_reason")
	default:
		columns = append(columns, "substr(COALESCE(fail_reason, ''), 1, 4096) AS fail_reason")
	}
	return columns
}

func taskDetailGroupColumn() string {
	if common.UsingMySQL {
		return "`group`"
	}
	return "\"group\""
}

func taskDetailRow2Dto(row lightweightTaskDetailRow, self bool) *dto.TaskDetailDto {
	result := taskResultSummaryForLightweightDetail(row.ID, row.Action, row.Status, row.FailReason, row.Properties, self)
	return &dto.TaskDetailDto{
		ID:         row.ID,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		TaskID:     row.TaskID,
		Platform:   string(row.Platform),
		UserId:     row.UserId,
		Group:      row.Group,
		ChannelId:  row.ChannelId,
		Quota:      row.Quota,
		Action:     row.Action,
		Status:     string(row.Status),
		FailReason: row.FailReason,
		SubmitTime: row.SubmitTime,
		StartTime:  row.StartTime,
		FinishTime: row.FinishTime,
		Progress:   row.Progress,
		Properties: row.Properties,
		Username:   row.Username,
		Result:     result,
		DataSummary: dto.TaskDataSummary{
			Omitted: true,
		},
	}
}

func taskModel2DetailDto(task *model.Task, self bool) *dto.TaskDetailDto {
	result := taskResultSummary(task, self)
	return &dto.TaskDetailDto{
		ID:           task.ID,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
		TaskID:       task.TaskID,
		Platform:     string(task.Platform),
		UserId:       task.UserId,
		Group:        task.Group,
		ChannelId:    task.ChannelId,
		Quota:        task.Quota,
		Action:       task.Action,
		Status:       string(task.Status),
		FailReason:   task.FailReason,
		UpstreamKind: task.PrivateData.UpstreamKind,
		SubmitTime:   task.SubmitTime,
		StartTime:    task.StartTime,
		FinishTime:   task.FinishTime,
		Progress:     task.Progress,
		Properties:   task.Properties,
		Username:     task.Username,
		Result:       result,
		DataSummary: dto.TaskDataSummary{
			Bytes:   len(task.Data),
			Omitted: len(task.Data) > 0,
		},
	}
}

func taskResultSummary(task *model.Task, self bool) dto.TaskResultSummary {
	resultURL := strings.TrimSpace(task.GetResultURL())
	return taskResultSummaryFromFields(task.ID, resultURL, len(resultURL), task.PrivateData.UpstreamKind, self)
}

func taskResultSummaryForLightweightDetail(id int64, action string, status model.TaskStatus, failReason string, properties model.Properties, self bool) dto.TaskResultSummary {
	if status != model.TaskStatusSuccess && strings.TrimSpace(failReason) == "" {
		return dto.TaskResultSummary{}
	}
	path := fmt.Sprintf("/api/task/%d/result", id)
	if self {
		path = fmt.Sprintf("/api/task/self/%d/result", id)
	}
	return dto.TaskResultSummary{
		Available: true,
		Type:      inferTaskResultTypeFromTaskFields(action, failReason, properties),
		URL:       path,
	}
}

func taskResultSummaryFromFields(id int64, resultPrefix string, resultSize int, upstreamKind string, self bool) dto.TaskResultSummary {
	resultPrefix = strings.TrimSpace(resultPrefix)
	if resultPrefix == "" {
		return dto.TaskResultSummary{}
	}
	path := fmt.Sprintf("/api/task/%d/result", id)
	if self {
		path = fmt.Sprintf("/api/task/self/%d/result", id)
	}
	return dto.TaskResultSummary{
		Available: true,
		Inline:    strings.HasPrefix(resultPrefix, "data:"),
		Type:      inferTaskResultType(resultPrefix, upstreamKind),
		Size:      resultSize,
		URL:       path,
	}
}

func inferTaskResultTypeFromTaskFields(action string, failReason string, properties model.Properties) string {
	lowerFailReason := strings.ToLower(failReason)
	if strings.HasPrefix(lowerFailReason, "data:image/") ||
		strings.Contains(lowerFailReason, ".png") ||
		strings.Contains(lowerFailReason, ".jpg") ||
		strings.Contains(lowerFailReason, ".jpeg") ||
		strings.Contains(lowerFailReason, ".webp") {
		return "image"
	}
	if strings.HasPrefix(lowerFailReason, "data:video/") ||
		strings.Contains(lowerFailReason, ".mp4") ||
		strings.Contains(lowerFailReason, ".webm") ||
		strings.Contains(lowerFailReason, ".mov") {
		return "video"
	}
	modelHint := strings.ToLower(properties.UpstreamModelName + " " + properties.OriginModelName)
	if strings.Contains(modelHint, "seedream") || strings.Contains(modelHint, "image") {
		return "image"
	}
	if action == "MUSIC" {
		return "audio"
	}
	return "video"
}

func inferTaskResultType(resultURL string, upstreamKind string) string {
	mediaType, _, ok := strings.Cut(strings.TrimPrefix(resultURL, "data:"), ";")
	if ok {
		if strings.HasPrefix(mediaType, "image/") {
			return "image"
		}
		if strings.HasPrefix(mediaType, "video/") {
			return "video"
		}
		if strings.HasPrefix(mediaType, "audio/") {
			return "audio"
		}
	}
	if upstreamKind != "" {
		return upstreamKind
	}
	lower := strings.ToLower(resultURL)
	if strings.Contains(lower, ".png") || strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".webp") {
		return "image"
	}
	if strings.Contains(lower, ".mp4") || strings.Contains(lower, ".webm") || strings.Contains(lower, ".mov") {
		return "video"
	}
	return ""
}

func writeTaskResult(c *gin.Context, task *model.Task) {
	resultURL := strings.TrimSpace(task.GetResultURL())
	if resultURL == "" {
		taskNotFound(c)
		return
	}
	if strings.HasPrefix(resultURL, "data:") {
		contentType, data, ok := decodeDataURL(resultURL)
		if !ok {
			common.ApiErrorMsg(c, "invalid task result data")
			return
		}
		c.Data(http.StatusOK, contentType, data)
		return
	}
	c.Redirect(http.StatusFound, resultURL)
}

func decodeDataURL(raw string) (string, []byte, bool) {
	header, payload, ok := strings.Cut(raw, ",")
	if !ok || !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return "", nil, false
	}
	contentType := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		contentType = "application/octet-stream"
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, false
	}
	return contentType, data, true
}

func tasksToDto(tasks []*model.Task, fillUser bool) []*dto.TaskDto {
	var userIdMap map[int]*model.UserBase
	if fillUser {
		userIdMap = make(map[int]*model.UserBase)
		userIds := types.NewSet[int]()
		for _, task := range tasks {
			userIds.Add(task.UserId)
		}
		for _, userId := range userIds.Items() {
			cacheUser, err := model.GetUserCache(userId)
			if err == nil {
				userIdMap[userId] = cacheUser
			}
		}
	}
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		if fillUser {
			if user, ok := userIdMap[task.UserId]; ok {
				task.Username = user.Username
			}
		}
		result[i] = relay.TaskModel2Dto(task)
	}
	return result
}
