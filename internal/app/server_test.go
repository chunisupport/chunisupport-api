package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestServerAddress(t *testing.T) {
	// Given: アプリケーションの待受ポート
	const port = 3000

	// When
	actual := serverAddress(port)

	// Then: ループバックIPv4のみで待ち受ける
	assert.Equal(t, "127.0.0.1:3000", actual)
}

func TestNewServer_メンテナンス状態を復元できなければ起動準備に失敗する(t *testing.T) {
	// Given
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	// When
	server, err := NewServer(
		context.Background(),
		database,
		config.Config{
			CORS: config.CORS{
				AllowOrigins: []string{"https://example.com"},
			},
		},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	// Then
	require.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "failed to initialize system maintenance state")
}

func TestNewRouter_永続化済みメンテナンス状態を全体へ適用する(t *testing.T) {
	// Given
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	_, err = database.Exec(`
		CREATE TABLE system_maintenance (
			id INTEGER NOT NULL PRIMARY KEY,
			enabled BOOLEAN NOT NULL,
			comment TEXT NOT NULL,
			updated_by_user_id INTEGER NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)
	_, err = database.Exec(
		`INSERT INTO system_maintenance (id, enabled, comment, updated_by_user_id, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		1,
		true,
		"データ更新中です",
		nil,
		time.Date(2026, time.July, 26, 3, 30, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	cfg := config.Config{
		Location: time.FixedZone("JST", 9*60*60),
		CORS: config.CORS{
			AllowOrigins: []string{"https://example.com"},
		},
	}
	var accessLog bytes.Buffer
	router, err := NewRouter(
		context.Background(),
		database,
		cfg,
		nil,
		nil,
		tokenVerifierWithRecentSignIn{},
		nil,
		&accessLog,
	)
	require.NoError(t, err)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantCode   string
		wantBody   string
	}{
		{name: "ヘルスチェックは維持する", path: "/healthz", wantStatus: http.StatusNoContent},
		{
			name:       "状態確認は設定タイムゾーンで公開する",
			path:       "/internal/system/status",
			wantStatus: http.StatusOK,
			wantBody:   `"updated_at":"2026-07-26T12:30:00+09:00"`,
		},
		{name: "ルートは遮断する", path: "/", wantStatus: http.StatusServiceUnavailable, wantCode: `"code":"maintenance_mode"`},
		{name: "バージョンAPIは遮断する", path: "/version", wantStatus: http.StatusServiceUnavailable, wantCode: `"code":"maintenance_mode"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			// When
			router.ServeHTTP(rec, req)

			// Then
			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCode != "" {
				assert.Contains(t, rec.Body.String(), tt.wantCode)
				assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
			}
			if tt.wantBody != "" {
				assert.Contains(t, rec.Body.String(), tt.wantBody)
			}
		})
	}

	assert.Contains(t, accessLog.String(), "uri=/")
	assert.Contains(t, accessLog.String(), "status=503")
}
