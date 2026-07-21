package model

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
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
	defaultModelCatalogURL  = "https://models.dev/api.json"
	modelCatalogTimeout     = 60 * time.Second
	modelCatalogDefaultTTLH = 24 // hours — cache for at least one day
	modelCatalogRetryDelay  = 5 * time.Minute
)

var (
	// dateSuffixRe strips trailing version-date suffixes that new-api appends to
	// model names (e.g. "doubao-seedance-2-0-260128", "claude-3-5-sonnet-20241022")
	// so they can match models.dev's canonical ids.
	dateSuffixRe = regexp.MustCompile(`-(?:\d{8}|\d{6})$`)

	modelCatalogOnce        sync.Once
	modelCatalogMu          sync.RWMutex
	modelCatalogIndex       map[string]*ModelCatalogSpec
	modelCatalogLoadedAt    time.Time
	modelCatalogLastAttempt time.Time
	modelCatalogRunning     atomic.Bool
)

// ModelCatalogSpec is the subset of models.dev metadata exposed on the pricing
// detail page. Field json tags intentionally match PricingModel in the frontend
// (web/default/src/features/pricing/types.ts) so values flow through unchanged.
type ModelCatalogSpec struct {
	ContextLength         int      `json:"context_length,omitempty"`
	MaxInputTokens        int      `json:"max_input_tokens,omitempty"`
	MaxOutputTokens       int      `json:"max_output_tokens,omitempty"`
	KnowledgeCutoff       string   `json:"knowledge_cutoff,omitempty"`
	ReleaseDate           string   `json:"release_date,omitempty"`
	InputModalities       []string `json:"input_modalities,omitempty"`
	OutputModalities      []string `json:"output_modalities,omitempty"`
	Capabilities          []string `json:"capabilities,omitempty"`
	OpenWeights           bool     `json:"open_weights,omitempty"`
	SupportsTemperature   *bool    `json:"supports_temperature,omitempty"`
	ReasoningEffortValues []string `json:"reasoning_effort_values,omitempty"`
}

// raw catalog JSON shapes (a subset of the models.dev schema we care about).
type rawCatalogProvider struct {
	Models map[string]rawCatalogModel `json:"models"`
}

type rawCatalogModel struct {
	Limit            *rawCatalogLimit            `json:"limit"`
	Knowledge        string                      `json:"knowledge"`
	ReleaseDate      string                      `json:"release_date"`
	Modalities       *rawCatalogMod              `json:"modalities"`
	ToolCall         bool                        `json:"tool_call"`
	StructuredOut    bool                        `json:"structured_output"`
	Reasoning        bool                        `json:"reasoning"`
	ReasoningOptions []rawCatalogReasoningOption `json:"reasoning_options"`
	Attachment       bool                        `json:"attachment"`
	OpenWeights      bool                        `json:"open_weights"`
	Temperature      *bool                       `json:"temperature"`
}

// rawCatalogReasoningOption mirrors models.dev's reasoning_options entries,
// e.g. {"type":"effort","values":["low","medium","high"]} or {"type":"toggle"}.
type rawCatalogReasoningOption struct {
	Type   string   `json:"type"`
	Values []string `json:"values"`
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
	return dateSuffixRe.ReplaceAllString(exactModelName(name), "")
}

func exactModelName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// catalogModalityAllowlist maps models.dev modality strings to the frontend
// Modality vocabulary (text/image/audio/video/file). Unknown values are dropped
// so the UI's icon lookup (MODALITY_META) never receives a key it has no entry
// for — models.dev sends e.g. "pdf" which would otherwise crash ModalityIcons.
var catalogModalityAllowlist = map[string]string{
	"text":     "text",
	"image":    "image",
	"audio":    "audio",
	"video":    "video",
	"file":     "file",
	"pdf":      "file",
	"document": "file",
}

// normalizeModalities maps/dedups modality values to the known vocabulary,
// dropping anything unrecognized.
func normalizeModalities(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, m := range in {
		mapped, ok := catalogModalityAllowlist[strings.ToLower(strings.TrimSpace(m))]
		if !ok {
			continue
		}
		if _, dup := seen[mapped]; dup {
			continue
		}
		seen[mapped] = struct{}{}
		out = append(out, mapped)
	}
	return out
}

func containsString(slice []string, v string) bool {
	for _, s := range slice {
		if s == v {
			return true
		}
	}
	return false
}

// extractReasoningEffort returns the effort values (e.g. ["low","medium","high"])
// from a models.dev reasoning_options list, or nil if it has no effort option.
func extractReasoningEffort(opts []rawCatalogReasoningOption) []string {
	for _, o := range opts {
		if strings.EqualFold(o.Type, "effort") && len(o.Values) > 0 {
			return o.Values
		}
	}
	return nil
}

