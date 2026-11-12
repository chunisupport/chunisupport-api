package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
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
		{ID: 4, Name: "LUMINOUS", ShortName: "LMN", ReleasedAt: releasedAt},
	}).Once()

	req := httptest.NewRequest(http.MethodGet, "/v1/master/versions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetVersions(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	assert.JSONEq(t, `{"versions":[{"name":"LUMINOUS","short_name":"LMN","released_at":"2026-02-05"}]}`, rec.Body.String())
	usecaseMock.AssertExpectations(t)
}

func (m *mockV1MasterDataUsecase) GetPermissions(context.Context) []string {
	return nil
}

func (m *mockV1MasterDataUsecase) GetNameFolders(context.Context) []masterdata.NameFolder {
	return []masterdata.NameFolder{}
}

func (m *mockV1MasterDataUsecase) GetGenres(context.Context) []masterdata.Genre {
	return []masterdata.Genre{}
}
