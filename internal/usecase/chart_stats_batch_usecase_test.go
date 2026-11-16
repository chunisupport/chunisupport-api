package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/chartstatsbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChartStatsBatchRepository struct {
	ratingBands      []*ratingband.RatingBand
	chartRecords     []chartstatsbatch.ChartRecord
	worldsendRecords []chartstatsbatch.ChartRecord
	eligiblePlayers  []chartstatsbatch.EligiblePlayer
	bestSlotRecords  []chartstatsbatch.BestSlotRecord
	bestSlotCharts   []int
	courseRecords    []chartstatsbatch.CourseRecord
	streamErr        error
	replaceErr       error
	replaceCalled    bool
	replaced         *chartstatsbatch.Snapshot
}

func (r *fakeChartStatsBatchRepository) FindRatingBands(context.Context) ([]*ratingband.RatingBand, error) {
	return r.ratingBands, nil
}

func (r *fakeChartStatsBatchRepository) StreamSource(_ context.Context, callbacks repository.ChartStatsBatchSourceCallbacks) error {
	if r.streamErr != nil {
		return r.streamErr
	}
	for _, record := range r.chartRecords {
		if err := callbacks.ChartRecord(record); err != nil {
			return err
		}
	}
	for _, record := range r.worldsendRecords {
		if err := callbacks.WorldsendChartRecord(record); err != nil {
			return err
		}
	}
	for _, player := range r.eligiblePlayers {
		if err := callbacks.EligiblePlayer(player); err != nil {
			return err
		}
	}
	for _, record := range r.bestSlotRecords {
		if err := callbacks.BestSlotRecord(record); err != nil {
			return err
		}
	}
	for _, chartID := range r.bestSlotCharts {
		if err := callbacks.BestSlotChart(chartID); err != nil {
			return err
		}
	}
	for _, record := range r.courseRecords {
		if err := callbacks.CourseRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeChartStatsBatchRepository) ReplaceAll(_ context.Context, snapshot *chartstatsbatch.Snapshot) error {
	r.replaceCalled = true
	if r.replaceErr != nil {
		return r.replaceErr
	}
	r.replaced = snapshot
	return nil
}

func TestChartStatsBatchUsecase_Execute_集計結果で統計テーブルを入れ替える(t *testing.T) {
	// Given
	repo := &fakeChartStatsBatchRepository{
		ratingBands: []*ratingband.RatingBand{{ID: 0}},
		chartRecords: []chartstatsbatch.ChartRecord{
			{ChartID: 1, BestAverageRating: 17.0, Score: 1000000, ClearLampID: 2, ComboLampID: 1},
			{ChartID: 1, BestAverageRating: 16.0, Score: 990000, ClearLampID: 2, ComboLampID: 1},
			{ChartID: 2, BestAverageRating: 16.0, Score: 990000, ClearLampID: 2, ComboLampID: 1},
		},
		worldsendRecords: []chartstatsbatch.ChartRecord{
			{ChartID: 7, BestAverageRating: 17.0, Score: 1000000, ClearLampID: 2, ComboLampID: 1},
		},
		eligiblePlayers: []chartstatsbatch.EligiblePlayer{{BestAverageRating: 17.0}, {BestAverageRating: 16.0}},
		bestSlotRecords: []chartstatsbatch.BestSlotRecord{{ChartID: 1, BestAverageRating: 17.0}},
		bestSlotCharts:  []int{1, 2, 3},
	}
	uc := NewChartStatsBatchUsecase(repo)

	// When
	result, err := uc.Execute(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, ChartStatsBatchResult{
		ChartRecordCount:          3,
		ChartStatsCount:           2,
		WorldsendChartRecordCount: 1,
		WorldsendChartStatsCount:  1,
		EligiblePlayerCount:       2,
		BestSlotRecordCount:       1,
		BestSlotStatsCount:        3,
	}, result)
	require.NotNil(t, repo.replaced)
	require.Len(t, repo.replaced.ChartStats, 2)
	assert.Equal(t, 2, repo.replaced.ChartStats[0].PlayerCount)
	require.Len(t, repo.replaced.WorldsendChartStats, 1)
	assert.Equal(t, 7, repo.replaced.WorldsendChartStats[0].ChartID)
	require.Len(t, repo.replaced.BestSlotStats, 3)
	assert.Equal(t, 1, repo.replaced.BestSlotStats[0].BestPlayerCount)
	assert.Equal(t, 2, repo.replaced.BestSlotStats[0].EligiblePlayerCount)
}

func TestChartStatsBatchUsecase_Execute_エラー(t *testing.T) {
	tests := []struct {
		name                  string
		repo                  *fakeChartStatsBatchRepository
		expectedReplaceCalled bool
	}{
		{
			name: "集計元を読み取れない場合は統計テーブルを入れ替えない",
			repo: &fakeChartStatsBatchRepository{streamErr: errors.New("query failed")},
		},
		{
			name:                  "入れ替えに失敗した場合はエラーを返す",
			repo:                  &fakeChartStatsBatchRepository{replaceErr: errors.New("insert failed")},
			expectedReplaceCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := NewChartStatsBatchUsecase(tt.repo)

			// When
			_, err := uc.Execute(context.Background())

			// Then
			assert.Error(t, err)
			assert.Equal(t, tt.expectedReplaceCalled, tt.repo.replaceCalled)
		})
	}
}

func TestChartStatsBatchUsecase_Execute_コース統計を同じスナップショットへ含める(t *testing.T) {
	// Given: 2コース分の記録。コース11は未クリアだけ。
	repo := &fakeChartStatsBatchRepository{
		ratingBands: []*ratingband.RatingBand{{ID: 0}},
		courseRecords: []chartstatsbatch.CourseRecord{
			{CourseID: 10, BestAverageRating: 17.0, Score: 3_000_000, IsClear: true, ComboLampID: 2},
			{CourseID: 10, BestAverageRating: 16.0, Score: 2_800_000, IsClear: true, ComboLampID: 1},
			{CourseID: 11, BestAverageRating: 16.0, Score: 1_000_000, IsClear: false, ComboLampID: 1},
		},
	}

	// When
	result, err := NewChartStatsBatchUsecase(repo).Execute(context.Background())

	// Then: 他の統計と同じ入れ替え単位に入り、件数も記録される。
	require.NoError(t, err)
	assert.Equal(t, ChartStatsBatchResult{CourseRecordCount: 3, CourseStatsCount: 2}, result)
	require.NotNil(t, repo.replaced)
	require.Len(t, repo.replaced.CourseStats, 2)
	assert.Equal(t, 10, repo.replaced.CourseStats[0].CourseID)
	assert.Equal(t, 2, repo.replaced.CourseStats[0].PlayerCount)
	require.NotNil(t, repo.replaced.CourseStats[0].AverageScore)
	assert.Equal(t, float64(2_900_000), *repo.replaced.CourseStats[0].AverageScore)
	assert.Equal(t, 11, repo.replaced.CourseStats[1].CourseID)
	assert.Nil(t, repo.replaced.CourseStats[1].AverageScore)
}
