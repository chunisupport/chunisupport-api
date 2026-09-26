package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type songBatchJobUsecaseStub struct {
	startCalls int
	requester  entity.SongBatchJobRequester
	request    songbatch.RunRequest
	startErr   error
	jobs       []*entity.SongBatchJob
	getID      string
	getErr     error
}

func (s *songBatchJobUsecaseStub) StartFromAdmin(_ context.Context, requester entity.SongBatchJobRequester, req songbatch.RunRequest) (*entity.SongBatchJob, error) {
	s.startCalls++
	s.requester = requester
	s.request = req
	if s.startErr != nil {
		return nil, s.startErr
	}
	return entity.StartSongBatchJobFromAdmin(uuid.NewV4(), req, requester, songBatchHandlerStartedAt), nil
}

func (s *songBatchJobUsecaseStub) List(context.Context) ([]*entity.SongBatchJob, error) {
	return s.jobs, nil
}

func (s *songBatchJobUsecaseStub) Get(_ context.Context, id string) (*entity.SongBatchJob, error) {
	s.getID = id
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.jobs[0], nil
}

var songBatchHandlerStartedAt = time.Date(2026, 9, 26, 3, 0, 0, 0, time.UTC)

func newSongBatchHandlerContext(method, target, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), method, target, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 42, Username: username.MustNewUserName("adminuser")})
	return c, rec
}

func TestSongBatchHandler_Start_管理者の実行要求を受け付ける(t *testing.T) {
	// Given
	stub := &songBatchJobUsecaseStub{}
	handler := NewSongBatchHandler(stub)
	c, rec := newSongBatchHandlerContext(http.MethodPost, "/internal/admin/song-batch/jobs", `{"mode":"MAJOR_UPDATE","fill_missing_release_date":true}`)

	// When
	err := handler.Start(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.Equal(t, entity.SongBatchJobRequester{UserID: 42, Username: username.MustNewUserName("adminuser")}, stub.requester)
	assert.Equal(t, songbatch.RunRequest{Mode: songbatch.RunModeMajorUpdate, FillMissingReleaseDate: true}, stub.request)

	var response internaldto.SongBatchJobDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "MAJOR_UPDATE", response.Mode)
	assert.True(t, response.FillMissingReleaseDate)
	assert.Equal(t, "ADMIN", response.Trigger)
	require.NotNil(t, response.RequestedBy)
	assert.Equal(t, "adminuser", *response.RequestedBy)
	assert.Equal(t, "RUNNING", response.Status)
	assert.Nil(t, response.FinishedAt)
	_, parseErr := uuid.Parse(response.ID)
	assert.NoError(t, parseErr)
	assert.NotContains(t, rec.Body.String(), "user_id")
}

func TestSongBatchHandler_Start_エラー(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		startErr           error
		expectedStatus     int
		expectedCode       string
		expectedStartCalls int
	}{
		{
			name:           "未定義のモードは400",
			body:           `{"mode":"normal"}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   apierror.CodeInvalidSongBatchMode,
		},
		{
			name:           "未知のフィールドは400",
			body:           `{"mode":"NORMAL","command":"rm"}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   apierror.CodeBadRequest,
		},
		{
			name:               "別の楽曲バッチが実行中の場合は409",
			body:               `{"mode":"NORMAL"}`,
			startErr:           usecase.ErrSongBatchAlreadyRunning,
			expectedStatus:     http.StatusConflict,
			expectedCode:       apierror.CodeSongBatchAlreadyRunning,
			expectedStartCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &songBatchJobUsecaseStub{startErr: tt.startErr}
			handler := NewSongBatchHandler(stub)
			c, _ := newSongBatchHandlerContext(http.MethodPost, "/internal/admin/song-batch/jobs", tt.body)

			// When
			err := handler.Start(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.expectedStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.expectedCode, apiErr.Code)
			assert.Equal(t, tt.expectedStartCalls, stub.startCalls)
		})
	}
}

func TestSongBatchHandler_List(t *testing.T) {
	// Given
	finished := entity.StartSongBatchJobFromCLI(uuid.NewV4(), songbatch.NewRunRequest(false, false), songBatchHandlerStartedAt)
	require.NoError(t, finished.Fail(1, "required datasource official failed", songBatchHandlerStartedAt.Add(time.Minute)))
	handler := NewSongBatchHandler(&songBatchJobUsecaseStub{jobs: []*entity.SongBatchJob{finished}})
	c, rec := newSongBatchHandlerContext(http.MethodGet, "/internal/admin/song-batch/jobs", "")

	// When
	err := handler.List(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response internaldto.SongBatchJobListDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Jobs, 1)
	job := response.Jobs[0]
	assert.Equal(t, finished.ID().String(), job.ID)
	assert.Equal(t, "CLI", job.Trigger)
	assert.Nil(t, job.RequestedBy)
	assert.Equal(t, "FAILED", job.Status)
	assert.Equal(t, 1, job.WarningCount)
	require.NotNil(t, job.ErrorMessage)
	assert.Equal(t, "required datasource official failed", *job.ErrorMessage)
	require.NotNil(t, job.FinishedAt)
}

func TestSongBatchHandler_Get(t *testing.T) {
	job := entity.StartSongBatchJobFromCLI(uuid.NewV4(), songbatch.NewRunRequest(false, false), songBatchHandlerStartedAt)

	tests := []struct {
		name           string
		getErr         error
		expectedStatus int
		expectedCode   string
	}{
		{name: "存在するジョブは200", expectedStatus: http.StatusOK},
		{name: "存在しないジョブは404", getErr: repository.ErrSongBatchJobNotFound, expectedStatus: http.StatusNotFound, expectedCode: apierror.CodeSongBatchJobNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &songBatchJobUsecaseStub{jobs: []*entity.SongBatchJob{job}, getErr: tt.getErr}
			handler := NewSongBatchHandler(stub)
			c, rec := newSongBatchHandlerContext(http.MethodGet, "/internal/admin/song-batch/jobs/"+job.ID().String(), "")
			c.SetPathValues(echo.PathValues{{Name: "id", Value: job.ID().String()}})

			// When
			err := handler.Get(c)

			// Then
			assert.Equal(t, job.ID().String(), stub.getID)
			if tt.expectedCode != "" {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.expectedStatus, apiErr.HTTPStatus)
				assert.Equal(t, tt.expectedCode, apiErr.Code)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
