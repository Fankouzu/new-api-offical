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
		Title:        defaultSiteName + " | OpenAI 兼容多模型 API 网关",
		Description:  "Lizh AI 提供 OpenAI 兼容的大模型 API 聚合服务，统一接入 GPT、Gemini、DeepSeek、Qwen、豆包、GLM、MiniMax、Kimi 等模型。",
		CanonicalURL: canonicalURL(base, path),
		Robots:       noindexRobots,
		OGType:       "website",
	}

	switch {
	case path == "/":
		meta.Title = "Lizh AI | GPT、Gemini、DeepSeek、Qwen 多模型 API 聚合平台"
		meta.Description = "Lizh AI 提供 OpenAI 兼容的大模型 API 聚合服务，支持 GPT、Gemini、DeepSeek、Qwen、豆包、GLM、MiniMax、Kimi 等模型，统一计费、统一接口、快速接入。"
		meta.Robots = indexRobots
		meta.JSONLD = homepageJSONLD(base)
	case path == "/pricing":
		meta.Title = "AI 大模型 API 价格广场 | GPT、Gemini、DeepSeek、Qwen、豆包模型价格 - Lizh AI"
		meta.Description = "查看 Lizh AI 在售大模型 API 价格，覆盖 GPT、Gemini、DeepSeek、Qwen、GLM、豆包、MiniMax、Kimi 等 50+ 模型，支持文本、图像、工具调用和结构化输出。"
		meta.Robots = indexRobots
		meta.JSONLD = pricingJSONLD(base, catalog)
	case strings.HasPrefix(path, "/pricing/"):
		modelID, err := url.PathUnescape(strings.TrimPrefix(path, "/pricing/"))
		if err != nil {
			modelID = strings.TrimPrefix(path, "/pricing/")
		}
		if item, ok := findModel(catalog, modelID); ok {
			meta.Title = fmt.Sprintf("%s API 价格 | Lizh AI", item.Name)
			meta.Description = modelDescription(item)
			meta.CanonicalURL = base + modelURLPath(item.ID)
			meta.Robots = indexRobots
			meta.OGType = "product"
			meta.JSONLD = modelJSONLD(base, item)
		} else {
			meta.Title = "模型价格未找到 | Lizh AI"
			meta.Description = "该模型价格页面暂不可用，请返回 Lizh AI 大模型价格广场查看当前在售模型。"
			meta.Robots = noindexRobots
		}
	case path == "/rankings" && theme != "classic":
		meta.Title = "热门 AI 大模型排行榜 | Lizh AI"
		meta.Description = "查看 Lizh AI 大模型调用排行榜，了解 GPT、Gemini、DeepSeek、Qwen、豆包、GLM 等模型的热门程度和使用趋势。"
		meta.Robots = indexRobots
	case path == "/about":
		meta.Title = "关于 Lizh AI | 多模型 API 聚合与 OpenAI 兼容网关"
		meta.Description = "了解 Lizh AI 的多模型 API 聚合服务、OpenAI 兼容接口、统一计费能力和面向开发者的模型接入体验。"
		meta.Robots = indexRobots
	case path == "/privacy-policy":
		meta.Title = "隐私政策 | Lizh AI"
		meta.Description = "查看 Lizh AI 隐私政策，了解账号、API 调用、计费与服务数据的处理方式。"
		meta.Robots = indexRobots
	case path == "/user-agreement":
		meta.Title = "用户协议 | Lizh AI"
		meta.Description = "查看 Lizh AI 用户协议，了解 API 聚合服务使用、账号、计费与合规要求。"
		meta.Robots = indexRobots
	default:
		meta.Title = utilityTitle(path)
		meta.Description = "该页面用于账号、控制台或系统流程，不建议作为搜索结果收录。"
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
		description = fmt.Sprintf("%s API 属于 %s 模型系列", item.Name, item.Family)
	} else {
		description = fmt.Sprintf("%s API：%s", item.Name, description)
	}
	caps := strings.Join(item.Capabilities, "、")
	return fmt.Sprintf("%s，支持%s，输入价格约 %s，输出价格约 %s。", description, caps, formatPrice(item.InputPrice), formatPrice(item.OutputPrice))
}

func utilityTitle(path string) string {
	switch {
	case path == "/login" || path == "/sign-in":
		return "登录 | Lizh AI"
	case path == "/register" || path == "/sign-up":
		return "注册 | Lizh AI"
	case path == "/reset" || path == "/forgot-password" || path == "/user/reset":
		return "找回密码 | Lizh AI"
	case strings.HasPrefix(path, "/oauth"):
		return "授权登录 | Lizh AI"
	case strings.HasPrefix(path, "/console"):
		return "控制台 | Lizh AI"
	case path == "/setup":
		return "系统初始化 | Lizh AI"
	case path == "/401" || path == "/403" || path == "/forbidden":
		return "无权访问 | Lizh AI"
	case path == "/500" || path == "/503":
		return "服务错误 | Lizh AI"
	default:
		return "页面未找到 | Lizh AI"
	}
}
