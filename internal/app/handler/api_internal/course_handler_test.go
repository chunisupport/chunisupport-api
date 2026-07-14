package api_internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseUsecaseStub struct {
	get func(context.Context, string, bool) (*usecase.CourseOutput, error)
}

func (s *courseUsecaseStub) List(context.Context, bool) ([]*usecase.CourseOutput, error) {
	return nil, nil
}
func (s *courseUsecaseStub) Get(ctx context.Context, displayID string, includeDeleted bool) (*usecase.CourseOutput, error) {
	return s.get(ctx, displayID, includeDeleted)
}
func (s *courseUsecaseStub) Create(context.Context, usecase.CreateCourseInput) (*usecase.CourseOutput, error) {
	return nil, nil
}
func (s *courseUsecaseStub) Update(context.Context, string, usecase.UpdateCourseInput) (*usecase.CourseOutput, error) {
	return nil, nil
}
func (s *courseUsecaseStub) Delete(context.Context, string) error  { return nil }
func (s *courseUsecaseStub) Restore(context.Context, string) error { return nil }
func (s *courseUsecaseStub) GetUserRecords(context.Context, string, *entity.User, bool) (*usecase.CourseRecordResult, error) {
	return nil, nil
}
func (s *courseUsecaseStub) GetUserRecord(context.Context, string, *entity.User, string) (*usecase.CourseRecordOutput, error) {
	return nil, nil
}

func TestCourseHandler_Get_DisplayIDを渡してレスポンスへ含める(t *testing.T) {
	const displayID = "0123456789abcdef"
	handler := NewCourseHandler(&courseUsecaseStub{get: func(_ context.Context, value string, includeDeleted bool) (*usecase.CourseOutput, error) {
		assert.Equal(t, displayID, value)
		assert.False(t, includeDeleted)
		return &usecase.CourseOutput{DisplayID: displayID, Idx: "50020", Name: "通常コース", Class: "1"}, nil
	}})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/courses/"+displayID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "displayid", Value: displayID}})

	err := handler.Get(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response dto.CourseDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, displayID, response.DisplayID)
}

func TestCourseHandler_Get_不正なDisplayIDを拒否する(t *testing.T) {
	called := false
	handler := NewCourseHandler(&courseUsecaseStub{get: func(context.Context, string, bool) (*usecase.CourseOutput, error) {
		called = true
		return nil, nil
	}})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/courses/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "displayid", Value: "invalid"}})

	err := handler.Get(c)

	var apiErr *apierror.APIError
	if assert.ErrorAs(t, err, &apiErr) {
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	}
	assert.False(t, called)
}
