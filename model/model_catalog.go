package model

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
)

// model_catalog.go integrates the open-source models.dev catalog (MIT,
// https://github.com/sst/models.dev) as the source of model specification
// metadata (context window, knowledge cutoff, release date, modalities,
// capabilities) shown on the /pricing/:model detail page.
//
// The catalog is fetched from https://models.dev/api.json, cached in memory for
// at least 24h, and refreshed by a background task. Only fields models.dev
// covers are populated; uncovered fields are left empty (no client-side mock).
// Additional sources (tokenizer, data retention, full parameter tables) will be
// wired in later — see pricing-detail-data-sources memory.

const (
	defaultModelCatalogURL   = "https://models.dev/api.json"
	modelCatalogTimeout      = 60 * time.Second
	modelCatalogDefaultTTLH  = 24 // hours — cache for at least one day
)

var (
	// dateSuffixRe strips trailing version-date suffixes that new-api appends to
	// model names (e.g. "doubao-seedance-2-0-260128", "claude-3-5-sonnet-20241022")
	// so they can match models.dev's canonical ids.
	dateSuffixRe = regexp.MustCompile(`-(?:\d{8}|\d{6})$`)

	modelCatalogOnce    sync.Once
	modelCatalogMu      sync.RWMutex
	modelCatalogIndex   map[string]*ModelCatalogSpec
	modelCatalogLoadedAt time.Time
	modelCatalogRunning atomic.Bool
)

