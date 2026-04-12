package api_internal_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProfileHandler_Me(t *testing.T) {
	e := newTestEcho()
	h, userCredentialMock := newProfileHandlerWithMocks()

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
	h, userCredentialMock := newProfileHandlerWithMocks()

	t.Run("公開設定更新時にユースケースを呼び出して成功レスポンスを返す", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 10, IsPrivate: false}
		userCredentialMock.On("UpdatePrivacy", mock.Anything, 10, true).Return(nil).Once()

		body := `{"is_private": true}`
		req := httptest.NewRequest(http.MethodPut, "/internal/me/privacy", bytes.NewBufferString(body))
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
	h, userCredentialMock := newProfileHandlerWithMocks()

	t.Run("アカウント削除時にユーザー削除のみを行う", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 20}
		userCredentialMock.On("DeleteOwnAccount", mock.Anything, 20).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Result().Cookies())
		userCredentialMock.AssertExpectations(t)
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
	h, userCredentialMock := newProfileHandlerWithMocks()

	t.Run("正常系: パスワード変更", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 5}
		userCredentialMock.On("ChangePassword", mock.Anything, 5, "oldpass123", "newpass123").Return(nil).Once()

		body := `{"current_password":"oldpass123","new_password":"newpass123"}`
		req := httptest.NewRequest(http.MethodPut, "/internal/me/password", bytes.NewBufferString(body))
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

	t.Run("異常系: 新しいパスワードが短い場合はバリデーションエラー", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 5}

		body := `{"current_password":"oldpass123","new_password":"short"}`
		req := httptest.NewRequest(http.MethodPut, "/internal/me/password", bytes.NewBufferString(body))
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
		assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})
}
