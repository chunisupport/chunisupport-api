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

var testDataTransferHMACSecret = []byte("0123456789abcdef0123456789abcdef")

// shutdownOrderBatchJobs は Wait 実行時点でDBがまだ利用可能かを記録するスタブです。
// Shutdown が DB を閉じる前に Wait を終える順序の逆転を検知するために使います。
type shutdownOrderBatchJobs struct {
	db      *sqlx.DB
	pingErr chan error
}

func (s *shutdownOrderBatchJobs) Wait() {
	s.pingErr <- s.db.Ping()
}

// blockingBatchJobs は Wait の終了をテスト側のチャネルで制御するスタブです。
// 実時間の sleep を使わず、cancel 済みコンテキストとチャネルだけでタイムアウト到達を再現します。
type blockingBatchJobs struct {
	release chan struct{}
	exited  chan struct{}
}

func (b *blockingBatchJobs) Wait() {
	<-b.release
	close(b.exited)
}

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
		config.Config{DataTransferHMACSecret: testDataTransferHMACSecret,
			CORS: config.CORS{
				AllowOrigins: []string{"https://example.com"},
			},
		},
		nil,
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
	cfg := config.Config{DataTransferHMACSecret: testDataTransferHMACSecret,
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

func TestServerShutdown_バッチ終了後にDBを閉じてキャンセルを通知する(t *testing.T) {
	// Given: 実ネットリスンを使わない直接構築のサーバーと記録用sqlite
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	// Shutdown が DB を閉じるため、後片付けの二重 Close は結果を無視する。
	t.Cleanup(func() {
		_ = database.Close()
	})
	cancelNotified := make(chan struct{}, 1)
	pingErr := make(chan error, 1)
	server := &Server{
		db: database,
		cancelBatchJobs: func() {
			cancelNotified <- struct{}{}
		},
		batchJobs: []backgroundBatchJobs{
			&shutdownOrderBatchJobs{db: database, pingErr: pingErr},
		},
	}

	// When: 正常コンテキストで停止する
	err = server.Shutdown(context.Background())

	// Then: エラーなく停止する
	require.NoError(t, err)
	// Then: Wait 実行時点では DB がまだ利用可能だった（順序逆転があれば失敗する）
	require.Len(t, pingErr, 1)
	assert.NoError(t, <-pingErr)
	// Then: バッチのキャンセルが通知された（cancel 呼び出しの削除を検知する）
	select {
	case <-cancelNotified:
	default:
		assert.Fail(t, "バッチのキャンセルが通知されませんでした")
	}
	// Then: 終了後は DB が閉じられている
	assert.Error(t, database.Ping())
}

func TestServerShutdown_バッチ待ちがctx期限で失敗してもDBを閉じる(t *testing.T) {
	// Given: 終了しないバッチと記録用sqlite（実時間の待機は使わない）
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	// Shutdown が DB を閉じるため、後片付けの二重 Close は結果を無視する。
	t.Cleanup(func() {
		_ = database.Close()
	})
	release := make(chan struct{})
	exited := make(chan struct{})
	// Shutdown 前に失敗しても Wait を取り残さないため、解放だけを保証する。
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	cancelNotified := make(chan struct{}, 1)
	server := &Server{
		db: database,
		cancelBatchJobs: func() {
			cancelNotified <- struct{}{}
		},
		batchJobs: []backgroundBatchJobs{
			&blockingBatchJobs{release: release, exited: exited},
		},
	}
	// cancel 済みコンテキストで期限切れ到達を確定的に再現する。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// When: 終了しないバッチを期限切れコンテキストで停止する
	err = server.Shutdown(ctx)
	released = true
	close(release)

	// Then: 原因を保持したまま期限切れを返す
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	// Then: バッチのキャンセルが通知された（cancel 呼び出しの削除を検知する）
	select {
	case <-cancelNotified:
	default:
		assert.Fail(t, "バッチのキャンセルが通知されませんでした")
	}
	// Then: 期限切れでも DB は閉じられている
	assert.Error(t, database.Ping())

	// When: 残ったバッチを解放してゴルーチンの終了を待つ（リーク防止）
	<-exited
}
