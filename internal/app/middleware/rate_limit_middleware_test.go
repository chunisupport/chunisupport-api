package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// setupEchoWithErrorHandler はテスト用にエラーハンドラーを設定したEchoインスタンスを作成します
func setupEchoWithErrorHandler() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		if apiErr, ok := err.(*apierror.APIError); ok {
			c.JSON(apiErr.HTTPStatus, map[string]interface{}{
				"error": map[string]interface{}{
					"status": apiErr.HTTPStatus,
					"code":   apiErr.Code,
				},
			})
			return
		}
		e.DefaultHTTPErrorHandler(err, c)
	}
	return e
}

func TestAPIRateLimitMiddleware_AdminUnlimited(t *testing.T) {
	// ADMINユーザーはレートリミットを受けない
	e := setupEchoWithErrorHandler()

	// normalLimit=2, adminLimit=10000
	middleware := APIRateLimitMiddleware(2, 10000, 1*time.Minute)

	adminUser := &entity.User{
		ID:            1,
		AccountTypeID: AccountTypeAdmin,
	}

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// ADMINは通常の制限（2回）を超えてもリクエストできる
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", adminUser)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusOK, rec.Code)

		// ヘッダーが設定されていることを確認
		assert.Equal(t, "10000", rec.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	}
}

func TestAPIRateLimitMiddleware_NonAdminLimited(t *testing.T) {
	// ADMIN以外のユーザーはレートリミットを受ける
	e := setupEchoWithErrorHandler()

	middleware := APIRateLimitMiddleware(3, 10000, 1*time.Minute)

	playerUser := &entity.User{
		ID:            100, // 他のテストと衝突しないIDを使用
		AccountTypeID: AccountTypePlayer,
	}

	handler := middleware(func(c echo.Context) error {
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
			e.HTTPErrorHandler(err, c)
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
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// 429エラー時もヘッダーが設定されていることを確認
	assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestAPIRateLimitMiddleware_EditorLimited(t *testing.T) {
	// EDITORユーザーもレートリミットを受ける
	e := setupEchoWithErrorHandler()

	middleware := APIRateLimitMiddleware(2, 10000, 1*time.Minute)

	editorUser := &entity.User{
		ID:            200, // 他のテストと衝突しないIDを使用
		AccountTypeID: AccountTypeEditor,
	}

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// 制限回数までは成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", editorUser)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 制限を超えると429エラー
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", editorUser)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestAPIRateLimitMiddleware_DifferentUsersHaveSeparateLimits(t *testing.T) {
	// 異なるユーザーは別々のレートリミットを持つ
	e := setupEchoWithErrorHandler()

	middleware := APIRateLimitMiddleware(2, 10000, 1*time.Minute)

	user1 := &entity.User{
		ID:            300, // 他のテストと衝突しないIDを使用
		AccountTypeID: AccountTypePlayer,
	}
	user2 := &entity.User{
		ID:            400, // 他のテストと衝突しないIDを使用
		AccountTypeID: AccountTypePlayer,
	}

	handler := middleware(func(c echo.Context) error {
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
			e.HTTPErrorHandler(err, c)
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
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// ユーザー2はまだリクエスト可能
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set("userEntity", user2)

	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIRateLimitMiddleware_NoUserEntity(t *testing.T) {
	// ユーザー情報がない場合は認証エラー
	e := setupEchoWithErrorHandler()

	middleware := APIRateLimitMiddleware(10, 10000, 1*time.Minute)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// userEntityを設定しない

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIRateLimitMiddleware_InvalidUserEntity(t *testing.T) {
	// ユーザー情報が不正な型の場合は認証エラー
	e := setupEchoWithErrorHandler()

	middleware := APIRateLimitMiddleware(10, 10000, 1*time.Minute)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", "invalid_type")

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestIPRateLimitMiddleware(t *testing.T) {
	e := setupEchoWithErrorHandler()

	// 1秒間に3回までのリクエストを許可する設定
	config := RateLimitConfig{
		Requests: 3,
		Window:   1 * time.Second,
	}
	middleware := IPRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
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
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 2回目: OK
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 3回目: OK
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 4回目: NG (Too Many Requests)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// 別のIPからのリクエスト: OK
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.0.2.200:1234"
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	err = handler(c2)
	if err != nil {
		e.HTTPErrorHandler(err, c2)
	}
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestIPRateLimitMiddleware_XForwardedFor(t *testing.T) {
	e := setupEchoWithErrorHandler()

	// 1秒間に1回までのリクエストを許可する設定
	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Second,
	}
	middleware := IPRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// X-Forwarded-For ヘッダーを設定
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.100")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 1回目: OK
	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	// 2回目: NG (同じIPとみなされる)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.100")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestUserRateLimitMiddleware_Limited(t *testing.T) {
	e := setupEchoWithErrorHandler()

	config := RateLimitConfig{
		Requests: 2,
		Window:   1 * time.Minute,
	}
	middleware := UserRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	user := &entity.User{
		ID:            500,
		AccountTypeID: AccountTypePlayer,
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", user)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestUserRateLimitMiddleware_DifferentUsersHaveSeparateLimits(t *testing.T) {
	e := setupEchoWithErrorHandler()

	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Minute,
	}
	middleware := UserRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	user1 := &entity.User{
		ID:            600,
		AccountTypeID: AccountTypePlayer,
	}
	user2 := &entity.User{
		ID:            700,
		AccountTypeID: AccountTypePlayer,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", user1)
	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set("userEntity", user1)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set("userEntity", user2)
	err = handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserRateLimitMiddleware_NoUserEntity(t *testing.T) {
	e := setupEchoWithErrorHandler()

	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Minute,
	}
	middleware := UserRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserRateLimitMiddleware_InvalidUserEntity(t *testing.T) {
	e := setupEchoWithErrorHandler()

	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Minute,
	}
	middleware := UserRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", "invalid_type")

	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAnonymousIPRateLimitMiddleware_AnonymousLimited(t *testing.T) {
	e := setupEchoWithErrorHandler()

	config := RateLimitConfig{
		Requests: 2,
		Window:   1 * time.Minute,
	}
	middleware := AnonymousIPRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
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
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = testIP
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestAnonymousIPRateLimitMiddleware_AuthenticatedSkipsLimit(t *testing.T) {
	e := setupEchoWithErrorHandler()

	config := RateLimitConfig{
		Requests: 1,
		Window:   1 * time.Minute,
	}
	middleware := AnonymousIPRateLimitMiddleware(config)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	user := &entity.User{
		ID:            1,
		AccountTypeID: AccountTypePlayer,
	}

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.100:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		err := handler(c)
		if err != nil {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
