package sub2api_async

const (
	ChannelName    = "Sub2API-async"
	DefaultBaseURL = "https://api.sub2api.com"

	ModelGPTImage2TextToImage  = "gpt-image-2-text-to-image"
	ModelGPTImage2ImageToImage = "gpt-image-2-image-to-image"
	ModelGeminiFlashImage      = "gemini-3.1-flash-image"

	// Upstream URL paths for OpenAI-compatible image endpoints.
	UpstreamPathImagesGenerations = "/v1/images/generations"
	UpstreamPathImagesEdits       = "/v1/images/edits"

	// Upstream path for Gemini image generation. The model name is embedded in
	// the path; the API key is appended as a query param at call time.
	UpstreamPathGeminiFlashImage = "/v1beta/models/gemini-3.1-flash-image:generateContent"
)

// validGeminiAspectRatios is the set of aspect ratios supported by
// gemini-3.1-flash-image (14 values per the API docs).
var validGeminiAspectRatios = map[string]bool{
	"1:1": true, "1:4": true, "1:8": true,
	"2:3": true, "3:2": true, "3:4": true,
	"4:1": true, "4:3": true, "4:5": true,
	"5:4": true, "8:1": true, "9:16": true,
	"16:9": true, "21:9": true,
}

// validGeminiImageSizes is the set of imageSize values accepted by Gemini.
// "512" is only available on gemini-3.1-flash-image (not Pro).
var validGeminiImageSizes = map[string]bool{
	"512": true, "1K": true, "2K": true, "4K": true,
}

var ModelList = []string{
	ModelGPTImage2TextToImage,
	ModelGPTImage2ImageToImage,
	ModelGeminiFlashImage,
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
