package analytics

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

const (
	eventVoucherRedeemSuccess = "voucher_redeem_success"
	eventAPIKeyCreated        = "api_key_created"
	eventFirstAPICall         = "first_api_call"
	defaultVoucherSource      = "lizh_ai"
	defaultTimeoutMS          = 1500
	developmentHashSalt       = "ga4-development-hash-salt"
)

type EventParams map[string]any

type RedemptionAttribution struct {
	VoucherSource       string
	DigisellerInvoiceID string
	DigisellerProductID string
	PlatiCampaign       string
}

type UserAttribution struct {
	VoucherSource string
}

type Config struct {
	Enabled       bool
	MeasurementID string
	APISecret     string
	HashSalt      string
	Debug         bool
	Timeout       time.Duration
	Endpoint      string
}

type ga4Payload struct {
	ClientID           string     `json:"client_id"`
	UserID             string     `json:"user_id,omitempty"`
	NonPersonalizedAds bool       `json:"non_personalized_ads"`
	Events             []ga4Event `json:"events"`
}

type ga4Event struct {
	Name   string      `json:"name"`
	Params EventParams `json:"params"`
}

type sender interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	configMu sync.RWMutex
	config   = loadConfigFromEnv()

	httpSender sender = &http.Client{Timeout: config.Timeout}

	regexpGA4APISecret = regexp.MustCompile(`([?&]api_secret=)[^&\s]+`)
)

func loadConfigFromEnv() Config {
	timeoutMS := common.GetEnvOrDefault("GA4_EVENT_TIMEOUT_MS", defaultTimeoutMS)
	if timeoutMS <= 0 {
		timeoutMS = defaultTimeoutMS
	}

	cfg := Config{
		Enabled:       common.GetEnvOrDefaultBool("GA4_EVENT_ENABLED", true),
		MeasurementID: strings.TrimSpace(common.GetEnvOrDefaultString("GA4_MEASUREMENT_ID", "")),
		APISecret:     strings.TrimSpace(common.GetEnvOrDefaultString("GA4_API_SECRET", "")),
		HashSalt:      strings.TrimSpace(common.GetEnvOrDefaultString("GA4_EVENT_HASH_SALT", "")),
		Debug:         common.GetEnvOrDefaultBool("GA4_EVENT_DEBUG", false),
		Timeout:       time.Duration(timeoutMS) * time.Millisecond,
		Endpoint:      "https://www.google-analytics.com/mp/collect",
	}
	if cfg.Debug {
		cfg.Endpoint = "https://www.google-analytics.com/debug/mp/collect"
	}
	if cfg.HashSalt == "" {
		cfg.HashSalt = developmentHashSalt
		if cfg.Enabled && cfg.MeasurementID != "" && cfg.APISecret != "" {
			common.SysLog("GA4_EVENT_HASH_SALT is empty; using development fallback salt")
		}
	}
	if cfg.Enabled && (cfg.MeasurementID == "" || cfg.APISecret == "") {
		common.SysLog("GA4 server analytics disabled: GA4_MEASUREMENT_ID or GA4_API_SECRET is missing")
	}
	return cfg
}

func ConfigureForTest(cfg Config, s sender) func() {
	configMu.Lock()
	oldConfig := config
	oldSender := httpSender
	config = cfg
	if config.Timeout <= 0 {
		config.Timeout = time.Duration(defaultTimeoutMS) * time.Millisecond
	}
	if config.Endpoint == "" {
		config.Endpoint = "https://www.google-analytics.com/mp/collect"
	}
	httpSender = s
	if httpSender == nil {
		httpSender = &http.Client{Timeout: config.Timeout}
	}
	configMu.Unlock()

	return func() {
		configMu.Lock()
		config = oldConfig
		httpSender = oldSender
		configMu.Unlock()
	}
}

func currentConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return config
}

func trackingEnabled(cfg Config) bool {
	return cfg.Enabled && cfg.MeasurementID != "" && cfg.APISecret != ""
}

func Enabled() bool {
	return trackingEnabled(currentConfig())
}

func HashIdentifier(value string) string {
	cfg := currentConfig()
	return hashIdentifierWithSalt(value, cfg.HashSalt)
}

