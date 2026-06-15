package tencentvod

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func useTencentVODPricingFixture(t *testing.T) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	original, hadOriginal := common.OptionMap[tencentVODPricingMatrixOptionKey]
	common.OptionMap[tencentVODPricingMatrixOptionKey] = tencentVODPricingFixtureJSON(t)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if hadOriginal {
			common.OptionMap[tencentVODPricingMatrixOptionKey] = original
		} else {
			delete(common.OptionMap, tencentVODPricingMatrixOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})
}

func tencentVODPricingFixtureJSON(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/pricing-matrix.json")
	if err != nil {
		t.Fatalf("read Tencent VOD pricing fixture: %v", err)
	}
	return string(data)
}

func priceRowsForTest(t *testing.T) map[string]map[string]vodPriceRow {
	t.Helper()
	rows, err := loadTencentVODPriceRows()
	if err != nil {
		t.Fatalf("load Tencent VOD pricing rows: %v", err)
	}
	return rows
}

func TestTencentVODBillingFailsWithoutPublishedMatrix(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	original, hadOriginal := common.OptionMap[tencentVODPricingMatrixOptionKey]
	delete(common.OptionMap, tencentVODPricingMatrixOptionKey)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if hadOriginal {
			common.OptionMap[tencentVODPricingMatrixOptionKey] = original
		} else {
			delete(common.OptionMap, tencentVODPricingMatrixOptionKey)
		}
		common.OptionMapRWMutex.Unlock()
	})

	a := &TaskAdaptor{}
	c := taskContext(t, `{"model":"kling-2.1-image","prompt":"image","resolution":"4K"}`)
	_, err := a.EstimateBillingWithError(c, &relaycommon.RelayInfo{OriginModelName: "kling-2.1-image"})
	if err == nil || !strings.Contains(err.Error(), tencentVODPricingMatrixOptionKey) {
		t.Fatalf("expected missing matrix error, got %v", err)
	}
}

func TestParseConfigSupportsDelimitedAndJSONKeys(t *testing.T) {
	delimited, err := parseConfig("sid|skey|1500044236", "ap-guangzhou")
	if err != nil {
		t.Fatal(err)
	}
	if delimited.SecretID != "sid" || delimited.SecretKey != "skey" || delimited.SubAppID != 1500044236 || delimited.Region != "ap-guangzhou" {
		t.Fatalf("delimited config = %+v", delimited)
	}

	jsonCfg, err := parseConfig(`{"secret_id":"json_sid","secret_key":"json_skey","sub_app_id":123}`, "ap-shanghai")
	if err != nil {
		t.Fatal(err)
	}
	if jsonCfg.SecretID != "json_sid" || jsonCfg.SecretKey != "json_skey" || jsonCfg.SubAppID != 123 || jsonCfg.Region != "ap-shanghai" {
		t.Fatalf("json config = %+v", jsonCfg)
	}
}

func TestParseConfigRequiresRegionAndSecretFields(t *testing.T) {
	if _, err := parseConfig("sid|skey|1", ""); err == nil {
		t.Fatal("expected missing region error")
	}
	if _, err := parseConfig("sid|skey", "ap-guangzhou"); err == nil {
		t.Fatal("expected invalid key format error")
	}
}