// buildSpec maps a raw models.dev entry to the public spec, including a
// capabilities list expressed in the frontend's ModelCapability vocabulary.
func buildSpec(raw rawCatalogModel) *ModelCatalogSpec {
	spec := &ModelCatalogSpec{
		KnowledgeCutoff:       raw.Knowledge,
		ReleaseDate:           raw.ReleaseDate,
		OpenWeights:           raw.OpenWeights,
		SupportsTemperature:   raw.Temperature,
		ReasoningEffortValues: extractReasoningEffort(raw.ReasoningOptions),
	}
	if raw.Limit != nil {
		spec.ContextLength = raw.Limit.Context
		spec.MaxInputTokens = raw.Limit.Input
		spec.MaxOutputTokens = raw.Limit.Output
	}
	if raw.Modalities != nil {
		spec.InputModalities = normalizeModalities(raw.Modalities.Input)
		spec.OutputModalities = normalizeModalities(raw.Modalities.Output)
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
// providers, the alphabetically first provider wins so refreshes and process
// restarts resolve conflicts consistently.
func parseCatalog(data []byte) (map[string]*ModelCatalogSpec, error) {
	var providers map[string]rawCatalogProvider
	if err := common.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("failed to parse model catalog json: %w", err)
	}

	idx := make(map[string]*ModelCatalogSpec)
	aliasTargets := make(map[string]string)
	ambiguousAliases := make(map[string]struct{})
	providerNames := make([]string, 0, len(providers))
	for name := range providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)
	for _, providerName := range providerNames {
		provider := providers[providerName]
		modelIDs := make([]string, 0, len(provider.Models))
		for id := range provider.Models {
			modelIDs = append(modelIDs, id)
		}
		sort.Strings(modelIDs)
		for _, id := range modelIDs {
			raw := provider.Models[id]
			exactKey := exactModelName(id)
			if exactKey == "" {
				continue
			}
			if _, exists := idx[exactKey]; !exists {
				idx[exactKey] = buildSpec(raw)
			}
			alias := normalizeModelName(exactKey)
			if alias == exactKey {
				continue
			}
			if previous, exists := aliasTargets[alias]; !exists {
				aliasTargets[alias] = exactKey
			} else if previous != exactKey {
				delete(aliasTargets, alias)
				ambiguousAliases[alias] = struct{}{}
			}
		}
	}
	for alias, target := range aliasTargets {
		if _, ambiguous := ambiguousAliases[alias]; ambiguous {
			continue
		}
		if _, exactExists := idx[alias]; exactExists {
			continue
		}
		idx[alias] = idx[target]
	}
	if len(idx) == 0 {
		return nil, fmt.Errorf("model catalog contains no models")
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
	InvalidatePricingCache()
	common.SysLog(fmt.Sprintf("model catalog loaded: %d models", len(idx)))
}

// triggerModelCatalogLoad schedules at most one refresh. Lookups continue with
// the current (possibly empty or stale) snapshot instead of waiting on the
// external catalog. Failed attempts are cooled down so a pricing rebuild cannot
// retry once per model.
func triggerModelCatalogLoad(force bool) {
	modelCatalogMu.RLock()
	fresh := len(modelCatalogIndex) > 0 && time.Since(modelCatalogLoadedAt) < catalogTTL()
	inCooldown := !modelCatalogLastAttempt.IsZero() && time.Since(modelCatalogLastAttempt) < modelCatalogRetryDelay
	modelCatalogMu.RUnlock()
	if !force && (fresh || inCooldown) {
		return
	}
	if !modelCatalogRunning.CompareAndSwap(false, true) {
		return
	}
	modelCatalogMu.Lock()
	modelCatalogLastAttempt = time.Now()
	modelCatalogMu.Unlock()
	gopool.Go(func() {
		defer modelCatalogRunning.Store(false)
		loadModelCatalog()
	})
}

// GetModelCatalogSpec looks up the spec for a model name. Returns ok=false when
// the model is absent from the catalog (caller leaves the field empty).
func GetModelCatalogSpec(modelName string) (*ModelCatalogSpec, bool) {
	if strings.TrimSpace(modelName) == "" {
		return nil, false
	}
	triggerModelCatalogLoad(false)
	exactKey := exactModelName(modelName)
	modelCatalogMu.RLock()
	defer modelCatalogMu.RUnlock()
	spec, ok := modelCatalogIndex[exactKey]
	if !ok {
		spec, ok = modelCatalogIndex[normalizeModelName(exactKey)]
	}
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
			triggerModelCatalogLoad(false)
			ticker := time.NewTicker(ttl)
			defer ticker.Stop()
			for range ticker.C {
				triggerModelCatalogLoad(true)
			}
		})
	})
}
