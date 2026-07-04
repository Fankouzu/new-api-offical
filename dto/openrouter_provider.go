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
	SupportedParameters         []string                    `json:"supported_parameters,omitempty"`
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

type OpenRouterImageModelsResponse struct {
	Data []OpenRouterImageModel `json:"data"`
}

type OpenRouterImageModel struct {
	ID                  string                         `json:"id"`
	Name                string                         `json:"name"`
	Description         string                         `json:"description,omitempty"`
	Created             int64                          `json:"created"`
	Architecture        OpenRouterImageArchitecture    `json:"architecture"`
	InputModalities     []string                       `json:"input_modalities"`
	OutputModalities    []string                       `json:"output_modalities"`
	Pricing             OpenRouterProviderPricing      `json:"pricing"`
	SupportedParameters []OpenRouterSupportedParameter `json:"supported_parameters"`
	SupportsStreaming   bool                           `json:"supports_streaming"`
	IsReady             bool                           `json:"is_ready"`
	Endpoints           []OpenRouterImageModelEndpoint `json:"endpoints,omitempty"`
}

type OpenRouterImageModelEndpointsResponse struct {
	Endpoints []OpenRouterImageModelEndpoint `json:"endpoints"`
}

type OpenRouterImageModelEndpoint struct {
	Name                string                         `json:"name"`
	ContextLength       int64                          `json:"context_length,omitempty"`
	Pricing             OpenRouterProviderPricing      `json:"pricing"`
	ProviderName        string                         `json:"provider_name"`
	Tag                 string                         `json:"tag"`
	MaxCompletionTokens int64                          `json:"max_completion_tokens,omitempty"`
	SupportedParameters []OpenRouterSupportedParameter `json:"supported_parameters"`
	Status              string                         `json:"status"`
	SupportsStreaming   bool                           `json:"supports_streaming"`
}

type OpenRouterImageArchitecture struct {
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
}

type OpenRouterSupportedParameter struct {
	Name     string   `json:"name"`
	Type     string   `json:"type,omitempty"`
	Required bool     `json:"required,omitempty"`
	Values   []string `json:"values,omitempty"`
}
