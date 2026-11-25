package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeScorePercentileProvider struct {
	snapshot      *scorepercentile.Snapshot
	snapshotErr   error
	refreshErr    error
	refreshCalled int
}

func (p *fakeScorePercentileProvider) Snapshot(context.Context) (*scorepercentile.Snapshot, error) {
	if p.snapshotErr != nil {
		return nil, p.snapshotErr
	}
	return p.snapshot, nil
}

func (p *fakeScorePercentileProvider) Refresh(context.Context) error {
	p.refreshCalled++
	return p.refreshErr
}

func TestChartStatsBatchJobUsecase_成功時のみ上位パーセントキャッシュを再構築する(t *testing.T) {
	tests := []struct {
		name              string
		runnerErr         error
		wantRefreshCalled int
	}{
		{
			name:              "バッチ成功時はRefreshを呼ぶ",
			runnerErr:         nil,
			wantRefreshCalled: 1,
		},
		{
			name:              "バッチ失敗時はRefreshを呼ばない",
			runnerErr:         errors.New("replace failed"),
			wantRefreshCalled: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			lockProvider := &fakeSongBatchLockProvider{acquired: true}
			repo := newFakeChartStatsBatchJobRepository()
			runner := &fakeChartStatsBatchRunner{err: tt.runnerErr}
			provider := &fakeScorePercentileProvider{}
			uc := NewChartStatsBatchJobUsecase(context.Background(), lockProvider, repo, runner, provider)
			uc.now = func() time.Time { return chartStatsBatchJobNow }

			// When
			_, err := uc.RunFromCLI(context.Background())

			// Then
			if tt.runnerErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantRefreshCalled, provider.refreshCalled)
		})
	}
}

func TestApplyTopPercents(t *testing.T) {
	// Given
	snapshot := scorepercentile.NewSnapshot(
		1,
		[]scorepercentile.Entry{{TargetID: 10, Thresholds: scorepercentile.BuildThresholds([]int{1010000, 1000000, 990000, 980000})}},
		[]scorepercentile.Entry{{TargetID: 20, Thresholds: scorepercentile.BuildThresholds([]int{1005000, 950000})}},
		[]scorepercentile.Entry{{TargetID: 30, Thresholds: scorepercentile.BuildThresholds([]int{3030000, 2000000})}},
	)
	now := time.Now()
	s1, err := score.NewScore(1010000)
	require.NoError(t, err)
	s2, err := score.NewScore(950000)
	require.NoError(t, err)
	cs, err := coursescore.New(2000000)
	require.NoError(t, err)

	stdOutputs := []*PlayerRecordOutput{
		{PlayerRecord: &entity.PlayerRecord{ChartID: 10, Score: s1, UpdatedAt: now}, IsPlayed: true},
		{PlayerRecord: &entity.PlayerRecord{ChartID: 10}, IsPlayed: false},
		{PlayerRecord: &entity.PlayerRecord{ChartID: 99, Score: s1, UpdatedAt: now}, IsPlayed: true},
	}
	weOutputs := []*WorldsendRecordOutput{
		{PlayerWorldsendRecord: &entity.PlayerWorldsendRecord{WorldsendChartID: 20, Score: s2, ClearLampID: 1, UpdatedAt: now}, IsPlayed: true},
		{PlayerWorldsendRecord: &entity.PlayerWorldsendRecord{WorldsendChartID: 20}, IsPlayed: false},
	}

	// When
	applyPlayerRecordTopPercents(stdOutputs, snapshot)
	applyWorldsendRecordTopPercents(weOutputs, snapshot)

	// Then
	require.NotNil(t, stdOutputs[0].TopPercent)
	assert.Equal(t, 25.00, *stdOutputs[0].TopPercent)
	assert.Nil(t, stdOutputs[1].TopPercent)
	assert.Nil(t, stdOutputs[2].TopPercent)

	require.NotNil(t, weOutputs[0].TopPercent)
	assert.Equal(t, 100.00, *weOutputs[0].TopPercent)
	assert.Nil(t, weOutputs[1].TopPercent)

	courseRec := &entity.PlayerCourseRecord{CourseID: 30, Score: cs, IsClear: false, UpdatedAt: now}
	courseTop := snapshot.LookupCourse(courseRecordCourseID(courseRec), courseRec.Score.Uint32())
	require.NotNil(t, courseTop)
	assert.Equal(t, 100.00, *courseTop)
}

func TestLoadScorePercentiles_エラー時はnilを返して処理を継続する(t *testing.T) {
	provider := &fakeScorePercentileProvider{snapshotErr: errors.New("db down")}
	assert.Nil(t, loadScorePercentiles(context.Background(), provider))
	assert.Nil(t, loadScorePercentiles(context.Background(), nil))
}

func TestUserUsecase_GetUserUpdatedAt_統計世代を返す(t *testing.T) {
	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		playerUpdatedAt *time.Time
		provider        *fakeScorePercentileProvider
		wantRevision    uint64
	}{
		{
			name:            "公開中スナップショットの世代を返す",
			playerUpdatedAt: &now,
			provider:        &fakeScorePercentileProvider{snapshot: scorepercentile.NewSnapshot(42, nil, nil, nil)},
			wantRevision:    42,
		},
		{
			name:            "プレイヤー未連携でも公開中スナップショットの世代を返す",
			playerUpdatedAt: nil,
			provider:        &fakeScorePercentileProvider{snapshot: scorepercentile.NewSnapshot(7, nil, nil, nil)},
			wantRevision:    7,
		},
		{
			name:            "スナップショット取得失敗時は0を返して処理を継続する",
			playerUpdatedAt: &now,
			provider:        &fakeScorePercentileProvider{snapshotErr: errors.New("db down")},
			wantRevision:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := NewUserUsecase(
				nil,
				&stubUserRepository{},
				&stubPlayerRepository{},
				&stubPlayerRecordRepository{},
				nil, nil, nil, nil, nil, nil, nil, nil,
				&stubUserUpdatedAtQueryService{
					result: &repository.UserUpdatedAtQueryResult{
						User:            &entity.User{ID: 1},
						PlayerUpdatedAt: tt.playerUpdatedAt,
					},
				},
				nil, nil, nil, nil,
				tt.provider,
			)

			// When
			got, err := uc.GetUserUpdatedAt(context.Background(), "tester", nil)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.playerUpdatedAt, got.UpdatedAt)
			assert.Equal(t, tt.wantRevision, got.StatsRevision)
		})
	}
}
