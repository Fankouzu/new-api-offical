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
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

const (
	eventSignUp               = "sign_up"
	eventVoucherRedeemSuccess = "voucher_redeem_success"
	eventAPIKeyCreated        = "api_key_created"
	eventPurchase             = "purchase"
	eventFirstAPICall         = "first_api_call"
	eventTopUp                = eventPurchase
	defaultVoucherSource      = "lizh_ai"
	defaultRedeemSource       = "voucher"
	defaultTimeoutMS          = 1500
	developmentHashSalt       = "ga4-development-hash-salt"
)

var attributionURLParamAllowlist = map[string]struct{}{
	"utm_source":   {},
	"utm_medium":   {},
	"utm_campaign": {},
	"utm_term":     {},
	"utm_content":  {},
	"gclid":        {},
	"fbclid":       {},
	"ttclid":       {},
	"yclid":        {},
	"aff":          {},
}

type EventParams map[string]any

type RedemptionAttribution struct {
	TransactionID       string
	Value               float64
	Currency            string
	Source              string
	PageLocation        string
	PageReferrer        string
	VoucherSource       string
	DigisellerInvoiceID string
	DigisellerProductID string
	PlatiCampaign       string
}

type UserAttribution struct {
	VoucherSource string
	KeyType       string
	PageLocation  string
	PageReferrer  string
}

type PurchaseAttribution struct {
	TradeNo         string
	Value           float64
	Currency        string
	PaymentProvider string
	PaymentMethod   string
	ItemType        string
	QuotaAmount     int64
	PageLocation    string
	PageReferrer    string
	FallbackPath    string
}

type SignUpAttribution struct {
	ClientID     string `json:"client_id"`
	PageLocation string `json:"page_location"`
	PageReferrer string `json:"page_referrer"`
	Source       string `json:"source"`
	Medium       string `json:"medium"`
	Campaign     string `json:"campaign"`
	Term         string `json:"term"`
	Content      string `json:"content"`
	GCLID        string `json:"gclid"`
	FBCLID       string `json:"fbclid"`
	TTCLID       string `json:"ttclid"`
	YCLID        string `json:"yclid"`
	FirstVisitAt string `json:"first_visit_at"`
	Method       string `json:"method"`
}

type FirstAPIRequestAttribution struct {
	Model        string
	Endpoint     string
	StatusCode   int
	QuotaSpent   int
	PageLocation string
	PageReferrer string
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
	source := strings.TrimSpace(attrs.Source)
	if source == "" {
		source = defaultRedeemSource
	}
	value := attrs.Value
	if value <= 0 && quota > 0 {
		value = float64(quota) / common.QuotaPerUnit
	}
	transactionID := strings.TrimSpace(attrs.TransactionID)
	if transactionID == "" {
		transactionID = "voucher:" + HashIdentifier(voucherCode)
	}
	params := EventParams{
		"transaction_id":     transactionID,
		"value":              value,
		"currency":           normalizeCurrency(attrs.Currency),
		"source":             source,
		"voucher_code_hash":  HashIdentifier(voucherCode),
		"voucher_amount_usd": value,
		"voucher_source":     firstNonEmpty(attrs.VoucherSource, source),
		"redeem_result":      "success",
	}
	addUserIDParam(params, userID)
	addStringParam(params, "digiseller_invoice_id", attrs.DigisellerInvoiceID)
	addStringParam(params, "digiseller_product_id", attrs.DigisellerProductID)
	addStringParam(params, "plati_campaign", attrs.PlatiCampaign)
	addPageContext(c, params, attrs.PageLocation, attrs.PageReferrer, "/wallet")
	track(c, cfg, userID, 0, eventVoucherRedeemSuccess, params)
}

func TrackAPIKeyCreated(c *gin.Context, userID int, tokenID int, tokenKey string, attrs UserAttribution) {
	TrackAPIKeyCreatedWithResult(c, userID, tokenID, tokenKey, attrs, nil)
}

func TrackAPIKeyCreatedWithResult(c *gin.Context, userID int, tokenID int, tokenKey string, attrs UserAttribution, onResult func(error)) {
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
	addUserIDParam(params, userID)
	addStringParam(params, "key_type", attrs.KeyType)
	addPageContext(c, params, attrs.PageLocation, attrs.PageReferrer, "/keys")
	trackWithResult(c, cfg, userID, tokenID, eventAPIKeyCreated, params, onResult)
}

func TrackTopUp(c *gin.Context, userID int, attrs PurchaseAttribution) {
	TrackTopUpWithResult(c, userID, attrs, nil)
}

func TrackPurchase(c *gin.Context, userID int, attrs PurchaseAttribution) {
	TrackPurchaseWithResult(c, userID, attrs, nil)
}

func TrackTopUpWithResult(c *gin.Context, userID int, attrs PurchaseAttribution, onResult func(error)) {
	trackPurchaseEventWithResult(c, userID, eventTopUp, attrs, onResult)
}

