package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCloudflareCacheConfigFromEnv(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "token")
	t.Setenv("CLOUDFLARE_ZONE_ID", "575f883bc4eb7c2d89c56ee987c73873")
	t.Setenv("STATIC_DATA_PUBLIC_BASE_URL", " https://static.chunisup-dev.f5.si/ ")

	got, err := LoadCloudflareCacheConfigFromEnv()

	require.NoError(t, err)
	assert.Equal(t, CloudflareCacheConfig{
		APIToken:      "token",
		ZoneID:        "575f883bc4eb7c2d89c56ee987c73873",
		PublicBaseURL: "https://static.chunisup-dev.f5.si",
	}, got)
}

func TestLoadCloudflareCacheConfigFromEnv_環境ごとの公開先を許可する(t *testing.T) {
	tests := []struct {
		name    string
		zoneID  string
		baseURL string
	}{
		{name: "開発とステージング", zoneID: "575f883bc4eb7c2d89c56ee987c73873", baseURL: "https://static.chunisup-dev.f5.si"},
		{name: "beta", zoneID: "6ef634111241a2dc524992ed7cfcf20f", baseURL: "https://static.beta-chunisup.f5.si"},
		{name: "本番", zoneID: "c7e970656a686c79cce6fad84c888d2c", baseURL: "https://static.chunisupport.net"},
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
		{name: "トークンなし", zoneID: "575f883bc4eb7c2d89c56ee987c73873", baseURL: "https://static.chunisup-dev.f5.si"},
		{name: "Zone IDなし", token: "token", baseURL: "https://static.chunisup-dev.f5.si"},
		{name: "Zone IDが不正", token: "token", zoneID: "../purge_cache", baseURL: "https://static.chunisup-dev.f5.si"},
		{name: "Zone IDと公開先が不一致", token: "token", zoneID: "6ef634111241a2dc524992ed7cfcf20f", baseURL: "https://static.chunisup-dev.f5.si"},
		{name: "HTTP", token: "token", zoneID: "575f883bc4eb7c2d89c56ee987c73873", baseURL: "http://static.chunisup-dev.f5.si"},
		{name: "パスあり", token: "token", zoneID: "575f883bc4eb7c2d89c56ee987c73873", baseURL: "https://static.chunisup-dev.f5.si/path"},
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
