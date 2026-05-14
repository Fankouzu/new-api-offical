package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type taskAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type taskPageResponse struct {
	Items []taskListItem `json:"items"`
}

type taskListItem struct {
	TaskID    string          `json:"task_id"`
	ResultURL string          `json:"result_url"`
	Data      json.RawMessage `json:"data"`
}

func setupTaskControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.User{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func decodeTaskAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) taskAPIResponse {
	t.Helper()

	var response taskAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestGetAllTaskOmitsHeavyMediaPayloadFromList(t *testing.T) {
	db := setupTaskControllerTestDB(t)

	require.NoError(t, db.Create(&model.User{
		Id:       1,
		Username: "alice",
		Group:    "default",
	}).Error)

	largeDataURL := "data:image/png;base64," + strings.Repeat("a", 4096)
	require.NoError(t, db.Create(&model.Task{
		TaskID:     "task_large_media",
		Platform:   constant.TaskPlatform("61"),
		UserId:     1,
		ChannelId:  7,
		Action:     "generate",
		Status:     model.TaskStatusSuccess,
		SubmitTime: 100,
		FinishTime: 110,
		Progress:   "100%",
		PrivateData: model.TaskPrivateData{
			ResultURL: largeDataURL,
		},
		Data: json.RawMessage(`{"created":1,"data":[{"url":"` + largeDataURL + `"}]}`),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/task/?p=1&page_size=10", nil)

	GetAllTask(ctx)

	response := decodeTaskAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var page taskPageResponse
	require.NoError(t, common.Unmarshal(response.Data, &page))
	require.Len(t, page.Items, 1)
	require.Empty(t, page.Items[0].Data)
	require.Empty(t, page.Items[0].ResultURL)
	require.NotContains(t, recorder.Body.String(), largeDataURL)
}

func TestGetTaskByIDReturnsFullPayloadForOnDemandDetails(t *testing.T) {
	db := setupTaskControllerTestDB(t)

	largeDataURL := "data:image/png;base64," + strings.Repeat("b", 4096)
	task := &model.Task{
		TaskID:     "task_detail_media",
		Platform:   constant.TaskPlatform("61"),
		UserId:     1,
		ChannelId:  7,
		Action:     "generate",
		Status:     model.TaskStatusSuccess,
		SubmitTime: 100,
		FinishTime: 110,
		Progress:   "100%",
		PrivateData: model.TaskPrivateData{
			ResultURL: largeDataURL,
		},
		Data: json.RawMessage(`{"created":1,"data":[{"url":"` + largeDataURL + `"}]}`),
	}
	require.NoError(t, db.Create(task).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/task/%d", task.ID), nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", task.ID)}}

	GetTaskByID(ctx)

	response := decodeTaskAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	require.Contains(t, string(response.Data), largeDataURL)
}
