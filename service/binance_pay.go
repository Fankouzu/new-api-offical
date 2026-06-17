package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

const (
	binancePayProdBaseURL      = "https://bpay.binanceapi.com"
	binancePaySandboxBaseURL   = "https://bpay.binanceapi.com"
	binancePayCreateOrderPath  = "/binancepay/openapi/v3/order"
	binancePayDefaultTolerance = 5 * time.Minute
)

type BinancePayCreateOrderParams struct {
	Env                BinancePayEnv     `json:"env"`
	MerchantTradeNo    string            `json:"merchantTradeNo"`
	OrderAmount        string            `json:"orderAmount"`
	Currency           string            `json:"currency"`
	Description        string            `json:"description,omitempty"`
	GoodsDetails       []BinancePayGoods `json:"goodsDetails,omitempty"`
	ReturnURL          string            `json:"returnUrl,omitempty"`
	CancelURL          string            `json:"cancelUrl,omitempty"`
	WebhookURL         string            `json:"webhookUrl,omitempty"`
	ExpireTime         int64             `json:"orderExpireTime,omitempty"`
	SupportPayCurrency string            `json:"supportPayCurrency,omitempty"`
	PassThroughInfo    string            `json:"passThroughInfo,omitempty"`
}

type BinancePayEnv struct {
	TerminalType string `json:"terminalType"`
}

type BinancePayGoods struct {
	GoodsType        string `json:"goodsType"`
	GoodsCategory    string `json:"goodsCategory"`
	ReferenceGoodsID string `json:"referenceGoodsId"`
	GoodsName        string `json:"goodsName"`
}

type BinancePayOrder struct {
	PrepayID    string `json:"prepayId"`
	CheckoutURL string `json:"checkoutUrl"`
	Deeplink    string `json:"deeplink"`
}

type binancePayCreateOrderResponse struct {
	Status       string           `json:"status"`
	Code         string           `json:"code"`
	ErrorMessage string           `json:"errorMessage"`
	Data         *BinancePayOrder `json:"data"`
}

type binancePayWebhookEvent struct {
	BizType   string                `json:"bizType"`
	BizIDStr  string                `json:"bizIdStr"`
	BizStatus string                `json:"bizStatus"`
	DataRaw   string                `json:"data"`
	Data      binancePayWebhookData `json:"-"`
}

type binancePayWebhookData struct {
	MerchantTradeNo string `json:"merchantTradeNo"`
	TotalFee        string `json:"totalFee"`
	OrderAmount     string `json:"orderAmount"`
	Currency        string `json:"currency"`
	TransactionID   string `json:"transactionId"`
	OpenUserID      string `json:"openUserId"`
}

func CreateBinancePayOrder(ctx context.Context, params *BinancePayCreateOrderParams) (*BinancePayOrder, error) {
	if params == nil {
		return nil, fmt.Errorf("missing Binance Pay order params")
	}

	body, err := common.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal Binance Pay order payload: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce := common.GetUUID()
	signature := signBinancePayRequest(timestamp, nonce, string(body), setting.BinancePayApiSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, binancePayBaseURL()+binancePayCreateOrderPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build Binance Pay order request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("BinancePay-Timestamp", timestamp)
	req.Header.Set("BinancePay-Nonce", nonce)
	req.Header.Set("BinancePay-Certificate-SN", setting.BinancePayApiKey)
	req.Header.Set("BinancePay-Signature", signature)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Binance Pay order: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Binance Pay order response: %w", err)
	}

	var result binancePayCreateOrderResponse
	if err := common.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode Binance Pay order response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest || !strings.EqualFold(result.Status, "SUCCESS") || result.Data == nil {
		if strings.TrimSpace(result.ErrorMessage) != "" {
			return nil, fmt.Errorf("Binance Pay error (%d): %s", resp.StatusCode, result.ErrorMessage)
		}
		if strings.TrimSpace(result.Code) != "" {
			return nil, fmt.Errorf("Binance Pay error (%d): %s", resp.StatusCode, result.Code)
		}
		return nil, fmt.Errorf("Binance Pay order request failed with status %d", resp.StatusCode)
	}
	if strings.TrimSpace(result.Data.CheckoutURL) == "" && strings.TrimSpace(result.Data.Deeplink) == "" {
		return nil, fmt.Errorf("Binance Pay returned empty checkout URL")
	}
	return result.Data, nil
}

func signBinancePayRequest(timestamp string, nonce string, body string, apiSecret string) string {
	payload := timestamp + "\n" + nonce + "\n" + body + "\n"
	mac := hmac.New(sha512.New, []byte(apiSecret))
	_, _ = mac.Write([]byte(payload))
	return strings.ToUpper(fmt.Sprintf("%x", mac.Sum(nil)))
}

func VerifyConfiguredBinancePayWebhook(payload string, timestamp string, nonce string, signature string) (*binancePayWebhookEvent, error) {
	return verifyBinancePayWebhook(payload, timestamp, nonce, signature, setting.BinancePayWebhookPubKey, time.Now)
}

func verifyBinancePayWebhook(payload string, timestamp string, nonce string, signature string, rawPublicKey string, now func() time.Time) (*binancePayWebhookEvent, error) {
	if strings.TrimSpace(timestamp) == "" || strings.TrimSpace(nonce) == "" || strings.TrimSpace(signature) == "" {
		return nil, fmt.Errorf("missing Binance Pay signature headers")
	}
	timestampMs, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Binance Pay timestamp")
	}
	if now == nil {
		now = time.Now
	}
	if math.Abs(float64(now().UnixMilli()-timestampMs)) > float64(binancePayDefaultTolerance.Milliseconds()) {
		return nil, fmt.Errorf("Binance Pay webhook timestamp outside tolerance window")
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid Binance Pay signature encoding")
	}
	publicKey, err := parseRSAPublicKey(rawPublicKey)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(timestamp + "\n" + nonce + "\n" + payload + "\n"))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signatureBytes); err != nil {
		return nil, fmt.Errorf("invalid Binance Pay webhook signature")
	}

	var event binancePayWebhookEvent
	if err := common.Unmarshal([]byte(payload), &event); err != nil {
		return nil, fmt.Errorf("parse Binance Pay webhook payload: %w", err)
	}
	if strings.TrimSpace(event.DataRaw) != "" {
		if err := common.Unmarshal([]byte(event.DataRaw), &event.Data); err != nil {
			return nil, fmt.Errorf("parse Binance Pay webhook data: %w", err)
		}
	}
	return &event, nil
}

func ResolveBinancePayTradeNo(event *binancePayWebhookEvent) (string, error) {
	if event == nil {
		return "", fmt.Errorf("missing Binance Pay webhook event")
	}
	tradeNo := strings.TrimSpace(event.Data.MerchantTradeNo)
	if tradeNo == "" {
		return "", fmt.Errorf("missing Binance Pay merchantTradeNo")
	}
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp != nil && topUp.PaymentProvider == model.PaymentProviderBinancePay {
		return tradeNo, nil
	}
	return "", fmt.Errorf("Binance Pay order not found for merchantTradeNo=%s", tradeNo)
}

func binancePayBaseURL() string {
	if setting.BinancePaySandbox {
		return binancePaySandboxBaseURL
	}
	return binancePayProdBaseURL
}
