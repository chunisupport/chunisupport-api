package api_internal_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/reauthtoken"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
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

	t.Run("ユーザーが存在しない場合はuser_not_foundを返す", func(t *testing.T) {
		// Given
		user := &entity.User{ID: 11, IsPrivate: false}
		userCredentialMock.On("UpdatePrivacy", mock.Anything, 11, true).Return(usecase.ErrUserNotFound).Once()

		body := `{"is_private": true}`
		req := httptest.NewRequest(http.MethodPut, "/internal/me/privacy", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.UpdatePrivacy(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
		userCredentialMock.AssertExpectations(t)
	})
}

func TestProfileHandler_DeleteAccount(t *testing.T) {
	e := newTestEcho()

	t.Run("アカウント削除時に204 No Contentを返す", func(t *testing.T) {
		h, userCredentialMock := newProfileHandlerWithMocks()
		// Given
		user := &entity.User{ID: 20}
		userCredentialMock.On("DeleteOwnAccount", mock.Anything, 20, reauthtoken.MustNew("reauth-token")).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		req.Header.Set("X-Reauth-Token", "reauth-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		userCredentialMock.AssertExpectations(t)
	})

	t.Run("再認証トークンがなければ recent sign-in required を返す", func(t *testing.T) {
		h, userCredentialMock := newProfileHandlerWithMocks()
		// Given
		user := &entity.User{ID: 21}
		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.ErrorIs(t, err, apierror.ErrRecentSignInRequired)
		userCredentialMock.AssertNotCalled(t, "DeleteOwnAccount", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("ユーザー未設定時は認証エラー", func(t *testing.T) {
		h, _ := newProfileHandlerWithMocks()
		// Given
		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.ErrorIs(t, err, apierror.ErrUnauthorized)
	})

	t.Run("UID不一致系はクライアントへ汎用認証エラーを返す", func(t *testing.T) {
		h, userCredentialMock := newProfileHandlerWithMocks()
		// Given
		user := &entity.User{ID: 22}
		userCredentialMock.On(
			"DeleteOwnAccount",
			mock.Anything,
			22,
			reauthtoken.MustNew("reauth-token"),
		).Return(usecase.ErrInvalidCredentials).Once()

		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		req.Header.Set("X-Reauth-Token", "reauth-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.DeleteAccount(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeInvalidCredentials, apiErr.Code)
			assert.Equal(t, apierror.ErrInvalidCredentials.HTTPStatus, apiErr.HTTPStatus)
		}
		userCredentialMock.AssertExpectations(t)
	})

	t.Run("再認証トークンを正規化して204 No Contentを返す", func(t *testing.T) {
		h, userCredentialMock := newProfileHandlerWithMocks()
		// Given
		user := &entity.User{ID: 23}
		userCredentialMock.On("DeleteOwnAccount", mock.Anything, 23, reauthtoken.MustNew("reauth-token")).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/internal/me", nil)
		req.Header.Set("X-Reauth-Token", "  reauth-token  ")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		// When
		err := h.DeleteAccount(c)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		userCredentialMock.AssertExpectations(t)
	})
}
