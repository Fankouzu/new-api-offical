package tencentvod

import (
	"math"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const (
	unitImage  = "image"
	unitSecond = "second"
	unitTask   = "task"
)

type vodPriceRow struct {
	Unit             string
	Prices           map[string]float64
	TaskPrice        float64
	DurationStep     int
	DurationMin      int
	BaseModeOverride string
	BaseTierOverride string
}

var tencentVODPriceRows = map[string]map[string]vodPriceRow{
	ModelHunyuan30Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.200, "2K": 0.280, "4K": 0.360}},
	},
	ModelQwen0925Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.300, "2K": 0.380, "4K": 0.460}},
	},
	ModelOGImage2Low: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.300, "2K": 0.338, "4K": 0.398}},
	},
	ModelOGImage2Medium: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.638, "2K": 1.050, "4K": 1.583}},
	},
	ModelOGImage2High: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 1.838, "2K": 3.450, "4K": 5.588}},
	},
	ModelGG31Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"512P": 0.333, "1K": 0.500, "2K": 0.750, "4K": 1.120}},
	},
	ModelGG30Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 1.000, "2K": 1.000, "4K": 1.800}},
	},
	ModelGG25Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.300, "2K": 0.380, "4K": 0.460}},
	},
	ModelViduQ2Image: {
		"text_to_image": {Unit: unitImage, Prices: map[string]float64{"1K": 0.188, "2K": 0.250, "4K": 0.313}},
		"reference_1_3": {Unit: unitImage, Prices: map[string]float64{"1K": 0.250, "2K": 0.375, "4K": 0.500}, BaseModeOverride: "text_to_image"},
		"reference_4_7": {Unit: unitImage, Prices: map[string]float64{"1K": 0.313, "2K": 0.625, "4K": 0.938}, BaseModeOverride: "text_to_image"},
	},
	ModelMJv7Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.300, "2K": 0.380, "4K": 0.460}},
	},
	ModelKlingImage30: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.200, "2K": 0.200, "4K": 0.400}},
	},
	ModelKlingImage30Omni: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.200, "2K": 0.200, "4K": 0.400}},
	},
	ModelKlingO1Image: {
		"default": {Unit: unitImage, Prices: map[string]float64{"1K": 0.200, "2K": 0.200, "4K": 0.400}},
	},
	ModelKling21Image: {
		"text_to_image":   {Unit: unitImage, Prices: map[string]float64{"1K": 0.100, "2K": 0.100, "4K": 0.260}},
		"multi_reference": {Unit: unitImage, Prices: map[string]float64{"1K": 0.400, "2K": 0.480, "4K": 0.560}, BaseModeOverride: "text_to_image"},
	},
	ModelHunyuan3DPanorama: {
		"task": {Unit: unitTask, TaskPrice: 5.000},
	},

	ModelHunyuan15: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.300, "1080P": 0.500, "2K": 0.750, "4K": 1.120}},
	},
	ModelViduQ3Mix: {
		"reference": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.782, "1080P": 0.938, "2K": 1.126, "4K": 1.352}},
	},
	ModelViduQ3: {
		"reference":          {Unit: unitSecond, Prices: map[string]float64{"540P": 0.313, "720P": 0.625, "1080P": 0.782, "2K": 0.939, "4K": 1.127}},
		"reference_off_peak": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.157, "720P": 0.313, "1080P": 0.391, "2K": 0.470, "4K": 0.564}, BaseModeOverride: "reference"},
	},
	ModelViduQ3Pro: {
		"default":          {Unit: unitSecond, Prices: map[string]float64{"540P": 0.313, "720P": 0.782, "1080P": 0.938, "2K": 1.126, "4K": 1.351}},
		"default_off_peak": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.157, "720P": 0.391, "1080P": 0.469, "2K": 0.563, "4K": 0.676}, BaseModeOverride: "default"},
	},
	ModelViduQ3Turbo: {
		"default":          {Unit: unitSecond, Prices: map[string]float64{"540P": 0.250, "720P": 0.375, "1080P": 0.438, "2K": 0.526, "4K": 0.631}},
		"default_off_peak": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.125, "720P": 0.188, "1080P": 0.219, "2K": 0.263, "4K": 0.316}, BaseModeOverride: "default"},
	},
	ModelViduQ2: {
		"text":               {Unit: unitSecond, Prices: map[string]float64{"720P": 0.320, "1080P": 0.470, "2K": 0.700, "4K": 1.050}},
		"text_off_peak":      {Unit: unitSecond, Prices: map[string]float64{"720P": 0.160, "1080P": 0.235, "2K": 0.350, "4K": 0.525}, BaseModeOverride: "text"},
		"reference":          {Unit: unitSecond, Prices: map[string]float64{"540P": 0.240, "720P": 0.320, "1080P": 0.820, "2K": 1.230, "4K": 1.845}, BaseModeOverride: "text"},
		"reference_off_peak": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.120, "720P": 0.160, "1080P": 0.410, "2K": 0.615, "4K": 0.923}, BaseModeOverride: "text"},
	},
	ModelViduQ2Pro: {
		"i2v_first_last":          {Unit: unitSecond, Prices: map[string]float64{"720P": 0.350, "1080P": 0.700, "2K": 1.000, "4K": 1.500}},
		"i2v_first_last_off_peak": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.175, "1080P": 0.350, "2K": 0.500, "4K": 0.750}, BaseModeOverride: "i2v_first_last"},
		"reference":               {Unit: unitSecond, Prices: map[string]float64{"540P": 0.270, "720P": 0.350, "1080P": 0.900, "2K": 1.350, "4K": 2.025}, BaseModeOverride: "i2v_first_last"},
		"reference_off_peak":      {Unit: unitSecond, Prices: map[string]float64{"540P": 0.135, "720P": 0.175, "1080P": 0.450, "2K": 0.675, "4K": 1.013}, BaseModeOverride: "i2v_first_last"},
	},
	ModelViduQ2Turbo: {
		"i2v_first_last":          {Unit: unitSecond, Prices: map[string]float64{"720P": 0.250, "1080P": 0.470, "2K": 0.700, "4K": 1.050}},
		"i2v_first_last_off_peak": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.125, "1080P": 0.235, "2K": 0.350, "4K": 0.525}, BaseModeOverride: "i2v_first_last"},
	},
	ModelKling30Omni: {
		"no_reference_no_audio": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.600, "1080P": 0.800, "2K": 1.000, "4K": 3.000}},
		"no_reference_audio":    {Unit: unitSecond, Prices: map[string]float64{"720P": 0.800, "1080P": 1.000, "2K": 1.200, "4K": 3.000}, BaseModeOverride: "no_reference_no_audio"},
		"reference_no_audio":    {Unit: unitSecond, Prices: map[string]float64{"720P": 0.900, "1080P": 1.200, "2K": 1.500, "4K": 2.000}, BaseModeOverride: "no_reference_no_audio"},
		"reference_audio":       {Unit: unitSecond, Prices: map[string]float64{"720P": 1.100, "1080P": 1.400, "2K": 1.800, "4K": 2.400}, BaseModeOverride: "no_reference_no_audio"},
	},
	ModelKling30: {
		"silent":         {Unit: unitSecond, Prices: map[string]float64{"720P": 0.600, "1080P": 0.800, "2K": 1.000, "4K": 3.000}},
		"audio_no_voice": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.900, "1080P": 1.200, "2K": 1.500, "4K": 3.000}, BaseModeOverride: "silent"},
		"audio_voice":    {Unit: unitSecond, Prices: map[string]float64{"720P": 1.100, "1080P": 1.400, "2K": 1.800, "4K": 2.400}, BaseModeOverride: "silent"},
	},
	ModelKling26: {
		"silent": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.300, "1080P": 0.500, "2K": 0.750, "4K": 1.120}},
		"audio":  {Unit: unitSecond, Prices: map[string]float64{"1080P": 1.000, "2K": 1.500, "4K": 2.250}, BaseModeOverride: "silent"},
	},
	ModelKling26MotionControl: {
		"motion_control": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.500, "1080P": 0.800, "2K": 1.200, "4K": 1.800}},
	},
	ModelKling25Pro: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.300, "1080P": 0.500, "2K": 0.750, "4K": 1.120}},
	},
	ModelKling16: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.400, "1080P": 0.700, "2K": 1.000, "4K": 1.500}},
	},
	ModelKling20: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.400, "1080P": 0.700, "2K": 1.000, "4K": 1.500}},
	},
	ModelKling21: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.400, "1080P": 0.700, "2K": 1.000, "4K": 1.500}},
	},
	ModelKlingAvatar: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.400, "1080P": 0.800, "2K": 1.200, "4K": 1.800}},
	},
	ModelKlingIdentifyFace: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.100}, DurationStep: 5, DurationMin: 5},
	},
	ModelH210: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.900, "1080P": 1.600, "2K": 1.920, "4K": 2.304}},
	},
	ModelHailuo02: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"768P": 0.330, "1080P": 0.580, "2K": 0.930, "4K": 1.490}},
	},
	ModelHailuo23: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"768P": 0.330, "1080P": 0.580, "2K": 0.930, "4K": 1.490}},
	},
	ModelHailuo23Fast: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"768P": 0.225, "1080P": 0.385, "2K": 0.580, "4K": 0.870}},
	},
	ModelGV31: {
		"audio":  {Unit: unitSecond, Prices: map[string]float64{"720P": 3.000, "1080P": 3.000, "2K": 3.750, "4K": 4.500}, BaseModeOverride: "silent"},
		"silent": {Unit: unitSecond, Prices: map[string]float64{"720P": 1.500, "1080P": 1.500, "2K": 2.250, "4K": 3.000}},
	},
	ModelGV31Fast: {
		"audio":  {Unit: unitSecond, Prices: map[string]float64{"720P": 1.125, "1080P": 1.125, "2K": 1.875, "4K": 2.625}, BaseModeOverride: "silent"},
		"silent": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.750, "1080P": 0.750, "2K": 1.500, "4K": 2.250}},
	},
	ModelGV31Lite: {
		"audio":  {Unit: unitSecond, Prices: map[string]float64{"720P": 0.375, "1080P": 0.600, "2K": 0.900, "4K": 1.125}, BaseModeOverride: "silent"},
		"silent": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.225, "1080P": 0.375, "2K": 0.600, "4K": 0.750}},
	},
	ModelOS20: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.750, "1080P": 1.125, "2K": 1.688, "4K": 2.531}},
	},
	ModelPixVerseV56: {
		"silent": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.245, "720P": 0.315, "1080P": 0.525, "2K": 0.735, "4K": 1.029}},
	},
	ModelPixVerseV6: {
		"silent": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.205, "720P": 0.264, "1080P": 0.528, "2K": 0.634, "4K": 0.760}},
		"audio":  {Unit: unitSecond, Prices: map[string]float64{"540P": 0.264, "720P": 0.352, "1080P": 0.675, "2K": 0.810, "4K": 0.971}, BaseModeOverride: "silent"},
	},
	ModelPixVerseC1: {
		"silent": {Unit: unitSecond, Prices: map[string]float64{"540P": 0.235, "720P": 0.293, "1080P": 0.557, "2K": 0.669, "4K": 0.803}},
		"audio":  {Unit: unitSecond, Prices: map[string]float64{"540P": 0.293, "720P": 0.381, "1080P": 0.704, "2K": 0.845, "4K": 1.014}, BaseModeOverride: "silent"},
	},
	ModelMingmou10: {
		"default": {Unit: unitSecond, Prices: map[string]float64{"720P": 0.300, "1080P": 0.500, "2K": 0.750, "4K": 1.120}},
	},
	ModelHunyuan3DScene: {
		"task": {Unit: unitTask, TaskPrice: 200.000},
	},
}

