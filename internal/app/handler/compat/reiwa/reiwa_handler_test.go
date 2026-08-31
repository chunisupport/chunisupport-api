package reiwa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubReiwaMasterDataUsecase struct {
	versions []domainmasterdata.Version
}

func (s stubReiwaMasterDataUsecase) GetMasterData(context.Context) *usecase.MasterDataOutput {
	return nil
}

func (s stubReiwaMasterDataUsecase) GetVersions(context.Context) []domainmasterdata.Version {
	return s.versions
}

func (s stubReiwaMasterDataUsecase) GetHonorTypes(context.Context) []domainmasterdata.Item {
	return nil
}

func TestReiwaHandler_GetChunithmVersions(t *testing.T) {
	// Given
	handler := NewReiwaHandler(nil, stubReiwaMasterDataUsecase{
		versions: []domainmasterdata.Version{
			{ID: 1, Name: "CHUNITHM", ReleasedAt: time.Date(2015, 7, 16, 0, 0, 0, 0, time.UTC)},
		},
	}, nil)
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/reiwa/1/chunithm_versions.json", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handler.GetChunithmVersions(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[
		{
			"version": "CHUNITHM",
			"release": "2015-07-16"
		}
	]`, rec.Body.String())
}