func hashIdentifierWithSalt(value string, salt string) string {
	if salt == "" {
		salt = developmentHashSalt
	}
	mac := hmac.New(sha256.New, []byte(salt))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func ParseGAClientID(cookieValue string) string {
	parts := strings.Split(strings.TrimSpace(cookieValue), ".")
	if len(parts) < 4 {
		return ""
	}
	first := parts[len(parts)-2]
	second := parts[len(parts)-1]
	if first == "" || second == "" {
		return ""
	}
	if _, err := strconv.ParseUint(first, 10, 64); err != nil {
		return ""
	}
	if _, err := strconv.ParseUint(second, 10, 64); err != nil {
		return ""
	}
	return first + "." + second
}

func ResolveGAClientID(c *gin.Context, userID int, tokenID int) string {
	if c != nil {
		if cookieValue, err := c.Cookie("_ga"); err == nil {
			if clientID := ParseGAClientID(cookieValue); clientID != "" {
				return clientID
			}
		}
	}
	base := fmt.Sprintf("%d:%d", userID, tokenID)
	hash := HashIdentifier(base)
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return "server." + hash
}

func TrackVoucherRedeemSuccess(c *gin.Context, userID int, voucherCode string, quota int, attrs RedemptionAttribution) {
	cfg := currentConfig()
	if !trackingEnabled(cfg) {
		return
	}
	source := strings.TrimSpace(attrs.VoucherSource)
	if source == "" {
		source = defaultVoucherSource
	}
	params := EventParams{
		"voucher_code_hash":  HashIdentifier(voucherCode),
		"voucher_amount_usd": float64(quota) / common.QuotaPerUnit,
		"voucher_source":     source,
		"redeem_result":      "success",
	}
	addStringParam(params, "digiseller_invoice_id", attrs.DigisellerInvoiceID)
	addStringParam(params, "digiseller_product_id", attrs.DigisellerProductID)
	addStringParam(params, "plati_campaign", attrs.PlatiCampaign)
	track(c, cfg, userID, 0, eventVoucherRedeemSuccess, params)
}

func TrackAPIKeyCreated(c *gin.Context, userID int, tokenID int, tokenKey string, attrs UserAttribution) {
	cfg := currentConfig()
	if !trackingEnabled(cfg) {
		return
	}
	hashSource := strconv.Itoa(tokenID)
	if tokenID <= 0 {
		hashSource = tokenKey
	}
	source := strings.TrimSpace(attrs.VoucherSource)
	if source == "" {
		source = defaultVoucherSource
	}
	params := EventParams{
		"api_key_id_hash": HashIdentifier(hashSource),
		"voucher_source":  source,
	}
	track(c, cfg, userID, tokenID, eventAPIKeyCreated, params)
}

func TrackFirstAPICall(c *gin.Context, userID int, tokenID int, tokenKey string, modelID string, quotaSpent int) {
	TrackFirstAPICallWithResult(c, userID, tokenID, tokenKey, modelID, quotaSpent, nil)
}

func TrackFirstAPICallWithResult(c *gin.Context, userID int, tokenID int, tokenKey string, modelID string, quotaSpent int, onResult func(error)) {
	cfg := currentConfig()
	if !trackingEnabled(cfg) {
		if onResult != nil {
			onResult(nil)
		}
		return
	}
	hashSource := strconv.Itoa(tokenID)
	if tokenID <= 0 {
		hashSource = tokenKey
	}
	params := EventParams{
		"api_key_id_hash": HashIdentifier(hashSource),
		"model_id":        modelID,
		"quota_spent":     quotaSpent,
		"voucher_source":  defaultVoucherSource,
	}
	trackWithResult(c, cfg, userID, tokenID, eventFirstAPICall, params, onResult)
}

func addStringParam(params EventParams, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		params[key] = value
	}
}

func track(c *gin.Context, cfg Config, userID int, tokenID int, eventName string, params EventParams) {
	trackWithResult(c, cfg, userID, tokenID, eventName, params, nil)
}

func trackWithResult(c *gin.Context, cfg Config, userID int, tokenID int, eventName string, params EventParams, onResult func(error)) {
	payload := buildPayload(c, userID, tokenID, eventName, params)
	gopool.Go(func() {
		if err := sendPayload(context.Background(), cfg, payload); err != nil {
			common.SysLog(fmt.Sprintf("GA4 event send failed: event=%s error=%s", eventName, sanitizeError(err).Error()))
			if onResult != nil {
				onResult(err)
			}
			return
		}
		if onResult != nil {
			onResult(nil)
		}
	})
}

func buildPayload(c *gin.Context, userID int, tokenID int, eventName string, params EventParams) ga4Payload {
	return ga4Payload{
		ClientID:           ResolveGAClientID(c, userID, tokenID),
		UserID:             HashIdentifier(strconv.Itoa(userID)),
		NonPersonalizedAds: true,
		Events: []ga4Event{
			{
				Name:   eventName,
				Params: params,
			},
		},
	}
}

func sendPayload(ctx context.Context, cfg Config, payload ga4Payload) error {
	if !trackingEnabled(cfg) {
		return nil
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint, err := buildEndpoint(cfg)
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	configMu.RLock()
	s := httpSender
	configMu.RUnlock()
	if s == nil {
		s = &http.Client{Timeout: cfg.Timeout}
	}
	resp, err := s.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("status=%d", resp.StatusCode)
	}
	return nil
}

func buildEndpoint(cfg Config) (string, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("measurement_id", cfg.MeasurementID)
	q.Set("api_secret", cfg.APISecret)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", sanitizeGA4Secrets(err.Error()))
}

func sanitizeGA4Secrets(message string) string {
	if message == "" {
		return ""
	}
	return regexpGA4APISecret.ReplaceAllString(message, "${1}[redacted]")
}
