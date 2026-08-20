package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type stubV1CourseUsecase struct {
	usecase.CourseUsecase
	get func(context.Context, string, bool) (*usecase.CourseOutput, error)
}

func (s *stubV1CourseUsecase) Get(ctx context.Context, displayID string, includeDeleted bool) (*usecase.CourseOutput, error) {
	return s.get(ctx, displayID, includeDeleted)
}

func TestV1CourseHandler_GetUsesIDPathParameter(t *testing.T) {
	// Given
	const id = "0123456789abcdef"
	var actualID string
	handler := NewV1CourseHandler(&stubV1CourseUsecase{
		get: func(ctx context.Context, displayID string, includeDeleted bool) (*usecase.CourseOutput, error) {
			actualID = displayID
			return &usecase.CourseOutput{DisplayID: displayID}, nil
		},
	})
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/v1/courses/"+id, nil), httptest.NewRecorder())
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

	// When
	err := handler.Get(c)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, id, actualID)
}
