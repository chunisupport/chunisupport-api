package api_v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockV1MasterDataUsecase struct {
	mock.Mock
}

func (m *mockV1MasterDataUsecase) GetMasterData(ctx context.Context) *usecase.MasterDataOutput {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*usecase.MasterDataOutput)
}

func (m *mockV1MasterDataUsecase) GetVersions(ctx context.Context) []masterdata.Version {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]masterdata.Version)
}

func (m *mockV1MasterDataUsecase) GetHonorTypes(ctx context.Context) []masterdata.Item {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]masterdata.Item)
}

func TestV1VersionHandler_GetVersions(t *testing.T) {
	e := echo.New()
	usecaseMock := new(mockV1MasterDataUsecase)
	handler := NewV1VersionHandler(usecaseMock)

	releasedAt := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	usecaseMock.On("GetVersions", mock.Anything).Return([]masterdata.Version{
		{ID: 4, Name: "LUMINOUS", ReleasedAt: releasedAt},
	}).Once()

	req := httptest.NewRequest(http.MethodGet, "/v1/master/versions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetVersions(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response dto.VersionSummariesResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Versions, 1)
	assert.Equal(t, "LUMINOUS", response.Versions[0].Name)
	assert.Equal(t, "2026-02-05T00:00:00Z", response.Versions[0].ReleasedAt)
	usecaseMock.AssertExpectations(t)
}