func TestConvertImageRequestPayload(t *testing.T) {
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt: "make an image",
		Images: []string{"https://example.com/in.png"},
		Size:   "2K",
		Metadata: map[string]any{
			"n":            3,
			"aspect_ratio": "16:9",
		},
	}, &relaycommon.RelayInfo{
		OriginModelName: "kling-image-3.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-image-3.0",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if action != actionCreateImageTask {
		t.Fatalf("action = %q", action)
	}
	if body.SubAppID != 1500044236 || body.ModelName != "Kling" || body.ModelVersion != "3.0" || body.Prompt != "make an image" {
		t.Fatalf("body = %+v", body)
	}
	if body.OutputConfig["Resolution"] != "2K" || body.OutputConfig["AspectRatio"] != "16:9" {
		t.Fatalf("output config = %#v", body.OutputConfig)
	}
	if _, ok := body.OutputConfig["Count"]; ok {
		t.Fatalf("Tencent VOD does not accept OutputConfig.Count: %#v", body.OutputConfig)
	}
	if len(body.FileInfos) != 1 || body.FileInfos[0].URL != "https://example.com/in.png" {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	if body.FileInfos[0].Type != "Url" || body.FileInfos[0].Category != "" || body.FileInfos[0].Usage != "" {
		t.Fatalf("image file info should only use Tencent image input fields: %#v", body.FileInfos[0])
	}
}

func TestConvertImageRequestPayloadAcceptsOpenAIImageArray(t *testing.T) {
	c := taskContext(t, `{
		"model": "og-image2-high",
		"prompt": "edit image",
		"image": ["https://example.com/input.jpg"],
		"size": "2496x1664"
	}`)
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		t.Fatal(err)
	}
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&req, &relaycommon.RelayInfo{
		OriginModelName: "og-image2-high",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "og-image2-high",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if action != actionCreateImageTask {
		t.Fatalf("action = %q", action)
	}
	if body.ModelName != "OG" || body.ModelVersion != "image2_high" {
		t.Fatalf("model = %s/%s", body.ModelName, body.ModelVersion)
	}
	if len(body.FileInfos) != 1 {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	got := body.FileInfos[0]
	if got.Type != "Url" || got.Category != "" || got.URL != "https://example.com/input.jpg" || got.Usage != "" || got.ID != "" {
		t.Fatalf("file info = %#v", got)
	}
}

func TestConvertVideoRequestPayloadPassesViduOffPeak(t *testing.T) {
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Model:      "vidu-q2",
		Prompt:     "make a video",
		Resolution: "1080P",
		Duration:   5,
		Metadata: map[string]any{
			"off_peak": true,
		},
	}, &relaycommon.RelayInfo{
		OriginModelName: "vidu-q2",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q2",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if action != actionCreateVideoTask {
		t.Fatalf("action = %q", action)
	}
	if body.OutputConfig["OffPeak"] != "Enabled" {
		t.Fatalf("expected OffPeak=Enabled, output config = %#v", body.OutputConfig)
	}
	if body.OutputConfig["Duration"] != 5 || body.OutputConfig["Resolution"] != "1080P" {
		t.Fatalf("output config = %#v", body.OutputConfig)
	}
}

func TestConvertVideoRequestPayload(t *testing.T) {
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt:     "make a video",
		Image:      "https://example.com/first.png",
		Resolution: "1080P",
		Duration:   5,
	}, &relaycommon.RelayInfo{
		OriginModelName: "vidu-q3-turbo",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3-turbo",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if action != actionCreateVideoTask {
		t.Fatalf("action = %q", action)
	}
	if body.ModelName != "Vidu" || body.ModelVersion != "q3-turbo" {
		t.Fatalf("model = %s/%s", body.ModelName, body.ModelVersion)
	}
	if body.OutputConfig["Resolution"] != "1080P" || body.OutputConfig["Duration"] != 5 {
		t.Fatalf("output config = %#v", body.OutputConfig)
	}
	if len(body.FileInfos) != 1 {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	if body.FileInfos[0].Type != "Url" || body.FileInfos[0].Category != "Image" || body.FileInfos[0].Usage != "FirstFrame" {
		t.Fatalf("video single-image input should be sent as first frame: %#v", body.FileInfos[0])
	}
}

func TestConvertVideoRequestPayloadMapsMultipleImagesToFrames(t *testing.T) {
	a := &TaskAdaptor{}
	body, action, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt: "make a video",
		Images: []string{
			"https://example.com/first.png",
			"https://example.com/last.png",
			"https://example.com/ref.png",
		},
	}, &relaycommon.RelayInfo{
		OriginModelName: "kling-3.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-3.0",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if action != actionCreateVideoTask {
		t.Fatalf("action = %q", action)
	}
	if len(body.FileInfos) != 2 {
		t.Fatalf("file infos = %#v lastFrame=%q", body.FileInfos, body.LastFrameURL)
	}
	if body.FileInfos[0].Usage != "FirstFrame" || body.LastFrameURL != "https://example.com/last.png" || body.FileInfos[1].Usage != "Reference" {
		t.Fatalf("video file inputs = %#v lastFrame=%q", body.FileInfos, body.LastFrameURL)
	}
}

func TestConvertVideoRequestPayloadKeepsInputReferenceAsReference(t *testing.T) {
	a := &TaskAdaptor{}
	body, _, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt:         "make a video",
		Image:          "https://example.com/first.png",
		InputReference: "https://example.com/ref.png",
	}, &relaycommon.RelayInfo{
		OriginModelName: "kling-3.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-3.0",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(body.FileInfos) != 2 {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
	if body.FileInfos[0].Usage != "FirstFrame" || body.FileInfos[1].Usage != "Reference" {
		t.Fatalf("video file usages = %#v", body.FileInfos)
	}
}

func TestConvertVideoRequestPayloadMapsSecondFileIDToLastFrameID(t *testing.T) {
	a := &TaskAdaptor{}
	body, _, err := a.convertToTencentPayload(&relaycommon.TaskSubmitReq{
		Prompt: "make a video",
		Images: []string{
			"https://example.com/first.png",
			"vod-file-id-last",
		},
	}, &relaycommon.RelayInfo{
		OriginModelName: "kling-3.0",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-3.0",
			ApiVersion:        "ap-guangzhou",
			ApiKey:            "sid|skey|1500044236",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if body.LastFrameID != "vod-file-id-last" || body.LastFrameURL != "" {
		t.Fatalf("last frame fields = url:%q id:%q", body.LastFrameURL, body.LastFrameID)
	}
	if len(body.FileInfos) != 1 || body.FileInfos[0].Usage != "FirstFrame" {
		t.Fatalf("file infos = %#v", body.FileInfos)
	}
}

func TestDocumentedModelMatrixIncludesRequestedVendors(t *testing.T) {
	cases := []struct {
		publicModel  string
		kind         string
		modelName    string
		modelVersion string
	}{
		{publicModel: ModelGG31Image, kind: modelKindImage, modelName: "GG", modelVersion: "3.1"},
		{publicModel: ModelMJv7Image, kind: modelKindImage, modelName: "MJ", modelVersion: "v7"},
		{publicModel: ModelQwen0925Image, kind: modelKindImage, modelName: "Qwen", modelVersion: "0925"},
		{publicModel: ModelSI50LiteImage, kind: modelKindImage, modelName: "SI", modelVersion: "5.0-lite"},
		{publicModel: ModelJimeng40, kind: modelKindVideo, modelName: "Jimeng", modelVersion: "4.0"},
		{publicModel: ModelSV10Pro, kind: modelKindVideo, modelName: "SV", modelVersion: "1.0-pro"},
		{publicModel: ModelOS20, kind: modelKindVideo, modelName: "OS", modelVersion: "2.0"},
	}

	for _, tc := range cases {
		t.Run(tc.publicModel, func(t *testing.T) {
			spec, ok := lookupModelSpec(tc.publicModel)
			if !ok {
				t.Fatalf("missing model spec for %s", tc.publicModel)
			}
			if spec.Kind != tc.kind || spec.TencentModelName != tc.modelName || spec.TencentModelVersion != tc.modelVersion {
				t.Fatalf("spec = %+v", spec)
			}
			if !containsModel(ModelList, tc.publicModel) {
				t.Fatalf("%s missing from ModelList", tc.publicModel)
			}
		})
	}
}

func TestEstimateBillingUsesResolutionDurationAndCount(t *testing.T) {
	useTencentVODPricingFixture(t)
	a := &TaskAdaptor{}
	c := taskContext(t, `{"model":"vidu-q3-turbo","prompt":"video","resolution":"1080P","duration":5}`)
	video := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "vidu-q3-turbo"})
	if video["duration"] != 5 || !floatClose(video["resolution"], 0.070/0.040) {
		t.Fatalf("video ratios = %#v", video)
	}

	c = taskContext(t, `{"model":"kling-image-3.0","prompt":"image","resolution":"4K","n":4}`)
	image := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-image-3.0"})
	if image["resolution"] != 2 || image["count"] != 4 {
		t.Fatalf("image ratios = %#v", image)
	}
}

func TestEstimateBillingUsesTencentVODModeSpecificPrices(t *testing.T) {
	useTencentVODPricingFixture(t)
	a := &TaskAdaptor{}

	c := taskContext(t, `{"model":"kling-3.0","prompt":"video","resolution":"1080P","duration":5,"metadata":{"audio_generation":"Enabled","voice_id":"voice-1"}}`)
	kling := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-3.0"})
	if kling["duration"] != 5 || !floatClose(kling["resolution"], 0.196/0.084) {
		t.Fatalf("kling audio voice ratios = %#v", kling)
	}

	c = taskContext(t, `{"model":"kling-3.0","prompt":"video","resolution":"1080P","duration":5,"metadata":{"audio_generation":"Disabled"}}`)
	klingSilent := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-3.0"})
	if !floatClose(klingSilent["resolution"], 0.112/0.084) {
		t.Fatalf("kling disabled audio should use silent ratios = %#v", klingSilent)
	}

	c = taskContext(t, `{"model":"gv-3.1-fast","prompt":"video","resolution":"2K","duration":8,"metadata":{"audio":true}}`)
	gv := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "gv-3.1-fast"})
	if gv["duration"] != 8 || !floatClose(gv["resolution"], 0.2500/0.1000) {
		t.Fatalf("gv audio ratios = %#v", gv)
	}

	c = taskContext(t, `{"model":"gv-3.1","prompt":"video","resolution":"4K","duration":8,"metadata":{"audio":true}}`)
	gv31Audio := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "gv-3.1"})
	if gv31Audio["duration"] != 8 || !floatClose(gv31Audio["resolution"], 0.6000/0.2000) {
		t.Fatalf("gv 3.1 audio ratios = %#v", gv31Audio)
	}

	c = taskContext(t, `{"model":"gv-3.1","prompt":"video","resolution":"4K","duration":8}`)
	gv31Silent := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "gv-3.1"})
	if gv31Silent["duration"] != 8 || !floatClose(gv31Silent["resolution"], 0.4000/0.2000) {
		t.Fatalf("gv 3.1 silent ratios = %#v", gv31Silent)
	}

	c = taskContext(t, `{"model":"gv-3.1-lite","prompt":"video","resolution":"1080P","duration":8,"generate_audio":true}`)
	gvLiteAudio := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "gv-3.1-lite"})
	if gvLiteAudio["duration"] != 8 || !floatClose(gvLiteAudio["resolution"], 0.0800/0.0300) {
		t.Fatalf("gv lite top-level generate_audio should use audio ratios = %#v", gvLiteAudio)
	}

	c = taskContext(t, `{"model":"gv-3.1-lite","prompt":"video","resolution":"1080P","duration":8}`)
	gvLiteSilent := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "gv-3.1-lite"})
	if gvLiteSilent["duration"] != 8 || !floatClose(gvLiteSilent["resolution"], 0.0500/0.0300) {
		t.Fatalf("gv lite without audio should use silent ratios = %#v", gvLiteSilent)
	}

	c = taskContext(t, `{"model":"vidu-q2-image","prompt":"image","resolution":"4K","image":["https://example.com/a.png","https://example.com/b.png","https://example.com/c.png","https://example.com/d.png"]}`)
	viduImage := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "vidu-q2-image"})
	if !floatClose(viduImage["resolution"], 0.1442/0.0288) || viduImage["count"] != 1 {
		t.Fatalf("vidu image reference ratios = %#v", viduImage)
	}
}

