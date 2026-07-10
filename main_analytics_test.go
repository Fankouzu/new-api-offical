package main

import (
	"strings"
	"testing"
)

func TestBuildGoogleAnalyticsSnippetsSeparatesDefaultAndClassicInitialization(t *testing.T) {
	defaultSnippet, classicSnippet := buildGoogleAnalyticsSnippets("G-TEST123")

	if !strings.Contains(defaultSnippet, `window.__GOOGLE_ANALYTICS_ID__="G-TEST123"`) {
		t.Fatalf("default snippet does not expose the runtime id: %s", defaultSnippet)
	}
	if strings.Contains(defaultSnippet, "googletagmanager.com") {
		t.Fatalf("default snippet initialized gtag outside the SPA: %s", defaultSnippet)
	}
	if !strings.Contains(classicSnippet, "googletagmanager.com/gtag/js") {
		t.Fatalf("classic snippet no longer initializes gtag: %s", classicSnippet)
	}
}

func TestBuildGoogleAnalyticsSnippetsEscapesInlineMeasurementID(t *testing.T) {
	maliciousID := "G-TEST\";alert(1)//"
	defaultSnippet, classicSnippet := buildGoogleAnalyticsSnippets(maliciousID)
	for name, snippet := range map[string]string{
		"default": defaultSnippet,
		"classic": classicSnippet,
	} {
		if strings.Contains(snippet, maliciousID) {
			t.Fatalf("%s snippet contains an unescaped inline id: %s", name, snippet)
		}
	}
}
