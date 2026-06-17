package setting

var (
	BinancePayEnabled       bool
	BinancePaySandbox       bool
	BinancePayApiKey        string
	BinancePayApiSecret     string
	BinancePayWebhookPubKey string
	BinancePayReturnURL     string
	BinancePayCurrency      string  = "USDT"
	BinancePayUnitPrice     float64 = 1.0
	BinancePayMinTopUp      int     = 1
)