func TestEstimateBillingUsesKling21ImageReferenceMatrix(t *testing.T) {
	useTencentVODPricingFixture(t)
	a := &TaskAdaptor{}

	c := taskContext(t, `{"model":"kling-2.1-image","prompt":"image","resolution":"4K"}`)
	textToImage := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-2.1-image"})
	if !floatClose(textToImage["resolution"], 0.037/0.014) || textToImage["count"] != 1 {
		t.Fatalf("kling 2.1 image no-reference ratios = %#v", textToImage)
	}

	c = taskContext(t, `{"model":"kling-2.1-image","prompt":"image","resolution":"4K","image":"https://example.com/ref.png"}`)
	singleReference := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-2.1-image"})
	if !floatClose(singleReference["resolution"], 0.056/0.014) || singleReference["count"] != 1 {
		t.Fatalf("kling 2.1 image single-reference ratios = %#v", singleReference)
	}

	c = taskContext(t, `{"model":"kling-2.1-image","prompt":"image","resolution":"4K","images":["https://example.com/a.png","https://example.com/b.png"]}`)
	multiReference := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-2.1-image"})
	if !floatClose(multiReference["resolution"], 0.079/0.014) || multiReference["count"] != 1 {
		t.Fatalf("kling 2.1 image multi-reference ratios = %#v", multiReference)
	}
}

