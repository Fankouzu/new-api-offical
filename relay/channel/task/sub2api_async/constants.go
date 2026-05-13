package sub2api_async

const (
	ChannelName    = "Sub2API-async"
	DefaultBaseURL = "https://api.sub2api.com"

	ModelGPTImage2TextToImage  = "gpt-image-2-text-to-image"
	ModelGPTImage2ImageToImage = "gpt-image-2-image-to-image"

	// Upstream URL paths. Centralized here so the async submit endpoint
	// (BuildRequestURL) and the deferred sync call (syncImageGenerationRequestPath)
	// reference one source. Sub2API uses the OpenAI Images API shape; if the
	// upstream renames or versions these paths, update them here only.
	UpstreamPathImagesGenerations = "/v1/images/generations"
	UpstreamPathImagesEdits       = "/v1/images/edits"
)

var ModelList = []string{
	ModelGPTImage2TextToImage,
	ModelGPTImage2ImageToImage,
}

type modelConfig struct {
	ImageKey    string
	ImageURLKey string
}

var modelConfigs = map[string]modelConfig{
	ModelGPTImage2TextToImage:  {},
	ModelGPTImage2ImageToImage: {ImageKey: "images", ImageURLKey: "image_url"},
}

func getModelConfig(modelName string) modelConfig {
	if cfg, ok := modelConfigs[modelName]; ok {
		return cfg
	}
	return modelConfig{}
}
