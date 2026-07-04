package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenRouterProviderModelsConvertsTextPricingAndMetadata(t *testing.T) {
	items := []model.Pricing{{
		ModelName:              "acme/test-chat",
		Description:            "A test model",
		Tags:                   "or:hf=acme/test-chat-hf,or:name=Acme Test Chat,or:context=128000,or:max_output=8192,or:quantization=fp16,or:dc=US,or:feature=tools",
		QuotaType:              0,
		ModelRatio:             0.001,
		CompletionRatio:        2,
		SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		EnableGroup:            []string{"default"},
	}}

	result := BuildOpenRouterProviderModels(items)

	require.Len(t, result, 1)
	got := result[0]
	require.Equal(t, "acme/test-chat", got.ID)
	require.Equal(t, "acme/test-chat-hf", got.HuggingFaceID)
	require.Equal(t, "Acme Test Chat", got.Name)
	require.Equal(t, int64(128000), got.ContextLength)
	require.Equal(t, int64(8192), got.MaxOutputLength)
	require.Equal(t, "fp16", got.Quantization)
	require.Equal(t, []string{"text"}, got.InputModalities)
	require.Equal(t, []string{"text"}, got.OutputModalities)
	require.Contains(t, got.SupportedSamplingParameters, "temperature")
	require.Contains(t, got.SupportedFeatures, "tools")
	require.Equal(t, "0.000000002", got.Pricing.Prompt)
	require.Equal(t, "0.000000004", got.Pricing.Completion)
	require.Equal(t, "0", got.Pricing.Image)
	require.Equal(t, "0", got.Pricing.Request)
	require.Equal(t, "0", got.Pricing.InputCacheRead)
	require.True(t, got.IsReady)
	require.False(t, got.IsFree)
	require.Equal(t, "acme/test-chat", got.OpenRouter.Slug)
	require.Len(t, got.Datacenters, 1)
	require.Equal(t, "US", got.Datacenters[0].CountryCode)
}

func TestBuildOpenRouterProviderModelsMarksFreeModels(t *testing.T) {
	items := []model.Pricing{{
		ModelName:              "acme/free-chat",
		QuotaType:              0,
		ModelRatio:             0,
		CompletionRatio:        1,
		SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
		EnableGroup:            []string{"default"},
	}}

	result := BuildOpenRouterProviderModels(items)

	require.Len(t, result, 1)
	require.True(t, result[0].IsFree)
	require.Equal(t, "0", result[0].Pricing.Prompt)
	require.Equal(t, "0", result[0].Pricing.Completion)
}

func TestBuildOpenRouterProviderModelsHonorsAllowlist(t *testing.T) {
	t.Setenv("OPENROUTER_PROVIDER_MODEL_ALLOWLIST", "z-image-turbo-2k")
	items := []model.Pricing{
		{
			ModelName:              "z-image-turbo-2k",
			QuotaType:              1,
			ModelPrice:             0.01,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeImageGeneration},
			EnableGroup:            []string{"default"},
		},
		{
			ModelName:              "acme/chat",
			QuotaType:              0,
			ModelRatio:             0.001,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
			EnableGroup:            []string{"default"},
		},
	}

	result := BuildOpenRouterProviderModels(items)

	require.Len(t, result, 1)
	require.Equal(t, "z-image-turbo-2k", result[0].ID)
}

func TestBuildOpenRouterProviderImageModelsOnlyIncludesImageModels(t *testing.T) {
	imageRatio := 0.01
	items := []model.Pricing{
		{
			ModelName:              "z-image-turbo-2k",
			Description:            "2K image model",
			Tags:                   "or:name=Z Image Turbo 2K",
			QuotaType:              1,
			ModelPrice:             0.01,
			ImageRatio:             &imageRatio,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeImageGeneration},
			EnableGroup:            []string{"default"},
		},
		{
			ModelName:              "acme/chat",
			QuotaType:              0,
			ModelRatio:             0.001,
			SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeOpenAI},
			EnableGroup:            []string{"default"},
		},
	}

	result := BuildOpenRouterProviderImageModels(items)

	require.Len(t, result, 1)
	require.Equal(t, "z-image-turbo-2k", result[0].ID)
	require.Equal(t, []string{"text", "image"}, result[0].InputModalities)
	require.Equal(t, []string{"image"}, result[0].OutputModalities)
	require.Contains(t, openRouterSupportedParameterNames(result[0].SupportedParameters), "input_references")
	require.Contains(t, openRouterSupportedParameterNames(result[0].SupportedParameters), "provider.options")
	require.Len(t, result[0].Endpoints, 1)
	require.Equal(t, "lizh-ai", result[0].Endpoints[0].Tag)
}

func openRouterSupportedParameterNames(parameters []dto.OpenRouterSupportedParameter) []string {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		names = append(names, parameter.Name)
	}
	return names
}
