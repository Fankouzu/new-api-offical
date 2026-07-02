package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	defaultOpenRouterImageMaxDownloadBytes = int64(32 << 20)
	defaultOpenRouterImageDownloadTimeout  = 20 * time.Second
)

type OpenRouterImageConvertOptions struct {
	HTTPClient        *http.Client
	MaxDownloadSize   int64
	Timeout           time.Duration
	AllowedHosts      []string
	AllowInsecureHTTP bool
	AllowPrivateHosts bool
}

func ConvertOpenRouterImageResponse(ctx context.Context, body []byte, options OpenRouterImageConvertOptions) ([]byte, error) {
	var payload dto.ImageResponse
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse image response: %w", err)
	}
	if len(payload.Data) == 0 {
		return body, nil
	}

	changed := false
	for index := range payload.Data {
		item := &payload.Data[index]
		if item.B64Json != "" {
			continue
		}
		if strings.TrimSpace(item.Url) == "" {
			continue
		}
		imageBytes, err := downloadOpenRouterImage(ctx, item.Url, options)
		if err != nil {
			return nil, err
		}
		item.B64Json = base64.StdEncoding.EncodeToString(imageBytes)
		item.Url = ""
		changed = true
	}
	if !changed {
		return body, nil
	}

	converted, err := common.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal converted image response: %w", err)
	}
	return converted, nil
}

func DefaultOpenRouterImageConvertOptions() OpenRouterImageConvertOptions {
	timeoutSeconds := common.GetEnvOrDefault("OPENROUTER_IMAGE_DOWNLOAD_TIMEOUT_SECONDS", int(defaultOpenRouterImageDownloadTimeout/time.Second))
	maxMB := common.GetEnvOrDefault("OPENROUTER_IMAGE_MAX_DOWNLOAD_MB", int(defaultOpenRouterImageMaxDownloadBytes>>20))
	return OpenRouterImageConvertOptions{
		HTTPClient:      GetHttpClient(),
		MaxDownloadSize: int64(maxMB) << 20,
		Timeout:         time.Duration(timeoutSeconds) * time.Second,
		AllowedHosts:    parseCSVEnv(common.GetEnvOrDefaultString("OPENROUTER_IMAGE_ALLOWED_HOSTS", "")),
	}
}

func downloadOpenRouterImage(ctx context.Context, rawURL string, options OpenRouterImageConvertOptions) ([]byte, error) {
	parsed, err := validateOpenRouterImageURL(rawURL, options)
	if err != nil {
		return nil, err
	}
	client := options.HTTPClient
	if client == nil {
		client = GetHttpClient()
		if client == nil {
			client = http.DefaultClient
		}
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultOpenRouterImageDownloadTimeout
	}
	maxBytes := options.MaxDownloadSize
	if maxBytes <= 0 {
		maxBytes = defaultOpenRouterImageMaxDownloadBytes
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create image download request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download image output: %w", err)
	}
	defer CloseResponseBodyGracefully(response)

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download image output failed with status %d", response.StatusCode)
	}
	if response.ContentLength > maxBytes {
		return nil, fmt.Errorf("image output exceeds %d bytes", maxBytes)
	}

	limited := io.LimitReader(response.Body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read image output: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("image output exceeds %d bytes", maxBytes)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("image output is empty")
	}
	return data, nil
}

func validateOpenRouterImageURL(rawURL string, options OpenRouterImageConvertOptions) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid image url: %w", err)
	}
	if parsed.Scheme != "https" && !(options.AllowInsecureHTTP && parsed.Scheme == "http") {
		return nil, errors.New("image url must use https")
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, errors.New("image url host is required")
	}
	if len(options.AllowedHosts) > 0 && !hostMatchesAllowedList(host, options.AllowedHosts) {
		return nil, fmt.Errorf("image url host %q is not allowed", host)
	}
	if !options.AllowPrivateHosts && isPrivateHostname(host) {
		return nil, fmt.Errorf("image url host %q resolves to private address", host)
	}
	return parsed, nil
}

func isPrivateHostname(host string) bool {
	ip := net.ParseIP(host)
	if ip != nil {
		return isPrivateIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, resolved := range ips {
		if isPrivateIP(resolved) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}

func hostMatchesAllowedList(host string, allowedHosts []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, allowed := range allowedHosts {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}
		if allowed == host {
			return true
		}
		if strings.HasPrefix(allowed, "*.") && strings.HasSuffix(host, strings.TrimPrefix(allowed, "*")) {
			return true
		}
	}
	return false
}

func parseCSVEnv(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '|'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
