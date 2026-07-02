package service

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenRouterImageRequestMapsGenerationFields(t *testing.T) {
	req := OpenRouterImageRequest{
		Model:        "z-image-turbo-2k",
		Prompt:       "lychee product photo",
		AspectRatio:  "16:9",
		OutputFormat: json.RawMessage(`"png"`),
		N:            common.GetPointer(uint(1)),
	}

	got, mode, err := ConvertOpenRouterImageRequest(req)

	require.NoError(t, err)
	require.Equal(t, OpenRouterImageGenerationMode, mode)
	require.Equal(t, "z-image-turbo-2k", got.Model)
	require.Equal(t, "lychee product photo", got.Prompt)
	require.Equal(t, "1536x864", got.Size)
	require.Equal(t, `"png"`, string(got.OutputFormat))
	require.Contains(t, string(got.ExtraFields), `"parameters"`)
}

func TestConvertOpenRouterImageRequestRejectsMultipleImages(t *testing.T) {
	req := OpenRouterImageRequest{
		Model:  "z-image-turbo",
		Prompt: "one image",
		N:      common.GetPointer(uint(2)),
	}

	_, _, err := ConvertOpenRouterImageRequest(req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "n=1")
}

func TestConvertOpenRouterImageRequestMapsEditReference(t *testing.T) {
	req := OpenRouterImageRequest{
		Model:  "Qwen-Image-Edit-2k",
		Prompt: "make it red",
		InputReferences: []OpenRouterImageInputReference{{
			ImageURL: &OpenRouterImageURL{URL: "https://example.com/input.png"},
		}},
		Provider: &OpenRouterImageProviderOptions{
			Options: map[string]any{
				"lizh-ai": map[string]any{
					"guidance_scale": 7.5,
				},
			},
		},
	}

	got, mode, err := ConvertOpenRouterImageRequest(req)

	require.NoError(t, err)
	require.Equal(t, OpenRouterImageEditMode, mode)
	require.Equal(t, "Qwen-Image-Edit-2k", got.Model)
	require.Equal(t, "make it red", got.Prompt)
	require.Equal(t, `"https://example.com/input.png"`, string(got.Image))
	require.Contains(t, string(got.ExtraFields), `"guidance_scale"`)
	require.Contains(t, string(got.ExtraFields), `"input"`)
}

func TestConvertOpenRouterImageRequestRejectsEditWithoutReference(t *testing.T) {
	req := OpenRouterImageRequest{
		Model:  "Qwen-Image-Edit",
		Prompt: "make it red",
	}

	_, _, err := ConvertOpenRouterImageRequest(req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "input_references")
}

func TestConvertOpenRouterImageRequestExtraFieldsCanHydrateImageRequestExtra(t *testing.T) {
	req := OpenRouterImageRequest{
		Model:  "z-image-turbo",
		Prompt: "lychee",
		Provider: &OpenRouterImageProviderOptions{
			Options: map[string]any{
				"lizh-ai": map[string]any{
					"prompt_extend": true,
				},
			},
		},
	}
	got, _, err := ConvertOpenRouterImageRequest(req)
	require.NoError(t, err)

	encoded, err := common.Marshal(got)
	require.NoError(t, err)

	var decoded struct {
		ExtraFields map[string]json.RawMessage `json:"extra_fields"`
	}
	require.NoError(t, common.Unmarshal(encoded, &decoded))
	require.Contains(t, decoded.ExtraFields, "parameters")
}
