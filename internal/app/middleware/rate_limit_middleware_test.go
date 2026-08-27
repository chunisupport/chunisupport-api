package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/info"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

// setupFixedWindowStoreCleanup はFixedWindowStoreのクリーンアップgoroutineを停止できるようにします
func setupFixedWindowStoreCleanup(t *testing.T) {
	t.Helper()

	fixedWindowStoreCleanupHookMu.Lock()
	previousHook := fixedWindowStoreCleanupHook
	fixedWindowStoreCleanupHook = func(stop func()) {
		t.Cleanup(stop)
	}
	fixedWindowStoreCleanupHookMu.Unlock()

	t.Cleanup(func() {
		fixedWindowStoreCleanupHookMu.Lock()
		fixedWindowStoreCleanupHook = previousHook
		fixedWindowStoreCleanupHookMu.Unlock()
	})
}

// setupEchoWithErrorHandler はテスト用にエラーハンドラーを設定したEchoインスタンスを作成します
func setupEchoWithErrorHandler(t *testing.T) *echo.Echo {
	t.Helper()
	setupFixedWindowStoreCleanup(t)

	e := echo.New()
	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		if response, _ := echo.UnwrapResponse(c.Response()); response != nil && response.Committed {
			return
		}
		if apiErr, ok := err.(*apierror.APIError); ok {
			c.JSON(apiErr.HTTPStatus, map[string]any{
				"error": map[string]any{
					"status": apiErr.HTTPStatus,
					"code":   apiErr.Code,
				},
			})
			return
		}
		echo.DefaultHTTPErrorHandler(true)(c, err)
	}
	return e
}

// setupUserRateLimitTest はユーザーIDベースのレートリミットテスト用に共通のセットアップを行います
func setupUserRateLimitTest(t *testing.T) (*echo.Echo, echo.HandlerFunc) {
	t.Helper()

	e := setupEchoWithErrorHandler(t)
	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Second,
	}
	middleware := UserRateLimitMiddleware(config)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	return e, handler
}

// performUserRateLimitRequest はユーザーIDベースのレートリミット用にリクエストを実行します
func performUserRateLimitRequest(t *testing.T, e *echo.Echo, handler echo.HandlerFunc, userEntity any) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if userEntity != nil {
		c.Set("userEntity", userEntity)
	}

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	return rec
}

func TestAPIRateLimitMiddleware_ADMINには管理者用上限を適用する(t *testing.T) {
	e := setupEchoWithErrorHandler(t)

	const adminLimit = 3
	middleware := APIRateLimitMiddleware(2, 50, adminLimit, 1*time.Minute)

	adminUser := &entity.User{
		ID:            1,
		AccountTypeID: info.AccountTypeAdmin,
	}

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	performRequest := func() *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", adminUser)

		if err := handler(c); err != nil {
			e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	for i, expectedRemaining := range []string{"2", "1", "0"} {
		rec := performRequest()
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, expectedRemaining, rec.Header().Get("X-RateLimit-Remaining"), "リクエスト回数: %d", i+1)
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	}

	rec := performRequest()
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
}

func TestAPIRateLimitMiddleware_NonAdminLimited(t *testing.T) {
	// ADMIN以外のユーザーはレートリミットを受ける
	e := setupEchoWithErrorHandler(t)

	middleware := APIRateLimitMiddleware(3, 50, 10000, 1*time.Minute)

	playerUser := &entity.User{
		ID:            100, // 他のテストと衝突しないIDを使用
		AccountTypeID: info.AccountTypePlayer,
	}

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// 制限回数までは成功
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", playerUser)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(c, err)
		}
		assert.Equal(t, http.StatusOK, rec.Code)

		// ヘッダーが設定されていることを確認
		assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	}

	// 制限を超えると429エラー
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", playerUser)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// 429エラー時もヘッダーが設定されていることを確認
	assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestAPIRateLimitMiddleware_EditorHasSeparateLimit(t *testing.T) {
	// EDITORユーザーはeditorLimitが適用され、超過時に429を返す
	e := setupEchoWithErrorHandler(t)

	// normalLimit=2, editorLimit=3, adminLimit=10000
	middleware := APIRateLimitMiddleware(2, 3, 10000, 1*time.Minute)

	editorUser := &entity.User{
		ID:            200,
		AccountTypeID: info.AccountTypeEditor,
	}

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// 通常の制限（2回）を超えてもeditorLimit（3回）までは成功
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", editorUser)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(c, err)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// editorLimit（3回）を超えると429エラー
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", editorUser)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
}

func TestAPIRateLimitMiddleware_ExtDevHasEditorLimit(t *testing.T) {
	// EXTDEVユーザーもeditorLimitが適用され、超過時に429を返す
	e := setupEchoWithErrorHandler(t)

	middleware := APIRateLimitMiddleware(2, 3, 10000, 1*time.Minute)

	extDevUser := &entity.User{
		ID:            210,
		AccountTypeID: info.AccountTypeExtDev,
	}

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", extDevUser)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(c, err)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", extDevUser)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
}

