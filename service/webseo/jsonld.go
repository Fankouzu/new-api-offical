package webseo

func homepageJSONLD(base string) []map[string]any {
	return []map[string]any{
		{
			"@context": "https://schema.org",
			"@type":    "Organization",
			"name":     defaultSiteName,
			"url":      base + "/",
			"logo":     siteLogoURL,
			"contactPoint": []map[string]any{
				{
					"@type":       "ContactPoint",
					"email":       supportEmail,
					"contactType": "customer support",
					"areaServed":  "Worldwide",
				},
			},
		},
		{
			"@context":        "https://schema.org",
			"@type":           "WebSite",
			"name":            defaultSiteName,
			"url":             base + "/",
			"description":     "OpenAI 兼容的大模型 API 聚合平台",
			"inLanguage":      "zh-CN",
			"potentialAction": searchAction(base),
		},
		{
			"@context":            "https://schema.org",
			"@type":               "SoftwareApplication",
			"name":                defaultSiteName,
			"applicationCategory": "DeveloperApplication",
			"operatingSystem":     "Web",
			"url":                 base + "/",
		},
	}
}

func pricingJSONLD(base string, items []ModelSEOItem) []map[string]any {
	list := make([]map[string]any, 0, min(len(items), 50))
	for idx, item := range items {
		if idx >= 50 {
			break
		}
		list = append(list, map[string]any{
			"@type":    "ListItem",
			"position": idx + 1,
			"url":      base + modelURLPath(item.ID),
			"name":     item.Name,
		})
	}
	return []map[string]any{
		{
			"@context":        "https://schema.org",
			"@type":           "ItemList",
			"name":            "Lizh AI 大模型 API 价格广场",
			"itemListElement": list,
		},
		breadcrumbJSONLD(base, []breadcrumbItem{
			{Name: "首页", URL: base + "/"},
			{Name: "价格广场", URL: base + "/pricing"},
		}),
	}
}

func modelJSONLD(base string, item ModelSEOItem) []map[string]any {
	offer := map[string]any{
		"@type":         "Offer",
		"url":           base + modelURLPath(item.ID),
		"priceCurrency": "USD",
		"availability":  "https://schema.org/InStock",
	}
	if item.InputPrice > 0 {
		offer["price"] = item.InputPrice
	}
	return []map[string]any{
		{
			"@context":    "https://schema.org",
			"@type":       "Product",
			"name":        item.Name + " API",
			"description": modelDescription(item),
			"brand": map[string]any{
				"@type": "Brand",
				"name":  defaultSiteName,
			},
			"offers": offer,
		},
		breadcrumbJSONLD(base, []breadcrumbItem{
			{Name: "首页", URL: base + "/"},
			{Name: "价格广场", URL: base + "/pricing"},
			{Name: item.Name, URL: base + modelURLPath(item.ID)},
		}),
		faqJSONLD([]faqItem{
			{Question: "How do I access the " + item.Name + " API?", Answer: "Use the model ID " + item.ID + " from an API key in Lizh AI. The gateway is designed for OpenAI-compatible client usage."},
			{Question: "Is " + item.Name + " compatible with the OpenAI SDK?", Answer: "Lizh AI exposes OpenAI-compatible access patterns for supported models."},
		}),
	}
}

func aboutJSONLD(base string) []map[string]any {
	return []map[string]any{
		{
			"@context":    "https://schema.org",
			"@type":       "AboutPage",
			"name":        "About Lizh AI",
			"url":         base + "/about",
			"description": "Lizh AI is an AI model marketplace for multi-model API access and OpenAI-compatible integration.",
		},
		breadcrumbJSONLD(base, []breadcrumbItem{
			{Name: "首页", URL: base + "/"},
			{Name: "关于", URL: base + "/about"},
		}),
	}
}

func topicJSONLD(base string, topic topicDefinition) []map[string]any {
	return []map[string]any{
		{
			"@context":    "https://schema.org",
			"@type":       "Article",
			"headline":    topic.H1,
			"name":        topic.Title,
			"description": topic.Description,
			"url":         canonicalURL(base, topic.Path),
			"author": map[string]any{
				"@type": "Organization",
				"name":  defaultSiteName,
			},
			"publisher": map[string]any{
				"@type": "Organization",
				"name":  defaultSiteName,
				"logo": map[string]any{
					"@type": "ImageObject",
					"url":   siteLogoURL,
				},
			},
		},
		breadcrumbJSONLD(base, []breadcrumbItem{
			{Name: "首页", URL: base + "/"},
			{Name: topic.H1, URL: canonicalURL(base, topic.Path)},
		}),
		faqJSONLD(topic.FAQ),
	}
}

func faqJSONLD(items []faqItem) map[string]any {
	entities := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entities = append(entities, map[string]any{
			"@type": "Question",
			"name":  item.Question,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  item.Answer,
			},
		})
	}
	return map[string]any{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": entities,
	}
}

type breadcrumbItem struct {
	Name string
	URL  string
}

func breadcrumbJSONLD(_ string, items []breadcrumbItem) map[string]any {
	elements := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		elements = append(elements, map[string]any{
			"@type":    "ListItem",
			"position": idx + 1,
			"name":     item.Name,
			"item":     item.URL,
		})
	}
	return map[string]any{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": elements,
	}
}

func searchAction(base string) map[string]any {
	return map[string]any{
		"@type":       "SearchAction",
		"target":      base + "/pricing?keyword={search_term_string}",
		"query-input": "required name=search_term_string",
	}
}
