package dto

type OpenRouterProviderModelsResponse struct {
	Data []OpenRouterProviderModel `json:"data"`
}

type OpenRouterProviderModel struct {
	ID                          string                      `json:"id"`
	HuggingFaceID               string                      `json:"hugging_face_id"`
	Name                        string                      `json:"name"`
	Created                     int64                       `json:"created"`
	InputModalities             []string                    `json:"input_modalities"`
	OutputModalities            []string                    `json:"output_modalities"`
	Quantization                string                      `json:"quantization"`
	ContextLength               int64                       `json:"context_length"`
	MaxOutputLength             int64                       `json:"max_output_length"`
	Pricing                     OpenRouterProviderPricing   `json:"pricing"`
	SupportedSamplingParameters []string                    `json:"supported_sampling_parameters"`
	SupportedFeatures           []string                    `json:"supported_features"`
	Description                 string                      `json:"description,omitempty"`
	DeprecationDate             string                      `json:"deprecation_date,omitempty"`
	IsReady                     bool                        `json:"is_ready"`
	IsFree                      bool                        `json:"is_free"`
	OpenRouter                  OpenRouterProviderSlug      `json:"openrouter"`
	Datacenters                 []OpenRouterProviderDataCtr `json:"datacenters,omitempty"`
}

type OpenRouterProviderPricing struct {
	Prompt         string `json:"prompt"`
	Completion     string `json:"completion"`
	Image          string `json:"image"`
	Request        string `json:"request"`
	InputCacheRead string `json:"input_cache_read"`
}

type OpenRouterProviderSlug struct {
	Slug string `json:"slug"`
}

type OpenRouterProviderDataCtr struct {
	CountryCode string `json:"country_code"`
}
