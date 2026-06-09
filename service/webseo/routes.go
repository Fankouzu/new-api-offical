package webseo

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

const (
	defaultSiteName = "Lizh AI"
	indexRobots     = "index,follow"
	noindexRobots   = "noindex,nofollow"
)

func ResolveMeta(requestURI string, baseURL string, pricings []model.Pricing) Meta {
	return ResolveMetaForTheme(requestURI, baseURL, pricings, "")
}

func ResolveMetaForTheme(requestURI string, baseURL string, pricings []model.Pricing, theme string) Meta {
	path := normalizePath(requestURI)
	base := normalizeBaseURL(baseURL)
	catalog := BuildCatalog(pricings)
	topic := topicPage(path)
	meta := Meta{
		Title:        defaultSiteName + " | AI Model Marketplace and API Gateway",
		Description:  "Lizh AI is an AI model marketplace with OpenAI-compatible API access for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream models.",
		CanonicalURL: canonicalURL(base, path),
		Robots:       noindexRobots,
		OGType:       "website",
	}

	switch {
	case path == "/":
		meta.Title = "Lizh AI | AI Model Marketplace and OpenAI-Compatible API Gateway"
		meta.Description = "Lizh AI is an AI model marketplace with OpenAI-compatible access to GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream models."
		meta.Robots = indexRobots
		meta.JSONLD = homepageJSONLD(base)
	case path == "/pricing":
		meta.Title = "AI Model API Pricing Marketplace | GPT, Gemini, DeepSeek, Qwen - Lizh AI"
		meta.Description = "Compare AI model API prices in Lizh AI, including GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other mainstream models."
		meta.Robots = indexRobots
		meta.JSONLD = pricingJSONLD(base, catalog)
	case strings.HasPrefix(path, "/pricing/"):
		modelID, err := url.PathUnescape(strings.TrimPrefix(path, "/pricing/"))
		if err != nil {
			modelID = strings.TrimPrefix(path, "/pricing/")
		}
		if item, ok := findModel(catalog, modelID); ok {
			meta.Title = fmt.Sprintf("%s API Pricing | Lizh AI", item.Name)
			meta.Description = modelDescription(item)
			meta.CanonicalURL = base + modelURLPath(item.ID)
			meta.Robots = indexRobots
			meta.OGType = "product"
			meta.JSONLD = modelJSONLD(base, item)
		} else {
			meta.Title = "Model Pricing Not Found | Lizh AI"
			meta.Description = "This model pricing page is not currently available. Return to the Lizh AI model pricing marketplace to view available models."
			meta.Robots = noindexRobots
		}
	case path == "/rankings" && theme != "classic":
		meta.Title = "Popular AI Model Rankings | Lizh AI"
		meta.Description = "Explore popular AI model rankings for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, and other model families available through Lizh AI."
		meta.Robots = indexRobots
	case path == "/about":
		meta.Title = "About Lizh AI | AI Model Marketplace and API Gateway"
		meta.Description = "Learn about Lizh AI, an AI model marketplace for multi-model API access, OpenAI-compatible integration, and unified account settlement."
		meta.Robots = indexRobots
		meta.JSONLD = aboutJSONLD(base)
	case topic != nil:
		meta.Title = topic.Title
		meta.Description = topic.Description
		meta.Robots = indexRobots
		meta.JSONLD = topicJSONLD(base, *topic)
	case path == "/privacy-policy":
		meta.Title = "Privacy Policy | Lizh AI"
		meta.Description = "Review the Lizh AI privacy policy for account, API usage, billing, and service data handling."
		meta.Robots = indexRobots
	case path == "/user-agreement":
		meta.Title = "User Agreement | Lizh AI"
		meta.Description = "Review the Lizh AI user agreement for API gateway usage, account, billing, and compliance requirements."
		meta.Robots = indexRobots
	default:
		meta.Title = defaultSiteName
		meta.Description = "This account, console, or system workflow page should not be indexed as a search result."
		meta.Robots = noindexRobots
	}

	return meta
}

func normalizePath(requestURI string) string {
	raw := strings.TrimSpace(requestURI)
	if raw == "" {
		return "/"
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Path != "" {
		raw = parsed.Path
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return "/"
	}
	return raw
}

func normalizeBaseURL(baseURL string) string {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		return "http://localhost:3000"
	}
	return strings.TrimRight(base, "/")
}

func canonicalURL(baseURL, path string) string {
	if path == "/" {
		return baseURL + "/"
	}
	return baseURL + path
}

func modelURLPath(modelID string) string {
	return "/pricing/" + url.PathEscape(modelID)
}

func modelDescription(item ModelSEOItem) string {
	description := strings.TrimSpace(item.Description)
	if description == "" {
		description = fmt.Sprintf("%s API is part of the %s model family", item.Name, item.Family)
	} else {
		description = fmt.Sprintf("%s API: %s", item.Name, description)
	}
	caps := strings.Join(item.Capabilities, ", ")
	return fmt.Sprintf("%s. Capabilities: %s. Approximate input price: %s. Approximate output price: %s.", description, caps, formatPrice(item.InputPrice), formatPrice(item.OutputPrice))
}
