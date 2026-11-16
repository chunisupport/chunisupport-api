package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// concurrencyTestRequest は同時実行数制限のテストで送るリクエストの送信元です
type concurrencyTestRequest struct {
	user any
	ip   string
}

// blockingHandler は解放されるまで処理中のまま留まるハンドラーを作成します
func blockingHandler(started chan<- struct{}, release <-chan struct{}) echo.HandlerFunc {
	return func(c *echo.Context) error {
		started <- struct{}{}
		<-release
		return c.String(http.StatusOK, "OK")
	}
}

// performConcurrencyLimitRequest は送信元を設定してリクエストを実行します
func performConcurrencyLimitRequest(e *echo.Echo, handler echo.HandlerFunc, request concurrencyTestRequest) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = request.ip + ":1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if request.user != nil {
		c.Set("userEntity", request.user)
	}
	if err := handler(c); err != nil {
		e.HTTPErrorHandler(c, err)
	}
	return rec.Code
}

func TestConcurrencyLimitMiddleware(t *testing.T) {
	tests := []struct {
		name string
		// Given: 送信元ごとの上限2本・全体の上限3本で、処理中のまま留めるリクエスト
		inFlight []concurrencyTestRequest
		// When: 追加で送るリクエスト
		next concurrencyTestRequest
		// Then: 追加リクエストの期待ステータス
		expected int
	}{
		{
			name: "同じユーザーが上限を超えると429になる",
			inFlight: []concurrencyTestRequest{
				{user: &entity.User{ID: 1}, ip: "192.0.2.1"},
				{user: &entity.User{ID: 1}, ip: "192.0.2.2"},
			},
			next:     concurrencyTestRequest{user: &entity.User{ID: 1}, ip: "192.0.2.3"},
			expected: http.StatusTooManyRequests,
		},
		{
			name: "別のユーザーは送信元ごとの上限を共有しない",
			inFlight: []concurrencyTestRequest{
				{user: &entity.User{ID: 1}, ip: "192.0.2.1"},
				{user: &entity.User{ID: 1}, ip: "192.0.2.1"},
			},
			next:     concurrencyTestRequest{user: &entity.User{ID: 2}, ip: "192.0.2.1"},
			expected: http.StatusOK,
		},
		{
			name: "未認証は同じIPで上限を共有する",
			inFlight: []concurrencyTestRequest{
				{ip: "192.0.2.1"},
				{ip: "192.0.2.1"},
			},
			next:     concurrencyTestRequest{ip: "192.0.2.1"},
			expected: http.StatusTooManyRequests,
		},
		{
			name: "全体の上限を超えると429になる",
			inFlight: []concurrencyTestRequest{
				{user: &entity.User{ID: 1}, ip: "192.0.2.1"},
				{user: &entity.User{ID: 2}, ip: "192.0.2.1"},
				{user: &entity.User{ID: 3}, ip: "192.0.2.1"},
			},
			next:     concurrencyTestRequest{user: &entity.User{ID: 4}, ip: "192.0.2.1"},
			expected: http.StatusTooManyRequests,
		},
		{
			name:     "不正なユーザー情報は401になる",
			next:     concurrencyTestRequest{user: "invalid", ip: "192.0.2.1"},
			expected: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := setupEchoWithErrorHandler(t)
			limit := ConcurrencyLimitMiddleware(2, 3)
			started := make(chan struct{})
			release := make(chan struct{})
			blocking := limit(blockingHandler(started, release))
			// テスト終了時に必ず解放し、処理中のゴルーチンを残さないようにします。
			var releaseOnce sync.Once
			releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(releaseAll)
			var wg sync.WaitGroup
			for _, request := range tt.inFlight {
				done := make(chan int, 1)
				wg.Go(func() {
					done <- performConcurrencyLimitRequest(e, blocking, request)
				})
				// 前提のリクエストが拒否された場合にハンドラー到達を待ち続けないよう、完了も監視します。
				select {
				case <-started:
				case code := <-done:
					require.Failf(t, "前提のリクエストが処理中になりませんでした", "status: %d", code)
				}
			}

			// When
			code := performConcurrencyLimitRequest(e, limit(func(c *echo.Context) error {
				return c.String(http.StatusOK, "OK")
			}), tt.next)

			// Then
			assert.Equal(t, tt.expected, code)
			releaseAll()
			wg.Wait()
		})
	}
}

func TestConcurrencyLimitMiddleware_処理完了後は枠を解放する(t *testing.T) {
	tests := []struct {
		name string
		// Given: 送信元ごとの上限1本で、1本目を処理するハンドラーの結果
		handlerErr error
	}{
		{
			name:       "正常終了した場合",
			handlerErr: nil,
		},
		{
			name:       "ハンドラーがエラーを返した場合",
			handlerErr: apierror.ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := setupEchoWithErrorHandler(t)
			limit := ConcurrencyLimitMiddleware(1, 1)
			request := concurrencyTestRequest{user: &entity.User{ID: 1}, ip: "192.0.2.1"}
			performConcurrencyLimitRequest(e, limit(func(c *echo.Context) error {
				return tt.handlerErr
			}), request)

			// When
			code := performConcurrencyLimitRequest(e, limit(func(c *echo.Context) error {
				return c.String(http.StatusOK, "OK")
			}), request)

			// Then
			assert.Equal(t, http.StatusOK, code)
		})
	}
}
