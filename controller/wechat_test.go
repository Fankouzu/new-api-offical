package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/analytics"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const wechatTestServerToken = "wechat-server-token"

type wechatGA4CaptureSender struct {
	mu     sync.Mutex
	bodies []string
	done   chan struct{}
}

func (s *wechatGA4CaptureSender) Do(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	s.mu.Lock()
	s.bodies = append(s.bodies, string(body))
	s.mu.Unlock()
	if s.done != nil {
		s.done <- struct{}{}
	}
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func (s *wechatGA4CaptureSender) snapshotBodies() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.bodies...)
}

type wechatAuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type wechatGA4Payload struct {
	ClientID string `json:"client_id"`
	Events   []struct {
		Name   string         `json:"name"`
		Params map[string]any `json:"params"`
	} `json:"events"`
}

func setupWeChatAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldSQLite := common.UsingSQLite
	oldMySQL := common.UsingMySQL
	oldPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled

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
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldSQLite
		common.UsingMySQL = oldMySQL
		common.UsingPostgreSQL = oldPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func setupWeChatAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	store := cookie.NewStore([]byte("wechat-login-test-secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	})
	router.Use(sessions.Sessions("session", store))
	router.GET("/api/oauth/wechat", WeChatAuth)
	return router
}

func setupWeChatAuthTestConfig(t *testing.T, upstreamURL string) {
	t.Helper()

	oldWeChatAuthEnabled := common.WeChatAuthEnabled
	oldRegisterEnabled := common.RegisterEnabled
	oldWeChatServerAddress := common.WeChatServerAddress
	oldWeChatServerToken := common.WeChatServerToken
	oldQuotaForNewUser := common.QuotaForNewUser
	oldQuotaForInviter := common.QuotaForInviter
	oldQuotaForInvitee := common.QuotaForInvitee
	oldServerAddress := system_setting.ServerAddress
	common.WeChatAuthEnabled = true
	common.RegisterEnabled = true
	common.WeChatServerAddress = upstreamURL
	common.WeChatServerToken = wechatTestServerToken
	common.QuotaForNewUser = 0
	common.QuotaForInviter = 0
	common.QuotaForInvitee = 0
	system_setting.ServerAddress = "https://lizh.ai"
	t.Cleanup(func() {
		common.WeChatAuthEnabled = oldWeChatAuthEnabled
		common.RegisterEnabled = oldRegisterEnabled
		common.WeChatServerAddress = oldWeChatServerAddress
		common.WeChatServerToken = oldWeChatServerToken
		common.QuotaForNewUser = oldQuotaForNewUser
		common.QuotaForInviter = oldQuotaForInviter
		common.QuotaForInvitee = oldQuotaForInvitee
		system_setting.ServerAddress = oldServerAddress
	})
}

func newWeChatAuthTestUpstream(t *testing.T, wechatIDsByCode map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		wechatID, ok := wechatIDsByCode[code]
		if r.URL.Path != "/api/wechat/user" || r.Header.Get("Authorization") != wechatTestServerToken || !ok {
			t.Errorf("unexpected WeChat upstream request: path=%q code=%q authorization=%q", r.URL.Path, code, r.Header.Get("Authorization"))
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		body, err := common.Marshal(wechatLoginResponse{Success: true, Data: wechatID})
		if err != nil {
			t.Errorf("marshal WeChat upstream response: %v", err)
			http.Error(w, "marshal response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(body); err != nil {
			t.Errorf("write WeChat upstream response: %v", err)
		}
	}))
}

