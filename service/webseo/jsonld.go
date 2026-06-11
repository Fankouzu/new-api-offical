package webseo

func homepageJSONLD(base string) []map[string]any {
	return []map[string]any{
		{
			"@context": "https://schema.org",
			"@type":    "Organization",
			"name":     defaultSiteName,
			"url":      base + "/",
		},
		{
			"@context":        "https://schema.org",
			"@type":           "WebSite",
			"name":            defaultSiteName,
			"url":             base + "/",
			"description":     "OpenAI-compatible AI model API marketplace",
			"inLanguage":      "en",
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
			"name":            "Lizh AI AI model API pricing marketplace",
			"itemListElement": list,
		},
		breadcrumbJSONLD(base, []breadcrumbItem{
			{Name: "Home", URL: base + "/"},
			{Name: "Pricing", URL: base + "/pricing"},
		}),
	}
}

func modelJSONLD(base string, item ModelSEOItem) []map[string]any {
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
			"offers": map[string]any{
				"@type":         "Offer",
				"url":           base + modelURLPath(item.ID),
				"priceCurrency": "USD",
				"availability":  "https://schema.org/InStock",
			},
		},
		breadcrumbJSONLD(base, []breadcrumbItem{
			{Name: "Home", URL: base + "/"},
			{Name: "Pricing", URL: base + "/pricing"},
			{Name: item.Name, URL: base + modelURLPath(item.ID)},
		}),
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
