package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenRouterImageResponseDownloadsURLAsBase64(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/image.png", r.URL.Path)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer upstream.Close()

	body, err := common.Marshal(dto.ImageResponse{
		Created: 1782978351,
		Data: []dto.ImageData{{
			Url:           upstream.URL + "/image.png",
			RevisedPrompt: "clean prompt",
		}},
	})
	require.NoError(t, err)

	got, err := ConvertOpenRouterImageResponse(context.Background(), body, OpenRouterImageConvertOptions{
		HTTPClient:        upstream.Client(),
		MaxDownloadSize:   1024,
		AllowInsecureHTTP: true,
		AllowPrivateHosts: true,
	})

	require.NoError(t, err)
	var payload dto.ImageResponse
	require.NoError(t, common.Unmarshal(got, &payload))
	require.Equal(t, int64(1782978351), payload.Created)
	require.Len(t, payload.Data, 1)
	require.Empty(t, payload.Data[0].Url)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("png-bytes")), payload.Data[0].B64Json)
	require.Equal(t, "clean prompt", payload.Data[0].RevisedPrompt)
}

func TestConvertOpenRouterImageResponseKeepsExistingBase64(t *testing.T) {
	body, err := common.Marshal(dto.ImageResponse{
		Data: []dto.ImageData{{
			B64Json: "already-base64",
		}},
	})
	require.NoError(t, err)

	got, err := ConvertOpenRouterImageResponse(context.Background(), body, OpenRouterImageConvertOptions{})

	require.NoError(t, err)
	require.JSONEq(t, string(body), string(got))
}

func TestConvertOpenRouterImageResponseRejectsPrivateImageURL(t *testing.T) {
	body, err := common.Marshal(dto.ImageResponse{
		Data: []dto.ImageData{{
			Url: "https://127.0.0.1/image.png",
		}},
	})
	require.NoError(t, err)

	_, err = ConvertOpenRouterImageResponse(context.Background(), body, OpenRouterImageConvertOptions{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestConvertOpenRouterImageResponseRejectsSpecialRangeImageURL(t *testing.T) {
	_, err := validateOpenRouterImageURL("https://100.64.0.1/image.png", OpenRouterImageConvertOptions{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestConvertOpenRouterImageResponseValidatesRedirectPolicy(t *testing.T) {
	var redirected bool
	redirectTarget := "https://example.com/image.png"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected = true
		http.Redirect(w, r, redirectTarget, http.StatusFound)
	}))
	defer upstream.Close()

	body, err := common.Marshal(dto.ImageResponse{
		Data: []dto.ImageData{{
			Url: upstream.URL + "/image.png",
		}},
	})
	require.NoError(t, err)

	_, err = ConvertOpenRouterImageResponse(context.Background(), body, OpenRouterImageConvertOptions{
		HTTPClient:        upstream.Client(),
		MaxDownloadSize:   1024,
		AllowedHosts:      []string{mustURLHost(t, upstream.URL)},
		AllowInsecureHTTP: true,
		AllowPrivateHosts: true,
	})

	require.True(t, redirected)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("image url host %q is not allowed", "example.com"))
}

func TestConvertOpenRouterImageResponseEnforcesMaxDownloadSize(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("too-large"))
	}))
	defer upstream.Close()

	body, err := common.Marshal(dto.ImageResponse{
		Data: []dto.ImageData{{
			Url: upstream.URL,
		}},
	})
	require.NoError(t, err)

	_, err = ConvertOpenRouterImageResponse(context.Background(), body, OpenRouterImageConvertOptions{
		HTTPClient:        upstream.Client(),
		MaxDownloadSize:   4,
		AllowInsecureHTTP: true,
		AllowPrivateHosts: true,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds")
}

func mustURLHost(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := http.NewRequest(http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	return parsed.URL.Hostname()
}
