package pingxingshijie

// AssetPlaceholderModel is used when /v1/assets/upload JSON omits "model" (distributor still needs a model for routing).
const AssetPlaceholderModel = "pingxingshijie-asset"

// ModelList combines video + image + common text model ids for channel configuration (align with PingXingShiJie / Ark console).
var ModelList = []string{
	"doubao-seedance-1-0-pro-250528",
	"doubao-seedance-1-0-lite-t2v",
	"doubao-seedance-1-0-lite-i2v",
	"doubao-seedance-1-5-pro-251215",
	"doubao-seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
	"doubao-seedream-4-0-250828",
	"doubao-seedream-4-0-250815",
	// Text/chat (same host; upstream uses Volcengine Ark /api/v3/chat/completions — use exact endpoint IDs from your console when mapping models).
	"doubao-pro-32k",
	"doubao-lite-32k",
}

var ChannelName = "pingxingshijie-video"

// videoInputRatioMap discount when video input is present (with-video / without-video pricing).
// Admins should set ModelRatio to the higher "without video" rate;
// the system multiplies by this ratio when video input is detected.
var videoInputRatioMap = map[string]float64{
	"doubao-seedance-2-0-260128":      28.0 / 46.0, // ~0.6087
	"doubao-seedance-2-0-fast-260128": 22.0 / 37.0, // ~0.5946
}

func GetVideoInputRatio(modelName string) (float64, bool) {
	r, ok := videoInputRatioMap[modelName]
	return r, ok
}
