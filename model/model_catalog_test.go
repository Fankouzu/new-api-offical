package model

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestNormalizeModelName(t *testing.T) {
	cases := map[string]string{
		"doubao-seedance-2-0-260128":   "doubao-seedance-2-0",
		"Claude-3-5-Sonnet-20241022":   "claude-3-5-sonnet",
		"gpt-4o":                       "gpt-4o",
		"deepseek-chat":                "deepseek-chat",
		"  Qwen-Max  ":                 "qwen-max",
		"gemini-2.0-flash-001":         "gemini-2.0-flash-001", // 3 digits are not a date suffix
		"claude-3-7-sonnet-20250219":   "claude-3-7-sonnet",
	}
	for input, want := range cases {
		if got := normalizeModelName(input); got != want {
			t.Errorf("normalizeModelName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseCatalogBuildsNormalizedIndex(t *testing.T) {
	data := []byte(`{
		"openai": {
			"models": {
				"gpt-4o": {
					"knowledge": "2023-10",
					"release_date": "2024-05-13",
					"tool_call": true,
					"structured_output": true,
					"reasoning": false,
					"temperature": true,
					"modalities": {"input": ["text","image"], "output": ["text"]},
					"limit": {"context": 128000, "input": 128000, "output": 16384}
				}
			}
		},
		"nano-gpt": {
			"models": {
				"doubao-seed-1.6": {
					"knowledge": "2024-06",
					"release_date": "2025-06-01",
					"tool_call": true,
					"structured_output": false,
					"reasoning": true,
					"modalities": {"input": ["text"], "output": ["text"]},
					"limit": {"context": 256000, "output": 32768}
				}
			}
		}
	}`)

	idx, err := parseCatalog(data)
	if err != nil {
		t.Fatalf("parseCatalog returned error: %v", err)
	}
	if len(idx) != 2 {
		t.Fatalf("expected 2 entries in index, got %d (%v)", len(idx), idx)
	}

	spec, ok := idx["gpt-4o"]
	if !ok {
		t.Fatalf("gpt-4o missing from normalized index (keys: %v)", idx)
	}
	if spec.ContextLength != 128000 {
		t.Errorf("gpt-4o context = %d, want 128000", spec.ContextLength)
	}
	if spec.MaxOutputTokens != 16384 {
		t.Errorf("gpt-4o max output = %d, want 16384", spec.MaxOutputTokens)
	}
	if spec.KnowledgeCutoff != "2023-10" {
		t.Errorf("gpt-4o knowledge = %q, want 2023-10", spec.KnowledgeCutoff)
	}
	if spec.ReleaseDate != "2024-05-13" {
		t.Errorf("gpt-4o release = %q, want 2024-05-13", spec.ReleaseDate)
	}
	wantCaps := []string{"streaming", "system_prompt", "tools", "function_calling", "structured_output", "json_mode", "vision"}
	if !reflect.DeepEqual(spec.Capabilities, wantCaps) {
		t.Errorf("gpt-4o capabilities = %v, want %v", spec.Capabilities, wantCaps)
	}
	wantIn := []string{"text", "image"}
	if !reflect.DeepEqual(spec.InputModalities, wantIn) {
		t.Errorf("gpt-4o input modalities = %v, want %v", spec.InputModalities, wantIn)
	}

	// doubao-seed-1.6 has reasoning true, no vision, no structured output.
	db, ok := idx["doubao-seed-1.6"]
	if !ok {
		t.Fatalf("doubao-seed-1.6 missing from index")
	}
	if !contains(db.Capabilities, "reasoning") {
		t.Errorf("doubao-seed-1.6 missing reasoning capability: %v", db.Capabilities)
	}
	if contains(db.Capabilities, "vision") {
		t.Errorf("doubao-seed-1.6 should not have vision: %v", db.Capabilities)
	}
}

func TestParseCatalogInvalidJSON(t *testing.T) {
	if _, err := parseCatalog([]byte("{not json")); err == nil {
		t.Fatalf("expected error for invalid json, got nil")
	}
}

func contains(slice []string, v string) bool {
	for _, s := range slice {
		if s == v {
			return true
		}
	}
	return false
}

// TestParseCatalogLiveFetch validates the parser against the real models.dev
// api.json. Opt-in (network) — set MODEL_CATALOG_INTEGRATION=1 to run.
func TestParseCatalogLiveFetch(t *testing.T) {
	if os.Getenv("MODEL_CATALOG_INTEGRATION") != "1" {
		t.Skip("set MODEL_CATALOG_INTEGRATION=1 to run the live models.dev fetch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	data, err := fetchCatalog(ctx)
	if err != nil {
		t.Fatalf("fetchCatalog failed: %v", err)
	}
	idx, err := parseCatalog(data)
	if err != nil {
		t.Fatalf("parseCatalog failed: %v", err)
	}
	if len(idx) < 1000 {
		t.Fatalf("expected a large catalog, got %d entries", len(idx))
	}
	if _, ok := idx["gpt-4o"]; !ok {
		t.Errorf("expected gpt-4o in catalog, keys sample: %v", firstKeys(idx, 5))
	}
	t.Logf("live catalog parsed: %d models", len(idx))
}

func firstKeys(m map[string]*ModelCatalogSpec, n int) []string {
	out := make([]string, 0, n)
	for k := range m {
		out = append(out, k)
		if len(out) >= n {
			break
		}
	}
	return out
}
