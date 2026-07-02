package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestImageRequestUnmarshalHydratesExtraFromExtraFields(t *testing.T) {
	var request ImageRequest

	err := common.Unmarshal([]byte(`{
		"model":"z-image-turbo",
		"prompt":"lychee",
		"extra_fields":{
			"parameters":{"size":"1024x1024"},
			"input":{"prompt":"lychee"}
		}
	}`), &request)

	require.NoError(t, err)
	require.Contains(t, request.Extra, "parameters")
	require.Contains(t, request.Extra, "input")
}
