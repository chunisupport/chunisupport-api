package api_internal_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func newAPITokenTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	return e
}

func TestAPITokenHandler_GetStatus(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)

	t.Run("認証済みユーザーのトークン状態を返す", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 10}
		c.Set("userEntity", user)
		createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)

		mockUsecase.On("GetStatus", mock.Anything, user.ID).Return(&entity.APIToken{
			ID:        1,
			UserID:    user.ID,
			CreatedAt: createdAt,
		}, nil).Once()

		err := h.GetStatus(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"has_token":true,"created_at":"2026-04-16T12:34:56Z"}`, rec.Body.String())
		mockUsecase.AssertExpectations(t)
	})

	t.Run("トークン未発行ならhas_token=falseを返す", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 10}
		c.Set("userEntity", user)

		mockUsecase.On("GetStatus", mock.Anything, user.ID).Return(nil, nil).Once()

		err := h.GetStatus(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"has_token":false,"created_at":null}`, rec.Body.String())
		mockUsecase.AssertExpectations(t)
	})

	t.Run("ユーザー情報が存在しない場合は401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.GetStatus(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
	})

	t.Run("状態取得に失敗した場合は500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 10}
		c.Set("userEntity", user)

		mockUsecase.On("GetStatus", mock.Anything, user.ID).Return(nil, errors.New("failed")).Once()

		err := h.GetStatus(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockUsecase.AssertExpectations(t)
	})
}

func TestAPITokenHandler_Generate(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)

	t.Run("認証済みユーザーに新しいトークンを返す", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 10}
		c.Set("userEntity", user)

		mockUsecase.On("Generate", mock.Anything, user.ID).Return("plain-token", nil).Once()

		err := h.Generate(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("ユーザー情報が存在しない場合は401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Generate(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
	})

	t.Run("トークン生成に失敗した場合は500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 10}
		c.Set("userEntity", user)

		mockUsecase.On("Generate", mock.Anything, user.ID).Return("", errors.New("failed")).Once()

		err := h.Generate(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockUsecase.AssertExpectations(t)
	})
}

var _ usecase.APITokenUsecase = (*mockAPITokenUsecase)(nil)

func TestAPITokenHandler_Delete(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)

	t.Run("認証済みユーザーのトークンを削除", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 42}
		c.Set("userEntity", user)

		mockUsecase.On("Delete", mock.Anything, user.ID).Return(nil).Once()

		err := h.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("ユーザー情報が存在しない場合は401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Delete(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
	})

	t.Run("削除に失敗した場合は500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/internal/auth/api-tokens", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &entity.User{ID: 42}
		c.Set("userEntity", user)

		mockUsecase.On("Delete", mock.Anything, user.ID).Return(errors.New("failed")).Once()

		err := h.Delete(c)
		apiErr, ok := err.(*apierror.APIError)
		assert.True(t, ok, "error should be *apierror.APIError")
		assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
		mockUsecase.AssertExpectations(t)
	})
}
