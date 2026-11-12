package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachePurgerPurge_公開JSONのプレフィックスだけを削除する(t *testing.T) {
	var gotAuthorization string
	var gotBody map[string][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
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
			PublicBaseURL: "https://static.example.test",
		},
		server.Client(),
		server.URL,
		func(context.Context, time.Duration) error { return nil },
	)

	err := purger.Purge(context.Background(), []string{"v1/songs.json", "v1/worldsend-songs.json", "compat/reiwa/1/chunithm_record/original.json"})

	require.NoError(t, err)
	assert.Equal(t, "Bearer token", gotAuthorization)
	assert.Equal(t, []string{
		"static.example.test/v1/songs.json",
		"static.example.test/v1/worldsend-songs.json",
		"static.example.test/compat/reiwa/1/chunithm_record/original.json",
	}, gotBody["prefixes"])
	assert.Len(t, gotBody, 1)
}

func TestCachePurgerPurge_一時的な失敗を再試行する(t *testing.T) {
	attempts := 0
	var retryDelays []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		var body map[string][]string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string][]string{"prefixes": {"static.example.test/v1/songs.json"}}, body)
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
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
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
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
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
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
		server.Client(),
		server.URL,
		func(context.Context, time.Duration) error { return nil },
	)

	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	require.Error(t, err)
	assert.Equal(t, 3, attempts)
}

func TestCachePurgerPurge_本番コンストラクタはゾーンIDを含むURLへ送る(t *testing.T) {
	// Given: 本番コンストラクタで作ったクライアント。通信部分だけを差し替えて外部通信はしない。
	purger := NewCachePurger(config.CloudflareCacheConfig{
		APIToken:      "token",
		ZoneID:        "0123456789abcdef0123456789abcdef",
		PublicBaseURL: "https://static.example.test",
	})
	var gotRequest *http.Request
	purger.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotRequest = req
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
			Request:    req,
		}, nil
	})

	// When
	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	// Then: ゾーン ID を含む正規の API パスへ送る。誤った URL では古い JSON が配信され続けるため。
	require.NoError(t, err)
	require.NotNil(t, gotRequest)
	assert.Equal(t, "https", gotRequest.URL.Scheme)
	assert.Equal(t, "api.cloudflare.com", gotRequest.URL.Host)
	assert.Equal(t, "/client/v4/zones/0123456789abcdef0123456789abcdef/purge_cache", gotRequest.URL.Path)
}

func TestCachePurgerPurge_失敗応答の扱い(t *testing.T) {
	tests := []struct {
		name string
		// Given: API の応答
		status int
		body   string
		// Then: 期待する結果
		wantAttempts    int
		wantErrContains string
	}{
		{
			name:            "成功ステータスでも失敗を示す本文は再試行しない",
			status:          http.StatusOK,
			body:            `{"success":false,"errors":[{"code":1000,"message":"prefix not found"}]}`,
			wantAttempts:    1,
			wantErrContains: "status 200",
		},
		{
			name:            "JSONとして読めない応答は再試行しない",
			status:          http.StatusOK,
			body:            `not json`,
			wantAttempts:    1,
			wantErrContains: "invalid response",
		},
		{
			name:            "JSONとして読めない応答でも503は再試行する",
			status:          http.StatusServiceUnavailable,
			body:            `not json`,
			wantAttempts:    3,
			wantErrContains: "invalid response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			purger := newCachePurger(
				config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
				server.Client(),
				server.URL,
				func(context.Context, time.Duration) error { return nil },
			)

			// When
			err := purger.Purge(context.Background(), []string{"v1/songs.json"})

			// Then
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrContains)
			assert.Equal(t, tt.wantAttempts, attempts)
		})
	}
}

func TestCachePurgerPurge_通信失敗は上限まで再試行して原因を残す(t *testing.T) {
	// Given: 通信そのものが毎回失敗する。
	sentinel := errors.New("transport failure")
	attempts := 0
	purger := newCachePurger(
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, sentinel
		})},
		"http://example.test/zones/zone-id/purge_cache",
		func(context.Context, time.Duration) error { return nil },
	)

	// When
	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	// Then: 上限の 3 回まで再試行し、原因エラーを errors.Is で辿れる。
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to call cloudflare cache purge API")
	assert.True(t, errors.Is(err, sentinel))
	assert.Equal(t, 3, attempts)
}

