package dto

import (
	"encoding/json"
)

type TaskError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	StatusCode int    `json:"-"`
	LocalError bool   `json:"-"`
	Error      error  `json:"-"`
}

type TaskData interface {
	SunoDataResponse | []SunoDataResponse | string | any
}

const TaskSuccessCode = "success"

type TaskResponse[T TaskData] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (t *TaskResponse[T]) IsSuccess() bool {
	return t.Code == TaskSuccessCode
}

type TaskDto struct {
	ID         int64  `json:"id"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	TaskID     string `json:"task_id"`
	Platform   string `json:"platform"`
	UserId     int    `json:"user_id"`
	Group      string `json:"group"`
	ChannelId  int    `json:"channel_id"`
	Quota      int    `json:"quota"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	FailReason string `json:"fail_reason"`
	ResultURL  string `json:"result_url,omitempty"` // 任务结果 URL（视频地址等）
	// UpstreamKind is set for PingXingShiJie-style async tasks: video | image | asset (routes polling / admin UI).
	UpstreamKind string          `json:"upstream_kind,omitempty"`
	SubmitTime   int64           `json:"submit_time"`
	StartTime    int64           `json:"start_time"`
	FinishTime   int64           `json:"finish_time"`
	Progress     string          `json:"progress"`
	Properties   any             `json:"properties"`
	Username     string          `json:"username,omitempty"`
	Data         json.RawMessage `json:"data,omitempty"`
}

type TaskResultSummary struct {
	Available bool   `json:"available"`
	Inline    bool   `json:"inline"`
	Type      string `json:"type,omitempty"`
	Size      int    `json:"size,omitempty"`
	URL       string `json:"url,omitempty"`
}

type TaskDataSummary struct {
	Bytes   int  `json:"bytes"`
	Omitted bool `json:"omitted"`
}

type TaskDetailDto struct {
	ID           int64             `json:"id"`
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
	TaskID       string            `json:"task_id"`
	Platform     string            `json:"platform"`
	UserId       int               `json:"user_id"`
	Group        string            `json:"group"`
	ChannelId    int               `json:"channel_id"`
	Quota        int               `json:"quota"`
	Action       string            `json:"action"`
	Status       string            `json:"status"`
	FailReason   string            `json:"fail_reason,omitempty"`
	UpstreamKind string            `json:"upstream_kind,omitempty"`
	SubmitTime   int64             `json:"submit_time"`
	StartTime    int64             `json:"start_time"`
	FinishTime   int64             `json:"finish_time"`
	Progress     string            `json:"progress"`
	Properties   any               `json:"properties"`
	Username     string            `json:"username,omitempty"`
	Result       TaskResultSummary `json:"result"`
	DataSummary  TaskDataSummary   `json:"data_summary"`
}

type FetchReq struct {
	IDs []string `json:"ids"`
}
