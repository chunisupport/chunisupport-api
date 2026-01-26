package api_internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/app"
	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/app/handler/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/auth"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/username"
	dto_internal "github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// newMockMasterCache はテスト用のマスタデータキャッシュを作成します。
func newMockMasterCache() *masterdata.Cache {
	return &masterdata.Cache{
		AccountTypes: map[string]masterdata.Item{
			"PLAYER": {ID: 1, Name: "PLAYER"},
			"EDITOR": {ID: 2, Name: "EDITOR"},
			"ADMIN":  {ID: 3, Name: "ADMIN"},
		},
	}
}

// mockAuthService は service.AuthService のモックです。
type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) Register(ctx context.Context, username, password string) (*dto_internal.UserDTO, string, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*dto_internal.UserDTO), args.String(1), args.Error(2)
}

func (m *mockAuthService) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockAuthService) Authenticate(ctx context.Context, userID int, sessionID string) (*entity.User, error) {
	args := m.Called(ctx, userID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *mockAuthService) GetUser(ctx context.Context, id int) (*dto_internal.UserDTO, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserDTO), args.Error(1)
}

func (m *mockAuthService) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	args := m.Called(ctx, userID, isPrivate)
	return args.Error(0)
}

func (m *mockAuthService) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	args := m.Called(ctx, userID, currentPassword, newPassword)
	return args.Error(0)
}

func (m *mockAuthService) IssueRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockAuthService) RecoverWithRecoveryCode(ctx context.Context, recoveryCode, newPassword string) error {
	args := m.Called(ctx, recoveryCode, newPassword)
	return args.Error(0)
}

func (m *mockAuthService) DeleteUser(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	return e
}

func TestAuthHandler_Register(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockAuthService)
	h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

	t.Run("正常系: ユーザー登録", func(t *testing.T) {
		expectedUser := &dto_internal.UserDTO{Username: "testuser"}
		mockService.On("Register", mock.Anything, "testuser", "password123").Return(expectedUser, "test_token", nil).Once()

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

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザー名重複時は409エラー", func(t *testing.T) {
		mockService.On("Register", mock.Anything, "existinguser", "password123").Return(nil, "", usecase.ErrUsernameTaken).Once()

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

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザー名が短すぎる", func(t *testing.T) {
		mockService.On("Register", mock.Anything, "abc", "password123").Return(nil, "", usecase.ErrUsernameTooShort).Once()

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

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: パスワードが短すぎる", func(t *testing.T) {
		mockService.On("Register", mock.Anything, "testuser", "short").Return(nil, "", usecase.ErrPasswordTooShort).Once()

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

		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockAuthService)
	h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

	t.Run("正常系: ログイン", func(t *testing.T) {
		mockService.On("Login", mock.Anything, "testuser", "password123").Return("test_token", nil).Once()

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
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: 不正な資格情報", func(t *testing.T) {
		mockService.On("Login", mock.Anything, "testuser", "wrongpassword").Return("", usecase.ErrInvalidCredentials).Once()

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
		mockService.AssertExpectations(t)
	})

	t.Run("正常系: Secure属性がtrueの場合", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, true, http.SameSiteStrictMode, newMockMasterCache())

		mockService.On("Login", mock.Anything, "testuser", "password123").Return("test_token", nil).Once()

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
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("正常系: ログアウト", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		sessionID := uuid.New().String()
		claims := &auth.Claims{UserID: 1, SessionID: sessionID}
		mockService.On("Logout", mock.Anything, sessionID).Return(nil).Once()

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

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: クレームがコンテキストに存在しない", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Logout(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)

		mockService.AssertNotCalled(t, "Logout", mock.Anything)
	})
}

