package api_internal

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type honorTestValidator struct {
	validator *validator.Validate
}

func (v *honorTestValidator) Validate(i any) error {
	return v.validator.Struct(i)
}

type mockHonorUsecase struct {
	createErr    error
	deleteErr    error
	createCalled bool
	deleteCalled bool
}

func (m *mockHonorUsecase) ListHonors(context.Context) ([]*entity.Honor, error) {
	return []*entity.Honor{}, nil
}

func (m *mockHonorUsecase) GetHonor(context.Context, int) (*entity.Honor, error) {
	return &entity.Honor{ID: 1, Name: "称号A", TypeName: "gold"}, nil
}

func (m *mockHonorUsecase) CreateHonor(context.Context, usecase.HonorInput) (*entity.Honor, error) {
	m.createCalled = true
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &entity.Honor{ID: 1, Name: "称号A", TypeName: "gold"}, nil
}

func (m *mockHonorUsecase) UpdateHonor(context.Context, int, usecase.HonorInput) (*entity.Honor, error) {
	return &entity.Honor{ID: 1, Name: "称号A", TypeName: "gold"}, nil
}

func (m *mockHonorUsecase) DeleteHonor(context.Context, int) error {
	m.deleteCalled = true
	return m.deleteErr
}

func newHonorHandlerTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &honorTestValidator{validator: validator.New()}
	return e
}

func assertHonorHandlerAPIError(t *testing.T, err error, code string) {
	t.Helper()
	require.Error(t, err)

	apiErr := &apierror.APIError{}
	require.True(t, errors.As(err, &apiErr), "err type = %T", err)
	assert.Equal(t, code, apiErr.Code)
}

func TestHonorHandler_GetHonor_不正IDはValidationFailedを返す(t *testing.T) {
	// Given
	e := newHonorHandlerTestEcho()
	uc := &mockHonorUsecase{}
	handler := NewHonorHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/internal/honors/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// When
	err := handler.GetHonor(c)

	// Then
	assertHonorHandlerAPIError(t, err, apierror.CodeValidationFailed)
}

func TestHonorHandler_CreateHonor_重複時はConflictを返す(t *testing.T) {
	// Given
	e := newHonorHandlerTestEcho()
	uc := &mockHonorUsecase{createErr: repository.ErrHonorConflict}
	handler := NewHonorHandler(uc)
	body := `{"name":"称号A","type_name":"gold","image_url":"https://example.com/honor.png"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/honors", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handler.CreateHonor(c)

	// Then
	assertHonorHandlerAPIError(t, err, apierror.CodeConflict)
	assert.True(t, uc.createCalled)
}

func TestHonorHandler_CreateHonor_必須項目不足はUsecaseを呼ばずValidationFailedを返す(t *testing.T) {
	// Given
	e := newHonorHandlerTestEcho()
	uc := &mockHonorUsecase{}
	handler := NewHonorHandler(uc)
	body := `{"name":"","type_name":"gold"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/honors", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handler.CreateHonor(c)

	// Then
	assertHonorHandlerAPIError(t, err, apierror.CodeValidationFailed)
	assert.False(t, uc.createCalled)
}

func TestHonorHandler_DeleteHonor_参照中はConflictを返す(t *testing.T) {
	// Given
	e := newHonorHandlerTestEcho()
	uc := &mockHonorUsecase{deleteErr: repository.ErrHonorConflict}
	handler := NewHonorHandler(uc)
	req := httptest.NewRequest(http.MethodDelete, "/internal/honors/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	// When
	err := handler.DeleteHonor(c)

	// Then
	assertHonorHandlerAPIError(t, err, apierror.CodeConflict)
	assert.True(t, uc.deleteCalled)
}
