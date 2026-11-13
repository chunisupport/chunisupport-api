package api_internal

import (
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
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chartStatsBatchJobUsecaseStub struct {
	startCalls int
	requester  entity.ChartStatsBatchJobRequester
	startErr   error
	jobs       []*entity.ChartStatsBatchJob
	getID      string
	getErr     error
}

func (s *chartStatsBatchJobUsecaseStub) StartFromAdmin(_ context.Context, requester entity.ChartStatsBatchJobRequester) (*entity.ChartStatsBatchJob, error) {
	s.startCalls++
	s.requester = requester
	if s.startErr != nil {
		return nil, s.startErr
	}
	return entity.StartChartStatsBatchJobFromAdmin(uuid.NewV4(), requester, chartStatsBatchHandlerStartedAt), nil
}

func (s *chartStatsBatchJobUsecaseStub) List(context.Context) ([]*entity.ChartStatsBatchJob, error) {
	return s.jobs, nil
}

func (s *chartStatsBatchJobUsecaseStub) Get(_ context.Context, id string) (*entity.ChartStatsBatchJob, error) {
	s.getID = id
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.jobs[0], nil
}

var chartStatsBatchHandlerStartedAt = time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)

func newChartStatsBatchHandlerContext(t *testing.T, method, target string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), method, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 42, Username: usernametest.New(t, "adminuser")})
	return c, rec
}

func TestChartStatsBatchHandler_Start_管理者の実行要求を受け付ける(t *testing.T) {
	// Given
	stub := &chartStatsBatchJobUsecaseStub{}
	handler := NewChartStatsBatchHandler(stub)
	c, rec := newChartStatsBatchHandlerContext(t, http.MethodPost, "/internal/admin/chart-stats-batch/jobs")

	// When
	err := handler.Start(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.Equal(t, entity.ChartStatsBatchJobRequester{UserID: 42, Username: usernametest.New(t, "adminuser")}, stub.requester)

	var response internaldto.ChartStatsBatchJobDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "ADMIN", response.Trigger)
	require.NotNil(t, response.RequestedBy)
	assert.Equal(t, "adminuser", *response.RequestedBy)
	assert.Equal(t, "RUNNING", response.Status)
	assert.Nil(t, response.FinishedAt)
	assert.Nil(t, response.ErrorMessage)
	_, parseErr := uuid.Parse(response.ID)
	assert.NoError(t, parseErr)
	assert.NotContains(t, rec.Body.String(), "user_id")
}

func TestChartStatsBatchHandler_Start_エラー(t *testing.T) {
	tests := []struct {
		name           string
		startErr       error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "別の譜面統計バッチが実行中の場合は409",
			startErr:       usecase.ErrChartStatsBatchAlreadyRunning,
			expectedStatus: http.StatusConflict,
			expectedCode:   apierror.CodeChartStatsBatchAlreadyRunning,
		},
		{
			name:           "サーバー停止処理中は503",
			startErr:       usecase.ErrChartStatsBatchUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   apierror.CodeServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &chartStatsBatchJobUsecaseStub{startErr: tt.startErr}
			handler := NewChartStatsBatchHandler(stub)
			c, _ := newChartStatsBatchHandlerContext(t, http.MethodPost, "/internal/admin/chart-stats-batch/jobs")

			// When
			err := handler.Start(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.expectedStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.expectedCode, apiErr.Code)
		})
	}
}

func TestChartStatsBatchHandler_List(t *testing.T) {
	// Given
	finished := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchHandlerStartedAt)
	require.NoError(t, finished.Fail("replace chart stats: deadlock", chartStatsBatchHandlerStartedAt.Add(time.Minute)))
	handler := NewChartStatsBatchHandler(&chartStatsBatchJobUsecaseStub{jobs: []*entity.ChartStatsBatchJob{finished}})
	c, rec := newChartStatsBatchHandlerContext(t, http.MethodGet, "/internal/admin/chart-stats-batch/jobs")

	// When
	err := handler.List(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	var response internaldto.ChartStatsBatchJobListDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Jobs, 1)
	job := response.Jobs[0]
	assert.Equal(t, finished.ID().String(), job.ID)
	assert.Equal(t, "CLI", job.Trigger)
	assert.Nil(t, job.RequestedBy)
	assert.Equal(t, "FAILED", job.Status)
	require.NotNil(t, job.ErrorMessage)
	assert.Equal(t, "replace chart stats: deadlock", *job.ErrorMessage)
	require.NotNil(t, job.FinishedAt)
}

func TestChartStatsBatchHandler_Get(t *testing.T) {
	// ジョブIDは固定値にし、レスポンス全体を文字列リテラルと比較できるようにする。
	jobID := uuid.MustParse("0199a1b2-c3d4-4e5f-8a9b-0c1d2e3f4a5b")
	job := entity.StartChartStatsBatchJobFromCLI(jobID, chartStatsBatchHandlerStartedAt)

	tests := []struct {
		name           string
		getErr         error
		expectedStatus int
		expectedCode   string
	}{
		{name: "存在するジョブは200", expectedStatus: http.StatusOK},
		{name: "存在しないジョブは404", getErr: repository.ErrChartStatsBatchJobNotFound, expectedStatus: http.StatusNotFound, expectedCode: apierror.CodeChartStatsBatchJobNotFound},
		{name: "不正なジョブIDは400", getErr: usecase.ErrInvalidChartStatsBatchJobID, expectedStatus: http.StatusBadRequest, expectedCode: apierror.CodeInvalidChartStatsBatchJobID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stub := &chartStatsBatchJobUsecaseStub{jobs: []*entity.ChartStatsBatchJob{job}, getErr: tt.getErr}
			handler := NewChartStatsBatchHandler(stub)
			c, rec := newChartStatsBatchHandlerContext(t, http.MethodGet, "/internal/admin/chart-stats-batch/jobs/"+job.ID().String())
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
			// ジョブ取得はキャッシュ禁止が契約のため、ヘッダーも確かめる。
			assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
			// 本文全体を固定値で確かめる。DTO経由のデコードではキー名の変更や null の有無を見逃すため。
			// 内部の数値IDは含めない契約であり、期待JSONにも含めていない。
			assert.JSONEq(t, `{"id":"0199a1b2-c3d4-4e5f-8a9b-0c1d2e3f4a5b","trigger":"CLI","requested_by":null,"status":"RUNNING","started_at":"2026-09-28T03:00:00Z","finished_at":null,"error_message":null}`, rec.Body.String())
			assert.NotContains(t, rec.Body.String(), "user_id")
		})
	}
}