func TrackPurchaseWithResult(c *gin.Context, userID int, attrs PurchaseAttribution, onResult func(error)) {
	trackPurchaseEventWithResult(c, userID, eventPurchase, attrs, onResult)
}

func trackPurchaseEventWithResult(c *gin.Context, userID int, eventName string, attrs PurchaseAttribution, onResult func(error)) {
	cfg := currentConfig()
	if !trackingEnabled(cfg) {
		return
	}
	paymentMethod := strings.TrimSpace(attrs.PaymentProvider)
	if paymentMethod == "" {
		paymentMethod = strings.TrimSpace(attrs.PaymentMethod)
	}
	params := EventParams{
		"transaction_id": attrs.TradeNo,
		"value":          attrs.Value,
		"currency":       normalizeCurrency(attrs.Currency),
		"payment_method": paymentMethod,
	}
	itemID := strings.TrimSpace(attrs.ItemType)
	if itemID == "" {
		itemID = "purchase"
	}
	params["items"] = []EventParams{
		{
			"item_id":   itemID,
			"item_name": purchaseItemName(itemID),
			"price":     attrs.Value,
			"quantity":  1,
		},
	}
	addUserIDParam(params, userID)
	addStringParam(params, "payment_provider", attrs.PaymentProvider)
	if strings.TrimSpace(attrs.PaymentMethod) != "" && strings.TrimSpace(attrs.PaymentMethod) != paymentMethod {
		addStringParam(params, "payment_method_detail", attrs.PaymentMethod)
	}
	addStringParam(params, "item_type", attrs.ItemType)
	if attrs.QuotaAmount > 0 {
		params["quota_amount"] = attrs.QuotaAmount
	}
	fallbackPath := firstNonEmpty(attrs.FallbackPath, "/wallet")
	addPageContext(c, params, attrs.PageLocation, attrs.PageReferrer, fallbackPath)
	trackWithResult(c, cfg, userID, 0, eventName, params, onResult)
}

func purchaseItemName(itemID string) string {
	switch itemID {
	case "top_up":
		return "Balance top-up"
	case "subscription":
		return "Subscription"
	default:
		return itemID
	}
}

func TrackSignUp(c *gin.Context, userID int, attrs SignUpAttribution) {
	cfg := currentConfig()
	if !trackingEnabled(cfg) {
		return
	}
	method := strings.TrimSpace(attrs.Method)
	if method == "" {
		method = "unknown"
	}
	params := EventParams{
		"method": method,
	}
	addUserIDParam(params, userID)
	addPageContext(c, params, attrs.PageLocation, attrs.PageReferrer, "/sign-up")
	addStringParam(params, "source", attrs.Source)
	addStringParam(params, "medium", attrs.Medium)
	addStringParam(params, "campaign", attrs.Campaign)
	addStringParam(params, "term", attrs.Term)
	addStringParam(params, "content", attrs.Content)
	addStringParam(params, "gclid", attrs.GCLID)
	addStringParam(params, "fbclid", attrs.FBCLID)
	addStringParam(params, "ttclid", attrs.TTCLID)
	addStringParam(params, "yclid", attrs.YCLID)
	addStringParam(params, "first_visit_at", attrs.FirstVisitAt)
	trackWithClientID(c, cfg, userID, 0, eventSignUp, params, attrs.ClientID, nil)
}

func TrackFirstAPICall(c *gin.Context, userID int, tokenID int, tokenKey string, modelID string, quotaSpent int) {
	TrackFirstAPIRequestSuccessWithResult(c, userID, tokenID, tokenKey, FirstAPIRequestAttribution{
		Model:      modelID,
		QuotaSpent: quotaSpent,
		StatusCode: http.StatusOK,
	}, nil)
}

func TrackFirstAPICallWithResult(c *gin.Context, userID int, tokenID int, tokenKey string, modelID string, quotaSpent int, onResult func(error)) {
	TrackFirstAPIRequestSuccessWithResult(c, userID, tokenID, tokenKey, FirstAPIRequestAttribution{
		Model:      modelID,
		QuotaSpent: quotaSpent,
		StatusCode: http.StatusOK,
	}, onResult)
}

func TrackFirstAPIRequestSuccessWithResult(c *gin.Context, userID int, tokenID int, tokenKey string, attrs FirstAPIRequestAttribution, onResult func(error)) {
	cfg := currentConfig()
	if !trackingEnabled(cfg) {
		return
	}
	hashSource := strconv.Itoa(tokenID)
	if tokenID <= 0 {
		hashSource = tokenKey
	}
	statusCode := attrs.StatusCode
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	params := EventParams{
		"api_key_id_hash": HashIdentifier(hashSource),
		"model":           attrs.Model,
		"model_id":        attrs.Model,
		"endpoint":        normalizeEndpoint(attrs.Endpoint),
		"status_code":     statusCode,
		"quota_spent":     attrs.QuotaSpent,
		"voucher_source":  defaultVoucherSource,
	}
	addUserIDParam(params, userID)
	addPageContext(c, params, attrs.PageLocation, attrs.PageReferrer, normalizeEndpoint(attrs.Endpoint))
	trackWithResult(c, cfg, userID, tokenID, eventFirstAPICall, params, onResult)
}