func estimatePreciseBillingRatios(req *relaycommon.TaskSubmitReq, spec modelSpec) (map[string]float64, bool) {
	modelRows, ok := tencentVODPriceRows[spec.PublicModel]
	if !ok {
		return nil, false
	}
	mode := resolvePricingMode(req, spec)
	row, ok := modelRows[mode]
	if !ok {
		row, ok = modelRows["default"]
	}
	if !ok {
		row, ok = modelRows["task"]
	}
	if !ok {
		return nil, false
	}

	ratios := map[string]float64{}
	switch row.Unit {
	case unitImage:
		resolution := normalizeResolution(firstString(req.Resolution, req.Size, metadataString(req.Metadata, "resolution"), metadataString(req.Metadata, "size"), spec.DefaultResolution))
		ratios["resolution"] = preciseResolutionRatio(modelRows, row, mode, resolution, spec.DefaultResolution)
		ratios["count"] = imageOutputCount(req)
	case unitSecond:
		duration := resolveDuration(req, spec)
		ratios["duration"] = billableDuration(duration, row)
		resolution := normalizeResolution(firstString(req.Resolution, req.Size, metadataString(req.Metadata, "resolution"), metadataString(req.Metadata, "size"), spec.DefaultResolution))
		ratios["resolution"] = preciseResolutionRatio(modelRows, row, mode, resolution, spec.DefaultResolution)
	case unitTask:
		ratios["task"] = 1
	default:
		return nil, false
	}
	return ratios, true
}

