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
	meta := Meta{
		Title:        defaultSiteName + " | OpenAI-compatible multi-model API gateway",
		Description:  "Lizh AI provides an OpenAI-compatible AI model API marketplace with unified access to GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and more.",
		CanonicalURL: canonicalURL(base, path),
		Robots:       noindexRobots,
		OGType:       "website",
	}

	switch {
	case path == "/":
		meta.Title = defaultSiteName
		meta.Description = "Access GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, MiniMax, Kimi, and other AI models through one OpenAI-compatible API gateway with unified billing."
		meta.Robots = indexRobots
		meta.JSONLD = homepageJSONLD(base)
	case path == "/pricing":
		meta.Title = "AI Model API Pricing Marketplace | GPT, Gemini, DeepSeek, Qwen - Lizh AI"
		meta.Description = "Compare Lizh AI model API pricing for GPT, Gemini, DeepSeek, Qwen, GLM, Doubao, MiniMax, Kimi, and 50+ models with text, image, tool, and structured-output support."
		meta.Robots = indexRobots
		meta.JSONLD = pricingJSONLD(base, catalog)
	case strings.HasPrefix(path, "/pricing/"):
		modelID, err := url.PathUnescape(strings.TrimPrefix(path, "/pricing/"))
		if err != nil {
			modelID = strings.TrimPrefix(path, "/pricing/")
		}
		if item, ok := findModel(catalog, modelID); ok {
			meta.Title = fmt.Sprintf("%s API pricing | Lizh AI", item.Name)
			meta.Description = modelDescription(item)
			meta.CanonicalURL = base + modelURLPath(item.ID)
			meta.Robots = indexRobots
			meta.OGType = "website"
			meta.JSONLD = modelJSONLD(base, item)
		} else {
			meta.Title = "Model pricing not found | Lizh AI"
			meta.Description = "This model pricing page is not available. Return to the Lizh AI model marketplace to view currently available models."
			meta.Robots = noindexRobots
		}
	case path == "/rankings" && theme != "classic":
		meta.Title = "Popular AI Model Rankings | Lizh AI"
		meta.Description = "Explore Lizh AI model usage rankings and compare demand trends for GPT, Gemini, DeepSeek, Qwen, Doubao, GLM, and other AI models."
		meta.Robots = indexRobots
	case path == "/about":
		meta.Title = "About Lizh AI | Multi-model API marketplace"
		meta.Description = "Learn about Lizh AI's multi-model API marketplace, OpenAI-compatible gateway, unified billing, and developer-focused model access experience."
		meta.Robots = indexRobots
	case path == "/privacy-policy":
		meta.Title = "Privacy Policy | Lizh AI"
		meta.Description = "Read the Lizh AI privacy policy to understand how account, API usage, billing, and service data are processed."
		meta.Robots = indexRobots
	case path == "/user-agreement":
		meta.Title = "User Agreement | Lizh AI"
		meta.Description = "Read the Lizh AI user agreement covering API marketplace usage, accounts, billing, and compliance requirements."
		meta.Robots = indexRobots
	default:
		meta.Title = utilityTitle(path)
		meta.Description = "This page is used for account, console, or system workflows and should not be indexed by search engines."
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
		description = fmt.Sprintf("%s API belongs to the %s model family", item.Name, item.Family)
	} else {
		description = fmt.Sprintf("%s API: %s", item.Name, description)
	}
	caps := strings.Join(item.Capabilities, ", ")
	return fmt.Sprintf("%s. Supports %s. Approximate input price %s and output price %s.", description, caps, formatPrice(item.InputPrice), formatPrice(item.OutputPrice))
}

func utilityTitle(path string) string {
	switch {
	case path == "/login" || path == "/sign-in":
		return "Sign in | Lizh AI"
	case path == "/register" || path == "/sign-up":
		return "Sign up | Lizh AI"
	case path == "/reset" || path == "/forgot-password" || path == "/user/reset":
		return "Reset password | Lizh AI"
	case strings.HasPrefix(path, "/oauth"):
		return "OAuth authorization | Lizh AI"
	case isAuthenticatedAppPath(path):
		return "Console | Lizh AI"
	case path == "/setup":
		return "System setup | Lizh AI"
	case path == "/401" || path == "/403" || path == "/forbidden":
		return "Access denied | Lizh AI"
	case path == "/500" || path == "/503":
		return "Service error | Lizh AI"
	default:
		return "Page not found | Lizh AI"
	}
}

func isAuthenticatedAppPath(path string) bool {
	authPrefixes := []string{
		"/_authenticated",
		"/console",
		"/usage-logs",
		"/playground",
		"/wallet",
		"/tokens",
		"/settings",
		"/user",
		"/users",
		"/channels",
		"/redemption",
		"/topup",
		"/subscription",
		"/billing",
		"/logs",
	}
	for _, prefix := range authPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
