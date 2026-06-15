package tencentvod

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const defaultBaseURL = "https://vod.tencentcloudapi.com"

type config struct {
	SecretID  string
	SecretKey string
	SubAppID  int64
	Region    string
}

type keyJSON struct {
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
	SubAppID  int64  `json:"sub_app_id"`
}

func parseConfig(key, region string) (config, error) {
	cfg := config{Region: strings.TrimSpace(region)}
	if cfg.Region == "" {
		return cfg, fmt.Errorf("X-TC-Region is required")
	}

	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "{") {
		var parsed keyJSON
		if err := common.Unmarshal([]byte(key), &parsed); err != nil {
			return cfg, fmt.Errorf("invalid api key JSON: %w", err)
		}
		cfg.SecretID = strings.TrimSpace(parsed.SecretID)
		cfg.SecretKey = strings.TrimSpace(parsed.SecretKey)
		cfg.SubAppID = parsed.SubAppID
	} else {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			return cfg, fmt.Errorf("api key must be SecretId|SecretKey|SubAppId")
		}
		subAppID, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("invalid SubAppId: %w", err)
		}
		cfg.SecretID = strings.TrimSpace(parts[0])
		cfg.SecretKey = strings.TrimSpace(parts[1])
		cfg.SubAppID = subAppID
	}

	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.SubAppID <= 0 {
		return cfg, fmt.Errorf("SecretId, SecretKey and SubAppId are required")
	}
	return cfg, nil
}