// ModelCatalogSpec is the subset of models.dev metadata exposed on the pricing
// detail page. Field json tags intentionally match PricingModel in the frontend
// (web/default/src/features/pricing/types.ts) so values flow through unchanged.
type ModelCatalogSpec struct {
	ContextLength    int      `json:"context_length,omitempty"`
	MaxInputTokens   int      `json:"max_input_tokens,omitempty"`
	MaxOutputTokens  int      `json:"max_output_tokens,omitempty"`
	KnowledgeCutoff  string   `json:"knowledge_cutoff,omitempty"`
	ReleaseDate      string   `json:"release_date,omitempty"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	OpenWeights      bool     `json:"open_weights,omitempty"`
}

// raw catalog JSON shapes (a subset of the models.dev schema we care about).
type rawCatalogProvider struct {
	Models map[string]rawCatalogModel `json:"models"`
}

type rawCatalogModel struct {
	Limit           *rawCatalogLimit `json:"limit"`
	Knowledge       string           `json:"knowledge"`
	ReleaseDate     string           `json:"release_date"`
	Modalities      *rawCatalogMod   `json:"modalities"`
	ToolCall        bool             `json:"tool_call"`
	StructuredOut   bool             `json:"structured_output"`
	Reasoning       bool             `json:"reasoning"`
	Attachment      bool             `json:"attachment"`
	OpenWeights     bool             `json:"open_weights"`
	Temperature     bool             `json:"temperature"`
}

type rawCatalogLimit struct {
	Context int `json:"context"`
	Input   int `json:"input"`
	Output  int `json:"output"`
}

type rawCatalogMod struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// normalizeModelName lower-cases and strips trailing 6/8-digit date suffixes.
func normalizeModelName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	return dateSuffixRe.ReplaceAllString(s, "")
}

func containsString(slice []string, v string) bool {
	for _, s := range slice {
		if s == v {
			return true
		}
	}
	return false
}

// buildSpec maps a raw models.dev entry to the public spec, including a
// capabilities list expressed in the frontend's ModelCapability vocabulary.
func buildSpec(raw rawCatalogModel) *ModelCatalogSpec {
	spec := &ModelCatalogSpec{
		KnowledgeCutoff: raw.Knowledge,
		ReleaseDate:     raw.ReleaseDate,
		OpenWeights:     raw.OpenWeights,
	}
	if raw.Limit != nil {
		spec.ContextLength = raw.Limit.Context
		spec.MaxInputTokens = raw.Limit.Input
		spec.MaxOutputTokens = raw.Limit.Output
	}
	if raw.Modalities != nil {
		spec.InputModalities = raw.Modalities.Input
		spec.OutputModalities = raw.Modalities.Output
	}

	caps := make([]string, 0, 8)
	hasTextOutput := raw.Modalities != nil && containsString(raw.Modalities.Output, "text")
	if hasTextOutput {
		caps = append(caps, "streaming", "system_prompt")
	}
	if raw.ToolCall {
		caps = append(caps, "tools", "function_calling")
	}
	if raw.StructuredOut {
		caps = append(caps, "structured_output", "json_mode")
	}
	if raw.Reasoning {
		caps = append(caps, "reasoning")
	}
	if raw.Modalities != nil && (containsString(raw.Modalities.Input, "image") || raw.Attachment) {
		caps = append(caps, "vision")
	}
	spec.Capabilities = caps
	return spec
}

// parseCatalog parses the models.dev api.json payload into a lookup index keyed
// by normalized model id. When the same model id appears under multiple
// providers, the first occurrence wins (their specs are normally identical).
func parseCatalog(data []byte) (map[string]*ModelCatalogSpec, error) {
	var providers map[string]rawCatalogProvider
	if err := common.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("failed to parse model catalog json: %w", err)
	}

	idx := make(map[string]*ModelCatalogSpec)
	for _, provider := range providers {
		for id, raw := range provider.Models {
			key := normalizeModelName(id)
			if key == "" {
				continue
			}
			if _, exists := idx[key]; !exists {
				idx[key] = buildSpec(raw)
			}
		}
	}
	return idx, nil
}

// catalogTTL returns the configured cache lifetime (>= 1 day by default).
func catalogTTL() time.Duration {
	hours := common.GetEnvOrDefault("MODEL_CATALOG_CACHE_HOURS", modelCatalogDefaultTTLH)
	if hours < modelCatalogDefaultTTLH {
		hours = modelCatalogDefaultTTLH
	}
	return time.Duration(hours) * time.Hour
}

func catalogURL() string {
	return common.GetEnvOrDefaultString("MODEL_CATALOG_URL", defaultModelCatalogURL)
}

// fetchCatalog downloads the catalog payload with a bounded timeout.
func fetchCatalog(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "new-api/model-catalog")
	client := &http.Client{Timeout: modelCatalogTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model catalog fetch returned status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024)) // hard cap 50MB
}

// loadModelCatalog fetches, parses, and atomically swaps the cached index.
// Failures are logged and leave the previous cache (if any) intact.
func loadModelCatalog() {
	ctx, cancel := context.WithTimeout(context.Background(), modelCatalogTimeout)
	defer cancel()

	data, err := fetchCatalog(ctx)
	if err != nil {
		common.SysLog(fmt.Sprintf("model catalog fetch failed (keeping previous cache): %v", err))
		return
	}
	idx, err := parseCatalog(data)
	if err != nil {
		common.SysLog(fmt.Sprintf("model catalog parse failed: %v", err))
		return
	}

	modelCatalogMu.Lock()
	modelCatalogIndex = idx
	modelCatalogLoadedAt = time.Now()
	modelCatalogMu.Unlock()
	common.SysLog(fmt.Sprintf("model catalog loaded: %d models", len(idx)))
}

// ensureModelCatalogLoaded performs a lazy first load if the cache is empty or
// stale. It is safe to call concurrently.
func ensureModelCatalogLoaded() {
	modelCatalogMu.RLock()
	loaded := len(modelCatalogIndex) > 0 && time.Since(modelCatalogLoadedAt) < catalogTTL()
	modelCatalogMu.RUnlock()
	if loaded {
		return
	}
	if !modelCatalogRunning.CompareAndSwap(false, true) {
		return // another goroutine is loading
	}
	defer modelCatalogRunning.Store(false)
	loadModelCatalog()
}

// GetModelCatalogSpec looks up the spec for a model name. Returns ok=false when
// the model is absent from the catalog (caller leaves the field empty).
func GetModelCatalogSpec(modelName string) (*ModelCatalogSpec, bool) {
	if strings.TrimSpace(modelName) == "" {
		return nil, false
	}
	ensureModelCatalogLoaded()
	key := normalizeModelName(modelName)
	modelCatalogMu.RLock()
	defer modelCatalogMu.RUnlock()
	spec, ok := modelCatalogIndex[key]
	return spec, ok
}

// StartModelCatalogRefresh launches the background refresh task (master node
// only). It loads once immediately, then refreshes on a 24h cadence.
func StartModelCatalogRefresh() {
	modelCatalogOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			ttl := catalogTTL()
			common.SysLog(fmt.Sprintf("model catalog refresh task started: cache ttl=%s", ttl))
			loadModelCatalog()
			ticker := time.NewTicker(ttl)
			defer ticker.Stop()
			for range ticker.C {
				loadModelCatalog()
			}
		})
	})
}
