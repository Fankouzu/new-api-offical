package model_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSyncImageModelMatchesCaseInsensitively(t *testing.T) {
	require.True(t, IsSyncImageModel("Qwen-Image-2512-2k"))
	require.True(t, IsSyncImageModel("Qwen-Image-Edit-2k"))
}
