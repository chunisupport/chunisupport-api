package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRecordFilterUsecase struct {
	listFilterType string
	createCalled   bool
}

func (m *mockRecordFilterUsecase) List(ctx context.Context, userID int, filterType string) ([]*usecase.RecordFilterOutput, error) {
	m.listFilterType = filterType
	return []*usecase.RecordFilterOutput{
		{
			ID:            "11111111-1111-1111-1111-111111111111",
			Name:          "高難度",
			FilterType:    usecase.RecordFilterTypeStandard,
			SchemaVersion: 3,
			Filter:        []byte(`{"title":"","difficulties":["MASTER"]}`),
			CreatedAt:     "2026-06-15T12:00:00Z",
			UpdatedAt:     "2026-06-15T12:00:00Z",
		},
	}, nil
}

func (m *mockRecordFilterUsecase) Create(ctx context.Context, userID int, input *usecase.RecordFilterInput) (*usecase.RecordFilterOutput, error) {
	m.createCalled = true
	return &usecase.RecordFilterOutput{
		ID:            "11111111-1111-1111-1111-111111111111",
		Name:          input.Name,
		FilterType:    input.FilterType,
		SchemaVersion: input.SchemaVersion,
		Filter:        input.Filter,
		CreatedAt:     "2026-06-15T12:00:00Z",
		UpdatedAt:     "2026-06-15T12:00:00Z",
	}, nil
}

func (m *mockRecordFilterUsecase) Update(ctx context.Context, userID int, id string, input *usecase.RecordFilterInput) (*usecase.RecordFilterOutput, error) {
	return nil, nil
}

func (m *mockRecordFilterUsecase) Delete(ctx context.Context, userID int, id string) error {
	return nil
}

func TestRecordFilterHandlerList(t *testing.T) {
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/internal/me/record-filters?filter_type=standard", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})

	err := h.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "standard", uc.listFilterType)

	var response map[string][]map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response["filters"], 1)
	assert.Equal(t, "高難度", response["filters"][0]["name"])
	assert.Equal(t, "standard", response["filters"][0]["filter_type"])
	assert.Equal(t, float64(3), response["filters"][0]["schema_version"])
}

func TestRecordFilterHandlerCreateRejectsUnknownTopLevelKey(t *testing.T) {
	e := echo.New()
	uc := &mockRecordFilterUsecase{}
	h := NewRecordFilterHandler(uc)
	body := `{"name":"条件","filter_type":"standard","schema_version":3,"filter":{"title":""},"unknown":1}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/record-filters", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 10})

	err := h.Create(c)
	require.Error(t, err)
	apiErr := &apierror.APIError{}
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, apierror.CodeBadRequest, apiErr.Code)
	assert.False(t, uc.createCalled)
}