func TestEstimateBillingAddsOGInputImageMixedCost(t *testing.T) {
	useTencentVODPricingFixture(t)
	a := &TaskAdaptor{}
	c := taskContext(t, `{"model":"og-image2-high","prompt":"edit","resolution":"4K","image":["https://example.com/input.png"]}`)
	ratios := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "og-image2-high"})

	if !floatClose(ratios["resolution"], 0.712/0.211) {
		t.Fatalf("resolution ratio = %#v", ratios)
	}
	if !floatClose(ratios["input_image"], 1+0.0154/0.712) {
		t.Fatalf("input image mixed ratio = %#v", ratios)
	}
}

func TestTencentVODMatrixPricingModesAreReachable(t *testing.T) {
	useTencentVODPricingFixture(t)
	cases := []struct {
		name            string
		model           string
		body            string
		expectedMode    string
		expectedRatio   float64
		expectedCount   float64
		expectedSeconds float64
	}{
		{
			name:          "vidu q2 image no reference",
			model:         "vidu-q2-image",
			body:          `{"model":"vidu-q2-image","prompt":"image","resolution":"4K"}`,
			expectedMode:  "text_to_image",
			expectedRatio: 0.0481 / 0.0288,
			expectedCount: 1,
		},
		{
			name:          "vidu q2 image one to three references",
			model:         "vidu-q2-image",
			body:          `{"model":"vidu-q2-image","prompt":"image","resolution":"4K","images":["https://example.com/a.png","https://example.com/b.png","https://example.com/c.png"]}`,
			expectedMode:  "reference_1_3",
			expectedRatio: 0.0769 / 0.0288,
			expectedCount: 1,
		},
		{
			name:          "vidu q2 image four to seven references",
			model:         "vidu-q2-image",
			body:          `{"model":"vidu-q2-image","prompt":"image","resolution":"4K","images":["https://example.com/a.png","https://example.com/b.png","https://example.com/c.png","https://example.com/d.png"]}`,
			expectedMode:  "reference_4_7",
			expectedRatio: 0.1442 / 0.0288,
			expectedCount: 1,
		},
		{
			name:          "kling 2.1 image no reference",
			model:         "kling-2.1-image",
			body:          `{"model":"kling-2.1-image","prompt":"image","resolution":"4K"}`,
			expectedMode:  "text_to_image",
			expectedRatio: 0.037 / 0.014,
			expectedCount: 1,
		},
		{
			name:          "kling 2.1 image single reference",
			model:         "kling-2.1-image",
			body:          `{"model":"kling-2.1-image","prompt":"image","resolution":"4K","image":"https://example.com/ref.png"}`,
			expectedMode:  "single_reference",
			expectedRatio: 0.056 / 0.014,
			expectedCount: 1,
		},
		{
			name:          "kling 2.1 image multi reference",
			model:         "kling-2.1-image",
			body:          `{"model":"kling-2.1-image","prompt":"image","resolution":"4K","images":["https://example.com/a.png","https://example.com/b.png"]}`,
			expectedMode:  "multi_reference",
			expectedRatio: 0.079 / 0.014,
			expectedCount: 1,
		},
		{
			name:            "vidu q2 text normal",
			model:           "vidu-q2",
			body:            `{"model":"vidu-q2","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "text",
			expectedRatio:   0.1615 / 0.0492,
			expectedSeconds: 5,
		},
		{
			name:            "vidu q2 text off peak",
			model:           "vidu-q2",
			body:            `{"model":"vidu-q2","prompt":"video","resolution":"4K","duration":5,"metadata":{"off_peak":true}}`,
			expectedMode:    "text_off_peak",
			expectedRatio:   0.0808 / 0.0492,
			expectedSeconds: 5,
		},
		{
			name:            "vidu q2 reference off peak",
			model:           "vidu-q2",
			body:            `{"model":"vidu-q2","prompt":"video","resolution":"4K","duration":5,"image":"https://example.com/ref.png","metadata":{"offpeak":true}}`,
			expectedMode:    "reference_off_peak",
			expectedRatio:   0.1419 / 0.0492,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 omni no reference no audio",
			model:           "kling-3.0-omni",
			body:            `{"model":"kling-3.0-omni","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "no_reference_no_audio",
			expectedRatio:   0.420 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 omni no reference audio",
			model:           "kling-3.0-omni",
			body:            `{"model":"kling-3.0-omni","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true}}`,
			expectedMode:    "no_reference_audio",
			expectedRatio:   0.420 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 omni reference no audio",
			model:           "kling-3.0-omni",
			body:            `{"model":"kling-3.0-omni","prompt":"video","resolution":"4K","duration":5,"image":"https://example.com/ref.png"}`,
			expectedMode:    "reference_no_audio",
			expectedRatio:   0.280 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 omni reference audio",
			model:           "kling-3.0-omni",
			body:            `{"model":"kling-3.0-omni","prompt":"video","resolution":"4K","duration":5,"image":"https://example.com/ref.png","metadata":{"audio":true}}`,
			expectedMode:    "reference_audio",
			expectedRatio:   0.336 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 silent",
			model:           "kling-3.0",
			body:            `{"model":"kling-3.0","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "silent",
			expectedRatio:   0.420 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 audio no voice",
			model:           "kling-3.0",
			body:            `{"model":"kling-3.0","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true}}`,
			expectedMode:    "audio_no_voice",
			expectedRatio:   0.420 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 audio voice",
			model:           "kling-3.0",
			body:            `{"model":"kling-3.0","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true,"voice_id":"voice-1"}}`,
			expectedMode:    "audio_voice",
			expectedRatio:   0.336 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 3 motion control",
			model:           "kling-3.0",
			body:            `{"model":"kling-3.0","prompt":"video","resolution":"4K","duration":5,"metadata":{"motion_control":true}}`,
			expectedMode:    "motion_control",
			expectedRatio:   0.378 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling o1 no reference",
			model:           "kling-o1",
			body:            `{"model":"kling-o1","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "no_reference",
			expectedRatio:   0.252 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling o1 reference",
			model:           "kling-o1",
			body:            `{"model":"kling-o1","prompt":"video","resolution":"4K","duration":5,"image":"https://example.com/ref.png"}`,
			expectedMode:    "reference",
			expectedRatio:   0.378 / 0.084,
			expectedSeconds: 5,
		},
		{
			name:            "kling 2.6 silent",
			model:           "kling-2.6",
			body:            `{"model":"kling-2.6","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "silent",
			expectedRatio:   0.1568 / 0.042,
			expectedSeconds: 5,
		},
		{
			name:            "kling 2.6 audio",
			model:           "kling-2.6",
			body:            `{"model":"kling-2.6","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true}}`,
			expectedMode:    "audio",
			expectedRatio:   0.315 / 0.042,
			expectedSeconds: 5,
		},
		{
			name:            "gv 3.1 silent",
			model:           "gv-3.1",
			body:            `{"model":"gv-3.1","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "silent",
			expectedRatio:   0.400 / 0.200,
			expectedSeconds: 5,
		},
		{
			name:            "gv 3.1 audio",
			model:           "gv-3.1",
			body:            `{"model":"gv-3.1","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true}}`,
			expectedMode:    "audio",
			expectedRatio:   0.600 / 0.200,
			expectedSeconds: 5,
		},
		{
			name:            "pixverse v6 silent",
			model:           "pixverse-v6",
			body:            `{"model":"pixverse-v6","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "silent",
			expectedRatio:   0.1152 / 0.0311,
			expectedSeconds: 5,
		},
		{
			name:            "pixverse v6 audio",
			model:           "pixverse-v6",
			body:            `{"model":"pixverse-v6","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true}}`,
			expectedMode:    "audio",
			expectedRatio:   0.1472 / 0.0311,
			expectedSeconds: 5,
		},
		{
			name:            "sv 1.5 pro audio",
			model:           "sv-1.5-pro",
			body:            `{"model":"sv-1.5-pro","prompt":"video","resolution":"4K","duration":5,"metadata":{"audio":true}}`,
			expectedMode:    "audio",
			expectedRatio:   0.4785 / 0.0123,
			expectedSeconds: 5,
		},
		{
			name:            "jv 3 pro",
			model:           "jv-3.0-pro",
			body:            `{"model":"jv-3.0-pro","prompt":"video","resolution":"4K","duration":5}`,
			expectedMode:    "default",
			expectedRatio:   0.3462 / 0.1538,
			expectedSeconds: 5,
		},
	}

	a := &TaskAdaptor{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := taskContext(t, tc.body)
			req, err := relaycommon.GetTaskRequest(c)
			if err != nil {
				t.Fatal(err)
			}
			spec, ok := lookupModelSpec(tc.model)
			if !ok {
				t.Fatalf("missing model spec for %s", tc.model)
			}
			modelRows, ok := priceRowsForTest(t)[spec.PublicModel]
			if !ok {
				t.Fatalf("missing pricing rows for %s", spec.PublicModel)
			}
			if mode := resolvePricingMode(&req, spec, modelRows); mode != tc.expectedMode {
				t.Fatalf("mode = %q, want %q", mode, tc.expectedMode)
			}
			ratios := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: tc.model})
			if !floatClose(ratios["resolution"], tc.expectedRatio) {
				t.Fatalf("resolution ratio = %v, want %v; ratios = %#v", ratios["resolution"], tc.expectedRatio, ratios)
			}
			if tc.expectedCount > 0 && !floatClose(ratios["count"], tc.expectedCount) {
				t.Fatalf("count = %v, want %v; ratios = %#v", ratios["count"], tc.expectedCount, ratios)
			}
			if tc.expectedSeconds > 0 && !floatClose(ratios["duration"], tc.expectedSeconds) {
				t.Fatalf("duration = %v, want %v; ratios = %#v", ratios["duration"], tc.expectedSeconds, ratios)
			}
		})
	}
}

func TestEstimateBillingRoundsTencentVODDurationRules(t *testing.T) {
	useTencentVODPricingFixture(t)
	a := &TaskAdaptor{}
	c := taskContext(t, `{"model":"kling-identifyface","prompt":"video","resolution":"1080P","duration":6}`)
	ratios := a.EstimateBilling(c, &relaycommon.RelayInfo{OriginModelName: "kling-identifyface"})
	if ratios["duration"] != 10 || ratios["resolution"] != 1 {
		t.Fatalf("identifyface ratios = %#v", ratios)
	}
}

func floatClose(a, b float64) bool {
	if a > b {
		return a-b < 0.000001
	}
	return b-a < 0.000001
}

func containsModel(models []string, target string) bool {
	for _, model := range models {
		if model == target {
			return true
		}
	}
	return false
}

func TestBuildRequestHeaderSignsTencentVODRequest(t *testing.T) {
	a := &TaskAdaptor{pendingAction: actionCreateVideoTask}
	body := []byte(`{"SubAppId":1500044236}`)
	req := httptest.NewRequest(http.MethodPost, "https://vod.tencentcloudapi.com/", bytes.NewReader(body))

	err := a.BuildRequestHeader(nil, req, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ApiKey:     "sid|skey|1500044236",
		ApiVersion: "ap-guangzhou",
	}})
	if err != nil {
		t.Fatal(err)
	}

	if req.Header.Get("X-TC-Action") != actionCreateVideoTask || req.Header.Get("X-TC-Region") != "ap-guangzhou" {
		t.Fatalf("headers = %#v", req.Header)
	}
	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "TC3-HMAC-SHA256 Credential=sid/") || strings.Contains(auth, "skey") {
		t.Fatalf("authorization header leaks secret or has wrong format: %q", auth)
	}
}

