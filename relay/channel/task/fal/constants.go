package fal

const (
	ChannelName    = "fal-ai"
	DefaultBaseURL = "https://queue.fal.run"

	ModelGPTImage2T2I = "openai/gpt-image-2"
	ModelGPTImage2I2I = "openai/gpt-image-2/edit"
)

var ModelList = []string{
	ModelGPTImage2T2I,
	ModelGPTImage2I2I,
}

var modelConfigs = map[string]modelConfig{
	ModelGPTImage2T2I: {OutputKind: outputKindImage},
	ModelGPTImage2I2I: {OutputKind: outputKindImage, ImageKey: "image_urls"},
}

const (
	outputKindImage = "image"
)

type modelConfig struct {
	OutputKind string
	ImageKey   string
}

func getModelConfig(modelName string) modelConfig {
	if cfg, ok := modelConfigs[modelName]; ok {
		return cfg
	}
	return modelConfig{OutputKind: outputKindImage}
}

// pricingMatrix maps size to quality-to-multiplier for t2i (openai/gpt-image-2).
// Base: 1024×1024 low = $0.006 = 1×.
var pricingMatrix = map[string]map[string]float64{
	"1024x768":  {"low": 0.8333, "medium": 6.1667, "high": 24.1667},
	"1024x1024": {"low": 1.0000, "medium": 8.8333, "high": 35.1667},
	"1024x1536": {"low": 0.8333, "medium": 7.0000, "high": 27.5000},
	"1920x1080": {"low": 0.8333, "medium": 6.6667, "high": 26.3333},
	"2560x1440": {"low": 1.1667, "medium": 9.3333, "high": 37.0000},
	"3840x2160": {"low": 2.0000, "medium": 16.8333, "high": 66.8333},
}

// pricingMatrixEdit maps size to quality-to-multiplier for i2i (openai/gpt-image-2/edit).
// Base: same as t2i, 1024×1024 low t2i = $0.006 = 1×.
var pricingMatrixEdit = map[string]map[string]float64{
	"1024x768":  {"low": 1.8333, "medium": 7.1667, "high": 25.1667},
	"1024x1024": {"low": 2.5000, "medium": 10.1667, "high": 36.5000},
	"1024x1536": {"low": 3.0000, "medium": 9.0000, "high": 29.6667},
	"1920x1080": {"low": 2.8333, "medium": 8.8333, "high": 26.3333},
	"2560x1440": {"low": 3.1667, "medium": 11.3333, "high": 39.0000},
	"3840x2160": {"low": 4.0000, "medium": 18.8333, "high": 68.8333},
}

func getPricingMultiplier(model, size, quality string) (float64, bool) {
	matrix := pricingMatrix
	if model == ModelGPTImage2I2I {
		matrix = pricingMatrixEdit
	}
	qmap, ok := matrix[size]
	if !ok {
		return 0, false
	}
	m, ok := qmap[quality]
	return m, ok
}
