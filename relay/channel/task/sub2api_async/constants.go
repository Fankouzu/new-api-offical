package sub2api_async

const (
	ChannelName    = "Sub2API-async"
	DefaultBaseURL = "https://api.sub2api.com"

	ModelGPTImage2TextToImage  = "gpt-image-2-text-to-image"
	ModelGPTImage2ImageToImage = "gpt-image-2-image-to-image"
)

var ModelList = []string{
	ModelGPTImage2TextToImage,
	ModelGPTImage2ImageToImage,
}

type modelConfig struct {
	ImageKey string
}

var modelConfigs = map[string]modelConfig{
	ModelGPTImage2TextToImage:  {},
	ModelGPTImage2ImageToImage: {ImageKey: "input_urls"},
}

func getModelConfig(modelName string) modelConfig {
	if cfg, ok := modelConfigs[modelName]; ok {
		return cfg
	}
	return modelConfig{}
}
