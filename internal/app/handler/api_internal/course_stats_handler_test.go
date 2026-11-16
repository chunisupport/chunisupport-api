package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCourseStatsUsecase struct {
	get func(context.Context, string) (*entity.CourseStats, error)
}

func (s *stubCourseStatsUsecase) GetByDisplayID(ctx context.Context, displayID string) (*entity.CourseStats, error) {
	return s.get(ctx, displayID)
}

type stubCourseStatsMasterProvider struct{}

func (stubCourseStatsMasterProvider) RatingBands() []*ratingband.RatingBand {
	return []*ratingband.RatingBand{{ID: 0, Label: "ALL"}, {ID: 22, Label: "17.0"}}
}

func newCourseStatsContext(displayID string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/courses/"+displayID+"/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: displayID}})
	return c, rec
}

func TestCourseStatsHandler_Get_レーティング帯ごとの統計を返す(t *testing.T) {
	// Given
	average := 2950000.0
	var gotDisplayID string
	h := NewCourseStatsHandler(&stubCourseStatsUsecase{get: func(_ context.Context, displayID string) (*entity.CourseStats, error) {
		gotDisplayID = displayID
		return &entity.CourseStats{CourseDisplayID: displayID, Stats: []*entity.CourseStatsByRatingBand{
			{
				RatingBandID: 0, PlayerCount: 3,
				Combo: entity.CourseComboStats{None: 1, FC: 1, AJ: 1}, Clear: entity.CourseClearStats{Failed: 1, Clear: 2},
				AverageScore: &average, MedianScore: &average,
			},
			{RatingBandID: 22, PlayerCount: 1, Combo: entity.CourseComboStats{None: 1}, Clear: entity.CourseClearStats{Failed: 1}},
		}}, nil
	}}, stubCourseStatsMasterProvider{})
	c, rec := newCourseStatsContext("0123456789abcdef")

	// When
	err := h.Get(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "0123456789abcdef", gotDisplayID)
	assert.JSONEq(t, `{"course_id":"0123456789abcdef","stats":[
		{"rating_band":"ALL","combo":{"none":1,"fc":1,"aj":1},"clear":{"failed":1,"clear":2},"average_score":2950000,"median_score":2950000,"player_count":3},
		{"rating_band":"17.0","combo":{"none":1,"fc":0,"aj":0},"clear":{"failed":1,"clear":0},"average_score":null,"median_score":null,"player_count":1}
	]}`, rec.Body.String())
}

func TestCourseStatsHandler_Get_エラー(t *testing.T) {
	tests := []struct {
		name string
		// Given
		displayID  string
		usecaseErr error
		// Then
		wantStatus int
		wantCode   string
	}{
		{name: "不正なコースIDはUsecaseへ渡さない", displayID: "invalid", wantStatus: http.StatusUnprocessableEntity, wantCode: apierror.CodeValidationFailed},
		{name: "コースなしは404", displayID: "0123456789abcdef", usecaseErr: repository.ErrCourseNotFound, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: usecaseErr がないケースでUsecaseが呼ばれるとnil関数呼び出しで失敗する。
			stub := &stubCourseStatsUsecase{}
			if tt.usecaseErr != nil {
				stub.get = func(context.Context, string) (*entity.CourseStats, error) { return nil, tt.usecaseErr }
			}
			c, _ := newCourseStatsContext(tt.displayID)

			// When
			err := NewCourseStatsHandler(stub, stubCourseStatsMasterProvider{}).Get(c)

			// Then
			apiErr, ok := err.(*apierror.APIError)
			require.True(t, ok)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, apiErr.Code)
			}
		})
	}
}
