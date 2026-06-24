package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type telegramAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setupTelegramControllerTestDB(t *testing.T) *gorm.DB {
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

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func setupTelegramControllerRouter() *gin.Engine {
	r := gin.New()
	store := cookie.NewStore([]byte("telegram-login-test-secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	})
	r.Use(sessions.Sessions("session", store))
	r.GET("/api/oauth/telegram/login", TelegramLogin)
	return r
}

func signedTelegramLoginURL(token string, values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	dataCheck := make([]string, 0, len(keys))
	for _, key := range keys {
		dataCheck = append(dataCheck, key+"="+values[key])
	}

	secret := sha256.New()
	io.WriteString(secret, token)
	mac := hmac.New(sha256.New, secret.Sum(nil))
	io.WriteString(mac, strings.Join(dataCheck, "\n"))
	hash := hex.EncodeToString(mac.Sum(nil))

	query := make([]string, 0, len(values)+1)
	for _, key := range keys {
		query = append(query, key+"="+values[key])
	}
	query = append(query, "hash="+hash)
	return "/api/oauth/telegram/login?" + strings.Join(query, "&")
}

func TestTelegramLoginRejectsDisabledBoundUser(t *testing.T) {
	db := setupTelegramControllerTestDB(t)

	const botToken = "123456:telegram-test-token"
	common.TelegramOAuthEnabled = true
	common.TelegramBotToken = botToken
	t.Cleanup(func() {
		common.TelegramOAuthEnabled = false
		common.TelegramBotToken = ""
	})

	require.NoError(t, db.Create(&model.User{
		Id:          1,
		Username:    "telegram_disabled",
		DisplayName: "Telegram Disabled",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusDisabled,
		Group:       "default",
		TelegramId:  "987654",
	}).Error)

	r := setupTelegramControllerRouter()
	req := httptest.NewRequest(http.MethodGet, signedTelegramLoginURL(botToken, map[string]string{
		"id":         "987654",
		"first_name": "Telegram",
		"auth_date":  fmt.Sprintf("%d", time.Now().Unix()),
	}), nil)
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response telegramAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, "用户已被封禁", response.Message)
	require.Empty(t, recorder.Result().Cookies())
}
