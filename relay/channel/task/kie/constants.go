package kie

import "strings"

const (
	ChannelName    = "kie-ai"
	DefaultBaseURL = "https://api.kie.ai"

	ModelSeedance2              = "bytedance/seedance-2"
	ModelSeedream45TextToImage  = "seedream-4.5-text-to-image"
	ModelSeedream45ImageToImage = "seedream-4.5-image-to-image"
	ModelGPTImage2TextToImage   = "gpt-image-2-text-to-image"
	ModelGPTImage2ImageToImage  = "gpt-image-2-image-to-image"
	ModelNanoBanana2            = "nano-banana-2"
	ModelHappyHorseTextToVideo  = "happyhorse/text-to-video"
	ModelHappyHorseImageToVideo = "happyhorse/image-to-video"

	DefaultImageModel = ModelSeedream45TextToImage
	DefaultVideoModel = ModelSeedance2
)

var ModelList = []string{
	ModelSeedance2,
	ModelSeedream45TextToImage,
	ModelSeedream45ImageToImage,
	ModelGPTImage2TextToImage,
	ModelGPTImage2ImageToImage,
	ModelNanoBanana2,
	ModelHappyHorseTextToVideo,
	ModelHappyHorseImageToVideo,
}

const (
	outputKindImage = "image"
	outputKindVideo = "video"
)

type modelConfig struct {
	OutputKind string
	ImageKey   string
}

var modelConfigs = map[string]modelConfig{
	ModelSeedance2:              {OutputKind: outputKindVideo},
	ModelSeedream45TextToImage:  {OutputKind: outputKindImage},
	ModelSeedream45ImageToImage: {OutputKind: outputKindImage, ImageKey: "input_urls"},
	ModelGPTImage2TextToImage:   {OutputKind: outputKindImage},
	ModelGPTImage2ImageToImage:  {OutputKind: outputKindImage, ImageKey: "input_urls"},
	ModelNanoBanana2:            {OutputKind: outputKindImage, ImageKey: "image_input"},
	ModelHappyHorseTextToVideo:  {OutputKind: outputKindVideo},
	ModelHappyHorseImageToVideo: {OutputKind: outputKindVideo, ImageKey: "image_urls"},
}

func getModelConfig(modelName string) modelConfig {
	if cfg, ok := modelConfigs[modelName]; ok {
		return cfg
	}
	return modelConfig{OutputKind: outputKindVideo}
}

func DefaultModelForRequest(path string, hasImage bool) string {
	if strings.HasPrefix(path, "/v1/images/") {
		if hasImage {
			return ModelSeedream45ImageToImage
		}
		return DefaultImageModel
	}
	return DefaultVideoModel
}
