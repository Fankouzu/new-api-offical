package tencentvod

import "strings"

const (
	ModelOGImage2Low       = "og-image2-low"
	ModelOGImage2Medium    = "og-image2-medium"
	ModelOGImage2High      = "og-image2-high"
	ModelGG25Image         = "gg-2.5-image"
	ModelGG30Image         = "gg-3.0-image"
	ModelGG31Image         = "gg-3.1-image"
	ModelSI40Image         = "si-4.0-image"
	ModelSI45Image         = "si-4.5-image"
	ModelSI50LiteImage     = "si-5.0-lite-image"
	ModelQwen0925Image     = "qwen-0925-image"
	ModelHunyuan30Image    = "hunyuan-3.0-image"
	ModelHunyuan3DPanorama = "hunyuan-3d-panorama"
	ModelViduQ2Image       = "vidu-q2-image"
	ModelKling21Image      = "kling-2.1-image"
	ModelKlingImage30      = "kling-image-3.0"
	ModelKlingImage30Omni  = "kling-image-3.0-omni"
	ModelKlingO1Image      = "kling-o1-image"
	ModelKlingSceneImage   = "kling-scene-image"
	ModelMJv7Image         = "mj-v7-image"
	ModelJimeng40Image     = "jimeng-4.0-image"

	ModelKling16              = "kling-1.6"
	ModelKling20              = "kling-2.0"
	ModelKling21              = "kling-2.1"
	ModelKling25Pro           = "kling-2.5-pro"
	ModelKling26              = "kling-2.6"
	ModelKling26MotionControl = "kling-2.6-motion-control"
	ModelKling30              = "kling-3.0"
	ModelKling30Omni          = "kling-3.0-omni"
	ModelKlingAvatar          = "kling-avatar"
	ModelKlingIdentifyFace    = "kling-identifyface"
	ModelJimeng30Pro          = "jimeng-3.0-pro"
	ModelJimeng40             = "jimeng-4.0"
	ModelSV10Pro              = "sv-1.0-pro"
	ModelSV10LiteI2V          = "sv-1.0-lite-i2v"
	ModelViduQ2               = "vidu-q2"
	ModelViduQ2Turbo          = "vidu-q2-turbo"
	ModelViduQ2Pro            = "vidu-q2-pro"
	ModelViduQ2ProFast        = "vidu-q2-pro-fast"
	ModelViduQ3               = "vidu-q3"
	ModelViduQ3Turbo          = "vidu-q3-turbo"
	ModelViduQ3Pro            = "vidu-q3-pro"
	ModelViduQ3Mix            = "vidu-q3-mix"
	ModelHunyuan15            = "hunyuan-1.5"
	ModelHunyuan3DScene       = "hunyuan-3d-scene"
	ModelH210                 = "h2-1.0"
	ModelHailuo02             = "hailuo-02"
	ModelHailuo23             = "hailuo-2.3"
	ModelHailuo23Fast         = "hailuo-2.3-fast"
	ModelGV31                 = "gv-3.1"
	ModelGV31Fast             = "gv-3.1-fast"
	ModelGV31Lite             = "gv-3.1-lite"
	ModelOS20                 = "os-2.0"
	ModelPixVerseV56          = "pixverse-v5.6"
	ModelPixVerseV6           = "pixverse-v6"
	ModelPixVerseC1           = "pixverse-c1"
	ModelMingmou10            = "mingmou-1.0"
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
	ModelOGImage2Low:       {PublicModel: ModelOGImage2Low, Kind: modelKindImage, TencentModelName: "OG", TencentModelVersion: "image2_low", DefaultResolution: "1K"},
	ModelOGImage2Medium:    {PublicModel: ModelOGImage2Medium, Kind: modelKindImage, TencentModelName: "OG", TencentModelVersion: "image2_medium", DefaultResolution: "1K"},
	ModelOGImage2High:      {PublicModel: ModelOGImage2High, Kind: modelKindImage, TencentModelName: "OG", TencentModelVersion: "image2_high", DefaultResolution: "1K"},
	ModelGG25Image:         {PublicModel: ModelGG25Image, Kind: modelKindImage, TencentModelName: "GG", TencentModelVersion: "2.5", DefaultResolution: "1K"},
	ModelGG30Image:         {PublicModel: ModelGG30Image, Kind: modelKindImage, TencentModelName: "GG", TencentModelVersion: "3.0", DefaultResolution: "1K"},
	ModelGG31Image:         {PublicModel: ModelGG31Image, Kind: modelKindImage, TencentModelName: "GG", TencentModelVersion: "3.1", DefaultResolution: "512P"},
	ModelSI40Image:         {PublicModel: ModelSI40Image, Kind: modelKindImage, TencentModelName: "SI", TencentModelVersion: "4.0", DefaultResolution: "1K"},
	ModelSI45Image:         {PublicModel: ModelSI45Image, Kind: modelKindImage, TencentModelName: "SI", TencentModelVersion: "4.5", DefaultResolution: "1K"},
	ModelSI50LiteImage:     {PublicModel: ModelSI50LiteImage, Kind: modelKindImage, TencentModelName: "SI", TencentModelVersion: "5.0-lite", DefaultResolution: "1K"},
	ModelQwen0925Image:     {PublicModel: ModelQwen0925Image, Kind: modelKindImage, TencentModelName: "Qwen", TencentModelVersion: "0925", DefaultResolution: "1K"},
	ModelHunyuan30Image:    {PublicModel: ModelHunyuan30Image, Kind: modelKindImage, TencentModelName: "Hunyuan", TencentModelVersion: "3.0", DefaultResolution: "1K"},
	ModelHunyuan3DPanorama: {PublicModel: ModelHunyuan3DPanorama, Kind: modelKindImage, TencentModelName: "Hunyuan", TencentModelVersion: "3d_2.0", SceneType: "3d_panorama", DefaultResolution: "1K"},
	ModelViduQ2Image:       {PublicModel: ModelViduQ2Image, Kind: modelKindImage, TencentModelName: "Vidu", TencentModelVersion: "q2", DefaultResolution: "1K"},
	ModelKling21Image:      {PublicModel: ModelKling21Image, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "2.1", DefaultResolution: "1K"},
	ModelKlingImage30:      {PublicModel: ModelKlingImage30, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "3.0", DefaultResolution: "1K"},
	ModelKlingImage30Omni:  {PublicModel: ModelKlingImage30Omni, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "3.0-Omni", DefaultResolution: "1K"},
	ModelKlingO1Image:      {PublicModel: ModelKlingO1Image, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "O1", DefaultResolution: "1K"},
	ModelKlingSceneImage:   {PublicModel: ModelKlingSceneImage, Kind: modelKindImage, TencentModelName: "Kling", TencentModelVersion: "scene", DefaultResolution: "1K"},
	ModelMJv7Image:         {PublicModel: ModelMJv7Image, Kind: modelKindImage, TencentModelName: "MJ", TencentModelVersion: "v7", DefaultResolution: "1K"},
	ModelJimeng40Image:     {PublicModel: ModelJimeng40Image, Kind: modelKindImage, TencentModelName: "Jimeng", TencentModelVersion: "4.0", DefaultResolution: "1K"},

	ModelKling16:              {PublicModel: ModelKling16, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "1.6", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling20:              {PublicModel: ModelKling20, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "2.0", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling21:              {PublicModel: ModelKling21, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "2.1", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling25Pro:           {PublicModel: ModelKling25Pro, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "2.5-pro", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling26:              {PublicModel: ModelKling26, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "2.6", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling26MotionControl: {PublicModel: ModelKling26MotionControl, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "2.6", SceneType: "motion_control", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling30:              {PublicModel: ModelKling30, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "3.0", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKling30Omni:          {PublicModel: ModelKling30Omni, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "3.0-Omni", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKlingAvatar:          {PublicModel: ModelKlingAvatar, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "avater", DefaultResolution: "720P", DefaultDuration: 5},
	ModelKlingIdentifyFace:    {PublicModel: ModelKlingIdentifyFace, Kind: modelKindVideo, TencentModelName: "Kling", TencentModelVersion: "Identifyface", DefaultResolution: "720P", DefaultDuration: 5},
	ModelJimeng30Pro:          {PublicModel: ModelJimeng30Pro, Kind: modelKindVideo, TencentModelName: "Jimeng", TencentModelVersion: "3.0pro", DefaultResolution: "720P", DefaultDuration: 5},
	ModelJimeng40:             {PublicModel: ModelJimeng40, Kind: modelKindVideo, TencentModelName: "Jimeng", TencentModelVersion: "4.0", DefaultResolution: "720P", DefaultDuration: 5},
	ModelSV10Pro:              {PublicModel: ModelSV10Pro, Kind: modelKindVideo, TencentModelName: "SV", TencentModelVersion: "1.0-pro", DefaultResolution: "720P", DefaultDuration: 5},
	ModelSV10LiteI2V:          {PublicModel: ModelSV10LiteI2V, Kind: modelKindVideo, TencentModelName: "SV", TencentModelVersion: "1.0-lite-i2v", DefaultResolution: "720P", DefaultDuration: 5},
	ModelViduQ2:               {PublicModel: ModelViduQ2, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q2", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ2Turbo:          {PublicModel: ModelViduQ2Turbo, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q2-turbo", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ2Pro:            {PublicModel: ModelViduQ2Pro, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q2-pro", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ2ProFast:        {PublicModel: ModelViduQ2ProFast, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q2-pro-fast", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ3:               {PublicModel: ModelViduQ3, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ3Turbo:          {PublicModel: ModelViduQ3Turbo, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3-turbo", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ3Pro:            {PublicModel: ModelViduQ3Pro, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3-pro", DefaultResolution: "540P", DefaultDuration: 5},
	ModelViduQ3Mix:            {PublicModel: ModelViduQ3Mix, Kind: modelKindVideo, TencentModelName: "Vidu", TencentModelVersion: "q3-mix", DefaultResolution: "540P", DefaultDuration: 5},
	ModelHunyuan15:            {PublicModel: ModelHunyuan15, Kind: modelKindVideo, TencentModelName: "Hunyuan", TencentModelVersion: "1.5", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHunyuan3DScene:       {PublicModel: ModelHunyuan3DScene, Kind: modelKindVideo, TencentModelName: "Hunyuan", TencentModelVersion: "3d_2.0", SceneType: "3d_scene", DefaultResolution: "720P", DefaultDuration: 5},
	ModelH210:                 {PublicModel: ModelH210, Kind: modelKindVideo, TencentModelName: "H2", TencentModelVersion: "1.0", DefaultResolution: "720P", DefaultDuration: 5},
	ModelHailuo02:             {PublicModel: ModelHailuo02, Kind: modelKindVideo, TencentModelName: "Hailuo", TencentModelVersion: "02", DefaultResolution: "768P", DefaultDuration: 5},
	ModelHailuo23:             {PublicModel: ModelHailuo23, Kind: modelKindVideo, TencentModelName: "Hailuo", TencentModelVersion: "2.3", DefaultResolution: "768P", DefaultDuration: 5},
	ModelHailuo23Fast:         {PublicModel: ModelHailuo23Fast, Kind: modelKindVideo, TencentModelName: "Hailuo", TencentModelVersion: "2.3-fast", DefaultResolution: "768P", DefaultDuration: 5},
	ModelGV31:                 {PublicModel: ModelGV31, Kind: modelKindVideo, TencentModelName: "GV", TencentModelVersion: "3.1", DefaultResolution: "720P", DefaultDuration: 5},
	ModelGV31Fast:             {PublicModel: ModelGV31Fast, Kind: modelKindVideo, TencentModelName: "GV", TencentModelVersion: "3.1-fast", DefaultResolution: "720P", DefaultDuration: 5},
	ModelGV31Lite:             {PublicModel: ModelGV31Lite, Kind: modelKindVideo, TencentModelName: "GV", TencentModelVersion: "3.1-lite", DefaultResolution: "720P", DefaultDuration: 5},
	ModelOS20:                 {PublicModel: ModelOS20, Kind: modelKindVideo, TencentModelName: "OS", TencentModelVersion: "2.0", DefaultResolution: "720P", DefaultDuration: 5},
	ModelPixVerseV56:          {PublicModel: ModelPixVerseV56, Kind: modelKindVideo, TencentModelName: "PixVerse", TencentModelVersion: "v5.6", DefaultResolution: "540P", DefaultDuration: 5},
	ModelPixVerseV6:           {PublicModel: ModelPixVerseV6, Kind: modelKindVideo, TencentModelName: "PixVerse", TencentModelVersion: "v6", DefaultResolution: "540P", DefaultDuration: 5},
	ModelPixVerseC1:           {PublicModel: ModelPixVerseC1, Kind: modelKindVideo, TencentModelName: "PixVerse", TencentModelVersion: "c1", DefaultResolution: "540P", DefaultDuration: 5},
	ModelMingmou10:            {PublicModel: ModelMingmou10, Kind: modelKindVideo, TencentModelName: "Mingmou", TencentModelVersion: "1.0", DefaultResolution: "720P", DefaultDuration: 5},
}

var ModelList = []string{
	ModelOGImage2Low,
	ModelOGImage2Medium,
	ModelOGImage2High,
	ModelGG25Image,
	ModelGG30Image,
	ModelGG31Image,
	ModelSI40Image,
	ModelSI45Image,
	ModelSI50LiteImage,
	ModelQwen0925Image,
	ModelHunyuan30Image,
	ModelHunyuan3DPanorama,
	ModelViduQ2Image,
	ModelKling21Image,
	ModelKlingImage30,
	ModelKlingImage30Omni,
	ModelKlingO1Image,
	ModelKlingSceneImage,
	ModelMJv7Image,
	ModelJimeng40Image,
	ModelKling16,
	ModelKling20,
	ModelKling21,
	ModelKling25Pro,
	ModelKling26,
	ModelKling26MotionControl,
	ModelKling30,
	ModelKling30Omni,
	ModelKlingAvatar,
	ModelKlingIdentifyFace,
	ModelJimeng30Pro,
	ModelJimeng40,
	ModelSV10Pro,
	ModelSV10LiteI2V,
	ModelViduQ2,
	ModelViduQ2Turbo,
	ModelViduQ2Pro,
	ModelViduQ2ProFast,
	ModelViduQ3,
	ModelViduQ3Turbo,
	ModelViduQ3Pro,
	ModelViduQ3Mix,
	ModelHunyuan15,
	ModelHunyuan3DScene,
	ModelH210,
	ModelHailuo02,
	ModelHailuo23,
	ModelHailuo23Fast,
	ModelGV31,
	ModelGV31Fast,
	ModelGV31Lite,
	ModelOS20,
	ModelPixVerseV56,
	ModelPixVerseV6,
	ModelPixVerseC1,
	ModelMingmou10,
}

func lookupModelSpec(model string) (modelSpec, bool) {
	spec, ok := modelSpecs[strings.TrimSpace(model)]
	return spec, ok
}