func TestDoResponseReturnsPublicTaskAndStoresUpstreamID(t *testing.T) {
	a := &TaskAdaptor{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"Response":{"TaskId":"vod-task-123","RequestId":"req-1"}}`)),
	}
	taskID, rawBody, taskErr := a.DoResponse(c, resp, &relaycommon.RelayInfo{
		OriginModelName: "gv-3.1",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	})
	if taskErr != nil {
		t.Fatalf("DoResponse error = %+v", taskErr)
	}
	if taskID != "vod-task-123" {
		t.Fatalf("taskID = %q", taskID)
	}
	if !strings.Contains(string(rawBody), "vod-task-123") || w.Code != http.StatusOK {
		t.Fatalf("response code=%d raw=%s", w.Code, rawBody)
	}
	var response map[string]any
	if err := common.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["object"]; ok {
		t.Fatalf("Tencent VOD submit response should not include object: %#v", response)
	}
	if response["id"] != "task_public" || response["task_id"] != "task_public" || response["status"] != dto.VideoStatusQueued {
		t.Fatalf("response = %#v", response)
	}
}

func TestConvertToOpenAIVideoReturnsNeutralTaskStatus(t *testing.T) {
	a := &TaskAdaptor{}
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 1781495472,
		UpdatedAt: 1781495572,
		Properties: model.Properties{
			OriginModelName: "og-image2-high",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/out.png",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := common.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["object"]; ok {
		t.Fatalf("Tencent VOD poll response should not include video object: %#v", response)
	}
	if response["id"] != "task_public" || response["status"] != dto.VideoStatusCompleted || response["url"] != "https://example.com/out.png" {
		t.Fatalf("response = %#v", response)
	}
}

func TestConvertToOpenAIVideoSuppressesGatewayProxyURL(t *testing.T) {
	a := &TaskAdaptor{}
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:   "task_public",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Properties: model.Properties{
			OriginModelName: "hailuo-2.3-fast",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://lizh.ai/v1/videos/task_public/content",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := common.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response["url"]; ok {
		t.Fatalf("Tencent VOD response must not expose gateway proxy URL: %#v", response)
	}
}

func TestConvertToOpenAIVideoRecoversDirectURLFromStoredTencentData(t *testing.T) {
	a := &TaskAdaptor{}
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:   "task_public",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Properties: model.Properties{
			OriginModelName: "hailuo-2.3-fast",
		},
		PrivateData: model.TaskPrivateData{
			UpstreamKind: "video",
			ResultURL:    "https://lizh.ai/v1/videos/task_public/content",
		},
		Data: []byte(`{"Response":{"Status":"FINISH","Output":{"FileInfos":[{"MediaBasicInfo":{"MediaUrl":"https://example.com/out.mp4"}}]}}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := common.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response["url"] != "https://example.com/out.mp4" {
		t.Fatalf("response = %#v", response)
	}
}

func TestParseTaskResultMapsTencentStatesAndURLs(t *testing.T) {
	a := &TaskAdaptor{}
	success, err := a.ParseTaskResult([]byte(`{"Response":{"Status":"FINISH","Output":{"VideoUrl":"https://example.com/out.mp4"},"RequestId":"req-1"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if success.Status != model.TaskStatusSuccess || success.Url != "https://example.com/out.mp4" {
		t.Fatalf("success = %+v", success)
	}

	failed, err := a.ParseTaskResult([]byte(`{"Response":{"Status":"FAIL","ErrMsg":"bad prompt"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != model.TaskStatusFailure || failed.Reason != "bad prompt" {
		t.Fatalf("failed = %+v", failed)
	}

	running, err := a.ParseTaskResult([]byte(`{"Response":{"TaskStatus":"PROCESSING","Progress":42}}`))
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != model.TaskStatusInProgress || running.Progress != "42%" {
		t.Fatalf("running = %+v", running)
	}
}

func TestParseTaskResultFindsNestedTencentMediaURL(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{
		"Response": {
			"Status": "FINISH",
			"Output": {
				"FileInfos": [
					{"Name": "preview"},
					{"MediaBasicInfo": {"MediaUrl": "https://example.com/direct-video.mp4"}}
				]
			},
			"RequestId": "req-1"
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != model.TaskStatusSuccess || info.Url != "https://example.com/direct-video.mp4" {
		t.Fatalf("info = %+v", info)
	}
}

func TestParseTaskResultTreatsNestedAIGCErrCodeAsFailure(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{
		"Response": {
			"TaskType": "AigcVideoTask",
			"Status": "FINISH",
			"AigcVideoTask": {
				"TaskId": "1442393297-AigcVideoTask-failed",
				"Status": "FINISH",
				"ErrCode": 70000,
				"ErrCodeExt": "InternalError",
				"Message": "task failed with status: FAIL, message: invalid params",
				"Progress": 100,
				"Output": {"FileInfos": []}
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != model.TaskStatusFailure || !strings.Contains(info.Reason, "invalid params") {
		t.Fatalf("info = %+v", info)
	}
}

func TestParseTaskResultPrefersNestedAIGCTaskStatus(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{
		"Response": {
			"Status": "FINISH",
			"AigcVideoTask": {
				"TaskStatus": "PROCESSING",
				"Progress": 35
			},
			"RequestId": "req-1"
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != model.TaskStatusInProgress || info.Progress != "35%" || info.Url != "" {
		t.Fatalf("info = %+v", info)
	}
}

func TestParseTaskResultKeepsCompletedWithoutMediaURLInProgress(t *testing.T) {
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult([]byte(`{
		"Response": {
			"Status": "FINISH",
			"AigcVideoTask": {
				"TaskStatus": "SUCCESS",
				"Progress": 100
			},
			"RequestId": "req-1"
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != model.TaskStatusInProgress || info.Progress != "95%" || info.Url != "" {
		t.Fatalf("info = %+v", info)
	}
}

func taskContext(t *testing.T, raw string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	var req relaycommon.TaskSubmitReq
	if err := common.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	c.Set("task_request", req)
	return c
}
