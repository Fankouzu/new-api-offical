package tencentvod

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
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

const tencentVODPricingMatrixOptionKey = "TencentVODPricingMatrix"

type vodPricingSnapshot struct {
	Image []vodPricingSnapshotRow `json:"image"`
	Video []vodPricingSnapshotRow `json:"video"`
	Rows  []vodPricingSnapshotRow `json:"rows"`
}

type vodPricingSnapshotRow struct {
	Model            string             `json:"model"`
	Mode             string             `json:"mode"`
	Unit             string             `json:"unit"`
	Prices           map[string]float64 `json:"prices"`
	TaskPrice        float64            `json:"task_price"`
	DurationRounding struct {
		Step    int `json:"step"`
		Minimum int `json:"minimum"`
	} `json:"duration_rounding"`
	Metadata map[string]any `json:"metadata"`
}

func loadTencentVODPriceRows() (map[string]map[string]vodPriceRow, error) {
	common.OptionMapRWMutex.RLock()
	raw := strings.TrimSpace(common.OptionMap[tencentVODPricingMatrixOptionKey])
	common.OptionMapRWMutex.RUnlock()
	return parseTencentVODPriceRows(raw)
}

func parseTencentVODPriceRows(raw string) (map[string]map[string]vodPriceRow, error) {
	if raw == "" {
		return nil, fmt.Errorf("%s is not configured", tencentVODPricingMatrixOptionKey)
	}

	var snapshot vodPricingSnapshot
	if err := common.UnmarshalJsonStr(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("parse %s failed: %w", tencentVODPricingMatrixOptionKey, err)
	}

	rows := make(map[string]map[string]vodPriceRow)
	appendRow := func(item vodPricingSnapshotRow) error {
		model := strings.TrimSpace(item.Model)
		if model == "" {
			return fmt.Errorf("Tencent VOD pricing row has empty model")
		}
		mode := normalizePricingMode(firstString(item.Mode, "default"))
		if mode == "" {
			mode = "default"
		}
		unit := strings.TrimSpace(item.Unit)
		if unit == "" {
			return fmt.Errorf("Tencent VOD pricing row %s/%s has empty unit", model, mode)
		}
		row := vodPriceRow{
			Unit:         unit,
			Prices:       item.Prices,
			TaskPrice:    item.TaskPrice,
			DurationStep: item.DurationRounding.Step,
			DurationMin:  item.DurationRounding.Minimum,
		}
		if row.TaskPrice <= 0 {
			row.TaskPrice = metadataFloat(item.Metadata, "task_price")
		}
		row.BaseModeOverride = metadataString(item.Metadata, "base_mode_override")
		row.BaseTierOverride = metadataString(item.Metadata, "base_tier_override")
		if rows[model] == nil {
			rows[model] = make(map[string]vodPriceRow)
		}
		rows[model][mode] = row
		return nil
	}

	for _, item := range snapshot.Image {
		if err := appendRow(item); err != nil {
			return nil, err
		}
	}
	for _, item := range snapshot.Video {
		if err := appendRow(item); err != nil {
			return nil, err
		}
	}
	for _, item := range snapshot.Rows {
		if err := appendRow(item); err != nil {
			return nil, err
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s contains no pricing rows", tencentVODPricingMatrixOptionKey)
	}
	return rows, nil
}

func ValidatePricingMatrix(raw string) error {
	_, err := parseTencentVODPriceRows(raw)
	return err
}

func estimatePreciseBillingRatios(req *relaycommon.TaskSubmitReq, spec modelSpec) (map[string]float64, error) {
	priceRows, err := loadTencentVODPriceRows()
	if err != nil {
		return nil, err
	}
	modelRows, ok := priceRows[spec.PublicModel]
	if !ok {
		return nil, fmt.Errorf("Tencent VOD pricing missing model %s", spec.PublicModel)
	}
	mode := resolvePricingMode(req, spec, modelRows)
	row, ok := modelRows[mode]
	if !ok {
		return nil, fmt.Errorf("Tencent VOD pricing missing mode %s for model %s", mode, spec.PublicModel)
	}

	ratios := map[string]float64{}
	switch row.Unit {
	case unitImage:
		resolution := normalizeResolution(firstString(req.Resolution, req.Size, metadataString(req.Metadata, "resolution"), metadataString(req.Metadata, "size"), spec.DefaultResolution))
		resolutionRatio, err := preciseResolutionRatio(modelRows, row, mode, resolution, spec.DefaultResolution)
		if err != nil {
			return nil, err
		}
		ratios["resolution"] = resolutionRatio
		ratios["count"] = imageOutputCount(req)
		inputRatio, err := mixedInputImageRatio(req, modelRows, row, mode, resolution, spec.DefaultResolution)
		if err != nil {
			return nil, err
		}
		if inputRatio > 0 {
			ratios["input_image"] = inputRatio
		}
	case unitSecond:
		duration := resolveDuration(req, spec)
		ratios["duration"] = billableDuration(duration, row)
		resolution := normalizeResolution(firstString(req.Resolution, req.Size, metadataString(req.Metadata, "resolution"), metadataString(req.Metadata, "size"), spec.DefaultResolution))
		resolutionRatio, err := preciseResolutionRatio(modelRows, row, mode, resolution, spec.DefaultResolution)
		if err != nil {
			return nil, err
		}
		ratios["resolution"] = resolutionRatio
	case unitTask:
		if row.TaskPrice <= 0 {
			return nil, fmt.Errorf("Tencent VOD pricing missing task price for model %s mode %s", spec.PublicModel, mode)
		}
		ratios["task"] = 1
	default:
		return nil, fmt.Errorf("Tencent VOD pricing unit %s is unsupported for model %s mode %s", row.Unit, spec.PublicModel, mode)
	}
	return ratios, nil
}

func preciseResolutionRatio(modelRows map[string]vodPriceRow, row vodPriceRow, mode string, resolution string, defaultResolution string) (float64, error) {
	currentPrice, ok := priceForResolution(row.Prices, resolution)
	if !ok {
		return 0, fmt.Errorf("Tencent VOD pricing missing resolution %s for mode %s", resolution, mode)
	}
	baseMode := mode
	if row.BaseModeOverride != "" {
		baseMode = row.BaseModeOverride
	} else if inferredBaseMode := inferBasePricingMode(modelRows, mode); inferredBaseMode != "" {
		baseMode = inferredBaseMode
	}
	baseRow := row
	if candidate, ok := modelRows[baseMode]; ok {
		baseRow = candidate
	}
	baseTier := normalizeResolution(firstString(row.BaseTierOverride, defaultResolution, firstPriceTier(baseRow.Prices)))
	basePrice, ok := priceForResolution(baseRow.Prices, baseTier)
	if !ok || basePrice <= 0 {
		return 0, fmt.Errorf("Tencent VOD pricing missing base resolution %s for mode %s", baseTier, baseMode)
	}
	return currentPrice / basePrice, nil
}

func mixedInputImageRatio(req *relaycommon.TaskSubmitReq, modelRows map[string]vodPriceRow, outputRow vodPriceRow, outputMode string, resolution string, defaultResolution string) (float64, error) {
	inputRow, ok := modelRows["input_image"]
	if !ok || requestReferenceCount(req) == 0 {
		return 0, nil
	}
	inputPrice, ok := priceForResolution(inputRow.Prices, "input")
	if !ok || inputPrice <= 0 {
		return 0, fmt.Errorf("Tencent VOD pricing missing input image price for mode input_image")
	}
	outputPrice, ok := priceForResolution(outputRow.Prices, resolution)
	if !ok || outputPrice <= 0 {
		outputPrice, ok = priceForResolution(outputRow.Prices, normalizeResolution(firstString(defaultResolution, firstPriceTier(outputRow.Prices))))
	}
	if !ok || outputPrice <= 0 {
		return 0, fmt.Errorf("Tencent VOD pricing missing output image price for mode %s resolution %s", outputMode, resolution)
	}
	if _, err := preciseResolutionRatio(modelRows, outputRow, outputMode, resolution, defaultResolution); err != nil {
		return 0, err
	}
	inputCount := float64(requestReferenceCount(req))
	return 1 + inputPrice*inputCount/outputPrice, nil
}

func priceForResolution(prices map[string]float64, resolution string) (float64, bool) {
	if len(prices) == 0 {
		return 0, false
	}
	if price, ok := prices[strings.TrimSpace(resolution)]; ok {
		return price, true
	}
	resolution = normalizeResolution(resolution)
	if price, ok := prices[resolution]; ok {
		return price, true
	}
	if price, ok := prices[strings.ToLower(resolution)]; ok {
		return price, true
	}
	return 0, false
}

func inferBasePricingMode(modelRows map[string]vodPriceRow, mode string) string {
	if _, ok := modelRows[mode]; !ok {
		return ""
	}
	candidates := []string{}
	switch {
	case mode == "reference_off_peak":
		candidates = append(candidates, "text", "i2v_first_last", "reference")
	case strings.HasSuffix(mode, "_off_peak"):
		candidates = append(candidates, strings.TrimSuffix(mode, "_off_peak"))
	case mode == "reference_1_3" || mode == "reference_4_7" || mode == "single_reference" || mode == "multi_reference":
		candidates = append(candidates, "text_to_image", "default")
	case mode == "reference":
		candidates = append(candidates, "text", "i2v_first_last", "no_reference", "default")
	case mode == "reference_no_audio" || mode == "reference_audio":
		candidates = append(candidates, "no_reference_no_audio")
	case mode == "no_reference_audio":
		candidates = append(candidates, "no_reference_no_audio")
	case mode == "audio" || mode == "audio_no_voice" || mode == "audio_voice":
		candidates = append(candidates, "silent", "no_reference_no_audio", "default")
	case mode == "motion_control":
		candidates = append(candidates, "silent", "default")
	}
	for _, candidate := range candidates {
		if _, ok := modelRows[candidate]; ok {
			return candidate
		}
	}
	return mode
}

func firstPriceTier(prices map[string]float64) string {
	for _, tier := range []string{"360P", "480P", "540P", "720P", "768P", "512P", "1K", "1080P", "2K", "4K"} {
		if _, ok := prices[tier]; ok {
			return tier
		}
	}
	return ""
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

func resolvePricingMode(req *relaycommon.TaskSubmitReq, spec modelSpec, modelRows map[string]vodPriceRow) string {
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
	hasMode := func(mode string) bool {
		_, ok := modelRows[mode]
		return ok
	}

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
		refCount := requestReferenceCount(req)
		if refCount == 1 {
			return "single_reference"
		}
		if refCount > 1 {
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
		if requestHasMotionControl(req) && hasMode("motion_control") {
			return "motion_control"
		}
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
		if hasMode("expand") {
			return "expand"
		}
		if hasMode("no_reference") || hasMode("reference") {
			if hasReference && hasMode("reference") {
				return "reference"
			}
			if hasMode("no_reference") {
				return "no_reference"
			}
		}
		if audio && hasMode("audio") {
			return "audio"
		}
		if hasMode("silent") {
			return "silent"
		}
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
	if metadataBool(req.Metadata, "audio") ||
		metadataBool(req.Metadata, "has_audio") ||
		metadataBool(req.Metadata, "generate_audio") ||
		metadataBool(req.Metadata, "with_audio") ||
		metadataBool(req.Metadata, "enable_audio") {
		return true
	}
	for _, key := range []string{"audio_generation", "generate_audio", "with_audio", "enable_audio"} {
		switch strings.ToLower(metadataString(req.Metadata, key)) {
		case "enabled", "true", "1", "yes", "on", "audio", "sound":
			return true
		}
	}
	return false
}

func requestHasMotionControl(req *relaycommon.TaskSubmitReq) bool {
	if metadataBool(req.Metadata, "motion_control") {
		return true
	}
	for _, key := range []string{"control_mode", "motion_mode", "mode", "pricing_mode"} {
		switch normalizePricingMode(metadataString(req.Metadata, key)) {
		case "motion_control":
			return true
		}
	}
	return normalizePricingMode(req.Mode) == "motion_control"
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

func metadataFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed
	}
	return 0
}
