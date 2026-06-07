package webseo

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

func BuildCatalog(pricings []model.Pricing) []ModelSEOItem {
	items := make([]ModelSEOItem, 0, len(pricings))
	seen := make(map[string]bool)
	for _, pricing := range pricings {
		id := strings.TrimSpace(pricing.ModelName)
		if id == "" || seen[id] || len(pricing.EnableGroup) == 0 {
			continue
		}
		seen[id] = true
		input, output := pricingUSDPerMillion(pricing)
		items = append(items, ModelSEOItem{
			ID:           id,
			Name:         displayModelName(id),
			Description:  strings.TrimSpace(pricing.Description),
			Family:       modelFamily(id),
			InputPrice:   input,
			OutputPrice:  output,
			Capabilities: endpointCapabilities(pricing.SupportedEndpointTypes),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].ID) < strings.ToLower(items[j].ID)
	})
	return items
}

func findModel(items []ModelSEOItem, id string) (ModelSEOItem, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return ModelSEOItem{}, false
}

func pricingUSDPerMillion(pricing model.Pricing) (float64, float64) {
	if pricing.QuotaType == 1 {
		return pricing.ModelPrice, 0
	}
	input := pricing.ModelRatio * 2
	output := input * pricing.CompletionRatio
	return input, output
}

func displayModelName(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool {
		return r == '-' || r == '_' || r == '/' || unicode.IsSpace(r)
	})
	for i, part := range parts {
		lower := strings.ToLower(part)
		switch lower {
		case "gpt", "glm", "api", "json", "vl", "tts", "ocr", "ai":
			parts[i] = strings.ToUpper(part)
		case "qwen":
			parts[i] = "Qwen"
		case "deepseek":
			parts[i] = "DeepSeek"
		case "gemini":
			parts[i] = "Gemini"
		case "doubao":
			parts[i] = "Doubao"
		case "minimax":
			parts[i] = "MiniMax"
		case "kimi":
			parts[i] = "Kimi"
		default:
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func modelFamily(id string) string {
	lower := strings.ToLower(id)
	families := []string{"gpt", "gemini", "deepseek", "qwen", "doubao", "glm", "minimax", "kimi", "claude", "wan", "seedance"}
	for _, family := range families {
		if strings.Contains(lower, family) {
			return displayModelName(family)
		}
	}
	if idx := strings.IndexAny(id, "-_/"); idx > 0 {
		return displayModelName(id[:idx])
	}
	return displayModelName(id)
}

func endpointCapabilities(endpoints []constant.EndpointType) []string {
	capabilities := []string{"文本生成"}
	for _, endpoint := range endpoints {
		switch endpoint {
		case constant.EndpointTypeImageGeneration:
			capabilities = appendUnique(capabilities, "图像生成")
		case constant.EndpointTypeEmbeddings:
			capabilities = appendUnique(capabilities, "向量嵌入")
		case constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAIResponseCompact:
			capabilities = appendUnique(capabilities, "Responses API")
		case constant.EndpointTypeGemini:
			capabilities = appendUnique(capabilities, "Gemini 兼容")
		case constant.EndpointTypeAnthropic:
			capabilities = appendUnique(capabilities, "Claude 兼容")
		}
	}
	return capabilities
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func formatPrice(price float64) string {
	if price <= 0 {
		return "按请求或配置计费"
	}
	return fmt.Sprintf("$%.4f / 1M tokens", price)
}
