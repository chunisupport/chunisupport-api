package config

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/info"
)

// CloudflareCacheConfig はS3認証とキャッシュパージ権限を分離するため、専用の認証情報を保持します。
type CloudflareCacheConfig struct {
	APIToken      string
	ZoneID        string
	PublicBaseURL string
}

// LoadCloudflareCacheConfigFromEnv はAPIトークンを設定ファイルへ保存させないため、環境変数だけを参照します。
func LoadCloudflareCacheConfigFromEnv() (CloudflareCacheConfig, error) {
	apiToken := strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN"))
	zoneID := strings.TrimSpace(os.Getenv("CLOUDFLARE_ZONE_ID"))
	publicBaseURL := strings.TrimSpace(os.Getenv("STATIC_DATA_PUBLIC_BASE_URL"))

	var validationErrors []string
	if apiToken == "" {
		validationErrors = append(validationErrors, "CLOUDFLARE_API_TOKEN environment variable is required")
	}
	if zoneID == "" {
		validationErrors = append(validationErrors, "CLOUDFLARE_ZONE_ID environment variable is required")
	} else if decodedZoneID, err := hex.DecodeString(zoneID); err != nil || len(decodedZoneID) != 16 {
		validationErrors = append(validationErrors, "CLOUDFLARE_ZONE_ID must be a 32-character hexadecimal ID")
	}

	normalizedBaseURL := ""
	if publicBaseURL == "" {
		validationErrors = append(validationErrors, "STATIC_DATA_PUBLIC_BASE_URL environment variable is required")
	} else {
		parsedURL, err := url.ParseRequestURI(publicBaseURL)
		if err != nil {
			validationErrors = append(validationErrors, "STATIC_DATA_PUBLIC_BASE_URL must be a valid HTTPS URL")
		} else {
			switch {
			case parsedURL.Scheme != "https":
				validationErrors = append(validationErrors, "STATIC_DATA_PUBLIC_BASE_URL must use HTTPS")
			case parsedURL.Host == "":
				validationErrors = append(validationErrors, "STATIC_DATA_PUBLIC_BASE_URL must include a host")
			case parsedURL.User != nil || (parsedURL.Path != "" && parsedURL.Path != "/") || parsedURL.RawQuery != "" || parsedURL.Fragment != "":
				validationErrors = append(validationErrors, "STATIC_DATA_PUBLIC_BASE_URL must not include credentials, a path, query, or fragment")
			default:
				normalizedBaseURL = "https://" + parsedURL.Host
			}
		}
	}

	if len(validationErrors) > 0 {
		return CloudflareCacheConfig{}, fmt.Errorf("cloudflare cache configuration validation failed: %s", strings.Join(validationErrors, "; "))
	}
	allowedDestinations := map[string]string{
		info.DevelopStaticDataPublicBaseURL:    info.DevelopCloudflareZoneID,
		info.BetaStaticDataPublicBaseURL:       info.BetaCloudflareZoneID,
		info.ProductionStaticDataPublicBaseURL: info.ProductionCloudflareZoneID,
	}
	if expectedZoneID, ok := allowedDestinations[normalizedBaseURL]; !ok || expectedZoneID != zoneID {
		return CloudflareCacheConfig{}, fmt.Errorf("cloudflare cache configuration validation failed: CLOUDFLARE_ZONE_ID and STATIC_DATA_PUBLIC_BASE_URL must match a configured deployment destination")
	}

	return CloudflareCacheConfig{
		APIToken:      apiToken,
		ZoneID:        zoneID,
		PublicBaseURL: normalizedBaseURL,
	}, nil
}