func preciseResolutionRatio(modelRows map[string]vodPriceRow, row vodPriceRow, mode string, resolution string, defaultResolution string) float64 {
	currentPrice, ok := priceForResolution(row.Prices, resolution)
	if !ok {
		return 1
	}
	baseMode := mode
	if row.BaseModeOverride != "" {
		baseMode = row.BaseModeOverride
	}
	baseRow := row
	if candidate, ok := modelRows[baseMode]; ok {
		baseRow = candidate
	}
	baseTier := normalizeResolution(firstString(row.BaseTierOverride, defaultResolution, firstPriceTier(baseRow.Prices)))
	basePrice, ok := priceForResolution(baseRow.Prices, baseTier)
	if !ok || basePrice <= 0 {
		basePrice, ok = lowestPrice(baseRow.Prices)
	}
	if !ok || basePrice <= 0 {
		return 1
	}
	return currentPrice / basePrice
}

func priceForResolution(prices map[string]float64, resolution string) (float64, bool) {
	if len(prices) == 0 {
		return 0, false
	}
	resolution = normalizeResolution(resolution)
	if price, ok := prices[resolution]; ok {
		return price, true
	}
	return 0, false
}

func firstPriceTier(prices map[string]float64) string {
	for _, tier := range []string{"360P", "480P", "540P", "720P", "768P", "512P", "1K", "1080P", "2K", "4K"} {
		if _, ok := prices[tier]; ok {
			return tier
		}
	}
	return ""
}

