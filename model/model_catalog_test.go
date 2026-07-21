package model

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func TestNormalizeModelName(t *testing.T) {
	cases := map[string]string{
		"doubao-seedance-2-0-260128": "doubao-seedance-2-0",
		"Claude-3-5-Sonnet-20241022": "claude-3-5-sonnet",
		"gpt-4o":                     "gpt-4o",
		"deepseek-chat":              "deepseek-chat",
		"  Qwen-Max  ":               "qwen-max",
		"gemini-2.0-flash-001":       "gemini-2.0-flash-001", // 3 digits are not a date suffix
		"claude-3-7-sonnet-20250219": "claude-3-7-sonnet",
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
					"reasoning_options": [
						{"type": "effort", "values": ["high", "max"]},
						{"type": "toggle"}
					],
					"temperature": false,
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
	wantCaps := []string{"tools", "function_calling", "structured_output", "json_mode", "vision"}
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

	// temperature signal: gpt-4o supports it, doubao-seed-1.6 does not.
	if spec.SupportsTemperature == nil || !*spec.SupportsTemperature {
		t.Errorf("gpt-4o supports_temperature = %v, want true", spec.SupportsTemperature)
	}
	if db.SupportsTemperature == nil || *db.SupportsTemperature {
		t.Errorf("doubao-seed-1.6 supports_temperature = %v, want false", db.SupportsTemperature)
	}
	// reasoning effort values extracted from reasoning_options (effort type only).
	if !reflect.DeepEqual(db.ReasoningEffortValues, []string{"high", "max"}) {
		t.Errorf("doubao-seed-1.6 reasoning_effort_values = %v, want [high max]", db.ReasoningEffortValues)
	}
	if len(spec.ReasoningEffortValues) != 0 {
		t.Errorf("gpt-4o reasoning_effort_values = %v, want empty", spec.ReasoningEffortValues)
	}
}

func TestNormalizeModalities(t *testing.T) {
	// "pdf"/"document" map to the frontend's "file"; unknown values drop;
	// duplicates collapse. This guards the UI's MODALITY_META icon lookup.
	got := normalizeModalities([]string{"text", "image", "pdf", "PDF", "bogus", "audio", "document"})
	want := []string{"text", "image", "file", "audio"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeModalities = %v, want %v", got, want)
	}
}

func TestParseCatalogInvalidJSON(t *testing.T) {
	if _, err := parseCatalog([]byte("{not json")); err == nil {
		t.Fatalf("expected error for invalid json, got nil")
	}
}

func TestParseCatalogRejectsEmptyCatalog(t *testing.T) {
	if _, err := parseCatalog([]byte(`{}`)); err == nil {
		t.Fatal("expected an empty catalog to be rejected")
	}
}

func TestParseCatalogPreservesUnknownTemperature(t *testing.T) {
	idx, err := parseCatalog([]byte(`{
		"provider": {"models": {"model-without-temperature": {"limit": {"context": 8192}}}}
	}`))
	if err != nil {
		t.Fatalf("parseCatalog returned error: %v", err)
	}
	if got := idx["model-without-temperature"].SupportsTemperature; got != nil {
		t.Fatalf("supports_temperature = %v, want nil for a missing source field", *got)
	}
}

func TestBuildSpecDoesNotTreatAttachmentsAsVision(t *testing.T) {
	spec := buildSpec(rawCatalogModel{
		Attachment: true,
		Modalities: &rawCatalogMod{
			Input:  []string{"text", "pdf"},
			Output: []string{"text"},
		},
	})

	if contains(spec.Capabilities, "vision") {
		t.Fatalf("attachment-only model capabilities = %v, want no vision", spec.Capabilities)
	}
	if !reflect.DeepEqual(spec.InputModalities, []string{"text", "file"}) {
		t.Fatalf("attachment-only input modalities = %v, want [text file]", spec.InputModalities)
	}
}

func TestParseCatalogUsesStableProviderPrecedence(t *testing.T) {
	data := []byte(`{
		"z-provider": {"models": {"shared-model": {"limit": {"context": 200}}}},
		"a-provider": {"models": {"shared-model": {"limit": {"context": 100}}}}
	}`)
	for i := 0; i < 100; i++ {
		idx, err := parseCatalog(data)
		if err != nil {
			t.Fatalf("parseCatalog returned error: %v", err)
		}
		if got := idx["shared-model"].ContextLength; got != 100 {
			t.Fatalf("iteration %d: context = %d, want alphabetically first provider value 100", i, got)
		}
	}
}

func TestParseCatalogPreservesExactDatedModelsAndRejectsAmbiguousAlias(t *testing.T) {
	data := []byte(`{
		"provider": {"models": {
			"seed-1-6-250615": {"release_date": "2025-06-25", "limit": {"output": 8192}},
			"seed-1-6-250915": {"release_date": "2025-09-15", "limit": {"output": 32768}},
			"single-version-250101": {"release_date": "2025-01-01", "limit": {"output": 4096}}
		}}
	}`)
	idx, err := parseCatalog(data)
	if err != nil {
		t.Fatalf("parseCatalog returned error: %v", err)
	}
	if got := idx["seed-1-6-250615"]; got == nil || got.MaxOutputTokens != 8192 {
		t.Fatalf("first exact dated model = %#v, want output 8192", got)
	}
	if got := idx["seed-1-6-250915"]; got == nil || got.MaxOutputTokens != 32768 {
		t.Fatalf("second exact dated model = %#v, want output 32768", got)
	}
	if _, ok := idx["seed-1-6"]; ok {
		t.Fatal("ambiguous date-stripped alias should not be indexed")
	}
	if got := idx["single-version"]; got == nil || got.MaxOutputTokens != 4096 {
		t.Fatalf("unambiguous alias = %#v, want output 4096", got)
	}
}

func TestCatalogFailureIsNotRetriedForEveryLookup(t *testing.T) {
	var requests atomic.Int32
	requestSeen := make(chan struct{}, 1)
	releaseRequest := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		select {
		case requestSeen <- struct{}{}:
		default:
		}
		<-releaseRequest
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	t.Setenv("MODEL_CATALOG_URL", server.URL)

	modelCatalogMu.Lock()
	oldIndex := modelCatalogIndex
	oldLoadedAt := modelCatalogLoadedAt
	oldLastAttempt := modelCatalogLastAttempt
	modelCatalogIndex = nil
	modelCatalogLoadedAt = time.Time{}
	modelCatalogLastAttempt = time.Time{}
	modelCatalogMu.Unlock()
	modelCatalogRunning.Store(false)
	t.Cleanup(func() {
		modelCatalogMu.Lock()
		modelCatalogIndex = oldIndex
		modelCatalogLoadedAt = oldLoadedAt
		modelCatalogLastAttempt = oldLastAttempt
		modelCatalogMu.Unlock()
		modelCatalogRunning.Store(false)
	})

	lookupDone := make(chan bool, 1)
	go func() {
		_, ok := GetModelCatalogSpec("first-model")
		lookupDone <- ok
	}()
	select {
	case ok := <-lookupDone:
		if ok {
			t.Fatal("unexpected catalog hit")
		}
	case <-time.After(500 * time.Millisecond):
		close(releaseRequest)
		t.Fatal("catalog lookup blocked on the external refresh")
	}
	select {
	case <-requestSeen:
	case <-time.After(2 * time.Second):
		t.Fatal("catalog load did not start")
	}
	close(releaseRequest)
	for modelCatalogRunning.Load() {
		time.Sleep(time.Millisecond)
	}

	for _, name := range []string{"second-model", "third-model", "fourth-model"} {
		if _, ok := GetModelCatalogSpec(name); ok {
			t.Fatalf("unexpected catalog hit for %s", name)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("catalog requests = %d, want 1 during the failure cooldown", got)
	}
}

func TestSuccessfulCatalogLoadMarksPricingCacheStaleWithoutDiscardingSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"provider": {"models": {"test-model": {"limit": {"context": 8192}}}}
		}`))
	}))
	defer server.Close()
	t.Setenv("MODEL_CATALOG_URL", server.URL)

	updatePricingLock.Lock()
	oldPricingMap := pricingMap
	oldVendorsList := vendorsList
	oldLastGetPricingTime := lastGetPricingTime
	pricingMap = []Pricing{{ModelName: "cached-without-catalog"}}
	vendorsList = []PricingVendor{{ID: 1}}
	lastGetPricingTime = time.Now()
	updatePricingLock.Unlock()
	t.Cleanup(func() {
		updatePricingLock.Lock()
		pricingMap = oldPricingMap
		vendorsList = oldVendorsList
		lastGetPricingTime = oldLastGetPricingTime
		updatePricingLock.Unlock()
	})

	loadModelCatalog()

	updatePricingLock.Lock()
	defer updatePricingLock.Unlock()
	if len(pricingMap) != 1 || pricingMap[0].ModelName != "cached-without-catalog" || len(vendorsList) != 1 || !lastGetPricingTime.IsZero() {
		t.Fatalf("last published pricing snapshot was not preserved as stale: pricing=%v vendors=%v loadedAt=%v", pricingMap, vendorsList, lastGetPricingTime)
	}
}

func TestPricingCacheConcurrentReadAndPublish(t *testing.T) {
	updatePricingLock.Lock()
	oldPricingMap := pricingMap
	oldVendorsList := vendorsList
	oldLastGetPricingTime := lastGetPricingTime
	pricingMap = []Pricing{{ModelName: "cached"}}
	vendorsList = []PricingVendor{{ID: 1}}
	lastGetPricingTime = time.Now()
	updatePricingLock.Unlock()
	t.Cleanup(func() {
		updatePricingLock.Lock()
		pricingMap = oldPricingMap
		vendorsList = oldVendorsList
		lastGetPricingTime = oldLastGetPricingTime
		updatePricingLock.Unlock()
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			updatePricingLock.Lock()
			vendorsList = []PricingVendor{{ID: i + 1}}
			lastGetPricingTime = time.Now()
			updatePricingLock.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = GetVendors()
		}
	}()
	wg.Wait()
}

func TestGetPricingSnapshotReturnsOnePublishedVersion(t *testing.T) {
	updatePricingLock.Lock()
	oldPricingMap := pricingMap
	oldVendorsList := vendorsList
	oldSupportedEndpointMap := supportedEndpointMap
	oldLastGetPricingTime := lastGetPricingTime
	pricingMap = []Pricing{{ModelName: "version-1"}}
	vendorsList = []PricingVendor{{ID: 1}}
	supportedEndpointMap = map[string]common.EndpointInfo{"version": {Path: "/1"}}
	lastGetPricingTime = time.Now()
	updatePricingLock.Unlock()
	t.Cleanup(func() {
		updatePricingLock.Lock()
		pricingMap = oldPricingMap
		vendorsList = oldVendorsList
		supportedEndpointMap = oldSupportedEndpointMap
		lastGetPricingTime = oldLastGetPricingTime
		updatePricingLock.Unlock()
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 1; i <= 1000; i++ {
			updatePricingLock.Lock()
			pricingMap = []Pricing{{ModelName: fmt.Sprintf("version-%d", i)}}
			vendorsList = []PricingVendor{{ID: i}}
			supportedEndpointMap = map[string]common.EndpointInfo{
				"version": {Path: fmt.Sprintf("/%d", i)},
			}
			lastGetPricingTime = time.Now()
			updatePricingLock.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			snapshot := GetPricingSnapshot()
			if len(snapshot.Pricing) != 1 || len(snapshot.Vendors) != 1 {
				t.Errorf("incomplete snapshot: %+v", snapshot)
				return
			}
			wantModel := fmt.Sprintf("version-%d", snapshot.Vendors[0].ID)
			wantPath := fmt.Sprintf("/%d", snapshot.Vendors[0].ID)
			if snapshot.Pricing[0].ModelName != wantModel || snapshot.SupportedEndpoints["version"].Path != wantPath {
				t.Errorf("mixed snapshot: %+v", snapshot)
				return
			}
		}
	}()
	wg.Wait()
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
