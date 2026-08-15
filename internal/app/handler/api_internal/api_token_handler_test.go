package api_internal_test

import (
	"bytes"
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
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAPITokenUsecase struct {
	mock.Mock
}

func (m *mockAPITokenUsecase) Generate(ctx context.Context, userID int, name string) (*usecase.GeneratedAPITokenOutput, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.GeneratedAPITokenOutput), args.Error(1)
}

func (m *mockAPITokenUsecase) List(ctx context.Context, userID int) ([]*usecase.APITokenOutput, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*usecase.APITokenOutput), args.Error(1)
}

func (m *mockAPITokenUsecase) Rename(ctx context.Context, userID int, id string, name string) (*usecase.APITokenOutput, error) {
	args := m.Called(ctx, userID, id, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.APITokenOutput), args.Error(1)
}

func (m *mockAPITokenUsecase) Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error) {
	args := m.Called(ctx, rawToken)
	if args.Get(0) == nil || args.Get(1) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*entity.User), args.Get(1).(*entity.APIToken), args.Error(2)
}

func (m *mockAPITokenUsecase) Delete(ctx context.Context, userID int, id string) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func newAPITokenTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = app.NewCustomValidator()
	return e
}

func TestAPITokenHandler_List(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)
	createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)

	req := httptest.NewRequest(http.MethodGet, "/internal/auth/api-tokens", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})
	mockUsecase.On("List", mock.Anything, 10).Return([]*usecase.APITokenOutput{{
		ID:        1,
		Name:      "既存のトークン",
		CreatedAt: createdAt,
	}}, nil).Once()

	err := h.List(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"tokens":[{"id":1,"name":"既存のトークン","token_prefix":null,"last_used_at":null,"created_at":"2026-04-16T12:34:56Z"}]}`, rec.Body.String())
	mockUsecase.AssertExpectations(t)
}

func TestAPITokenHandler_Generate(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)
	createdAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	prefix := "abcde"

	req := httptest.NewRequest(http.MethodPost, "/internal/auth/api-tokens", bytes.NewBufferString(`{"name":"Discord Bot"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})
	mockUsecase.On("Generate", mock.Anything, 10, "Discord Bot").Return(&usecase.GeneratedAPITokenOutput{
		Token: "abcde-secret",
		Metadata: &usecase.APITokenOutput{
			ID:          2,
			Name:        "Discord Bot",
			TokenPrefix: &prefix,
			CreatedAt:   createdAt,
		},
	}, nil).Once()

	err := h.Generate(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.JSONEq(t, `{"id":2,"name":"Discord Bot","token":"abcde-secret","token_prefix":"abcde","last_used_at":null,"created_at":"2026-07-22T12:00:00Z"}`, rec.Body.String())
	mockUsecase.AssertExpectations(t)
}

func TestAPITokenHandler_GenerateRejectsUnknownField(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)
	req := httptest.NewRequest(http.MethodPost, "/internal/auth/api-tokens", bytes.NewBufferString(`{"name":"CLI","unknown":true}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})

	err := h.Generate(c)

	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	mockUsecase.AssertNotCalled(t, "Generate")
}

func TestAPITokenHandler_Rename(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)
	createdAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

	req := httptest.NewRequest(http.MethodPatch, "/internal/auth/api-tokens/2", bytes.NewBufferString(`{"name":"Batch"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/auth/api-tokens/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "2"}})
	c.Set("userEntity", &entity.User{ID: 10})
	mockUsecase.On("Rename", mock.Anything, 10, "2", "Batch").Return(&usecase.APITokenOutput{ID: 2, Name: "Batch", CreatedAt: createdAt}, nil).Once()

	err := h.Rename(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"id":2,"name":"Batch","token_prefix":null,"last_used_at":null,"created_at":"2026-07-22T12:00:00Z"}`, rec.Body.String())
	mockUsecase.AssertExpectations(t)
}

func TestAPITokenHandler_Delete(t *testing.T) {
	e := newAPITokenTestEcho()
	mockUsecase := new(mockAPITokenUsecase)
	h := api_internal.NewAPITokenHandler(mockUsecase)
	req := httptest.NewRequest(http.MethodDelete, "/internal/auth/api-tokens/2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/auth/api-tokens/:id")
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "2"}})
	c.Set("userEntity", &entity.User{ID: 10})
	mockUsecase.On("Delete", mock.Anything, 10, "2").Return(nil).Once()

	err := h.Delete(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	mockUsecase.AssertExpectations(t)
}

func TestAPITokenHandler_RequiresAuthenticatedUser(t *testing.T) {
	e := newAPITokenTestEcho()
	h := api_internal.NewAPITokenHandler(new(mockAPITokenUsecase))
	req := httptest.NewRequest(http.MethodGet, "/internal/auth/api-tokens", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.List(c)

	var apiErr *apierror.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, apierror.CodeUnauthorized, apiErr.Code)
}

var _ usecase.APITokenUsecase = (*mockAPITokenUsecase)(nil)