func TestAPIRateLimitMiddleware_DifferentUsersHaveSeparateLimits(t *testing.T) {
	// 異なるユーザーは別々のレートリミットを持つ
	e := setupEchoWithErrorHandler(t)

	middleware := APIRateLimitMiddleware(2, 50, 10000, 1*time.Minute)

	user1 := &entity.User{
		ID:            300, // 他のテストと衝突しないIDを使用
		AccountTypeID: info.AccountTypePlayer,
	}
	user2 := &entity.User{
		ID:            400, // 他のテストと衝突しないIDを使用
		AccountTypeID: info.AccountTypePlayer,
	}

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// ユーザー1が制限回数までリクエスト
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user1)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(c, err)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// ユーザー1は制限超過
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", user1)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// ユーザー2はまだリクエスト可能
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set("userEntity", user2)

	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIRateLimitMiddleware_NoUserEntity(t *testing.T) {
	// ユーザー情報がない場合は認証エラー
	e := setupEchoWithErrorHandler(t)

	middleware := APIRateLimitMiddleware(10, 50, 10000, 1*time.Minute)

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// userEntityを設定しない

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIRateLimitMiddleware_InvalidUserEntity(t *testing.T) {
	// ユーザー情報が不正な型の場合は認証エラー
	e := setupEchoWithErrorHandler(t)

	middleware := APIRateLimitMiddleware(10, 50, 10000, 1*time.Minute)

	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", "invalid_type")

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserRateLimitMiddleware_SameUserLimited(t *testing.T) {
	e, handler := setupUserRateLimitTest(t)

	user := &entity.User{
		ID:            500,
		AccountTypeID: info.AccountTypePlayer,
	}

	rec := performUserRateLimitRequest(t, e, handler, user)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = performUserRateLimitRequest(t, e, handler, user)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestUserRateLimitMiddleware_DifferentUsersHaveSeparateLimits(t *testing.T) {
	e, handler := setupUserRateLimitTest(t)

	user1 := &entity.User{
		ID:            600,
		AccountTypeID: info.AccountTypePlayer,
	}
	user2 := &entity.User{
		ID:            700,
		AccountTypeID: info.AccountTypePlayer,
	}

	rec := performUserRateLimitRequest(t, e, handler, user1)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = performUserRateLimitRequest(t, e, handler, user1)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	rec = performUserRateLimitRequest(t, e, handler, user2)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserRateLimitMiddleware_NoUserEntity(t *testing.T) {
	e, handler := setupUserRateLimitTest(t)

	rec := performUserRateLimitRequest(t, e, handler, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserRateLimitMiddleware_InvalidUserEntity(t *testing.T) {
	e, handler := setupUserRateLimitTest(t)

	rec := performUserRateLimitRequest(t, e, handler, "invalid")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestIPRateLimitMiddleware(t *testing.T) {
	e := setupEchoWithErrorHandler(t)

	// 1秒間に3回までのリクエストを許可する設定
	config := RateLimitConfig{
		Requests: 3,
		Window:   1 * time.Second,
	}
	middleware := IPRateLimitMiddleware(config)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// IPアドレスを設定 (RemoteAddr)
	testIP := "192.0.2.100:1234"

	// 1回目: OK
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 2回目: OK
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 3回目: OK
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 4回目: NG (Too Many Requests)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// 別のIPからのリクエスト: OK
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.0.2.200:1234"
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	err = handler(c2)
	if err != nil {
		e.HTTPErrorHandler(c2, err)
	}
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestIPRateLimitMiddleware_IPExtractor未設定ではRemoteAddrを使用する(t *testing.T) {
	e := setupEchoWithErrorHandler(t)

	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Minute,
	}
	middleware := IPRateLimitMiddleware(config)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	performRequest := func(remoteAddr, forwardedFor string) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		req.Header.Set("X-Forwarded-For", forwardedFor)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := handler(c); err != nil {
			e.HTTPErrorHandler(c, err)
		}
		return rec
	}

	first := performRequest("192.0.2.1:10001", "203.0.113.100")
	second := performRequest("192.0.2.2:10002", "203.0.113.100")
	sameRemoteAddr := performRequest("192.0.2.1:10003", "203.0.113.200")

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, http.StatusTooManyRequests, sameRemoteAddr.Code)
}

func TestAnonymousIPRateLimitMiddleware_AnonymousLimited(t *testing.T) {
	e := setupEchoWithErrorHandler(t)

	config := RateLimitConfig{
		Requests: 2,
		Window:   1 * time.Minute,
	}
	middleware := AnonymousIPRateLimitMiddleware(config)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	testIP := "192.0.2.100:1234"

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = testIP
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(c, err)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(c, err)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestAnonymousIPRateLimitMiddleware_AuthenticatedSkipsLimit(t *testing.T) {
	e := setupEchoWithErrorHandler(t)

	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Minute,
	}
	middleware := AnonymousIPRateLimitMiddleware(config)
	handler := middleware(func(c *echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	user := &entity.User{
		ID:            1,
		AccountTypeID: info.AccountTypePlayer,
	}

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.100:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(c, err)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
