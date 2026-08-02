package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ObjectStorageConfig はオブジェクトストレージへ接続するための設定です。
type ObjectStorageConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Secure          bool
}

// LoadObjectStorageConfigFromEnv はエクスポート先の接続情報を環境変数から読み込みます。
// 秘密情報を設定ファイルへ保存しないため、JSON設定ファイルからは読み込みません。
func LoadObjectStorageConfigFromEnv() (ObjectStorageConfig, error) {
	endpointURL := strings.TrimSpace(os.Getenv("OBJECT_STORAGE_ENDPOINT_URL"))
	accessKeyID := strings.TrimSpace(os.Getenv("OBJECT_STORAGE_ACCESS_KEY_ID"))
	secretAccessKey := strings.TrimSpace(os.Getenv("OBJECT_STORAGE_SECRET_ACCESS_KEY"))
	bucketName := strings.TrimSpace(os.Getenv("OBJECT_STORAGE_BUCKET_NAME"))

	var validationErrors []string
	if endpointURL == "" {
		validationErrors = append(validationErrors, "OBJECT_STORAGE_ENDPOINT_URL environment variable is required")
	}
	if accessKeyID == "" {
		validationErrors = append(validationErrors, "OBJECT_STORAGE_ACCESS_KEY_ID environment variable is required")
	}
	if secretAccessKey == "" {
		validationErrors = append(validationErrors, "OBJECT_STORAGE_SECRET_ACCESS_KEY environment variable is required")
	}
	if bucketName == "" {
		validationErrors = append(validationErrors, "OBJECT_STORAGE_BUCKET_NAME environment variable is required")
	}

	var endpoint string
	if endpointURL != "" {
		parsedURL, err := url.ParseRequestURI(endpointURL)
		if err != nil {
			validationErrors = append(validationErrors, "OBJECT_STORAGE_ENDPOINT_URL must be a valid HTTPS URL")
		} else {
			switch {
			case parsedURL.Scheme != "https":
				validationErrors = append(validationErrors, "OBJECT_STORAGE_ENDPOINT_URL must use HTTPS")
			case parsedURL.Host == "":
				validationErrors = append(validationErrors, "OBJECT_STORAGE_ENDPOINT_URL must include a host")
			case parsedURL.User != nil || (parsedURL.Path != "" && parsedURL.Path != "/") || parsedURL.RawQuery != "" || parsedURL.Fragment != "":
				validationErrors = append(validationErrors, "OBJECT_STORAGE_ENDPOINT_URL must not include credentials, a path, query, or fragment")
			default:
				endpoint = parsedURL.Host
			}
		}
	}

	if len(validationErrors) > 0 {
		return ObjectStorageConfig{}, fmt.Errorf("object storage configuration validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return ObjectStorageConfig{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		BucketName:      bucketName,
		Secure:          true,
	}, nil
}
