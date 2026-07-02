package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsImageGenerationModelIncludesQwenAndZImageModels(t *testing.T) {
	require.True(t, IsImageGenerationModel("z-image-turbo-2k"))
	require.True(t, IsImageGenerationModel("Qwen-Image-2512-2k"))
}
