package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// R2Config はCloudflare R2のS3互換APIへ接続するための設定です。
type R2Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Secure          bool
}

// LoadR2ConfigFromEnv はR2エクスポート専用の接続情報を環境変数から読み込みます。
// 秘密情報を設定ファイルへ保存しないため、JSON設定ファイルからは読み込みません。
func LoadR2ConfigFromEnv() (R2Config, error) {
	endpointURL := strings.TrimSpace(os.Getenv("R2_ENDPOINT_URL"))
	accessKeyID := strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID"))
	secretAccessKey := strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY"))
	bucketName := strings.TrimSpace(os.Getenv("R2_BUCKET_NAME"))

	var validationErrors []string
	if endpointURL == "" {
		validationErrors = append(validationErrors, "R2_ENDPOINT_URL environment variable is required")
	}
	if accessKeyID == "" {
		validationErrors = append(validationErrors, "R2_ACCESS_KEY_ID environment variable is required")
	}
	if secretAccessKey == "" {
		validationErrors = append(validationErrors, "R2_SECRET_ACCESS_KEY environment variable is required")
	}
	if bucketName == "" {
		validationErrors = append(validationErrors, "R2_BUCKET_NAME environment variable is required")
	}

	var endpoint string
	if endpointURL != "" {
		parsedURL, err := url.ParseRequestURI(endpointURL)
		if err != nil {
			validationErrors = append(validationErrors, "R2_ENDPOINT_URL must be a valid HTTPS URL")
		} else {
			switch {
			case parsedURL.Scheme != "https":
				validationErrors = append(validationErrors, "R2_ENDPOINT_URL must use HTTPS")
			case parsedURL.Host == "":
				validationErrors = append(validationErrors, "R2_ENDPOINT_URL must include a host")
			case parsedURL.User != nil || (parsedURL.Path != "" && parsedURL.Path != "/") || parsedURL.RawQuery != "" || parsedURL.Fragment != "":
				validationErrors = append(validationErrors, "R2_ENDPOINT_URL must not include credentials, a path, query, or fragment")
			default:
				endpoint = parsedURL.Host
			}
		}
	}

	if len(validationErrors) > 0 {
		return R2Config{}, fmt.Errorf("R2 configuration validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return R2Config{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		BucketName:      bucketName,
		Secure:          true,
	}, nil
}
