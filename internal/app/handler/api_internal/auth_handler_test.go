package api_internal_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	return e
}

func TestAuthHandler_Register(t *testing.T) {
	e := newTestEcho()
	h, authMock := newAuthHandlerWithAuthMock(false, http.SameSiteLaxMode)

	t.Run("正常系: ユーザー登録", func(t *testing.T) {
		expectedUser := &dto_internal.UserDTO{Username: "testuser", IsPrivate: false}
		authMock.On("Register", mock.Anything, "testuser", "password123").Return(expectedUser, "test_token", nil).Once()

		body := `{"username": "testuser", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		cookie := rec.Result().Cookies()[0]
		assert.Equal(t, "token", cookie.Name)
		assert.Equal(t, "test_token", cookie.Value)
		assert.True(t, cookie.HttpOnly)
		authMock.AssertExpectations(t)
	})

	t.Run("異常系: ユーザー名重複時は409エラー", func(t *testing.T) {
		authMock.On("Register", mock.Anything, "existinguser", "password123").Return(nil, "", usecase.ErrUsernameTaken).Once()

		body := `{"username": "existinguser", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeRegistrationFailed, apiErr.Code)
		authMock.AssertExpectations(t)
	})

	t.Run("異常系: ユーザー名が短すぎる場合はバリデーションエラー", func(t *testing.T) {
		body := `{"username": "abc", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("異常系: ユーザー名に大文字が含まれる場合はバリデーションエラー", func(t *testing.T) {
		body := `{"username": "Testuser", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("異常系: パスワードが短すぎる場合はバリデーションエラー", func(t *testing.T) {
		body := `{"username": "testuser", "password": "short"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	e := newTestEcho()
	h, authMock := newAuthHandlerWithAuthMock(false, http.SameSiteLaxMode)

	t.Run("正常系: ログイン", func(t *testing.T) {
		authMock.On("Login", mock.Anything, "testuser", "password123").Return("test_token", nil).Once()

		body := `{"username": "testuser", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		cookie := rec.Result().Cookies()[0]
		assert.Equal(t, "token", cookie.Name)
		assert.Equal(t, "test_token", cookie.Value)
		assert.True(t, cookie.HttpOnly)
		assert.False(t, cookie.Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		authMock.AssertExpectations(t)
	})

	t.Run("異常系: 不正な資格情報", func(t *testing.T) {
		authMock.On("Login", mock.Anything, "testuser", "wrongpassword").Return("", usecase.ErrInvalidCredentials).Once()

		body := `{"username": "testuser", "password": "wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		authMock.AssertExpectations(t)
	})

	t.Run("異常系: バリデーションエラー時は400を返す", func(t *testing.T) {
		body := `{"username": "abc", "password": "short"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	})

	t.Run("正常系: Secure属性がtrueの場合", func(t *testing.T) {
		e := newTestEcho()
		h, authMock := newAuthHandlerWithAuthMock(true, http.SameSiteStrictMode)
		authMock.On("Login", mock.Anything, "testuser", "password123").Return("test_token", nil).Once()

		body := `{"username": "testuser", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Login(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		cookie := rec.Result().Cookies()[0]
		assert.Equal(t, "token", cookie.Name)
		assert.True(t, cookie.Secure)
		assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		authMock.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("正常系: ログアウト", func(t *testing.T) {
		e := newTestEcho()
		h, authMock := newAuthHandlerWithAuthMock(false, http.SameSiteLaxMode)

		sessionID := uuid.New().String()
		claims := &auth.Claims{UserID: 1, SessionID: sessionID}
		authMock.On("Logout", mock.Anything, sessionID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		err := h.Logout(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 1)
		cookie := cookies[0]
		assert.Equal(t, "token", cookie.Name)
		assert.Empty(t, cookie.Value)
		assert.Equal(t, -1, cookie.MaxAge)
		assert.True(t, cookie.Expires.Equal(time.Unix(0, 0).UTC()))
		assert.Empty(t, rec.Body.String())
		authMock.AssertExpectations(t)
	})

	t.Run("異常系: クレームがコンテキストに存在しない", func(t *testing.T) {
		e := newTestEcho()
		h, authMock := newAuthHandlerWithAuthMock(false, http.SameSiteLaxMode)

		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Logout(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		authMock.AssertNotCalled(t, "Logout", mock.Anything)
	})
}
