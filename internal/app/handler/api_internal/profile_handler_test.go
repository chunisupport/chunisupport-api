package api_internal_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newProfileHandlerWithMocks(secureCookie bool, sameSite http.SameSite) (*api_internal.ProfileHandler, *mockAuthUsecase, *mockUserCredentialUsecase) {
	authMock := new(mockAuthUsecase)
	userCredentialMock := new(mockUserCredentialUsecase)

	h := api_internal.NewProfileHandler(authMock, userCredentialMock, secureCookie, sameSite)

	return h, authMock, userCredentialMock
}

func TestProfileHandler_Me(t *testing.T) {
	e := newTestEcho()
	h, _, userCredentialMock := newProfileHandlerWithMocks(false, http.SameSiteLaxMode)

	t.Run("正常系: 自分のユーザー情報を取得できる", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 1}
		expected := &dto_internal.UserDTO{Username: "tester", IsPrivate: true}
		userCredentialMock.On("GetUser", mock.Anything, 1).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.Me(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "tester")
		userCredentialMock.AssertExpectations(t)
	})
}

func TestProfileHandler_UpdatePrivacy(t *testing.T) {
	e := newTestEcho()
	h, _, userCredentialMock := newProfileHandlerWithMocks(false, http.SameSiteLaxMode)

	t.Run("公開設定更新時にユースケースを呼び出して成功レスポンスを返す", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 10, IsPrivate: false}
		userCredentialMock.On("UpdatePrivacy", mock.Anything, 10, true).Return(nil).Once()

		body := `{"is_private": true}`
		req := httptest.NewRequest(http.MethodPatch, "/internal/me/privacy", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.UpdatePrivacy(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		userCredentialMock.AssertExpectations(t)
	})
}

func TestProfileHandler_DeleteAccount(t *testing.T) {
	e := newTestEcho()
	h, authMock, userCredentialMock := newProfileHandlerWithMocks(false, http.SameSiteLaxMode)

	t.Run("アカウント削除時にセッション無効化とCookie削除を行う", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 20}
		claims := &auth.Claims{SessionID: "session-1"}
		userCredentialMock.On("DeleteUser", mock.Anything, 20).Return(nil).Once()
		authMock.On("Logout", mock.Anything, "session-1").Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)
		c.Set("user", claims)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		cookies := rec.Result().Cookies()
		if assert.NotEmpty(t, cookies) {
			assert.Equal(t, "token", cookies[0].Name)
			assert.Equal(t, -1, cookies[0].MaxAge)
		}
		userCredentialMock.AssertExpectations(t)
		authMock.AssertExpectations(t)
	})

	t.Run("ユーザー未設定時は認証エラー", func(t *testing.T) {
		// Given
		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.ErrorIs(t, err, apierror.ErrUnauthorized)
	})
}

func TestProfileHandler_ChangePassword(t *testing.T) {
	e := newTestEcho()
	h, _, userCredentialMock := newProfileHandlerWithMocks(false, http.SameSiteLaxMode)

	t.Run("正常系: パスワード変更", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 5}
		userCredentialMock.On("ChangePassword", mock.Anything, 5, "oldpass123", "newpass123").Return(nil).Once()

		body := `{"current_password":"oldpass123","new_password":"newpass123"}`
		req := httptest.NewRequest(http.MethodPatch, "/internal/me/password", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.ChangePassword(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		userCredentialMock.AssertExpectations(t)
	})

	t.Run("異常系: 新しいパスワードが短い場合はユースケースエラーを返す", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 5}
		userCredentialMock.On("ChangePassword", mock.Anything, 5, "oldpass123", "short").Return(usecase.ErrPasswordTooShort).Once()

		body := `{"current_password":"oldpass123","new_password":"short"}`
		req := httptest.NewRequest(http.MethodPatch, "/internal/me/password", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.ChangePassword(c)

		// Then
		assert.Error(t, err)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		userCredentialMock.AssertExpectations(t)
	})
}