func addStringParam(params EventParams, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		params[key] = value
	}
}

func setStringParam(params EventParams, key string, value string) {
	params[key] = strings.TrimSpace(value)
}

func addUserIDParam(params EventParams, userID int) {
	if userID > 0 {
		params["user_id"] = userID
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return "USD"
	}
	return currency
}

func sanitizeAttributionURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	clean := url.URL{
		Scheme:  parsed.Scheme,
		Host:    parsed.Host,
		Path:    parsed.Path,
		RawPath: parsed.RawPath,
	}
	query := url.Values{}
	for key, values := range parsed.Query() {
		if _, ok := attributionURLParamAllowlist[key]; !ok {
			continue
		}
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				query.Add(key, value)
			}
		}
	}
	clean.RawQuery = query.Encode()
	return clean.String()
}

func addPageContext(c *gin.Context, params EventParams, pageLocation string, pageReferrer string, fallbackPath string) {
	resolvedLocation := resolvePageLocation(c, pageLocation, fallbackPath)
	setStringParam(params, "page_location", resolvedLocation)
	setStringParam(params, "page_referrer", sanitizeAttributionURL(pageReferrer))
	setStringParam(params, "hostname", resolveHostname(c, resolvedLocation))
}

func resolvePageLocation(c *gin.Context, pageLocation string, fallbackPath string) string {
	if sanitized := sanitizeAttributionURL(pageLocation); sanitized != "" {
		return sanitized
	}
	base := configuredSiteBaseURL()
	if base == nil {
		base = requestBaseURL(c)
	}
	if base == nil {
		return ""
	}
	fallbackPath, fallbackRawPath := normalizePagePath(fallbackPath)
	if fallbackPath == "" {
		return base.String()
	}
	resolved := *base
	resolved.Path = fallbackPath
	resolved.RawPath = fallbackRawPath
	resolved.RawQuery = ""
	resolved.Fragment = ""
	return resolved.String()
}

func resolveHostname(c *gin.Context, pageLocation string) string {
	if parsed, err := url.Parse(strings.TrimSpace(pageLocation)); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	if base := configuredSiteBaseURL(); base != nil {
		return base.Hostname()
	}
	if base := requestBaseURL(c); base != nil {
		return base.Hostname()
	}
	return ""
}

func configuredSiteBaseURL() *url.URL {
	raw := strings.TrimSpace(system_setting.ServerAddress)
	if raw == "" {
		return nil
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return nil
	}
	return cleanBaseURL(parsed)
}

func requestBaseURL(c *gin.Context) *url.URL {
	if c == nil || c.Request == nil {
		return nil
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if host == "" {
		return nil
	}
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	parsed := &url.URL{Scheme: scheme, Host: host}
	return cleanBaseURL(parsed)
}

func cleanBaseURL(parsed *url.URL) *url.URL {
	if parsed == nil {
		return nil
	}
	clean := *parsed
	clean.Path = ""
	clean.RawPath = ""
	clean.RawQuery = ""
	clean.Fragment = ""
	return &clean
}

func normalizePagePath(path string) (string, string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", ""
	}
	rawPath := ""
	if parsed, err := url.Parse(path); err == nil {
		path = parsed.Path
		rawPath = parsed.RawPath
	} else if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" {
		return "", ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
		if rawPath != "" {
			rawPath = "/" + rawPath
		}
	}
	return path, rawPath
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if parsed, err := url.Parse(endpoint); err == nil {
		if parsed.Scheme != "" && parsed.Host != "" {
			endpoint = parsed.Path
		} else if parsed.Path != "" {
			endpoint = parsed.Path
		}
	}
	if endpoint == "" {
		return ""
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	if idx := strings.Index(endpoint, "?"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func track(c *gin.Context, cfg Config, userID int, tokenID int, eventName string, params EventParams) {
	trackWithResult(c, cfg, userID, tokenID, eventName, params, nil)
}

func trackWithResult(c *gin.Context, cfg Config, userID int, tokenID int, eventName string, params EventParams, onResult func(error)) {
	trackWithClientID(c, cfg, userID, tokenID, eventName, params, "", onResult)
}

func trackWithClientID(c *gin.Context, cfg Config, userID int, tokenID int, eventName string, params EventParams, clientID string, onResult func(error)) {
	payload := buildPayloadWithClientID(c, userID, tokenID, eventName, params, clientID)
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
	return buildPayloadWithClientID(c, userID, tokenID, eventName, params, "")
}

func buildPayloadWithClientID(c *gin.Context, userID int, tokenID int, eventName string, params EventParams, clientID string) ga4Payload {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = ResolveGAClientID(c, userID, tokenID)
	}
	return ga4Payload{
		ClientID:           clientID,
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