func performWeChatAuthRequest(router *gin.Engine, code string, aff string, requestCookies []*http.Cookie) *httptest.ResponseRecorder {
	query := url.Values{"code": {code}}
	if aff != "" {
		query.Set("aff", aff)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/oauth/wechat?"+query.Encode(), nil)
	request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.123.456"})
	request.AddCookie(&http.Cookie{Name: "_ga_TEST", Value: "GS1.1.1740000000.1.1.1740000000.0.0.0"})
	for _, requestCookie := range requestCookies {
		request.AddCookie(requestCookie)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeWeChatAuthResponse(t *testing.T, recorder *httptest.ResponseRecorder) wechatAuthResponse {
	t.Helper()
	var response wechatAuthResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func stubWeChatSignUpTracker(t *testing.T) *int {
	t.Helper()
	calls := 0
	original := trackWeChatSignUp
	trackWeChatSignUp = func(_ *gin.Context, _ int, _ analytics.SignUpAttribution) {
		calls++
	}
	t.Cleanup(func() {
		trackWeChatSignUp = original
	})
	return &calls
}

func requireNoWeChatSecretsInLogs(t *testing.T, db *gorm.DB, secrets ...string) {
	t.Helper()
	var logs []model.Log
	require.NoError(t, db.Find(&logs).Error)
	for _, log := range logs {
		for _, secret := range secrets {
			require.NotContains(t, log.Content, secret)
		}
	}
}

func TestWeChatAuthTracksSignUpOnlyForNewUser(t *testing.T) {
	db := setupWeChatAuthTestDB(t)
	originalTrackWeChatSignUp := trackWeChatSignUp
	trackWeChatSignUp = analytics.TrackSignUp
	t.Cleanup(func() {
		trackWeChatSignUp = originalTrackWeChatSignUp
	})

	const (
		wechatCode = "wechat-login-code-secret"
		wechatID   = "wechat-user-id-secret"
	)
	upstream := newWeChatAuthTestUpstream(t, map[string]string{wechatCode: wechatID})
	defer upstream.Close()
	setupWeChatAuthTestConfig(t, upstream.URL)
	common.QuotaForNewUser = 100
	common.QuotaForInviter = 30
	common.QuotaForInvitee = 20

	inviter := model.User{
		Username:    "wechat_inviter",
		DisplayName: "WeChat Inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     "ABCD",
	}
	require.NoError(t, db.Create(&inviter).Error)

	sender := &wechatGA4CaptureSender{done: make(chan struct{}, 1)}
	restoreAnalytics := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	t.Cleanup(restoreAnalytics)

	router := setupWeChatAuthTestRouter()
	newUserRecorder := performWeChatAuthRequest(router, wechatCode, "ABCD", nil)

	require.Equal(t, http.StatusOK, newUserRecorder.Code)
	newUserResponse := decodeWeChatAuthResponse(t, newUserRecorder)
	require.True(t, newUserResponse.Success, newUserResponse.Message)
	var sessionCookie *http.Cookie
	for _, responseCookie := range newUserRecorder.Result().Cookies() {
		if responseCookie.Name == "session" {
			sessionCookie = responseCookie
			break
		}
	}
	require.NotNil(t, sessionCookie)
	require.NotEmpty(t, sessionCookie.Value)
	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WeChat sign_up event")
	}

	bodies := sender.snapshotBodies()
	require.Len(t, bodies, 1)
	var payload wechatGA4Payload
	require.NoError(t, common.Unmarshal([]byte(bodies[0]), &payload))
	require.Equal(t, "123.456", payload.ClientID)
	require.Len(t, payload.Events, 1)
	require.Equal(t, "sign_up", payload.Events[0].Name)
	require.Equal(t, "wechat", payload.Events[0].Params["method"])
	require.Equal(t, float64(1740000000), payload.Events[0].Params["session_id"])
	require.Equal(t, float64(1), payload.Events[0].Params["engagement_time_msec"])
	require.NotContains(t, bodies[0], wechatCode)
	require.NotContains(t, bodies[0], wechatID)
	require.NotContains(t, newUserRecorder.Body.String(), wechatCode)
	require.NotContains(t, newUserRecorder.Body.String(), wechatID)

	var invitee model.User
	require.NoError(t, db.First(&invitee, "wechat_id = ?", wechatID).Error)
	require.Equal(t, inviter.Id, invitee.InviterId)
	require.Equal(t, common.QuotaForNewUser+common.QuotaForInvitee, invitee.Quota)
	var savedInviter model.User
	require.NoError(t, db.First(&savedInviter, inviter.Id).Error)
	require.Equal(t, 1, savedInviter.AffCount)
	require.Equal(t, common.QuotaForInviter, savedInviter.AffQuota)
	require.Equal(t, common.QuotaForInviter, savedInviter.AffHistoryQuota)
	requireNoWeChatSecretsInLogs(t, db, wechatCode, wechatID)

	existingSignUpCalls := stubWeChatSignUpTracker(t)
	existingUserRecorder := performWeChatAuthRequest(router, wechatCode, "", []*http.Cookie{sessionCookie})

	require.Equal(t, http.StatusOK, existingUserRecorder.Code)
	existingUserResponse := decodeWeChatAuthResponse(t, existingUserRecorder)
	require.True(t, existingUserResponse.Success, existingUserResponse.Message)
	require.Zero(t, *existingSignUpCalls)
	require.Len(t, sender.snapshotBodies(), 1)
	var userCount int64
	require.NoError(t, db.Model(&model.User{}).Count(&userCount).Error)
	require.Equal(t, int64(2), userCount)
}

func TestWeChatAuthRegistrationDisabledDoesNotTrackOrInsert(t *testing.T) {
	db := setupWeChatAuthTestDB(t)
	const (
		wechatCode = "wechat-disabled-code-secret"
		wechatID   = "wechat-disabled-id-secret"
	)
	upstream := newWeChatAuthTestUpstream(t, map[string]string{wechatCode: wechatID})
	defer upstream.Close()
	setupWeChatAuthTestConfig(t, upstream.URL)
	common.RegisterEnabled = false
	signUpCalls := stubWeChatSignUpTracker(t)

	recorder := performWeChatAuthRequest(setupWeChatAuthTestRouter(), wechatCode, "", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeWeChatAuthResponse(t, recorder)
	require.False(t, response.Success)
	require.Zero(t, *signUpCalls)
	var userCount int64
	require.NoError(t, db.Model(&model.User{}).Count(&userCount).Error)
	require.Zero(t, userCount)
	requireNoWeChatSecretsInLogs(t, db, wechatCode, wechatID)
	require.NotContains(t, recorder.Body.String(), wechatCode)
	require.NotContains(t, recorder.Body.String(), wechatID)
}

func TestWeChatAuthInsertFailureDoesNotTrackOrPersistUser(t *testing.T) {
	db := setupWeChatAuthTestDB(t)
	const (
		wechatCode = "wechat-insert-failure-code-secret"
		wechatID   = "wechat-insert-failure-id-secret"
	)
	upstream := newWeChatAuthTestUpstream(t, map[string]string{wechatCode: wechatID})
	defer upstream.Close()
	setupWeChatAuthTestConfig(t, upstream.URL)
	signUpCalls := stubWeChatSignUpTracker(t)

	sentinel := errors.New("sentinel WeChat user insert failure")
	callbackName := "test:fail_wechat_user_create:" + strings.ReplaceAll(t.Name(), "/", "_")
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		user, ok := tx.Statement.Dest.(*model.User)
		if ok && user.WeChatId == wechatID {
			tx.AddError(sentinel)
		}
	}))
	t.Cleanup(func() {
		require.NoError(t, db.Callback().Create().Remove(callbackName))
	})

	recorder := performWeChatAuthRequest(setupWeChatAuthTestRouter(), wechatCode, "", nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeWeChatAuthResponse(t, recorder)
	require.False(t, response.Success)
	require.Equal(t, sentinel.Error(), response.Message)
	require.Zero(t, *signUpCalls)
	var targetUserCount int64
	require.NoError(t, db.Model(&model.User{}).Where("wechat_id = ?", wechatID).Count(&targetUserCount).Error)
	require.Zero(t, targetUserCount)
	requireNoWeChatSecretsInLogs(t, db, wechatCode, wechatID)
	require.NotContains(t, recorder.Body.String(), wechatCode)
	require.NotContains(t, recorder.Body.String(), wechatID)
}
