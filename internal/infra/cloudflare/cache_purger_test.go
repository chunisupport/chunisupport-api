package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachePurgerPurge_URL単位でキャッシュを削除する(t *testing.T) {
	var gotAuthorization string
	var gotBody struct {
		Files []string `json:"files"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"purge-id"}}`))
	}))
	defer server.Close()
	purger := newCachePurger(
		config.CloudflareCacheConfig{
			APIToken:      "token",
			ZoneID:        "zone-id",
			PublicBaseURL: "https://static.chunisupport.net",
		},
		server.Client(),
		server.URL,
		func(context.Context, time.Duration) error { return nil },
	)

	err := purger.Purge(context.Background(), []string{"v1/songs.json", "v1/worldsend-songs.json"})

	require.NoError(t, err)
	assert.Equal(t, "Bearer token", gotAuthorization)
	assert.Equal(t, []string{
		"https://static.chunisupport.net/v1/songs.json",
		"https://static.chunisupport.net/v1/worldsend-songs.json",
	}, gotBody.Files)
}

func TestCachePurgerPurge_一時的な失敗を再試行する(t *testing.T) {
	attempts := 0
	var retryDelays []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1015,"message":"rate limited"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"purge-id"}}`))
	}))
	defer server.Close()
	purger := newCachePurger(
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.chunisupport.net"},
		server.Client(),
		server.URL,
		func(_ context.Context, duration time.Duration) error {
			retryDelays = append(retryDelays, duration)
			return nil
		},
	)

	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, []time.Duration{0}, retryDelays)
}

func TestCachePurgerPurge_APIエラーを返す(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"authentication error"}]}`))
	}))
	defer server.Close()
	purger := newCachePurger(
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.chunisupport.net"},
		server.Client(),
		server.URL,
		func(context.Context, time.Duration) error { return nil },
	)

	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication error")
	assert.Equal(t, 1, attempts)
}

func TestCachePurgerPurge_サーバーエラーは最大試行回数で終了する(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1000,"message":"unavailable"}]}`))
	}))
	defer server.Close()
	purger := newCachePurger(
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.chunisupport.net"},
		server.Client(),
		server.URL,
		func(context.Context, time.Duration) error { return nil },
	)

	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	require.Error(t, err)
	assert.Equal(t, 3, attempts)
}