func TestCachePurgerPurge_応答本文の読取失敗は再試行しない(t *testing.T) {
	// Given: 応答本文の読み取りが毎回失敗する。
	sentinel := errors.New("response body read failure")
	attempts := 0
	sleepCalls := 0
	purger := newCachePurger(
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
		&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(iotest.ErrReader(sentinel)),
				Request:    req,
			}, nil
		})},
		"http://example.test/zones/zone-id/purge_cache",
		func(context.Context, time.Duration) error {
			sleepCalls++
			return nil
		},
	)

	// When
	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	// Then: 原因を残して 1 回で終え、待機しない。
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read cloudflare cache purge response")
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, attempts)
	assert.Equal(t, 0, sleepCalls)
}

func TestCachePurgerPurge_待機中のキャンセルは再送せず中断する(t *testing.T) {
	// Given: 初回は再試行対象の 429、待機はキャンセルで失敗する。
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1015,"message":"rate limited"}]}`))
	}))
	defer server.Close()
	sleepCalls := 0
	purger := newCachePurger(
		config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
		server.Client(),
		server.URL,
		func(context.Context, time.Duration) error {
			sleepCalls++
			// sleepContext が待機中のキャンセルで返す値の代わり。
			return context.Canceled
		},
	)

	// When
	err := purger.Purge(context.Background(), []string{"v1/songs.json"})

	// Then: 中断エラーを返し、2 回目のリクエストは送らない。
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry interrupted")
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 1, sleepCalls)
	assert.Equal(t, 1, attempts)
}

func TestCachePurgerPurge_再試行の待機時間(t *testing.T) {
	tests := []struct {
		name string
		// Given: 初回の 429 に付ける Retry-After。空文字はヘッダーなし。
		retryAfter string
		// retryAfterFromNow が 0 でなければ、テスト開始時刻にこの時間を足した HTTP-date を動的に使う。
		retryAfterFromNow time.Duration
		// Then: 1 回目の再試行前に待つ時間。未来の HTTP-date は範囲で確かめるため使わない。
		wantDelay time.Duration
	}{
		{
			name:       "Retry-Afterの指定秒数だけ待つ",
			retryAfter: "2",
			wantDelay:  2 * time.Second,
		},
		{
			name:       "Retry-Afterがなければ1秒待つ",
			retryAfter: "",
			wantDelay:  time.Second,
		},
		{
			name:       "不正なRetry-Afterは指定なしと同じく1秒待つ",
			retryAfter: "invalid",
			wantDelay:  time.Second,
		},
		{
			// 固定の過去日時はつねに期限切れのため 0 待機になる。時刻境界の不安定化なしに ParseTime 分岐の削除を検知できる。
			name:       "過去のHTTP-dateは待たずに再試行する",
			retryAfter: "Mon, 02 Jan 2006 15:04:05 GMT",
			wantDelay:  0,
		},
		{
			// 未来の HTTP-date は現在時刻に依存するため、呼出前後の時刻から許容範囲で確かめる。固定遅延値を now から手計算しない。
			name:              "未来のHTTP-dateは指定時刻まで待つ",
			retryAfterFromNow: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			retryAfterValue := tt.retryAfter
			var retryAt time.Time
			if tt.retryAfterFromNow != 0 {
				retryAt = time.Now().Add(tt.retryAfterFromNow).UTC().Truncate(time.Second)
				retryAfterValue = retryAt.Format(http.TimeFormat)
			}
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts++
				w.Header().Set("Content-Type", "application/json")
				if attempts == 1 {
					if retryAfterValue != "" {
						w.Header().Set("Retry-After", retryAfterValue)
					}
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1015,"message":"rate limited"}]}`))
					return
				}
				_, _ = w.Write([]byte(`{"success":true,"result":{"id":"purge-id"}}`))
			}))
			defer server.Close()
			var delays []time.Duration
			purger := newCachePurger(
				config.CloudflareCacheConfig{APIToken: "token", ZoneID: "zone-id", PublicBaseURL: "https://static.example.test"},
				server.Client(),
				server.URL,
				func(_ context.Context, duration time.Duration) error {
					delays = append(delays, duration)
					return nil
				},
			)

			// When
			before := time.Now()
			err := purger.Purge(context.Background(), []string{"v1/songs.json"})
			after := time.Now()

			// Then: 実際に待たず、待機関数に渡る時間で確かめる。
			require.NoError(t, err)
			assert.Equal(t, 2, attempts)
			if tt.retryAfterFromNow != 0 {
				require.Len(t, delays, 1)
				assert.GreaterOrEqual(t, delays[0], max(retryAt.Sub(after), time.Duration(0)))
				assert.LessOrEqual(t, delays[0], max(retryAt.Sub(before), time.Duration(0)))
				return
			}
			assert.Equal(t, []time.Duration{tt.wantDelay}, delays)
		})
	}
}

// roundTripFunc は HTTP 通信部分だけを差し替えるテスト用の RoundTripper です。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
