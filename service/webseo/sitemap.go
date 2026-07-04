package webseo

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
)

func BuildRobotsTxt(baseURL string) string {
	base := normalizeBaseURL(baseURL)
	lines := []string{
		"User-agent: *",
		"Allow: /llms.txt",
		"Allow: /",
		"Disallow: /console/",
		"Disallow: /api/",
		"Disallow: /v1/",
		"Disallow: /openrouter/",
		"Disallow: /login",
		"Disallow: /register",
		"Disallow: /reset",
		"Disallow: /user/reset",
		"Disallow: /oauth/",
		"Disallow: /setup",
		"",
		"Sitemap: " + base + "/sitemap.xml",
		"",
	}
	return strings.Join(lines, "\n")
}

func BuildSitemapXML(baseURL string, pricings []model.Pricing) string {
	return BuildSitemapXMLForTheme(baseURL, pricings, "")
}

func BuildSitemapXMLForTheme(baseURL string, pricings []model.Pricing, theme string) string {
	base := normalizeBaseURL(baseURL)
	urlset := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}
	now := time.Now().UTC().Format("2006-01-02")
	staticPaths := []string{"/", "/pricing", "/compare/ai-api-pricing", "/rankings", "/about", "/privacy-policy", "/user-agreement"}
	if theme == "classic" {
		staticPaths = []string{"/", "/pricing", "/compare/ai-api-pricing", "/about", "/privacy-policy", "/user-agreement"}
	}
	for _, path := range staticPaths {
		if isNoindexSitemapPath(path, base, pricings, theme) {
			continue
		}
		urlset.URLs = append(urlset.URLs, sitemapURL{
			Loc:        canonicalURL(base, path),
			LastMod:    now,
			ChangeFreq: changeFreq(path),
			Priority:   priority(path),
		})
	}
	for _, item := range BuildCatalog(pricings) {
		path := modelURLPath(item.ID)
		if isNoindexSitemapPath(path, base, pricings, theme) {
			continue
		}
		urlset.URLs = append(urlset.URLs, sitemapURL{
			Loc:        base + path,
			LastMod:    now,
			ChangeFreq: "weekly",
			Priority:   "0.7",
		})
	}
	bytes, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(bytes) + "\n"
}

func isNoindexSitemapPath(path, baseURL string, pricings []model.Pricing, theme string) bool {
	return strings.Contains(ResolveMetaForTheme(path, baseURL, pricings, theme).Robots, "noindex")
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

func changeFreq(path string) string {
	switch path {
	case "/", "/pricing":
		return "daily"
	case "/compare/ai-api-pricing":
		return "daily"
	case "/rankings":
		return "hourly"
	default:
		return "monthly"
	}
}

func priority(path string) string {
	switch path {
	case "/":
		return "1.0"
	case "/pricing":
		return "0.9"
	case "/compare/ai-api-pricing":
		return "0.9"
	case "/rankings", "/about":
		return "0.8"
	default:
		return "0.5"
	}
}
