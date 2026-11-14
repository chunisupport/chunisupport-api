package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCloudflareCacheConfigFromEnv(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "token")
	t.Setenv("CLOUDFLARE_ZONE_ID", "11111111111111111111111111111111")
	t.Setenv("STATIC_DATA_PUBLIC_BASE_URL", " https://static-develop.example.test/ ")

	got, err := LoadCloudflareCacheConfigFromEnv()

	require.NoError(t, err)
	assert.Equal(t, CloudflareCacheConfig{
		APIToken:      "token",
		ZoneID:        "11111111111111111111111111111111",
		PublicBaseURL: "https://static-develop.example.test",
	}, got)
}

func TestLoadCloudflareCacheConfigFromEnv_環境ごとの公開先を許可する(t *testing.T) {
	tests := []struct {
		name    string
		zoneID  string
		baseURL string
	}{
		{name: "開発", zoneID: "11111111111111111111111111111111", baseURL: "https://static-develop.example.test"},
		{name: "ステージング", zoneID: "22222222222222222222222222222222", baseURL: "https://static-staging.example.test"},
		{name: "本番", zoneID: "33333333333333333333333333333333", baseURL: "https://static-production.example.test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLOUDFLARE_API_TOKEN", "token")
			t.Setenv("CLOUDFLARE_ZONE_ID", tt.zoneID)
			t.Setenv("STATIC_DATA_PUBLIC_BASE_URL", tt.baseURL)

			got, err := LoadCloudflareCacheConfigFromEnv()

			require.NoError(t, err)
			assert.Equal(t, tt.zoneID, got.ZoneID)
			assert.Equal(t, tt.baseURL, got.PublicBaseURL)
		})
	}
}

func TestLoadCloudflareCacheConfigFromEnv_不正な設定を拒否する(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		zoneID  string
		baseURL string
	}{
		{name: "トークンなし", zoneID: "11111111111111111111111111111111", baseURL: "https://static-develop.example.test"},
		{name: "Zone IDなし", token: "token", baseURL: "https://static-develop.example.test"},
		{name: "Zone IDが不正", token: "token", zoneID: "../purge_cache", baseURL: "https://static-develop.example.test"},
		{name: "HTTP", token: "token", zoneID: "11111111111111111111111111111111", baseURL: "http://static-develop.example.test"},
		{name: "パスあり", token: "token", zoneID: "11111111111111111111111111111111", baseURL: "https://static-develop.example.test/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLOUDFLARE_API_TOKEN", tt.token)
			t.Setenv("CLOUDFLARE_ZONE_ID", tt.zoneID)
			t.Setenv("STATIC_DATA_PUBLIC_BASE_URL", tt.baseURL)

			_, err := LoadCloudflareCacheConfigFromEnv()

			assert.Error(t, err)
		})
	}
}
