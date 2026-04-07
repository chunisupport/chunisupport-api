package api_internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockMasterDataUsecase struct {
	mock.Mock
}

func (m *mockMasterDataUsecase) GetMasterData(ctx context.Context) *usecase.MasterDataOutput {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*usecase.MasterDataOutput)
}

func (m *mockMasterDataUsecase) GetVersions(ctx context.Context) []masterdata.Version {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]masterdata.Version)
}

func TestMasterDataHandler_GetVersions(t *testing.T) {
	e := newTestEcho()
	usecaseMock := new(mockMasterDataUsecase)
	handler := api_internal.NewMasterDataHandler(usecaseMock)

	releasedAt := time.Date(2025, 10, 30, 15, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	usecaseMock.On("GetVersions", mock.Anything).Return([]masterdata.Version{
		{ID: 3, Name: "VERSE", ReleasedAt: releasedAt},
	}).Once()

	req := httptest.NewRequest(http.MethodGet, "/internal/master/versions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetVersions(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response dto.VersionSummariesResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Versions, 1)
	assert.Equal(t, "VERSE", response.Versions[0].Name)
	assert.Equal(t, "2025-10-30T15:00:00+09:00", response.Versions[0].ReleasedAt)
	usecaseMock.AssertExpectations(t)
}

func TestMasterDataHandler_GetMasterData_UsesVersionDTOShape(t *testing.T) {
	e := newTestEcho()
	usecaseMock := new(mockMasterDataUsecase)
	handler := api_internal.NewMasterDataHandler(usecaseMock)

	releasedAt := time.Date(2015, 7, 16, 0, 0, 0, 0, time.UTC)
	usecaseMock.On("GetMasterData", mock.Anything).Return(&usecase.MasterDataOutput{
		Versions: []masterdata.Version{
			{ID: 1, Name: "CHUNITHM", ReleasedAt: releasedAt},
		},
		Genres:           []masterdata.Item{},
		Difficulties:     []masterdata.Item{},
		AccountTypes:     []masterdata.Item{},
		RatingBands:      nil,
		AchievementTypes: []masterdata.Item{},
		ClassEmblems:     []masterdata.Item{},
		ClassEmblemBases: []masterdata.Item{},
		ClearLamps:       []masterdata.Item{},
		ComboLamps:       []masterdata.Item{},
		FullChains:       []masterdata.Item{},
		Slots:            []masterdata.Item{},
		HonorTypes:       []masterdata.Item{},
	}).Once()

	req := httptest.NewRequest(http.MethodGet, "/internal/master", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetMasterData(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"versions\":[{\"id\":1,\"name\":\"CHUNITHM\",\"released_at\":\"2015-07-16T00:00:00Z\"}]")
	usecaseMock.AssertExpectations(t)
}
