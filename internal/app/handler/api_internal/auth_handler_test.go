package api_internal_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAuthUsecase は usecase.AuthUsecase のモックです。
type mockAuthUsecase struct {
	mock.Mock
}

func (m *mockAuthUsecase) Register(ctx context.Context, username, password string) (*dto_internal.UserDTO, string, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*dto_internal.UserDTO), args.String(1), args.Error(2)
}

func (m *mockAuthUsecase) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *mockAuthUsecase) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockAuthUsecase) Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error) {
	args := m.Called(ctx, userID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

// mockUserCredentialUsecase は usecase.UserCredentialUsecase のモックです。
type mockUserCredentialUsecase struct {
	mock.Mock
}

func (m *mockUserCredentialUsecase) GetUser(ctx context.Context, id int) (*dto_internal.UserDTO, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserDTO), args.Error(1)
}

func (m *mockUserCredentialUsecase) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	args := m.Called(ctx, userID, isPrivate)
	return args.Error(0)
}

func (m *mockUserCredentialUsecase) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	args := m.Called(ctx, userID, currentPassword, newPassword)
	return args.Error(0)
}

func (m *mockUserCredentialUsecase) DeleteUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// mockRecoveryUsecase は usecase.RecoveryUsecase のモックです。
type mockRecoveryUsecase struct {
	mock.Mock
}

func (m *mockRecoveryUsecase) IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRecoveryUsecase) RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error {
	args := m.Called(ctx, recoveryCode, newPassword)
	return args.Error(0)
}

func newAuthHandlerWithMocks(secureCookie bool, sameSite http.SameSite) (*api_internal.AuthHandler, *mockAuthUsecase, *mockUserCredentialUsecase, *mockRecoveryUsecase) {
	authMock := new(mockAuthUsecase)
	userCredentialMock := new(mockUserCredentialUsecase)
	recoveryMock := new(mockRecoveryUsecase)

	h := api_internal.NewAuthHandler(authMock, secureCookie, sameSite)

	return h, authMock, userCredentialMock, recoveryMock
}

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	return e
}

func TestAuthHandler_Register(t *testing.T) {
	e := newTestEcho()
	h, authMock, _, _ := newAuthHandlerWithMocks(false, http.SameSiteLaxMode)

	t.Run("正常系: ユーザー登録", func(t *testing.T) {
		expectedUser := &dto_internal.UserDTO{
			Username:  "testuser",
			IsPrivate: false,
		}
		authMock.On("Register", mock.Anything, "testuser", "password123").Return(expectedUser, "test_token", nil).Once()

		body := `{"username": "testuser", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		// Cookieの検証
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
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)     // セキュリティ強化: 409→400
		assert.Equal(t, apierror.CodeRegistrationFailed, apiErr.Code) // username_taken→registration_failed

		authMock.AssertExpectations(t)
	})

	t.Run("異常系: ユーザー名が短すぎる", func(t *testing.T) {
		authMock.On("Register", mock.Anything, "abc", "password123").Return(nil, "", usecase.ErrUsernameTooShort).Once()

		body := `{"username": "abc", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeUsernameTooShort, apiErr.Code)

		authMock.AssertExpectations(t)
	})

	t.Run("異常系: パスワードが短すぎる", func(t *testing.T) {
		authMock.On("Register", mock.Anything, "testuser", "short").Return(nil, "", usecase.ErrPasswordTooShort).Once()

		body := `{"username": "testuser", "password": "short"}`
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Register(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodePasswordTooShort, apiErr.Code)

		authMock.AssertExpectations(t)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	e := newTestEcho()
	h, authMock, _, _ := newAuthHandlerWithMocks(false, http.SameSiteLaxMode)

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
		assert.False(t, cookie.Secure) // テストではfalseを期待
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
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		authMock.AssertExpectations(t)
	})

	t.Run("正常系: Secure属性がtrueの場合", func(t *testing.T) {
		e := newTestEcho()
		h, authMock, _, _ := newAuthHandlerWithMocks(true, http.SameSiteStrictMode)

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
		assert.True(t, cookie.Secure) // Secure=trueを期待
		assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		authMock.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("正常系: ログアウト", func(t *testing.T) {
		e := newTestEcho()
		h, authMock, _, _ := newAuthHandlerWithMocks(false, http.SameSiteLaxMode)

		sessionID := uuid.New().String()
		claims := &auth.Claims{UserID: 1, SessionID: sessionID}
		authMock.On("Logout", mock.Anything, sessionID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims) // ミドルウェアの代わり

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

		// ボディは空であることを確認
		assert.Empty(t, rec.Body.String())

		authMock.AssertExpectations(t)
	})

	t.Run("異常系: クレームがコンテキストに存在しない", func(t *testing.T) {
		e := newTestEcho()
		h, authMock, _, _ := newAuthHandlerWithMocks(false, http.SameSiteLaxMode)

		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Logout(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)

		authMock.AssertNotCalled(t, "Logout", mock.Anything)
	})
}
