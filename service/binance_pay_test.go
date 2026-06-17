package service

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBinancePayTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}))
	return db
}

func TestSignBinancePayRequest(t *testing.T) {
	signature := signBinancePayRequest("1700000000000", "nonce-123", `{"merchantTradeNo":"BINANCE_PAY-1"}`, "secret")

	require.Equal(t, "DE59E201B5E15EA719442534298212BE8E08389E967F4289BDA053B9072219B19FA9C58A209209BD64D73BC647D62F788EB3940EFAB409854C182B8FD788E442", signature)
}

func TestBinancePayCreateOrderResponseParsesCheckoutURL(t *testing.T) {
	var result binancePayCreateOrderResponse
	err := common.Unmarshal([]byte(`{
		"status": "SUCCESS",
		"code": "000000",
		"data": {
			"prepayId": "29383937493038367292",
			"checkoutUrl": "https://pay.binance.com/en/checkout/example",
			"deeplink": "bnc://app.binance.com/payment/secpay"
		}
	}`), &result)
	require.NoError(t, err)
	require.Equal(t, "SUCCESS", result.Status)
	require.NotNil(t, result.Data)
	require.Equal(t, "https://pay.binance.com/en/checkout/example", result.Data.CheckoutURL)
}

func TestVerifyBinancePayWebhook(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyDER}))

	timestamp := "1700000000000"
	nonce := "nonce-123"
	body := `{"bizType":"PAY","bizIdStr":"biz_123","bizStatus":"PAY_SUCCESS","data":"{\"merchantTradeNo\":\"BINANCE_PAY-1-1700000000000-abc123\",\"orderAmount\":\"10.00\",\"currency\":\"USDT\"}"}`
	digest := sha256.Sum256([]byte(timestamp + "\n" + nonce + "\n" + body + "\n"))
	signatureBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	require.NoError(t, err)

	event, err := verifyBinancePayWebhook(body, timestamp, nonce, base64.StdEncoding.EncodeToString(signatureBytes), publicKeyPEM, func() time.Time {
		return time.UnixMilli(1700000000000)
	})
	require.NoError(t, err)
	require.Equal(t, "PAY_SUCCESS", event.BizStatus)
	require.Equal(t, "BINANCE_PAY-1-1700000000000-abc123", event.Data.MerchantTradeNo)
}

func TestResolveBinancePayTradeNo(t *testing.T) {
	db := setupBinancePayTestDB(t)

	topUp := &model.TopUp{
		UserId:          1,
		Amount:          10,
		Money:           10,
		TradeNo:         "BINANCE_PAY-1-1700000000000-abc123",
		PaymentMethod:   model.PaymentMethodBinancePay,
		PaymentProvider: model.PaymentProviderBinancePay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(topUp).Error)

	tradeNo, err := ResolveBinancePayTradeNo(&binancePayWebhookEvent{
		Data: binancePayWebhookData{
			MerchantTradeNo: topUp.TradeNo,
		},
	})
	require.NoError(t, err)
	require.Equal(t, topUp.TradeNo, tradeNo)
}
