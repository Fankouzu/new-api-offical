package controller

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

type BinancePayRequest struct {
	Amount int64 `json:"amount"`
}

func RequestBinancePayAmount(c *gin.Context) {
	var req BinancePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < int64(setting.BinancePayMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.BinancePayMinTopUp)})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney := getBinancePayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success", "data": fmt.Sprintf("%.2f", payMoney)})
}

func RequestBinancePay(c *gin.Context) {
	if !setting.BinancePayEnabled {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "Binance Pay 支付未启用"})
		return
	}
	if strings.TrimSpace(setting.BinancePayApiKey) == "" ||
		strings.TrimSpace(setting.BinancePayApiSecret) == "" ||
		strings.TrimSpace(setting.BinancePayWebhookPubKey) == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "Binance Pay 配置不完整"})
		return
	}

	var req BinancePayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < int64(setting.BinancePayMinTopUp) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", setting.BinancePayMinTopUp)})
		return
	}

	id := c.GetInt("id")
	if _, err := model.GetUserById(id, false); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}

	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney := getBinancePayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	tradeNo := fmt.Sprintf("BINANCE_PAY-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          normalizeBinancePayTopUpAmount(req.Amount),
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodBinancePay,
		PaymentProvider: model.PaymentProviderBinancePay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Binance Pay 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	order, err := service.CreateBinancePayOrder(c.Request.Context(), &service.BinancePayCreateOrderParams{
		Env: service.BinancePayEnv{
			TerminalType: "WEB",
		},
		MerchantTradeNo: tradeNo,
		OrderAmount:     decimal.NewFromFloat(payMoney).StringFixed(2),
		Currency:        strings.ToUpper(strings.TrimSpace(setting.BinancePayCurrency)),
		Description:     "Balance top-up",
		GoodsDetails: []service.BinancePayGoods{
			{
				GoodsType:        "02",
				GoodsCategory:    "D000",
				ReferenceGoodsID: "balance-topup",
				GoodsName:        "Balance top-up",
			},
		},
		ReturnURL:  getBinancePayReturnURL(),
		CancelURL:  getBinancePayReturnURL(),
		ExpireTime: time.Now().Add(45 * time.Minute).UnixMilli(),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Binance Pay 创建订单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	checkoutURL := order.CheckoutURL
	if strings.TrimSpace(checkoutURL) == "" {
		checkoutURL = order.Deeplink
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url": checkoutURL,
			"prepay_id":    order.PrepayID,
			"order_id":     tradeNo,
		},
	})
}

func BinancePayWebhook(c *gin.Context) {
	if !isBinancePayWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Binance Pay webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.JSON(http.StatusForbidden, gin.H{"returnCode": "FAIL", "returnMessage": "webhook disabled"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Binance Pay webhook 读取请求体失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"returnCode": "FAIL", "returnMessage": "bad request"})
		return
	}

	event, err := service.VerifyConfiguredBinancePayWebhook(
		string(bodyBytes),
		c.GetHeader("Binancepay-Timestamp"),
		c.GetHeader("Binancepay-Nonce"),
		c.GetHeader("Binancepay-Signature"),
	)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Binance Pay webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.JSON(http.StatusUnauthorized, gin.H{"returnCode": "FAIL", "returnMessage": "invalid signature"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Binance Pay webhook 验签成功 biz_type=%s biz_status=%s biz_id=%s client_ip=%s", event.BizType, event.BizStatus, event.BizIDStr, c.ClientIP()))
	if event.BizStatus != "PAY_SUCCESS" {
		c.JSON(http.StatusOK, gin.H{"returnCode": "SUCCESS", "returnMessage": nil})
		return
	}

	tradeNo, err := service.ResolveBinancePayTradeNo(event)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Binance Pay webhook 订单号映射失败 biz_id=%s error=%q", event.BizIDStr, err.Error()))
		c.JSON(http.StatusOK, gin.H{"returnCode": "SUCCESS", "returnMessage": nil})
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	if err := model.RechargeBinancePay(tradeNo, c.ClientIP()); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Binance Pay 充值处理失败 trade_no=%s biz_id=%s client_ip=%s error=%q", tradeNo, event.BizIDStr, c.ClientIP(), err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"returnCode": "FAIL", "returnMessage": "retry"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Binance Pay 充值成功 trade_no=%s biz_id=%s client_ip=%s", tradeNo, event.BizIDStr, c.ClientIP()))
	c.JSON(http.StatusOK, gin.H{"returnCode": "SUCCESS", "returnMessage": nil})
}

func getBinancePayMoney(amount int64, group string) float64 {
	dAmount := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount = dAmount.Div(decimal.NewFromFloat(common.QuotaPerUnit))
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]; ok && ds > 0 {
		discount = ds
	}

	return dAmount.
		Mul(decimal.NewFromFloat(setting.BinancePayUnitPrice)).
		Mul(decimal.NewFromFloat(topupGroupRatio)).
		Mul(decimal.NewFromFloat(discount)).
		InexactFloat64()
}

func normalizeBinancePayTopUpAmount(amount int64) int64 {
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return amount
	}

	normalized := decimal.NewFromInt(amount).
		Div(decimal.NewFromFloat(common.QuotaPerUnit)).
		IntPart()
	if normalized < 1 {
		return 1
	}
	return normalized
}

func getBinancePayReturnURL() string {
	if strings.TrimSpace(setting.BinancePayReturnURL) != "" {
		return setting.BinancePayReturnURL
	}
	return strings.TrimRight(system_setting.ServerAddress, "/") + "/console/topup?show_history=true"
}
