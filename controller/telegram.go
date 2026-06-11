package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const telegramAuthMaxAge = 24 * time.Hour

type telegramAuthRequest struct {
	Id        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  string `json:"auth_date"`
	Hash      string `json:"hash"`
	Lang      string `json:"lang"`
}

func TelegramBind(c *gin.Context) {
	if !common.TelegramOAuthEnabled {
		c.JSON(200, gin.H{
			"message": "管理员未开启通过 Telegram 登录以及注册",
			"success": false,
		})
		return
	}
	params := c.Request.URL.Query()
	if !checkTelegramAuthorization(params, common.TelegramBotToken, true) {
		c.JSON(200, gin.H{
			"message": "无效的请求",
			"success": false,
		})
		return
	}
	telegramId := params["id"][0]
	if model.IsTelegramIdAlreadyTaken(telegramId) {
		c.JSON(200, gin.H{
			"message": "该 Telegram 账户已被绑定",
			"success": false,
		})
		return
	}

	session := sessions.Default(c)
	id := session.Get("id")
	userId, ok := id.(int)
	if !ok || userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "用户未登录",
			"success": false,
		})
		return
	}
	user := model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		c.JSON(200, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}
	if user.Id == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户已注销",
		})
		return
	}
	user.TelegramId = telegramId
	if err := user.Update(false); err != nil {
		c.JSON(200, gin.H{
			"message": err.Error(),
			"success": false,
		})
		return
	}

	c.Redirect(302, "/console/personal")
}

func TelegramLogin(c *gin.Context) {
	if !common.TelegramOAuthEnabled {
		c.JSON(200, gin.H{
			"message": "管理员未开启通过 Telegram 登录以及注册",
			"success": false,
		})
		return
	}
	params, ok := getTelegramLoginParams(c)
	if !ok || !checkTelegramAuthorization(params, common.TelegramBotToken, true) {
		c.JSON(200, gin.H{
			"message": "无效的请求",
			"success": false,
		})
		return
	}

	telegramId := params["id"][0]
	user := model.User{TelegramId: telegramId}
	if model.IsTelegramIdAlreadyTaken(telegramId) {
		if err := user.FillUserByTelegramId(); err != nil {
			c.JSON(200, gin.H{
				"message": err.Error(),
				"success": false,
			})
			return
		}
	} else {
		if !common.RegisterEnabled {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}

		session := sessions.Default(c)
		inviterId := 0
		if affCode := session.Get("aff"); affCode != nil {
			inviterId, _ = model.GetUserIdByAffCode(affCode.(string))
		}

		user.Username = getTelegramUsername(params)
		user.DisplayName = getTelegramDisplayName(params)
		user.Role = common.RoleCommonUser
		user.Status = common.UserStatusEnabled
		if err := user.Insert(inviterId); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	setupLogin(&user, c)
}

func getTelegramLoginParams(c *gin.Context) (url.Values, bool) {
	if c.Request.Method == http.MethodGet {
		return c.Request.URL.Query(), true
	}

	var req telegramAuthRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		return nil, false
	}
	params := url.Values{}
	setTelegramParam(params, "id", req.Id)
	setTelegramParam(params, "first_name", req.FirstName)
	setTelegramParam(params, "last_name", req.LastName)
	setTelegramParam(params, "username", req.Username)
	setTelegramParam(params, "photo_url", req.PhotoURL)
	setTelegramParam(params, "auth_date", req.AuthDate)
	setTelegramParam(params, "hash", req.Hash)
	setTelegramParam(params, "lang", req.Lang)
	return params, true
}

func setTelegramParam(params url.Values, key string, value string) {
	if value != "" {
		params.Set(key, value)
	}
}

func getTelegramUsername(params map[string][]string) string {
	if username := firstTelegramParam(params, "username"); username != "" {
		return getUniqueTelegramUsername(username)
	}
	return "telegram_" + strconv.Itoa(model.GetMaxUserId()+1)
}

func isTelegramUsernameTaken(username string) bool {
	return model.DB.Where("username = ?", username).Find(&model.User{}).RowsAffected == 1
}

func getUniqueTelegramUsername(username string) string {
	username = truncateTelegramField(username, 20)
	if !isTelegramUsernameTaken(username) {
		return username
	}
	for i := 1; i < 1000; i++ {
		suffix := "_" + strconv.Itoa(i)
		prefix := truncateTelegramField(username, 20-len(suffix))
		candidate := prefix + suffix
		if !isTelegramUsernameTaken(candidate) {
			return candidate
		}
	}
	return "telegram_" + strconv.Itoa(model.GetMaxUserId()+1)
}

func getTelegramDisplayName(params map[string][]string) string {
	firstName := firstTelegramParam(params, "first_name")
	lastName := firstTelegramParam(params, "last_name")
	displayName := strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))
	if displayName != "" {
		return truncateTelegramField(displayName, 20)
	}
	if username := firstTelegramParam(params, "username"); username != "" {
		return truncateTelegramField(username, 20)
	}
	return "Telegram User"
}

func truncateTelegramField(value string, maxLen int) string {
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}

func firstTelegramParam(params url.Values, key string) string {
	values := params[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func checkTelegramAuthorization(params url.Values, token string, enforceFreshness bool) bool {
	if token == "" {
		return false
	}
	if firstTelegramParam(params, "id") == "" ||
		firstTelegramParam(params, "auth_date") == "" ||
		firstTelegramParam(params, "hash") == "" {
		return false
	}
	if enforceFreshness && !isFreshTelegramAuthDate(firstTelegramParam(params, "auth_date")) {
		return false
	}

	strs := []string{}
	var hash = ""
	for k, v := range params {
		if len(v) == 0 {
			continue
		}
		if k == "hash" {
			hash = v[0]
			continue
		}
		strs = append(strs, k+"="+v[0])
	}
	sort.Strings(strs)
	var imploded = ""
	for _, s := range strs {
		if imploded != "" {
			imploded += "\n"
		}
		imploded += s
	}
	sha256hash := sha256.New()
	io.WriteString(sha256hash, token)
	hmachash := hmac.New(sha256.New, sha256hash.Sum(nil))
	io.WriteString(hmachash, imploded)
	expectedHash := hmachash.Sum(nil)
	actualHash, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}
	return hmac.Equal(actualHash, expectedHash)
}

func isFreshTelegramAuthDate(authDate string) bool {
	timestamp, err := strconv.ParseInt(authDate, 10, 64)
	if err != nil {
		return false
	}
	createdAt := time.Unix(timestamp, 0)
	return time.Since(createdAt) <= telegramAuthMaxAge && time.Until(createdAt) <= time.Minute
}
