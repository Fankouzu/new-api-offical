package tencentvod

import "strings"

const (
	ModelGV31              = "gv-3.1"
	ModelGV31Fast          = "gv-3.1-fast"
	ModelKling30Std        = "kling-3.0-std"
	ModelKling30Pro        = "kling-3.0-pro"
	ModelKling30Omni       = "kling-3.0-omni"
	ModelKling26           = "kling-2.6"
	ModelViduQ3Turbo       = "vidu-q3-turbo"
	ModelViduQ3Pro         = "vidu-q3-pro"
	ModelViduQ3Mix         = "vidu-q3-mix"
	ModelPixVerseV56       = "pixverse-v5.6"
	ModelPixVerseV6        = "pixverse-v6"
	ModelPixVerseC1        = "pixverse-c1"
	ModelHailuo02          = "hailuo-02"
	ModelHailuo23          = "hailuo-2.3"
	ModelHailuo23Fast      = "hailuo-2.3-fast"
	ModelH210              = "h2-1.0"
	ModelHunyuan3DScene    = "hunyuan-3d-scene"
	ModelHunyuan3DPanorama = "hunyuan-3d-panorama"
	ModelOGImage2Low       = "og-image2-low"
	ModelOGImage2Medium    = "og-image2-medium"
	ModelOGImage2High      = "og-image2-high"
	ModelKlingImage30      = "kling-image-3.0"
	ModelKlingImage30Omni  = "kling-image-3.0-omni"
	ModelViduImage         = "vidu-image"
)

const (
	modelKindImage = "image"
	modelKindVideo = "video"
)

type modelSpec struct {
	PublicModel         string
	Kind                string
	TencentModelName    string
	TencentModelVersion string
	SceneType           string
	DefaultResolution   string
	DefaultDuration     int
	TaskMultiplier      float64
}

var imageResolutionRatios = map[string]float64{
	"512P": 1,
	"1K":   1,
	"2K":   1.4,
	"4K":   1.8,
}

var videoResolutionRatios = map[string]float64{
	"360P":  1,
	"480P":  1,
	"540P":  1.1,
	"720P":  1.5,
	"768P":  1.5,
	"1080P": 1.75,
	"2K":    2.1,
	"4K":    2.5,
}

var modelSpecs = map[string]modelSpec{
	ModelGV31:              {PublicModel: ModelGV31, Kind: modelKindVideo, TencentModelName: "GV", TencentModelVersion: "3.1", DefaultResolution: "720P", DefaultDuration: 5},
	ModelGV31Fast:          {PublicModel: ModelGV31Fast, Kind: modelKindVideo, TencentModelName: "GV", TencentModelVersion: "3.1-fast", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling30Std:        {PublicModel: ModelKling30Std, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "3.0-std", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling30Pro:        {PublicModel: ModelKling30Pro, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "3.0-pro", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling30Omni:       {PublicModel: ModelKling30Omni, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "3.0-omni", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling26:           {PublicModel: ModelKling26, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "2.6", DefaultResolution: "720P", DefaultDuration: 5},
	ModelViduQ3Turbo:       {PublicModel: ModelViduQ3Turbo, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3-turbo", DefaultResolution: "480P", DefaultDuration: 5},
	ModelViduQ3Pro:         {PublicModel: ModelViduQ3Pro, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3-pro", DefaultResolution: "480P", DefaultDuration: 5},
	ModelViduQ3Mix:         {PublicModel: ModelViduQ3Mix, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3-mix", DefaultResolution: "480P", DefaultDuration: 5},
	ModelPixVerseV56:       {PublicModel: ModelPixVerseV56, Kind: modelKindVideo, TencentModelName: "PixVerse", TencentModelVersion: "5.6", DefaultResolution: "720P", DefaultDuration: 5},
	ModelPixVerseV6:        {PublicModel: ModelPixVerseV6, Kind: modelKindVideo, TencentModelName: "PixVerse", TencentModelVersion: "6", DefaultResolution: "720P", DefaultDuration: 5},
	ModelPixVerseC1:        {PublicModel: ModelPixVerseC1, Kind: modelKindVideo, TencentModelName: "PixVerse", TencentModelVersion: "c1", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHailuo02:          {PublicModel: ModelHailuo02, Kind: modelKindVideo, TencentModelName: "Hailuo", TencentModelVersion: "02", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHailuo23:          {PublicModel: ModelHailuo23, Kind: modelKindVideo, TencentModelName: "Hailuo", TencentModelVersion: "2.3", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHailuo23Fast:      {PublicModel: ModelHailuo23Fast, Kind: modelKindVideo, TencentModelName: "Hailuo", TencentModelVersion: "2.3-fast", DefaultResolution: "720P", DefaultDuration: 5},
	ModelH210:              {PublicModel: ModelH210, Kind: modelKindVideo, TencentModelName: "H2", TencentModelVersion: "1.0", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHunyuan3DScene:    {PublicModel: ModelHunyuan3DScene, Kind: modelKindVideo, TencentModelName: "Hunyuan", TencentModelVersion: "3d_2.0", SceneType: "3d_scene", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHunyuan3DPanorama: {PublicModel: ModelHunyuan3DPanorama, Kind: modelKindImage, TencentModelName: "Hunyuan", TencentModelVersion: "3d_2.0", SceneType: "3d_panorama", DefaultResolution: "1K"},
	ModelOGImage2Low:       {PublicModel: ModelOGImage2Low, Kind: modelKindImage, TencentModelName: "OG", TencentModelVersion: "image2_low", DefaultResolution: "1K"},
	ModelOGImage2Medium:    {PublicModel: ModelOGImage2Medium, Kind: modelKindImage, TencentModelName: "OG", TencentModelVersion: "image2_medium", DefaultResolution: "1K"},
	ModelOGImage2High:      {PublicModel: ModelOGImage2High, Kind: modelKindImage, TencentModelName: "OG", TencentModelVersion: "image2_high", DefaultResolution: "1K"},
	ModelKlingImage30:      {PublicModel: ModelKlingImage30, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "3.0", DefaultResolution: "1K"},
	ModelKlingImage30Omni:  {PublicModel: ModelKlingImage30Omni, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "3.0-omni", DefaultResolution: "1K"},
	ModelViduImage:         {PublicModel: ModelViduImage, Kind: modelKindImage, TencentModelName: "Vidu", TencentModelVersion: "image", DefaultResolution: "1K"},
}

var ModelList = []string{
	ModelGV31,
	ModelGV31Fast,
	ModelKling30Std,
	ModelKling30Pro,
	ModelKling30Omni,
	ModelKling26,
	ModelViduQ3Turbo,
	ModelViduQ3Pro,
	ModelViduQ3Mix,
	ModelPixVerseV56,
	ModelPixVerseV6,
	ModelPixVerseC1,
	ModelHailuo02,
	ModelHailuo23,
	ModelHailuo23Fast,
	ModelH210,
	ModelHunyuan3DScene,
	ModelHunyuan3DPanorama,
	ModelOGImage2Low,
	ModelOGImage2Medium,
	ModelOGImage2High,
	ModelKlingImage30,
	ModelKlingImage30Omni,
	ModelViduImage,
}

func lookupModelSpec(model string) (modelSpec, bool) {
	spec, ok := modelSpecs[strings.TrimSpace(model)]
	return spec, ok
}
