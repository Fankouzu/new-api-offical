package service

import (
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const openRouterDefaultCreated int64 = 1626777600

var openRouterDefaultSamplingParameters = []string{
	"temperature",
	"top_p",
	"frequency_penalty",
	"presence_penalty",
	"stop",
	"seed",
	"max_tokens",
	"logit_bias",
}

type openRouterProviderMetadata struct {
	huggingFaceID    string
	name             string
	quantization     string
	contextLength    int64
	maxOutputLength  int64
	deprecationDate  string
	isReady          bool
	slug             string
	datacenters      []dto.OpenRouterProviderDataCtr
	inputModalities  []string
	outputModalities []string
	features         []string
}

func BuildOpenRouterProviderModels(pricings []model.Pricing) []dto.OpenRouterProviderModel {
	models := make([]dto.OpenRouterProviderModel, 0, len(pricings))
	for _, pricing := range pricings {
		if strings.TrimSpace(pricing.ModelName) == "" || len(pricing.EnableGroup) == 0 {
			continue
		}

		meta := parseOpenRouterProviderMetadata(pricing)
		inputModalities, outputModalities := deriveOpenRouterModalities(pricing.SupportedEndpointTypes)
		if len(meta.inputModalities) > 0 {
			inputModalities = meta.inputModalities
		}
		if len(meta.outputModalities) > 0 {
			outputModalities = meta.outputModalities
		}

		features := deriveOpenRouterFeatures(pricing.SupportedEndpointTypes)
		features = mergeUniqueStrings(features, meta.features)
		sort.Strings(features)

		orPricing := buildOpenRouterPricing(pricing)
		models = append(models, dto.OpenRouterProviderModel{
			ID:                          pricing.ModelName,
			HuggingFaceID:               meta.huggingFaceID,
			Name:                        firstNonEmpty(meta.name, pricing.ModelName),
			Created:                     openRouterDefaultCreated,
			InputModalities:             inputModalities,
			OutputModalities:            outputModalities,
			Quantization:                meta.quantization,
			ContextLength:               meta.contextLength,
			MaxOutputLength:             meta.maxOutputLength,
			Pricing:                     orPricing,
			SupportedSamplingParameters: append([]string(nil), openRouterDefaultSamplingParameters...),
			SupportedFeatures:           features,
			Description:                 pricing.Description,
			DeprecationDate:             meta.deprecationDate,
			IsReady:                     meta.isReady,
			IsFree:                      isOpenRouterFreePricing(orPricing),
			OpenRouter: dto.OpenRouterProviderSlug{
				Slug: firstNonEmpty(meta.slug, pricing.ModelName),
			},
			Datacenters: meta.datacenters,
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	return models
}

func parseOpenRouterProviderMetadata(pricing model.Pricing) openRouterProviderMetadata {
	meta := openRouterProviderMetadata{
		isReady: true,
	}
	for _, rawTag := range strings.Split(pricing.Tags, ",") {
		tag := strings.TrimSpace(rawTag)
		if !strings.HasPrefix(tag, "or:") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimPrefix(tag, "or:"), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "hf":
			meta.huggingFaceID = value
		case "name":
			meta.name = value
		case "quantization":
			meta.quantization = value
		case "context":
			meta.contextLength = parsePositiveInt64(value)
		case "max_output":
			meta.maxOutputLength = parsePositiveInt64(value)
		case "deprecation":
			meta.deprecationDate = value
		case "ready":
			meta.isReady = strings.ToLower(value) != "false"
		case "slug":
			meta.slug = value
		case "dc":
			for _, country := range strings.Split(value, "|") {
				country = strings.ToUpper(strings.TrimSpace(country))
				if len(country) == 2 {
					meta.datacenters = append(meta.datacenters, dto.OpenRouterProviderDataCtr{CountryCode: country})
				}
			}
		case "input":
			meta.inputModalities = parseOpenRouterCSV(value)
		case "output":
			meta.outputModalities = parseOpenRouterCSV(value)
		case "feature":
			meta.features = parseOpenRouterCSV(value)
		}
	}
	return meta
}

func deriveOpenRouterModalities(endpointTypes []constant.EndpointType) ([]string, []string) {
	inputs := []string{}
	outputs := []string{}
	for _, endpointType := range endpointTypes {
		switch endpointType {
		case constant.EndpointTypeEmbeddings:
			inputs = append(inputs, "text")
			outputs = append(outputs, "embeddings")
		case constant.EndpointTypeImageGeneration:
			inputs = append(inputs, "text", "image")
			outputs = append(outputs, "image")
		case constant.EndpointTypeJinaRerank:
			inputs = append(inputs, "text")
			outputs = append(outputs, "rerank")
		case constant.EndpointTypeOpenAIVideo:
			inputs = append(inputs, "text", "image", "video")
			outputs = append(outputs, "video")
		default:
			inputs = append(inputs, "text")
			outputs = append(outputs, "text")
		}
	}
	if len(inputs) == 0 {
		inputs = append(inputs, "text")
	}
	if len(outputs) == 0 {
		outputs = append(outputs, "text")
	}
	return mergeUniqueStrings(nil, inputs), mergeUniqueStrings(nil, outputs)
}

func deriveOpenRouterFeatures(endpointTypes []constant.EndpointType) []string {
	features := []string{}
	for _, endpointType := range endpointTypes {
		switch endpointType {
		case constant.EndpointTypeOpenAI, constant.EndpointTypeOpenAIResponse, constant.EndpointTypeAnthropic, constant.EndpointTypeGemini:
			features = append(features, "tools", "json_mode", "structured_outputs")
		}
	}
	return mergeUniqueStrings(nil, features)
}

func buildOpenRouterPricing(pricing model.Pricing) dto.OpenRouterProviderPricing {
	result := dto.OpenRouterProviderPricing{
		Prompt:         "0",
		Completion:     "0",
		Image:          "0",
		Request:        "0",
		InputCacheRead: "0",
	}
	if pricing.QuotaType == 1 {
		if pricing.ModelPrice > 0 {
			result.Request = formatOpenRouterPrice(pricing.ModelPrice)
		}
		return result
	}
	if pricing.ModelRatio <= 0 {
		return result
	}

	prompt := pricing.ModelRatio / (1000 * ratio_setting.USD)
	completionRatio := pricing.CompletionRatio
	if completionRatio <= 0 {
		completionRatio = 1
	}
	result.Prompt = formatOpenRouterPrice(prompt)
	result.Completion = formatOpenRouterPrice(prompt * completionRatio)
	if pricing.CacheRatio != nil && *pricing.CacheRatio >= 0 {
		result.InputCacheRead = formatOpenRouterPrice(prompt * *pricing.CacheRatio)
	}
	if pricing.ImageRatio != nil && *pricing.ImageRatio > 0 {
		result.Image = formatOpenRouterPrice(*pricing.ImageRatio)
	}
	return result
}

func isOpenRouterFreePricing(pricing dto.OpenRouterProviderPricing) bool {
	return pricing.Prompt == "0" &&
		pricing.Completion == "0" &&
		pricing.Image == "0" &&
		pricing.Request == "0" &&
		pricing.InputCacheRead == "0"
}

func formatOpenRouterPrice(value float64) string {
	if value <= 0 {
		return "0"
	}
	formatted := strconv.FormatFloat(value, 'f', 12, 64)
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}

func parsePositiveInt64(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func parseOpenRouterCSV(value string) []string {
	values := make([]string, 0)
	for _, item := range strings.FieldsFunc(value, func(r rune) bool {
		return r == '|' || r == ';'
	}) {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}
	return mergeUniqueStrings(nil, values)
}

func mergeUniqueStrings(base []string, values []string) []string {
	seen := make(map[string]struct{}, len(base)+len(values))
	result := make([]string, 0, len(base)+len(values))
	for _, value := range append(base, values...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
