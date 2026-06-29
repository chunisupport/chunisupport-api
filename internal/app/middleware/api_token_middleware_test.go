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
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAPITokenUsecase struct {
	mock.Mock
}

func (m *mockAPITokenUsecase) Generate(ctx context.Context, userID int) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *mockAPITokenUsecase) GetStatus(ctx context.Context, userID int) (*entity.APIToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.APIToken), args.Error(1)
}

func (m *mockAPITokenUsecase) Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error) {
	args := m.Called(ctx, rawToken)
	if args.Get(0) == nil || args.Get(1) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*entity.User), args.Get(1).(*entity.APIToken), args.Error(2)
}

func (m *mockAPITokenUsecase) Delete(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestAPITokenMiddleware(t *testing.T) {
	e := echo.New()
	e.Validator = nil
	mockUsecase := new(mockAPITokenUsecase)
	middlewareFunc := middleware.APITokenMiddleware(mockUsecase)

	t.Run("Bearerトークンが有効な場合は次のハンドラーが実行される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		user := &entity.User{ID: 1}
		token := &entity.APIToken{ID: 2}
		mockUsecase.On("Validate", mock.Anything, "valid").Return(user, token, nil).Once()

		handlerCalled := false
		handler := middlewareFunc(func(c *echo.Context) error {
			handlerCalled = true
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("トークンが指定されていない場合は401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middlewareFunc(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeMissingToken, apiErr.Code)
	})

	t.Run("クエリパラメータのみでトークンが指定されている場合も401", func(t *testing.T) {
		// SEC-005対応: クエリパラメータでのトークン受け渡しはセキュリティリスクがあるため無効化
		req := httptest.NewRequest(http.MethodGet, "/v1/songs?token=querytoken", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middlewareFunc(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
	})

	t.Run("検証に失敗した場合は401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer invalid")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockUsecase.On("Validate", mock.Anything, "invalid").Return(nil, nil, usecase.ErrInvalidAPIToken).Once()

		handler := middlewareFunc(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("内部エラーの場合は500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer error")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockUsecase.On("Validate", mock.Anything, "error").Return(nil, nil, errors.New("boom")).Once()

		handler := middlewareFunc(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockUsecase.AssertExpectations(t)
	})
}

func TestOptionalAPITokenMiddleware(t *testing.T) {
	e := echo.New()

	t.Run("トークンなしでも次のハンドラーを実行する", func(t *testing.T) {
		mockUsecase := new(mockAPITokenUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/songs/1/score-history/master", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware.OptionalAPITokenMiddleware(mockUsecase)(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockUsecase.AssertNotCalled(t, "Validate")
	})

	t.Run("有効なトークンならユーザーを設定する", func(t *testing.T) {
		mockUsecase := new(mockAPITokenUsecase)
		user := &entity.User{ID: 1}
		token := &entity.APIToken{ID: 2}
		mockUsecase.On("Validate", mock.Anything, "valid").Return(user, token, nil).Once()
		req := httptest.NewRequest(http.MethodGet, "/v1/songs/1/score-history/master", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware.OptionalAPITokenMiddleware(mockUsecase)(func(c *echo.Context) error {
			assert.Same(t, user, c.Get("userEntity"))
			return c.NoContent(http.StatusOK)
		})(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("不正なトークンは401を返す", func(t *testing.T) {
		mockUsecase := new(mockAPITokenUsecase)
		mockUsecase.On("Validate", mock.Anything, "invalid").Return(nil, nil, usecase.ErrInvalidAPIToken).Once()
		req := httptest.NewRequest(http.MethodGet, "/v1/songs/1/score-history/master", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer invalid")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware.OptionalAPITokenMiddleware(mockUsecase)(func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		mockUsecase.AssertExpectations(t)
	})
}

var _ usecase.APITokenUsecase = (*mockAPITokenUsecase)(nil)
