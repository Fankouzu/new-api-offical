package router

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskResultAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = false

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

func setupTaskResultAuthRouter(role int) *gin.Engine {
	r := gin.New()
	store := cookie.NewStore([]byte("task-result-auth-test-secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	})
	r.Use(middleware.I18n())
	r.Use(sessions.Sessions("session", store))
	r.GET("/test/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 1)
		session.Set("username", "user")
		session.Set("role", role)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		if err := session.Save(); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(r)
	return r
}

func createInlineImageResultTask(t *testing.T, db *gorm.DB) {
	t.Helper()

	resultData := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png"))
	require.NoError(t, db.Create(&model.Task{
		TaskID:     "task_browser_image_result",
		Platform:   constant.TaskPlatform("58"),
		UserId:     1,
		ChannelId:  15,
		Action:     "generate",
		Status:     model.TaskStatusSuccess,
		SubmitTime: 100,
		FinishTime: 101,
		Progress:   "100%",
		PrivateData: model.TaskPrivateData{
			UpstreamKind: "image",
			ResultURL:    resultData,
		},
	}).Error)
}

func TestTaskResultRouteAllowsBrowserSessionImageRequestWithoutNewApiUserHeader(t *testing.T) {
	db := setupTaskResultAuthTestDB(t)
	createInlineImageResultTask(t, db)

	r := setupTaskResultAuthRouter(common.RoleAdminUser)
	loginRecorder := httptest.NewRecorder()
	r.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/test/login", nil))
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)
	require.NotEmpty(t, loginRecorder.Result().Cookies())

	req := httptest.NewRequest(http.MethodGet, "/api/task/1/result", nil)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	for _, c := range loginRecorder.Result().Cookies() {
		req.AddCookie(c)
	}

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(t, "png", recorder.Body.String())
}

func TestTaskResultRouteRejectsAnonymousBrowserImageRequest(t *testing.T) {
	db := setupTaskResultAuthTestDB(t)
	createInlineImageResultTask(t, db)

	r := setupTaskResultAuthRouter(common.RoleAdminUser)
	req := httptest.NewRequest(http.MethodGet, "/api/task/1/result", nil)
	req.Header.Set("Accept", "image/*")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAdminTaskResultRouteRejectsNonAdminBrowserSession(t *testing.T) {
	db := setupTaskResultAuthTestDB(t)
	createInlineImageResultTask(t, db)

	r := setupTaskResultAuthRouter(common.RoleCommonUser)
	loginRecorder := httptest.NewRecorder()
	r.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/test/login", nil))
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)

	req := httptest.NewRequest(http.MethodGet, "/api/task/1/result", nil)
	req.Header.Set("Accept", "image/*")
	for _, c := range loginRecorder.Result().Cookies() {
		req.AddCookie(c)
	}

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "insufficient")
}