func lowestPrice(prices map[string]float64) (float64, bool) {
	var value float64
	ok := false
	for _, price := range prices {
		if price <= 0 {
			continue
		}
		if !ok || price < value {
			value = price
			ok = true
		}
	}
	return value, ok
}

func billableDuration(duration int, row vodPriceRow) float64 {
	if duration <= 0 {
		duration = row.DurationMin
	}
	if row.DurationMin > 0 && duration < row.DurationMin {
		duration = row.DurationMin
	}
	if row.DurationStep > 0 {
		return math.Ceil(float64(duration)/float64(row.DurationStep)) * float64(row.DurationStep)
	}
	return float64(duration)
}

func imageOutputCount(req *relaycommon.TaskSubmitReq) float64 {
	return math.Max(1, float64(metadataInt(req.Metadata, "n", metadataInt(req.Metadata, "count", metadataInt(req.Metadata, "output_image_count", 1)))))
}

func resolvePricingMode(req *relaycommon.TaskSubmitReq, spec modelSpec) string {
	if mode := strings.TrimSpace(req.Mode); mode != "" {
		return normalizePricingMode(mode)
	}
	if mode := metadataString(req.Metadata, "pricing_mode"); mode != "" {
		return normalizePricingMode(mode)
	}
	if mode := metadataString(req.Metadata, "mode"); mode != "" {
		return normalizePricingMode(mode)
	}

	hasReference := requestHasReference(req)
	offPeak := metadataBool(req.Metadata, "off_peak") || metadataBool(req.Metadata, "offpeak")
	audio := requestHasAudio(req)

	switch spec.PublicModel {
	case ModelViduQ2Image:
		refCount := requestReferenceCount(req)
		if refCount == 0 {
			return "text_to_image"
		}
		if refCount <= 3 {
			return "reference_1_3"
		}
		return "reference_4_7"
	case ModelKling21Image:
		if hasReference {
			return "multi_reference"
		}
		return "text_to_image"
	case ModelViduQ2:
		if hasReference {
			return withOffPeak("reference", offPeak)
		}
		return withOffPeak("text", offPeak)
	case ModelViduQ2Pro:
		if requestHasExtraReference(req) {
			return withOffPeak("reference", offPeak)
		}
		return withOffPeak("i2v_first_last", offPeak)
	case ModelViduQ2Turbo:
		return withOffPeak("i2v_first_last", offPeak)
	case ModelViduQ3:
		return withOffPeak("reference", offPeak)
	case ModelViduQ3Mix:
		return "reference"
	case ModelViduQ3Pro, ModelViduQ3Turbo:
		return withOffPeak("default", offPeak)
	case ModelKling30Omni:
		if hasReference && audio {
			return "reference_audio"
		}
		if hasReference {
			return "reference_no_audio"
		}
		if audio {
			return "no_reference_audio"
		}
		return "no_reference_no_audio"
	case ModelKling30:
		if audio {
			if metadataString(req.Metadata, "voice_id") != "" || metadataBool(req.Metadata, "custom_voice") {
				return "audio_voice"
			}
			return "audio_no_voice"
		}
		return "silent"
	case ModelKling26:
		if audio {
			return "audio"
		}
		return "silent"
	case ModelKling26MotionControl:
		return "motion_control"
	case ModelGV31, ModelGV31Fast, ModelGV31Lite, ModelPixVerseV6, ModelPixVerseC1:
		if audio {
			return "audio"
		}
		return "silent"
	case ModelPixVerseV56:
		return "silent"
	case ModelHunyuan3DPanorama, ModelHunyuan3DScene:
		return "task"
	default:
		return "default"
	}
}

func normalizePricingMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	mode = strings.ReplaceAll(mode, "-", "_")
	mode = strings.ReplaceAll(mode, " ", "_")
	return mode
}

func withOffPeak(mode string, offPeak bool) string {
	if offPeak {
		return mode + "_off_peak"
	}
	return mode
}

func requestReferenceCount(req *relaycommon.TaskSubmitReq) int {
	count := 0
	if strings.TrimSpace(req.Image) != "" {
		count++
	}
	count += len(req.Images)
	if strings.TrimSpace(req.InputReference) != "" {
		count++
	}
	return count
}

func requestHasReference(req *relaycommon.TaskSubmitReq) bool {
	return requestReferenceCount(req) > 0
}

func requestHasExtraReference(req *relaycommon.TaskSubmitReq) bool {
	return len(req.Images) > 2 || strings.TrimSpace(req.InputReference) != ""
}

func requestHasAudio(req *relaycommon.TaskSubmitReq) bool {
	if metadataBool(req.Metadata, "audio") || metadataBool(req.Metadata, "has_audio") {
		return true
	}
	audioGeneration := strings.ToLower(metadataString(req.Metadata, "audio_generation"))
	return audioGeneration == "enabled" || audioGeneration == "true" || audioGeneration == "1"
}

func metadataBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, ok := m[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "enabled", "on":
			return true
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	}
	return false
}
