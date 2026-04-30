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
)

type mockAPITokenService struct {
	mock.Mock
}

func (m *mockAPITokenService) Generate(ctx context.Context, userID int, name string) (string, *entity.APIToken, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(1) == nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*entity.APIToken), args.Error(2)
}

func (m *mockAPITokenService) List(ctx context.Context, userID int) ([]*entity.APIToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.APIToken), args.Error(1)
}

func (m *mockAPITokenService) Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error) {
	args := m.Called(ctx, rawToken)
	if args.Get(0) == nil || args.Get(1) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*entity.User), args.Get(1).(*entity.APIToken), args.Error(2)
}

func (m *mockAPITokenService) Delete(ctx context.Context, userID int, tokenID int64) error {
	args := m.Called(ctx, userID, tokenID)
	return args.Error(0)
}

func (m *mockAPITokenService) DeleteAll(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestAPITokenMiddleware(t *testing.T) {
	e := echo.New()
	e.Validator = nil
	mockService := new(mockAPITokenService)
	middlewareFunc := middleware.APITokenMiddleware(mockService)

	t.Run("Bearerトークンが有効な場合は次のハンドラーが実行される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer valid")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		user := &entity.User{ID: 1}
		token := &entity.APIToken{ID: 2}
		mockService.On("Validate", mock.Anything, "valid").Return(user, token, nil).Once()

		handlerCalled := false
		handler := middlewareFunc(func(c echo.Context) error {
			handlerCalled = true
			return c.String(http.StatusOK, "ok")
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("トークンが指定されていない場合は401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middlewareFunc(func(c echo.Context) error {
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

		handler := middlewareFunc(func(c echo.Context) error {
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

		mockService.On("Validate", mock.Anything, "invalid").Return(nil, nil, usecase.ErrInvalidAPIToken).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusUnauthorized, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidToken, apiErr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("内部エラーの場合は500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/songs", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer error")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("Validate", mock.Anything, "error").Return(nil, nil, errors.New("boom")).Once()

		handler := middlewareFunc(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockService.AssertExpectations(t)
	})
}

var _ usecase.APITokenUsecase = (*mockAPITokenService)(nil)
