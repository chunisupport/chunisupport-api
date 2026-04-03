package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockFirebaseAuthenticator struct {
	mock.Mock
}

func (m *mockFirebaseAuthenticator) Authenticate(ctx context.Context, idToken string) (*entity.User, error) {
	args := m.Called(ctx, idToken)
	if user, ok := args.Get(0).(*entity.User); ok {
		return user, args.Error(1)
	}

	return nil, args.Error(1)
}

func TestFirebaseIDTokenMiddleware(t *testing.T) {
	e := echo.New()

	t.Run("Bearerトークンが有効な場合はuserEntityを設定して次のハンドラーを実行する", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer firebase-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		user := &entity.User{ID: 1}
		mockAuthenticator.On("Authenticate", mock.Anything, "firebase-token").Return(user, nil).Once()

		handlerCalled := false
		handler := middlewareFunc(func(c echo.Context) error {
			handlerCalled = true
			storedUser, ok := c.Get("userEntity").(*entity.User)
			require.True(t, ok)
			assert.Same(t, user, storedUser)
			return c.String(http.StatusOK, "ok")
		})

		// When
		err := handler(c)

		// Then
		require.NoError(t, err)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockAuthenticator.AssertExpectations(t)
	})

	t.Run("Authorizationヘッダがない場合は401を返す", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeMissingToken, apiErr.Code)
		mockAuthenticator.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
	})

	t.Run("IDトークンが不正な場合は401を返す", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockAuthenticator.On("Authenticate", mock.Anything, "invalid-token").Return(nil, usecase.ErrInvalidIDToken).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		mockAuthenticator.AssertExpectations(t)
	})

	t.Run("失効済みIDトークンの場合は401を返す", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer revoked-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockAuthenticator.On("Authenticate", mock.Anything, "revoked-token").Return(nil, usecase.ErrInvalidIDToken).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		mockAuthenticator.AssertExpectations(t)
	})

	t.Run("無効化済みユーザーのIDトークンの場合は401を返す", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer disabled-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockAuthenticator.On("Authenticate", mock.Anything, "disabled-token").Return(nil, usecase.ErrInvalidIDToken).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		mockAuthenticator.AssertExpectations(t)
	})

	t.Run("authenticatorがnilの場合は500を返す", func(t *testing.T) {
		// Given
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(nil)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
	})

	t.Run("authenticatorがnilのユーザーを返した場合は500を返す", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockAuthenticator.On("Authenticate", mock.Anything, "valid-token").Return(nil, nil).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockAuthenticator.AssertExpectations(t)
	})

	t.Run("内部エラーの場合は500を返す", func(t *testing.T) {
		// Given
		mockAuthenticator := new(mockFirebaseAuthenticator)
		middlewareFunc := middleware.FirebaseIDTokenMiddleware(mockAuthenticator)
		req := httptest.NewRequest(http.MethodGet, "/internal/me", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer error-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockAuthenticator.On("Authenticate", mock.Anything, "error-token").Return(nil, errors.New("boom")).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		// When
		err := handler(c)

		// Then
		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockAuthenticator.AssertExpectations(t)
	})
}

var _ middleware.FirebaseAuthenticator = (*mockFirebaseAuthenticator)(nil)
