package webseo

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var (
	titleTagPattern       = regexp.MustCompile(`(?is)<title>.*?</title>`)
	descriptionPattern    = regexp.MustCompile(`(?is)<meta\s+name=["']description["'][^>]*>`)
	metaTitlePattern      = regexp.MustCompile(`(?is)<meta\s+name=["']title["'][^>]*>`)
	robotsPattern         = regexp.MustCompile(`(?is)<meta\s+name=["']robots["'][^>]*>`)
	canonicalPattern      = regexp.MustCompile(`(?is)<link\s+rel=["']canonical["'][^>]*>`)
	openGraphPattern      = regexp.MustCompile(`(?is)<meta\s+property=["']og:[^"']+["'][^>]*>`)
	twitterPattern        = regexp.MustCompile(`(?is)<meta\s+name=["']twitter:[^"']+["'][^>]*>`)
	jsonLDPattern         = regexp.MustCompile(`(?is)<script\s+type=["']application/ld\+json["'][^>]*>.*?</script>`)
	seoInjectedTagPattern = regexp.MustCompile(`(?is)<!--seo:injected:start-->.*?<!--seo:injected:end-->\s*`)
	headClosePattern      = regexp.MustCompile(`(?is)</head>`)
)

func RenderIndexHTML(indexHTML []byte, meta Meta) []byte {
	output := string(indexHTML)
	output = seoInjectedTagPattern.ReplaceAllString(output, "")
	output = titleTagPattern.ReplaceAllString(output, "")
	output = descriptionPattern.ReplaceAllString(output, "")
	output = metaTitlePattern.ReplaceAllString(output, "")
	output = robotsPattern.ReplaceAllString(output, "")
	output = canonicalPattern.ReplaceAllString(output, "")
	output = openGraphPattern.ReplaceAllString(output, "")
	output = twitterPattern.ReplaceAllString(output, "")
	output = jsonLDPattern.ReplaceAllString(output, "")

	seo := buildSEOTags(meta)
	if headClosePattern.MatchString(output) {
		return []byte(headClosePattern.ReplaceAllLiteralString(output, seo+"</head>"))
	}
	return []byte(seo + output)
}

func buildSEOTags(meta Meta) string {
	var builder strings.Builder
	builder.WriteString("    <!--seo:injected:start-->\n")
	writeTag(&builder, "title", meta.Title)
	writeMeta(&builder, "name", "title", meta.Title)
	writeMeta(&builder, "name", "description", meta.Description)
	writeMeta(&builder, "name", "robots", firstNonEmpty(meta.Robots, noindexRobots))
	if meta.CanonicalURL != "" {
		builder.WriteString(`    <link rel="canonical" href="`)
		builder.WriteString(html.EscapeString(meta.CanonicalURL))
		builder.WriteString(`">` + "\n")
	}
	writeMeta(&builder, "property", "og:type", firstNonEmpty(meta.OGType, "website"))
	writeMeta(&builder, "property", "og:title", meta.Title)
	writeMeta(&builder, "property", "og:description", meta.Description)
	writeMeta(&builder, "property", "og:url", meta.CanonicalURL)
	writeMeta(&builder, "name", "twitter:card", "summary_large_image")
	writeMeta(&builder, "name", "twitter:title", meta.Title)
	writeMeta(&builder, "name", "twitter:description", meta.Description)
	for _, data := range meta.JSONLD {
		if jsonBytes, err := common.Marshal(data); err == nil {
			builder.WriteString(`    <script type="application/ld+json">`)
			builder.Write(bytes.TrimSpace(jsonBytes))
			builder.WriteString("</script>\n")
		}
	}
	builder.WriteString("    <!--seo:injected:end-->\n")
	return builder.String()
}

func writeTag(builder *strings.Builder, tag string, value string) {
	builder.WriteString("    <")
	builder.WriteString(tag)
	builder.WriteString(">")
	builder.WriteString(html.EscapeString(value))
	builder.WriteString("</")
	builder.WriteString(tag)
	builder.WriteString(">\n")
}

func writeMeta(builder *strings.Builder, attrName string, attrValue string, content string) {
	if content == "" {
		return
	}
	builder.WriteString("    <meta ")
	builder.WriteString(attrName)
	builder.WriteString(`="`)
	builder.WriteString(html.EscapeString(attrValue))
	builder.WriteString(`" content="`)
	builder.WriteString(html.EscapeString(content))
	builder.WriteString(`">` + "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