func TestAuthHandler_IssueRecoveryCodes(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockAuthService)
	h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

	t.Run("正常系: リカバリーコード発行", func(t *testing.T) {
		expectedCodes := []string{"ABCD-EFGH-IJKL", "MNOP-QRST-UVWX"}
		mockService.On("IssueRecoveryCodes", mock.Anything, 1).Return(expectedCodes, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/internal/me/recovery-codes", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.IssueRecoveryCodes(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp struct {
			RecoveryCodes []string `json:"recovery_codes"`
		}
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, expectedCodes, resp.RecoveryCodes)

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: 発行に失敗する", func(t *testing.T) {
		mockService.On("IssueRecoveryCodes", mock.Anything, 1).Return(nil, usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodPost, "/internal/me/recovery-codes", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", &entity.User{ID: 1})

		err := h.IssueRecoveryCodes(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)

		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_RecoverPassword(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockAuthService)
	h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

	t.Run("正常系: リカバリーコードで復旧", func(t *testing.T) {
		mockService.On("RecoverWithRecoveryCode", mock.Anything, "ABCD-EFGH-IJKL", "new-password").Return(nil).Once()

		body := `{"recovery_code": "ABCD-EFGH-IJKL", "new_password": "new-password"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/recovery-codes", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.RecoverPassword(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: リカバリーコードの形式が不正", func(t *testing.T) {
		body := `{"recovery_code": "invalid", "new_password": "new-password"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/recovery-codes", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.RecoverPassword(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	})

	t.Run("異常系: リカバリーコードが無効", func(t *testing.T) {
		mockService.On("RecoverWithRecoveryCode", mock.Anything, "ABCD-EFGH-IJKL", "new-password").Return(usecase.ErrInvalidRecoveryCredentials).Once()

		body := `{"recovery_code": "ABCD-EFGH-IJKL", "new_password": "new-password"}`
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/recovery-codes", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.RecoverPassword(c)
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidRecovery, apiErr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_Me(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockAuthService)
	h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

	t.Run("正常系: ユーザー情報取得", func(t *testing.T) {
		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		mockUserDTO := &dto_internal.UserDTO{
			Username:    un.String(),
			AccountType: "PLAYER",
		}

		mockService.On("GetUser", mock.Anything, mockUser.ID).Return(mockUserDTO, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser) // ミドルウェアの代わり

		err := h.Me(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var userDTO dto_internal.UserDTO
		err = json.Unmarshal(rec.Body.Bytes(), &userDTO)
		assert.NoError(t, err)
		assert.Equal(t, un.String(), userDTO.Username)

		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_UpdatePrivacy(t *testing.T) {
	t.Run("正常系: 非公開設定を有効化", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un, IsPrivate: false}
		mockService.On("UpdatePrivacy", mock.Anything, 1, true).Return(nil).Once()

		body := `{"is_private": true}`
		req := httptest.NewRequest(http.MethodPut, "/api/me/privacy", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser) // ミドルウェアの代わり

		err := h.UpdatePrivacy(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var respBody map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.True(t, respBody["is_private"].(bool))

		mockService.AssertExpectations(t)
	})

	t.Run("正常系: 非公開設定を無効化", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un, IsPrivate: true}
		mockService.On("UpdatePrivacy", mock.Anything, 1, false).Return(nil).Once()

		body := `{"is_private": false}`
		req := httptest.NewRequest(http.MethodPut, "/api/me/privacy", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser)

		err := h.UpdatePrivacy(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var respBody map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody["is_private"].(bool))

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーエンティティがコンテキストに存在しない", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		body := `{"is_private": true}`
		req := httptest.NewRequest(http.MethodPut, "/api/me/privacy", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.UpdatePrivacy(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)

		mockService.AssertNotCalled(t, "UpdatePrivacy", mock.Anything, mock.Anything)
	})

	t.Run("異常系: サービスエラー", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un, IsPrivate: false}
		mockService.On("UpdatePrivacy", mock.Anything, 1, true).Return(errors.New("database error")).Once()

		body := `{"is_private": true}`
		req := httptest.NewRequest(http.MethodPut, "/api/me/privacy", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser)

		err := h.UpdatePrivacy(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)

		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(*mockAuthService)
		expectedStatus int
		expectError    bool
	}{
		{
			name: "正常系: パスワード変更",
			body: `{"current_password": "oldpass123", "new_password": "newpass123"}`,
			setupMock: func(m *mockAuthService) {
				m.On("ChangePassword", mock.Anything, 1, "oldpass123", "newpass123").Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "異常系: バリデーションエラー - パスワードが短すぎる",
			body: `{"current_password": "short", "new_password": "short"}`,
			setupMock: func(m *mockAuthService) {
				m.On("ChangePassword", mock.Anything, 1, "short", "short").Return(usecase.ErrPasswordTooShort).Once()
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "異常系: 現在のパスワードが間違っている",
			body: `{"current_password": "wrongpass", "new_password": "newpass123"}`,
			setupMock: func(m *mockAuthService) {
				m.On("ChangePassword", mock.Anything, 1, "wrongpass", "newpass123").Return(usecase.ErrIncorrectPassword).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name: "異常系: 新しいパスワードが現在のパスワードと同じ",
			body: `{"current_password": "password123", "new_password": "password123"}`,
			setupMock: func(m *mockAuthService) {
				m.On("ChangePassword", mock.Anything, 1, "password123", "password123").Return(usecase.ErrInvalidPassword).Once() // セキュリティ強化
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "異常系: サービスエラー",
			body: `{"current_password": "oldpass123", "new_password": "newpass123"}`,
			setupMock: func(m *mockAuthService) {
				m.On("ChangePassword", mock.Anything, 1, "oldpass123", "newpass123").Return(errors.New("db error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEcho()
			mockService := new(mockAuthService)
			h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

			un, _ := username.NewUserName("testuser")
			mockUser := &entity.User{ID: 1, Username: un}

			tc.setupMock(mockService)

			req := httptest.NewRequest(http.MethodPut, "/internal/me/password", bytes.NewBufferString(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("userEntity", mockUser)

			err := h.ChangePassword(c)

			if tc.expectError {
				assert.Error(t, err)
				apiErr, ok := err.(*apierror.APIError)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedStatus, apiErr.HTTPStatus)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedStatus, rec.Code)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_DeleteAccount(t *testing.T) {
	t.Run("正常系: アカウント削除", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		claims := &auth.Claims{SessionID: "session-123"}

		mockService.On("DeleteUser", mock.Anything, 1).Return(nil).Once()
		mockService.On("Logout", mock.Anything, claims.SessionID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser)
		c.Set("user", claims)

		err := h.DeleteAccount(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 1)
		assert.Equal(t, "token", cookies[0].Name)
		assert.Equal(t, "", cookies[0].Value)
		assert.True(t, mockUser.IsDeleted)

		// ボディは空であることを確認
		assert.Empty(t, rec.Body.String())

		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーエンティティが存在しない", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		req := httptest.NewRequest(http.MethodDelete, "/api/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.DeleteAccount(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)

		mockService.AssertNotCalled(t, "DeleteUser", mock.Anything)
	})

	t.Run("異常系: 削除処理でエラー", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}

		mockService.On("DeleteUser", mock.Anything, 1).Return(errors.New("database error")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser)

		err := h.DeleteAccount(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)

		mockService.AssertExpectations(t)
		mockService.AssertNotCalled(t, "Logout", mock.Anything)
	})

	t.Run("異常系: セッション無効化でエラー", func(t *testing.T) {
		e := newTestEcho()
		mockService := new(mockAuthService)
		h := api_internal.NewAuthHandler(mockService, false, http.SameSiteLaxMode, newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		claims := &auth.Claims{SessionID: "session-123"}

		mockService.On("DeleteUser", mock.Anything, 1).Return(nil).Once()
		mockService.On("Logout", mock.Anything, claims.SessionID).Return(errors.New("session error")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", mockUser)
		c.Set("user", claims)

		err := h.DeleteAccount(c)
		assert.Error(t, err)

		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)

		mockService.AssertExpectations(t)
	})
}
