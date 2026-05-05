package fal

const (
	ChannelName    = "fal-ai"
	DefaultBaseURL = "https://fal.run"

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

var qualityRatioWeights = map[string]map[string]float64{
	ModelGPTImage2T2I: {"low": 1, "medium": 9, "high": 35},
	ModelGPTImage2I2I: {"low": 1, "medium": 4, "high": 15},
}
